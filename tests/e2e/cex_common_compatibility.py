"""Old/new artifact compatibility oracle for the CEX/common change.

Run after building both artifacts with identical Go/SDK/Version/tags:
python3 tests/e2e/cex_common_compatibility.py --baseline OLD --candidate NEW
This complements (does not replace) the unified tests/agentic feature suite.
Only the Gate HTTP peer is mocked; request signing and stdout/stderr/exit/wire
are compared. The old artifact must come from the recorded pre-change commit.
"""
import argparse
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
from urllib.parse import unquote_plus, urlsplit


class Compatibility(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.calls = []
        cls.status, cls.response = 200, {}

        class Peer(http.server.BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def respond(self):
                raw = self.rfile.read(int(self.headers.get('Content-Length', '0')))
                parsed = urlsplit(self.path)
                if self.headers.get('KEY'):
                    canonical = '\n'.join((self.command, parsed.path,
                        unquote_plus(parsed.query), hashlib.sha512(raw).hexdigest(),
                        self.headers.get('Timestamp', '')))
                    expected = hmac.new(b'fixture-secret', canonical.encode(), hashlib.sha512).hexdigest()
                    if not hmac.compare_digest(expected, self.headers.get('SIGN', '')):
                        self.send_error(401, 'invalid synthetic signature')
                        return
                cls.calls.append((self.command, self.path, raw,
                    self.headers.get('KEY'), self.headers.get('Authorization')))
                response = cls.response
                if '/positions/' in parsed.path:
                    response = {'contract': 'BTC_USDT', 'size': '5', 'mode': 'single'}
                self.send_response(cls.status)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
            do_GET = do_POST = do_PUT = do_DELETE = respond

        cls.peer = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Peer)
        cls.worker = threading.Thread(target=cls.peer.serve_forever, daemon=True)
        cls.worker.start()

    @classmethod
    def tearDownClass(cls):
        cls.peer.shutdown()
        cls.peer.server_close()
        cls.worker.join(timeout=3)

    def run_artifact(self, binary, argv, input_value=None, agent=False):
        with tempfile.TemporaryDirectory(prefix='cex-compat-') as home:
            env = {key: value for key, value in os.environ.items()
                   if not key.startswith('GATE_') and key not in
                   ('HOME', 'USERPROFILE', 'XDG_CONFIG_HOME', 'XDG_CACHE_HOME',
                    'APPDATA', 'LOCALAPPDATA', 'CLI_AGENTIC_TEST_ISSUER')}
            env.update(HOME=home, USERPROFILE=home, XDG_CACHE_HOME=home+'/cache',
                       XDG_CONFIG_HOME=home+'/config', APPDATA=home, LOCALAPPDATA=home,
                       GATE_API_KEY='fixture-key', GATE_API_SECRET='fixture-secret',
                       GATE_BASE_URL='http://127.0.0.1:'+str(self.peer.server_port),
                       GATE_CLI_SUPPRESS_FORMAT_NOTICE='1')
            if agent:
                env["GATE_CLI_AGENT"] = "1"
            type(self).calls = []
            done = subprocess.run([binary]+argv, input=input_value, capture_output=True,
                                  env=env, timeout=10)
            # The sole normalization is the fixture-owned config path printed by
            # config init. Do not ignore error fields, headers, bodies or hints.
            result = (done.returncode, done.stdout.replace(home.encode(), b'<HOME>'),
                      done.stderr.replace(home.encode(), b'<HOME>'), list(self.calls))
            path = Path(home)/'.gate-cli/config.yaml'
            saved = path.read_bytes() if path.exists() else None
            return result, saved

    def test_old_common_calls(self):
        cases = [(['version'], None), (['--version'], None),
                 (['version', '--format', 'json'], None),
                 (['info', '--help'], None), (['news', '--help'], None),
                 (['cex', 'spot', 'market', 'nosuch'], None),
                 (['help', 'nosuch'], None),
                 (['cex', 'futures', 'order', 'get'], None),
                 (['config', 'init'], b'default\r\nfixture-key\r\nfixture-secret'),
                 (['config', 'init'], b'default\nkey\n'),
                 (['config', 'init'], b'default\nkey\n'+b's'*65537+b'\n')]
        for argv, input_value in cases:
            with self.subTest(argv=argv, input_size=len(input_value or b'')):
                self.assertEqual(self.run_artifact(BASELINE, argv, input_value),
                                 self.run_artifact(CANDIDATE, argv, input_value))


    def test_discovery_contract(self):
        base = ['agent-index', '--format', 'json', '--max-output-bytes', '0']
        if not AGENT:
            for binary in (BASELINE, CANDIDATE):
                result, _ = self.run_artifact(binary, base)
                self.assertNotEqual(result[0], 0, 'standard build exposed agent command')
            return
        for domain in ('cex', 'config', 'info', 'news'):
            with self.subTest(domain=domain):
                argv = base+['--domain', domain]
                old, _ = self.run_artifact(BASELINE, argv)
                new, _ = self.run_artifact(CANDIDATE, argv)
                self.assertEqual(old[0], 0)
                self.assertEqual(new[0], 0)
                before, after = json.loads(old[1]), json.loads(new[1])
                if domain in ('info', 'news'):
                    self.assertEqual(old, new)
                    continue
                added = 'cex futures order get-by-text' if domain == 'cex' else 'config check'
                self.assertEqual(after['count'], before['count']+1)
                self.assertEqual([v for v in after['leaves'] if v['path'] != added], before['leaves'])
                self.assertEqual([v for v in after['commands'] if v != 'gate-cli '+added], before['commands'])
                self.assertEqual(new[2:], old[2:])
        old, _ = self.run_artifact(BASELINE, base, agent=True)
        new, _ = self.run_artifact(CANDIDATE, base, agent=True)
        self.assertEqual(old, new, 'default agent info/news discovery changed')

    def test_normal_cex_wire_and_errors(self):
        cases = []
        for direction in ('long', 'short', 'add', 'remove'):
            cases.append((['futures', 'order', direction, '--contract', 'BTC_USDT',
                           '--size', '1'], {'id': 9007199254740993}))
        for side in ('buy', 'sell'):
            cases.append((['spot', 'order', side, '--pair', 'BTC_USDT', '--amount',
                           '0.00000001', '--price', '10.00000001'], {'id': '9007199254740993'}))
        cases += [(['spot', 'order', 'get', '--id', 't-fixture', '--pair', 'BTC_USDT'], {}),
                  (['futures', 'order', 'get', '--id', '0'], {}),
                  (['futures', 'order', 'get', '--id', '9223372036854775807'], {})]
        for operation in ('create', 'amend', 'cancel'):
            args = ['futures', 'order', 'batch-'+operation]
            args += ['--ids', '9007199254740993'] if operation == 'cancel' else [
                '--orders-json', '[{"contract":"BTC_USDT","size":"1"}]']
            result_id = '9007199254740993' if operation == 'cancel' else 9007199254740993
            cases.append((args, [{'id': result_id, 'succeeded': True}]))
        for args, response in cases:
            for status in (200, 401, 403, 429, 503):
                with self.subTest(args=args, status=status):
                    type(self).status = status
                    type(self).response = response if status == 200 else {
                        'label': 'SYNTHETIC_REJECTED', 'message': 'synthetic rejection'}
                    argv = ['cex']+args+['--format', 'json', '--max-output-bytes', '0']
                    self.assertEqual(self.run_artifact(BASELINE, argv),
                                     self.run_artifact(CANDIDATE, argv))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--baseline', required=True)
    parser.add_argument('--candidate', required=True)
    parser.add_argument('--agent', action='store_true', help='Both artifacts were built with agent')
    args, remaining = parser.parse_known_args()
    AGENT = args.agent
    BASELINE, CANDIDATE = str(Path(args.baseline).resolve()), str(Path(args.candidate).resolve())
    unittest.main(argv=[__file__]+remaining)
