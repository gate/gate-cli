"""CLI -> released Go SDK -> local HTTP peer regression tests.

Run with the existing tests/e2e convention:
GATE_CLI_BINARY=/absolute/path/to/gate-cli python3 -m unittest discover -s tests/e2e -p '*_cli.py'
Only the remote API and preissued OAuth credential are mocked. The real CLI process,
argument parsing, config loading, transport, SDK, serialization and exit codes run.
"""
import contextlib
import hashlib
import hmac
import http.server
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import threading
import unittest
from urllib.parse import parse_qs, unquote_plus, urlsplit


class CEXCLIProcessTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.binary = os.environ['GATE_CLI_BINARY']
        if not Path(cls.binary).is_file():
            raise RuntimeError('Build the CLI before running E2E')
        cls.cases = json.loads((Path(__file__).resolve().parents[2] /
                               'testdata/cex-capability-gaps.json').read_text())
        if len(cls.cases) != 21:
            raise RuntimeError('Expected all 21 requested REST operations')
        cls.requests, cls.mode, cls.response = [], 'success', {}

        class Peer(http.server.BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def handle_request(self):
                self.connection.settimeout(2)
                length = int(self.headers.get('Content-Length', '0'))
                if not 0 <= length <= 65536:
                    self.send_error(413)
                    return
                raw = self.rfile.read(length)
                url = urlsplit(self.path)
                record = (self.command, url.path, parse_qs(url.query),
                          json.loads(raw) if raw else None, dict(self.headers))
                cls.requests.append(record)
                # Synthetic signing is verified against the actual transmitted bytes.
                if self.headers.get('KEY'):
                    canonical = '\n'.join((self.command, url.path, unquote_plus(url.query),
                        hashlib.sha512(raw).hexdigest(), self.headers.get('Timestamp', '')))
                    expected = hmac.new(b'fixture-secret', canonical.encode(), hashlib.sha512).hexdigest()
                    if not hmac.compare_digest(expected, self.headers.get('SIGN', '')):
                        self.send_error(401, 'invalid synthetic HMAC')
                        return
                if cls.mode == 'disconnect':
                    self.connection.shutdown(socket.SHUT_RDWR)
                    self.connection.close()
                    return
                status, payload = 200, cls.response
                if cls.mode in ('forbidden', 'unavailable'):
                    status = 403 if cls.mode == 'forbidden' else 503
                    payload = {'label': 'INSUFFICIENT_SCOPE', 'message': 'fixture rejected'}
                if cls.mode == 'optional_rejected':
                    # The CLI must preserve server eligibility errors instead of choosing a fallback.
                    status, payload = 400, {'label': 'FIXTURE_REJECTED', 'message': 'server rule rejected'}
                if cls.mode == 'trigger_ioc_only':
                    # Model the service rule independently of the CLI defaults.
                    if record[3]['put']['time_in_force'] == 'ioc':
                        status = 201
                    else:
                        status, payload = 400, {'label': 'INVALID_PARAM_VALUE',
                                                'message': 'put.time_in_force must be ioc'}
                if cls.mode == 'chase_wrapped':
                    payload = {'code': 0, 'message': 'ok', 'data': payload}
                elif cls.mode == 'chase_rejected':
                    payload = {'code': -1, 'message': 'Invalid request'}
                elif cls.mode == 'chase_incomplete':
                    payload = {'code': 0, 'message': 'ok', 'data': {}}
                body = b'not-json' if cls.mode == 'malformed' else (b'null' if cls.mode == 'null' else json.dumps(payload).encode())
                self.send_response(status)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            do_GET = do_POST = do_PATCH = do_DELETE = handle_request

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
            raise RuntimeError('Local HTTP peer did not stop')

    @contextlib.contextmanager
    def environment(self, auth):
        with tempfile.TemporaryDirectory(prefix='cex-cli-e2e-') as home:
            # Removing inherited Gate settings guarantees test account isolation.
            env = {k: v for k, v in os.environ.items() if not k.startswith('GATE_')}
            env.update(HOME=home, GATE_BASE_URL=self.base)
            if auth == 'hmac':
                env.update(GATE_API_KEY='fixture-key', GATE_API_SECRET='fixture-secret')
            elif auth == 'oauth':
                folder = Path(home) / '.gate-cli'
                folder.mkdir(mode=0o700)
                config = folder / 'config.yaml'
                config.write_text(json.dumps({'oauth2': {'token': {
                    'client_id': 'fixture-client', 'access_token': 'fixture-at',
                    'token_type': 'Bearer', 'resource': self.base + '/api/v4',
                    'expires_at': '2099-01-01T00:00:00Z', 'scopes': 'base'}}}))
                config.chmod(0o600)
            yield env

    def invoke(self, env, args, response=None, mode='success', error=False, sends=1):
        type(self).mode, type(self).response = mode, {} if response is None else response
        before = len(self.requests)
        result = subprocess.run([self.binary, '--format', 'json', 'cex', *args],
                                env=env, capture_output=True, text=True, timeout=8)
        self.assertEqual(len(self.requests) - before, sends, result.stderr)
        self.assertEqual(result.returncode != 0, error, result.stderr)
        self.assertEqual(result.stdout if error else result.stderr, '')
        payload = json.loads(result.stderr if error else result.stdout)
        self.assertIsNotNone(payload)
        return payload.get('error', payload) if error else payload

    def test_spot_batch_business_outcomes(self):
        # Keep the SDK array on stdout even on failure; only the HTTP peer is fake.
        order = {'currency_pair': 'BTC_USDT', 'type': 'limit', 'side': 'buy',
                 'amount': '0.0002', 'price': '10000'}
        commands = [('batch-create', [order, order], '/api/v4/spot/batch_orders'),
                    ('batch-amend', [{'currency_pair': 'BTC_USDT', 'order_id': 'one', 'price': '10000'}] * 2,
                     '/api/v4/spot/amend_batch_orders'),
                    ('batch-cancel', [{'currency_pair': 'BTC_USDT', 'id': 'one'}] * 2,
                     '/api/v4/spot/cancel_batch_orders')]
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for action, orders, path in commands:
                    success = {'id': '9223372036854775807', 'succeeded': True}
                    failure = {'label': 'INVALID_PARAM_VALUE', 'message': 'size too small'}
                    for name, reply, label, counts in [
                        ('success', [success, success], None, None),
                        ('partial', [success, failure], 'BATCH_ORDER_FAILED', '1 succeeded, 1 failed'),
                        ('failed', [failure, failure], 'BATCH_ORDER_FAILED', '0 succeeded, 2 failed'),
                        ('false-only', [{'succeeded': False}] * 2, 'BATCH_ORDER_FAILED', '0 succeeded, 2 failed'),
                        ('inconsistent', [dict(success, label='REJECTED'), success], 'BATCH_ORDER_FAILED', '1 succeeded, 1 failed'),
                        ('missing', [success], 'INVALID_RESPONSE', None),
                        ('extra', [success] * 3, 'INVALID_RESPONSE', None),
                        ('empty', [], 'INVALID_RESPONSE', None),
                        ('null', None, 'INVALID_RESPONSE', None),
                    ]:
                        with self.subTest(auth=auth, action=action, outcome=name):
                            type(self).mode, type(self).response = 'success', reply
                            before = len(self.requests)
                            result = subprocess.run([self.binary, '--format', 'json', 'cex', 'spot', 'order',
                                action, '--orders-json', json.dumps(orders)], env=env,
                                capture_output=True, text=True, timeout=8)
                            self.assertEqual(len(self.requests) - before, 1, result.stderr)
                            self.assertEqual(self.requests[-1][0:2], ('POST', path))
                            # SDK omitempty removes false flags; preserve its decoded result shape.
                            expected = None if reply is None else [
                                {k: v for k, v in row.items() if v is not False} for row in reply]
                            if expected is not None and action != 'batch-cancel':
                                # BatchOrder has value-typed protection fields, serialized as empty objects.
                                expected = [dict(row, stop_profit={}, stop_loss={}) for row in expected]
                            self.assertEqual(json.loads(result.stdout), expected)
                            if label is None:
                                self.assertEqual(result.returncode, 0, result.stderr)
                                self.assertEqual(result.stderr, '')
                            else:
                                self.assertNotEqual(result.returncode, 0)
                                error = json.loads(result.stderr)['error']
                                self.assertEqual(error['label'], label)
                                self.assertEqual(error['status'], 200)
                                self.assertFalse(error['retryable'])
                                self.assertIn('do not retry the entire batch', error['suggested_next_action'])
                                if counts:
                                    self.assertIn(counts, error['message'])
                for action, orders, _ in commands:
                    for mode in ('forbidden', 'unavailable', 'malformed', 'disconnect'):
                        with self.subTest(auth=auth, action=action, transport=mode):
                            self.invoke(env, ['spot', 'order', action, '--orders-json', json.dumps(orders)],
                                        mode=mode, error=True)
                    # Existing human formats must retain the SDK data and a visible failure summary.
                    if auth == 'hmac':
                        for format_name in ('pretty', 'table'):
                            with self.subTest(action=action, format=format_name):
                                type(self).mode, type(self).response = 'success', [success, failure]
                                before = len(self.requests)
                                result = subprocess.run([self.binary, '--format', format_name, 'cex', 'spot',
                                    'order', action, '--orders-json', json.dumps(orders)], env=env,
                                    capture_output=True, text=True, timeout=8)
                                self.assertNotEqual(result.returncode, 0)
                                self.assertEqual(len(self.requests) - before, 1)
                                self.assertIn('9223372036854775807', result.stdout)
                                self.assertIn('INVALID_PARAM_VALUE', result.stdout)
                                self.assertIn('1 succeeded, 1 failed', result.stderr)
                    for raw in ('[]', 'null', '{', '{}'):
                        with self.subTest(auth=auth, action=action, invalid=raw):
                            self.invoke(env, ['spot', 'order', action, '--orders-json', raw],
                                        error=True, sends=0)

    def test_spot_trigger_time_in_force(self):
        # The complete CLI/SDK path must put the default on the wire, not just in help.
        args = ['spot', 'price-trigger', 'create', '--market', 'BTC_USDT',
                '--trigger-price', '60000', '--side', 'buy', '--price', '59000', '--amount', '0.001']
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for flags, expected in [([], 'ioc'), (['--time-in-force', 'gtc'], 'gtc'),
                                        (['--time-in-force', 'ioc'], 'ioc')]:
                    with self.subTest(auth=auth, flags=flags):
                        self.invoke(env, args + flags, response={'id': 123})
                        method, path, _, body, _ = self.requests[-1]
                        self.assertEqual((method, path), ('POST', '/api/v4/spot/price_orders'))
                        self.assertEqual(body['put']['time_in_force'], expected)
                        self.assertEqual(body['put']['price'], '59000')
                        self.assertEqual(body['put']['amount'], '0.001')
                for value in ('', 'GTC', 'fok', 'poc', 'bad'):
                    with self.subTest(auth=auth, invalid=value):
                        self.invoke(env, args + ['--time-in-force', value], error=True, sends=0)
                for mode in ('forbidden', 'unavailable', 'malformed', 'disconnect'):
                    with self.subTest(auth=auth, mode=mode):
                        self.invoke(env, args, mode=mode, error=True)

    def test_spot_trigger_ioc_only_service(self):
        # Use the actual CLI/SDK wire; only the API response is simulated.
        args = ['spot', 'price-trigger', 'create', '--market', 'BTC_USDT',
                '--trigger-price', '60000', '--side', 'buy', '--price', '59000', '--amount', '0.001']
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for flags in ([], ['--time-in-force', 'ioc']):
                    with self.subTest(auth=auth, flags=flags):
                        result = self.invoke(env, args + flags, mode='trigger_ioc_only', response={'id': 123})
                        self.assertEqual(result['id'], 123)
                        self.assertEqual(self.requests[-1][3]['put']['time_in_force'], 'ioc')
                # Explicit gtc is never silently rewritten or retried as ioc.
                error = self.invoke(env, args + ['--time-in-force', 'gtc'],
                                    mode='trigger_ioc_only', error=True)
                self.assertEqual(error['status'], 400)
                self.assertEqual(error['label'], 'INVALID_PARAM_VALUE')
                self.assertEqual(self.requests[-1][3]['put']['time_in_force'], 'gtc')

    def test_all_21_sdk_operations(self):
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for case in self.cases:
                    with self.subTest(auth=auth, operation=case['name']):
                        payload = self.invoke(env, case['args'], response=case['response'])
                        method, path, query, body, headers = self.requests[-1]
                        self.assertEqual((method, path), (case['method'], case['path']))
                        self.assertEqual(query, {k: [v] for k, v in case['query'].items()})
                        self.assertEqual(body, case['body'])
                        if case['private']:
                            if auth == 'hmac':
                                self.assertEqual(headers.get('Key'), 'fixture-key')
                                self.assertNotIn('Authorization', headers)
                            else:
                                self.assertEqual(headers.get('Authorization'), 'Bearer fixture-at')
                                self.assertNotIn('Key', headers)
                        else:
                            self.assertNotIn('Authorization', headers)
                            self.assertNotIn('Key', headers)
                        # Large IDs and decimal strings must survive SDK decoding intact.
                        if case['name'] in ('pov-create', 'pov-get', 'pov-cancel', 'chase-create'):
                            self.assertEqual(payload['id'], '9223372036854775807')

    def test_authentication_and_api_failures(self):
        with self.environment('none') as env:
            for case in self.cases:
                with self.subTest(operation=case['name'], auth='none'):
                    self.invoke(env, case['args'], response=case['response'],
                                error=case['private'], sends=0 if case['private'] else 1)
        with self.environment('oauth') as env:
            for case in self.cases:
                for mode in ('forbidden', 'unavailable', 'malformed'):
                    with self.subTest(operation=case['name'], mode=mode):
                        self.invoke(env, case['args'], mode=mode, error=True)
            for case in self.cases:
                if case['method'] == 'GET':
                    continue
                with self.subTest(operation=case['name'], mode='disconnect'):
                    err = self.invoke(env, case['args'], mode='disconnect', error=True)
                    self.assertIn('reconcile', json.dumps(err).lower())

    def test_chase_live_protocol_envelopes(self):
        # These wire shapes were observed on the actual test service; no live writes here.
        with self.environment('hmac') as env:
            for case in self.cases:
                if case['args'][:2] != ['futures', 'chase']:
                    continue
                with self.subTest(operation=case['name']):
                    self.invoke(env, case['args'], response=case['response'], mode='chase_wrapped')
                    error = self.invoke(env, case['args'], mode='chase_rejected', error=True)
                    self.assertEqual(error['code'], -1)
                    self.assertEqual(error['status'], 200)
                    self.assertEqual(error['message'], 'Invalid request')
                    self.invoke(env, case['args'], mode='chase_incomplete', error=True)
            payload = self.invoke(env, ['futures', 'chase', 'list'],
                                  response={'orders': []}, mode='chase_wrapped')
            self.assertEqual(payload, {'orders': []})
            self.assertEqual(self.requests[-1][2]['is_finished'], ['false'])

    def test_margin_and_protection_parameters(self):
        with self.environment('oauth') as env:
            for account in ('spot', 'margin', 'unified', 'cross_margin'):
                args = ['spot', 'order', 'buy', '--pair', 'BTC_USDT', '--amount',
                        '0.000000000000000001', '--price', '100', '--account', account]
                if account != 'spot':
                    args += ['--auto-borrow']
                if account in ('unified', 'cross_margin'):
                    args += ['--auto-repay']
                args += ['--stop-profit-json', '{"trigger_price":"101","order_price":"102"}',
                         '--stop-loss-json', '{"trigger_price":"99","order_price":"98"}']
                self.invoke(env, args, response={'id': '123'})
                body = self.requests[-1][3]
                self.assertEqual(body['account'], account)
                self.assertEqual(body['amount'], '0.000000000000000001')
                self.assertEqual(body.get('auto_borrow', False), account != 'spot')
                self.assertEqual(body.get('auto_repay', False), account in ('unified', 'cross_margin'))
                self.assertEqual(body['stop_profit'], {'trigger_price': '101', 'order_price': '102'})
                self.assertEqual(body['stop_loss'], {'trigger_price': '99', 'order_price': '98'})
                commands = [
                    (['spot', 'order', 'get', '--pair', 'BTC_USDT', '--id', '123'], {}),
                    (['spot', 'order', 'list', '--pair', 'BTC_USDT'], []),
                    (['spot', 'order', 'cancel', '--pair', 'BTC_USDT', '--id', '123'], {}),
                    (['spot', 'order', 'cancel', '--pair', 'BTC_USDT', '--all'], []),
                    (['spot', 'order', 'my-trades', '--pair', 'BTC_USDT'], []),
                    (['spot', 'order', 'all-open'], []),
                ]
                for command, response in commands:
                    with self.subTest(account=account, command=command):
                        self.invoke(env, command + ['--account', account], response=response)
                        self.assertEqual(self.requests[-1][2]['account'], [account])
                for protection in ([], ['--stop-profit-json', '{}'],
                                   ['--stop-loss-json', '{}'], ['--stop-profit-json', 'null']):
                    self.invoke(env, ['spot', 'order', 'amend', '--pair', 'BTC_USDT',
                                     '--id', '123', '--price', '100', '--account', account] + protection)
                    body = self.requests[-1][3]
                    self.assertEqual(body['account'], account)
                    self.assertEqual(body['stop_profit'], {} if protection == ['--stop-profit-json', '{}'] else None)
                    self.assertEqual(body['stop_loss'], {} if protection == ['--stop-loss-json', '{}'] else None)
            # Existing invocations continue to let the server choose the default account.
            self.invoke(env, ['spot', 'order', 'sell', '--pair', 'BTC_USDT', '--amount', '1'])
            self.assertNotIn('account', self.requests[-1][3])

    def test_leverage_partial_failure(self):
        with self.environment('oauth') as env:
            error = self.invoke(env, ['unified', 'config', 'leverage-set-all', '--leverage', '3'],
                                response=[{'currency': 'BTC', 'reason': 'limit exceeded'}], error=True)
            self.assertIn('BTC', json.dumps(error))
            self.assertIn('already', json.dumps(error))
            self.invoke(env, ['unified', 'config', 'leverage-set-all', '--leverage', '3'], mode='null', error=True)

    def test_delivery_trigger_rule_sdk_fields(self):
        # Keep parsing, authentication and SDK encoding real; only the API peer is fake.
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for rule, size, price in (('1', '3', '19.5'), ('2', '-3', '19.5'), ('2', '3', None)):
                    with self.subTest(auth=auth, rule=rule, size=size, price=price):
                        args = ['delivery', 'price-trigger', 'create', '--contract', 'SYNTH',
                                '--size', size, '--trigger-price', '20.25', '--trigger-rule', rule]
                        if price is not None:
                            args += ['--order-price', price]
                        result = self.invoke(env, args, response={'id': 123})
                        self.assertEqual(result['id'], 123)
                        request = self.requests[-1]
                        self.assertEqual(request[:2], ('POST', '/api/v4/delivery/usdt/price_orders'))
                        if auth == 'hmac':
                            self.assertEqual(request[4]['Key'], 'fixture-key')
                        else:
                            self.assertEqual(request[4]['Authorization'], 'Bearer fixture-at')
                        self.assertEqual(request[3], {'initial': {'contract': 'SYNTH', 'size': int(size),
                                                                 'price': price or ''},
                                                      'trigger': {'price': '20.25', 'rule': int(rule)}})
                # Table mode must still display the SDK's returned order ID.
                before = len(self.requests)
                result = subprocess.run([self.binary, '--format', 'table', 'cex', *args], env=env,
                                        capture_output=True, text=True, timeout=8)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stderr, '')
                self.assertIn('123', result.stdout)
                self.assertEqual(len(self.requests), before + 1)

    def test_delivery_trigger_rule_failures_do_not_replay(self):
        base = ['delivery', 'price-trigger', 'create', '--contract', 'SYNTH', '--size', '3',
                '--trigger-price', '20.25']
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for rule in ('0', '3', '-2147483648', '2147483647'):
                    with self.subTest(auth=auth, rejected_rule=rule):
                        error = self.invoke(env, base + ['--trigger-rule', rule],
                                            mode='optional_rejected', error=True)
                        self.assertEqual(error['label'], 'FIXTURE_REJECTED')
                        self.assertIn('server rule rejected', error['message'])
                        self.assertEqual(self.requests[-1][3]['trigger']['rule'], int(rule))
                for mode in ('forbidden', 'unavailable', 'malformed', 'disconnect'):
                    with self.subTest(auth=auth, mode=mode):
                        self.invoke(env, base + ['--trigger-rule', '1'], mode=mode, error=True)
                        self.assertEqual(self.requests[-1][3]['trigger']['rule'], 1)
        with self.environment('none') as env:
            self.invoke(env, base + ['--trigger-rule', '1'], error=True, sends=0)

    def test_delivery_trigger_rule_parsing_and_help(self):
        base = ['delivery', 'price-trigger', 'create', '--contract', 'SYNTH', '--size', '3',
                '--trigger-price', '20.25']
        with self.environment('hmac') as env:
            # Required-flag diagnostics append the existing hint; type errors emit duplicate JSON.
            # Also check the old --contract requirement so this is not hidden by the new flag.
            parse_cases = [(base, 'trigger-rule', True),
                           (base[:3] + base[5:] + ['--trigger-rule', '1'], 'contract', True)]
            parse_cases += [(base + ['--trigger-rule', rule], 'trigger-rule', False)
                            for rule in ('bad', '1.5', '2147483648', '-2147483649')]
            for args, field, required in parse_cases:
                with self.subTest(parse_args=args):
                    before = len(self.requests)
                    result = subprocess.run([self.binary, '--format', 'json', 'cex', *args], env=env,
                                            capture_output=True, text=True, timeout=8)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, '')
                    lines = result.stderr.splitlines()
                    if required:
                        # Do not silently discard arbitrary non-JSON diagnostics.
                        self.assertIn(len(lines), (2, 3))
                        self.assertEqual(lines[1], 'Hint: check the leaf command --help once, then fix flags')
                        if len(lines) == 3:
                            # Agent builds additionally emit the existing prefixed diagnostic JSON.
                            self.assertTrue(lines[2].startswith('gate_cli_diagnostic='))
                            diagnostic = json.loads(lines[2].split('=', 1)[1])
                            self.assertEqual(diagnostic['error_type'], 'INVALID_ARGS')
                            self.assertIn(field, diagnostic['message'])
                            self.assertTrue(diagnostic['retryable'])
                        envelopes = [json.loads(lines[0])['error']]
                    else:
                        envelopes = [json.loads(line)['error'] for line in lines]
                    self.assertTrue(envelopes)
                    for error in envelopes:
                        self.assertEqual(error['error_type'], 'INVALID_ARGS')
                        self.assertIn(field, error['message'])
                    self.assertEqual(len(self.requests), before)
            before = len(self.requests)
            result = subprocess.run([self.binary, 'cex', *base, '--help'], env=env,
                                    capture_output=True, text=True, timeout=8)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, '')
            self.assertIn('--trigger-rule', result.stdout)
            self.assertIn('1=>=', result.stdout)
            self.assertIn('2=<=', result.stdout)
            self.assertIn('Trigger condition (required)', result.stdout)
            self.assertEqual(len(self.requests), before)

    def test_order_optional_sdk_fields(self):
        # Only the peer is mocked; both real authentication paths sign/serialize the selected fields.
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for size, flags, extra in (
                    ('1', [], {}), ('1', ['--mmp=false'], {}),
                    ('1', ['--mmp'], {'mmp': True}),
                    ('0', ['--close', '--mmp'], {'close': True, 'mmp': True}),
                    ('-1', ['--reduce-only', '--mmp'], {'reduce_only': True, 'mmp': True}),
                    ('1', ['--reduce-only', '--mmp'], {'reduce_only': True, 'mmp': True}),
                ):
                    with self.subTest(auth=auth, size=size, flags=flags):
                        result = self.invoke(env, ['options', 'order', 'create', '--contract', 'SYNTH',
                                                  '--size', size, '--price', '0', '--tif', 'ioc', *flags],
                                             response={'id': 123, 'is_mmp': bool(extra.get('mmp'))})
                        request = self.requests[-1]
                        self.assertEqual(request[:2], ('POST', '/api/v4/options/orders'))
                        self.assertEqual(request[3], dict(contract='SYNTH', size=int(size), price='0', tif='ioc', **extra))
                        self.assertEqual(result['id'], 123)
                        self.assertEqual(result.get('is_mmp', False), bool(extra.get('mmp')))
                for leverage in (None, '0', '100', '-1', '2147483647', '-2147483648'):
                    with self.subTest(auth=auth, leverage=leverage):
                        flags = [] if leverage is None else ['--leverage', leverage]
                        result = self.invoke(env, ['tradfi', 'order', 'create', '--symbol', 'SYNTH', '--side', '2',
                                                  '--volume', '0.01', '--tp', '200', '--sl', '50', *flags],
                                             response={'timestamp': 1, 'data': {'id': 'fixture-task'}})
                        request = self.requests[-1]
                        expected = dict(symbol='SYNTH', side=2, volume='0.01', price='0', price_type='market',
                                        price_tp='200', price_sl='50')
                        if leverage not in (None, '0'):
                            expected['leverage'] = int(leverage)
                        self.assertEqual(request[:2], ('POST', '/api/v4/tradfi/orders'))
                        self.assertEqual(request[3], expected)
                        self.assertEqual(result['data']['id'], 'fixture-task')

    def test_order_optional_failures_do_not_replay(self):
        commands = [
            ['options', 'order', 'create', '--contract', 'SYNTH', '--size', '1', '--mmp'],
            ['tradfi', 'order', 'create', '--symbol', 'SYNTH', '--side', '2', '--volume', '0.01', '--leverage', '-1'],
        ]
        for auth in ('hmac', 'oauth'):
            with self.environment(auth) as env:
                for args in commands:
                    for mode in ('optional_rejected', 'forbidden', 'unavailable', 'malformed', 'disconnect'):
                        with self.subTest(auth=auth, args=args, mode=mode):
                            error = self.invoke(env, args, mode=mode, error=True)
                            if mode == 'optional_rejected':
                                self.assertEqual(error['label'], 'FIXTURE_REJECTED')
                                self.assertIn('server rule rejected', error['message'])
                            body = self.requests[-1][3]
                            if args[0] == 'options':
                                self.assertTrue(body['mmp'])
                            else:
                                self.assertEqual(body['leverage'], -1)
            with self.environment('none') as env:
                for args in commands:
                    with self.subTest(auth='none', args=args):
                        self.invoke(env, args, error=True, sends=0)

    def test_order_optional_parsing_and_help(self):
        # Parse/type boundaries stop before HTTP; trading eligibility remains with the server.
        with self.environment('hmac') as env:
            option = ['options', 'order', 'create', '--contract', 'SYNTH', '--size', '1']
            tradfi = ['tradfi', 'order', 'create', '--symbol', 'SYNTH', '--side', '2', '--volume', '0.01']
            # The existing flag hook and final renderer can both emit an error envelope.
            # Keep their diagnostics intact; this regression checks rejection and zero writes.
            parse_cases = [(option + ['--mmp=invalid'], '--mmp'),
                           (option + ['--reduce-only=invalid'], '--reduce-only')]
            parse_cases += [(tradfi + ['--leverage', value], '--leverage')
                            for value in ('bad', '1.5', '2147483648', '-2147483649')]
            for args, flag in parse_cases:
                with self.subTest(parse_args=args):
                    before = len(self.requests)
                    result = subprocess.run([self.binary, '--format', 'json', 'cex', *args],
                                            env=env, capture_output=True, text=True, timeout=8)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, '')
                    envelopes = [json.loads(line)['error'] for line in result.stderr.splitlines()]
                    self.assertTrue(envelopes)
                    for error in envelopes:
                        self.assertEqual(error['error_type'], 'INVALID_ARGS')
                        self.assertIn(flag, error['message'])
                    self.assertEqual(len(self.requests), before)
            self.invoke(env, option + ['--close', '--mmp'], error=True, sends=0)
            for args, field in ((option, '--mmp'), (tradfi, '--leverage')):
                before = len(self.requests)
                result = subprocess.run([self.binary, 'cex', *args, '--help'], env=env,
                                        capture_output=True, text=True, timeout=8)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stderr, '')
                self.assertIn(field, result.stdout)
                if field == '--leverage':
                    self.assertIn('0 is omitted by the SDK', result.stdout)
                self.assertEqual(len(self.requests), before)

    def test_options_close_parameters(self):
        with self.environment('hmac') as env:
            for size, extra in [('0', ['--close']), ('-9223372036854775808', ['--reduce-only']),
                                ('9223372036854775807', ['--reduce-only']), ('1', [])]:
                self.invoke(env, ['options', 'order', 'create', '--contract', 'BTC_USDT-20261225-60000-C',
                                 '--price', '0', '--tif', 'ioc', '--size', size] + extra)
                body = self.requests[-1][3]
                self.assertEqual(body['size'], int(size))
                self.assertEqual(body.get('close', False), '--close' in extra)
                self.assertEqual(body.get('reduce_only', False), '--reduce-only' in extra)
            self.invoke(env, ['options', 'position', 'close', '--underlying', 'BTC_USDT'], response=[])
            self.assertEqual(self.requests[-1][0:2], ('GET', '/api/v4/options/position_close'))

    def test_invalid_intent_never_calls_api(self):
        commands = [
            ['spot', 'order', 'buy', '--pair', 'BTC_USDT', '--quote', '1', '--account', 'invalid'],
            ['spot', 'order', 'buy', '--pair', 'BTC_USDT', '--quote', '1', '--auto-borrow'],
            ['spot', 'order', 'buy', '--pair', 'BTC_USDT', '--quote', '1', '--account', 'margin', '--auto-repay'],
            ['spot', 'order', 'buy', '--pair', 'BTC_USDT', '--quote', '1', '--stop-profit-json', '{"trigger_price":"101"}'],
            ['spot', 'order', 'buy', '--pair', 'BTC_USDT', '--amount', '1', '--price', '100', '--stop-loss-json', '{"trigger_price":"101"}'],
            ['spot', 'order', 'amend', '--pair', 'BTC_USDT', '--id', '123', '--stop-loss-json', '{"typo":"1"}'],
            ['options', 'order', 'create', '--contract', 'BTC_USDT-20261225-60000-C', '--size', '1', '--close'],
            ['options', 'order', 'create', '--contract', 'BTC_USDT-20261225-60000-C', '--size', '0'],
            ['futures', 'chase', 'create', '--json', '{"contract":"BTC_USDT","amount":"1","price_limit":"0","unknown":true}'],
            ['futures', 'chase', 'create', '--json', '{"contract":"BTC_USDT","amount":"0","price_limit":"0"}'],
            ['futures', 'chase', 'create', '--json', '{"contract":"BTC_USDT","amount":"1","price_limit":"0","settle":"btc"}'],
            ['futures', 'chase', 'stop', '--id', '123', '--text', 't-chase'],
        ]
        for raw in ('{"contract":"BTC_USDT","amount":"1","price_limit":"0","is_dual_mode":true,"iſ_dual_mode":false}',
                    '{"contract":"BTC_USDT","amount":"1","price_limit":"0","reduce_only":null}',
                    '{"contract":"BTC_USDT","amount":"1","price_limit":"0","reduce_only":true,"reduce_only":false}'):
            commands.append(['futures', 'chase', 'create', '--json', raw])
        for raw in ('{"trigger_price":null}',  '{"trigger_price":""}', '{"order_price":null}'):
            commands.append(['spot', 'order', 'amend', '--pair', 'BTC_USDT', '--id', '123',
                             '--stop-loss-json', raw])
        with self.environment('hmac') as env:
            for args in commands:
                with self.subTest(args=args):
                    self.invoke(env, args, error=True, sends=0)


if __name__ == '__main__':
    unittest.main()
