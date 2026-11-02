"""REQ-7792 CLI-process checks using the released SDK and a local HTTP peer.

Run GATE_CLI_BINARY=/absolute/path/to/gate-cli python3 tests/e2e/bot_cli.py.
Synthetic HMAC keys and resource-bound OAuth credentials exercise the CLI→SDK
wire, not a real issuer, gateway authorization decision or trading backend.
"""
import contextlib
import hashlib
import hmac
import http.server
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import unittest
from urllib.parse import parse_qs, unquote_plus, urlsplit


ID = '9223372036854775807'
TYPES = ('spot_grid', 'margin_grid', 'infinite_grid', 'futures_grid',
         'spot_martingale', 'contract_martingale', 'smart_position', 'futures_combo')
RECOMMEND_TYPES = tuple(t for t in TYPES if t not in ('contract_martingale', 'futures_combo'))


class BotCLIProcessTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.binary = os.environ['GATE_CLI_BINARY']
        if not Path(cls.binary).is_file():
            raise RuntimeError('Build the CLI before running E2E')
        cls.requests, cls.mode = [], 'success'

        class Peer(http.server.BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_GET(self):
                self.connection.settimeout(2)
                length = int(self.headers.get('Content-Length', '0'))
                if not 0 <= length <= 65536:
                    self.send_error(413)
                    return
                raw = self.rfile.read(length)
                url = urlsplit(self.path)
                # Check the signature against exact transmitted bytes, including query escaping.
                if self.headers.get('KEY'):
                    canonical = '\n'.join((self.command, url.path, unquote_plus(url.query),
                        hashlib.sha512(raw).hexdigest(), self.headers.get('Timestamp', '')))
                    expected = hmac.new(b'fixture-secret', canonical.encode(), hashlib.sha512).hexdigest()
                    if not hmac.compare_digest(expected, self.headers.get('SIGN', '')):
                        self.send_error(401, 'invalid synthetic HMAC')
                        return
                cls.requests.append((self.command, self.path, dict(self.headers), raw.decode()))
                data = {'strategy_id': ID, 'status': 'starting',
                        'operation_status': 'pending', 'recovery_required': False, 'result_message': 'stop not confirmed'}
                query = parse_qs(url.query)
                if url.path.endswith('/recommend'):
                    data = {'scene': 'refresh', 'recommendations': [
                        {'strategy_type': query.get('strategy_type', ['smart_position'])[0],
                         'recommendation_id': 'smart_position|USDT|123|a'}]}
                elif url.path.endswith('/running'):
                    data = {'items': [{'strategy_id': ID, 'strategy_type': kind,
                        'status': 'running', 'assets': ['BTC', 'ETH'],
                        'markets': ['BTC_USDT', 'ETH_USDT']} for kind in TYPES],
                        'page': 1, 'page_size': 20, 'total': 100}
                elif url.path.endswith('/detail'):
                    data.update(strategy_type=query.get('strategy_type', ['futures_combo'])[0],
                        assets=['BTC', 'ETH'], markets=['BTC_USDT', 'ETH_USDT'],
                        positions=[{'asset': 'BTC', 'amount': '0.000000000000000001'}],
                        positions_status='partial', configuration={'invest_amount': '100.000000000000000001'},
                        stop_progress={'trading_finished': False, 'funds_returned': False, 'account_released': False})
                payload = {'code': 200, 'message': 'accepted', 'trace_id': 'fixture-trace', 'data': data}
                status = 200
                if cls.mode != 'success':
                    payload.update(code=400, label='RECOVERY_REQUIRED', message='verify')
                if cls.mode == 'missing_data':
                    payload.pop('data')
                elif cls.mode == 'incomplete':
                    payload.update(code=200, data={})
                elif cls.mode == 'forbidden':
                    status = 403
                    payload.update(label='INSUFFICIENT_SCOPE', message='base required')
                elif cls.mode == 'unavailable':
                    status = 503
                body = b'not-json' if cls.mode == 'malformed' else json.dumps(payload).encode()
                self.send_response(status)
                self.send_header('Content-Type', 'application/json')
                self.send_header('X-Gate-Trace-Id', 'fixture-trace')
                self.send_header('Content-Length', str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            do_POST = do_GET

        cls.peer = http.server.HTTPServer(('127.0.0.1', 0), Peer)
        cls.thread = threading.Thread(target=cls.peer.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = f'http://127.0.0.1:{cls.peer.server_port}'

    @classmethod
    def tearDownClass(cls):
        cls.peer.shutdown()
        cls.peer.server_close()
        cls.thread.join(3)
        if cls.thread.is_alive():
            raise RuntimeError('HTTP fixture did not stop')

    @contextlib.contextmanager
    def environment(self, auth):
        with tempfile.TemporaryDirectory(prefix='bot-cli-e2e-') as home:
            # Ignore inherited Gate flags/limits/credentials so this test cannot contact a real account.
            env = {k: v for k, v in os.environ.items() if not k.startswith('GATE_')}
            env.update(HOME=home, GATE_BASE_URL=self.base)
            if auth == 'key':
                env.update(GATE_API_KEY='fixture-key', GATE_API_SECRET='fixture-secret')
            elif auth == 'oauth':
                folder = Path(home) / '.gate-cli'
                folder.mkdir(mode=0o700)
                config = folder / 'config.yaml'
                config.write_text(json.dumps({'oauth2': {'token': {'client_id': 'fixture-client',
                    'access_token': 'fixture-at', 'token_type': 'Bearer', 'resource': self.base + '/api/v4',
                    'expires_at': '2099-01-01T00:00:00Z', 'scopes': 'base'}}}))
                config.chmod(0o600)
            yield env

    def invoke(self, env, args, mode='success', expect_error=False, sends=1):
        type(self).mode = mode
        before = len(self.requests)
        result = subprocess.run([self.binary, '--format', 'json', 'cex', 'bot', *args],
            env=env, capture_output=True, text=True, timeout=8)
        self.assertEqual(len(self.requests) - before, sends, result.stderr)
        self.assertEqual(result.returncode != 0, expect_error, result.stderr)
        self.assertEqual(result.stdout if expect_error else result.stderr, '')
        payload = json.loads(result.stderr if expect_error else result.stdout)
        return payload.get('error', payload) if expect_error else payload

    def assert_supplied_fields(self, supplied, wire):
        """SDK may add defaults but must never drop or change explicit user intent."""
        if isinstance(supplied, dict):
            self.assertIsInstance(wire, dict)
            for key, value in supplied.items():
                self.assertIn(key, wire)
                self.assert_supplied_fields(value, wire[key])
        elif isinstance(supplied, list):
            self.assertIsInstance(wire, list)
            self.assertEqual(len(supplied), len(wire))
            for expected, actual in zip(supplied, wire):
                self.assert_supplied_fields(expected, actual)
        else:
            self.assertEqual(type(supplied), type(wire))
            self.assertEqual(supplied, wire)

    def commands(self):
        grid = {'money': '100', 'low_price': '60000', 'high_price': '70000', 'grid_num': 10, 'price_type': 0}
        martingale = {'invest_amount': '100', 'price_deviation': '0.02', 'max_orders': 5, 'take_profit_ratio': '0.01'}
        commands = []
        for kind in TYPES[:6]:
            params = dict(grid if kind.endswith('grid') else martingale)
            if kind == 'infinite_grid':
                params = {'money': '100', 'price_floor': '60000', 'profit_per_grid': '0.005'}
            if kind in ('margin_grid', 'futures_grid', 'contract_martingale'):
                params['leverage'] = '3'
            if kind == 'contract_martingale':
                params.update(direction='buy', stop_loss_price='10.000000000000000001', stop_loss_price_type='mark')
            group, leaf = (('martingale', 'contract' if kind == 'contract_martingale' else 'spot')
                if kind.endswith('martingale') else ('grid', kind.split('_')[0]))
            commands.append(([group, leaf, '--json', json.dumps({'strategy_type': kind,
                'market': 'BTC_USDT', 'create_params': params})], f'/api/v4/bot/{kind.replace("_", "-")}/create'))
        for name in ('smart-position', 'futures-combo'):
            params = {'begin_money_amount': '100.000000000000000001', 'begin_money_currency': 'USDT', 'period': '1h'}
            if name == 'smart-position':
                params.update(asset_info=[{'asset': a, 'rate': '0.5'} for a in ('BTC', 'ETH')], close_flag=0)
            else:
                params.update(asset_info=[{'market': a, 'side': 'buy', 'rate': '0.5'} for a in ('BTC_USDT', 'ETH_USDT')], leverage=2)
            commands.append(([name, 'create', '--json', json.dumps({'strategy_type': name.replace('-', '_'),
                'create_params': params})], f'/api/v4/bot/{name}/create'))
        commands.extend([
            (['stop', '--strategy-id', ID, '--strategy-type', 'smart_position', '--stop-mode', 'limit',
              '--stop-options', '{"price":"10.000000000000000001","expiration":60}'], '/api/v4/bot/portfolio/stop'),
            (['recommend', '--scene', 'refresh', '--exclude-recommendation-id', 'smart_position|USDT|123|a'], '/api/v4/bot/strategy/recommend'),
            (['running'], '/api/v4/bot/portfolio/running'),
            (['detail', '--strategy-id', ID, '--strategy-type', 'futures_combo'], '/api/v4/bot/portfolio/detail')])
        return commands

    def test_all_twelve_methods_hmac_oauth_and_business_errors(self):
        for auth in ('key', 'oauth'):
            with self.environment(auth) as env:
                for mode in ('success', 'partial', 'missing_data'):
                    for args, path in self.commands():
                        with self.subTest(auth=auth, mode=mode, path=path):
                            payload = self.invoke(env, args, mode, mode != 'success')
                            method, actual, headers, raw = self.requests[-1]
                            self.assertEqual(urlsplit(actual).path, path)
                            headers = {k.lower(): v for k, v in headers.items()}
                            if auth == 'oauth':
                                self.assertEqual(headers.get('authorization'), 'Bearer fixture-at')
                                for key in ('key', 'sign', 'timestamp'):
                                    self.assertNotIn(key, headers)
                            else:
                                for key in ('key', 'sign', 'timestamp'):
                                    self.assertTrue(headers.get(key))
                            if mode != 'success':
                                self.assertEqual(payload['label'], 'RECOVERY_REQUIRED')
                                self.assertEqual(payload['trace_id'], 'fixture-trace')
                                if mode == 'partial' and method == 'POST':
                                    self.assertEqual(payload['data']['strategy_id'], ID)
                                    self.assertIs(payload['data']['recovery_required'], False)
                            elif method == 'POST':
                                self.assertEqual(payload['data']['strategy_id'], ID)
                                if '--json' in args:
                                    self.assert_supplied_fields(json.loads(args[args.index('--json') + 1]), json.loads(raw))
                                if args[0] == 'stop':
                                    self.assert_supplied_fields({'strategy_id': ID, 'strategy_type': 'smart_position',
                                        'stop_mode': 'limit', 'stop_options': {'price': '10.000000000000000001', 'expiration': 60}}, json.loads(raw))
                            elif args[0] == 'recommend':
                                query = parse_qs(urlsplit(actual).query)
                                self.assertEqual(query['scene'], ['refresh'])
                                self.assertEqual(query['exclude_recommendation_id'], ['smart_position|USDT|123|a'])
                            elif args[0] == 'running':
                                self.assertEqual(payload['data']['total'], 100)
                                self.assertEqual([i['strategy_type'] for i in payload['data']['items']], list(TYPES))
                            elif args[0] == 'detail':
                                query = parse_qs(urlsplit(actual).query)
                                self.assertEqual(query['strategy_id'], [ID])
                                self.assertEqual(query['strategy_type'], ['futures_combo'])
                                self.assertEqual(payload['data']['positions'][0]['amount'], '0.000000000000000001')
                                self.assertEqual(payload['data']['configuration']['invest_amount'], '100.000000000000000001')
                                self.assertEqual(payload['data']['stop_progress'], {'trading_finished': False, 'funds_returned': False, 'account_released': False})

    def test_gateway_errors_and_unknown_write_do_not_retry(self):
        for auth in ('key', 'oauth'):
            with self.environment(auth) as env:
                for mode in ('forbidden', 'unavailable', 'incomplete', 'malformed'):
                    with self.subTest(auth=auth, mode=mode):
                        args = self.commands()[8][0]
                        payload = self.invoke(env, args, mode, True)
                        if mode == 'forbidden':
                            self.assertEqual(payload['label'], 'INSUFFICIENT_SCOPE')
                            self.assertEqual(payload['data']['result_message'], 'stop not confirmed')
                        if mode == 'incomplete':
                            self.assertIn('unknown', payload['message'])
                            self.assertIn('query', payload['suggested_next_action'])

    def test_six_recommendations_and_eight_detail_types(self):
        with self.environment('oauth') as env:
            for kind in RECOMMEND_TYPES:
                with self.subTest(recommend=kind):
                    payload = self.invoke(env, ['recommend', '--strategy-type', kind])
                    self.assertEqual(payload['data']['recommendations'][0]['strategy_type'], kind)
                    self.assertEqual(parse_qs(urlsplit(self.requests[-1][1]).query)['strategy_type'], [kind])
            for kind in TYPES:
                with self.subTest(detail=kind):
                    payload = self.invoke(env, ['detail', '--strategy-id', ID, '--strategy-type', kind])
                    self.assertEqual(payload['data']['strategy_type'], kind)

    def test_invalid_intent_and_missing_auth_never_send(self):
        bad = [['smart-position', 'create', '--json', raw] for raw in (
            'null', '[]', '{} {}', '{"extra":true}',
            '{"strategy_type":"smart_position","create_params":{"period":"1h","threshold":"0.1"}}')]
        bad += [
            ['stop', '--strategy-id', ID, '--strategy-type', 'smart_position'],
            ['stop', '--strategy-id', ID, '--strategy-type', 'smart_position', '--stop-mode', 'limit'],
            ['stop', '--strategy-id', ID, '--strategy-type', 'futures_combo', '--stop-mode', 'market'],
            ['martingale', 'contract', '--json', '{"create_params":{"stop_loss_price":"10"}}']]
        with self.environment('oauth') as env:
            for args in bad:
                with self.subTest(invalid=args):
                    self.invoke(env, args, expect_error=True, sends=0)
        with self.environment('none') as env:
            for args, _ in self.commands():
                with self.subTest(no_auth=args[:2]):
                    self.invoke(env, args, expect_error=True, sends=0)


if __name__ == '__main__':
    unittest.main()
