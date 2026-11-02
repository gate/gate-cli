"""CLI cwd file containment through real MCP HTTP, using only a local peer.

Run GATE_CLI_BINARY=/absolute/path/to/gate-cli python3 tests/e2e/intel_args_file_cli.py.
No host credentials or real backend participate. Relative, absolute and home
paths share the same cwd boundary; only the external MCP service is mocked.
"""
import http.server
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import unittest


class IntelArgsFileProcessTest(unittest.TestCase):
    """The process, Cobra parsing, argument merge and MCP client stay real."""

    @classmethod
    def setUpClass(cls):
        cls.binary = os.environ['GATE_CLI_BINARY']
        if not Path(cls.binary).is_file():
            raise RuntimeError('Build the CLI before running file-input E2E')
        cls.calls = []

        class Peer(http.server.BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                self.connection.settimeout(3)
                length = int(self.headers.get('Content-Length', '0'))
                if not 0 <= length <= 2 * 1024 * 1024:
                    self.send_error(413)
                    return
                message = json.loads(self.rfile.read(length))
                method = message['method']
                if method == 'initialize':
                    result = {'protocolVersion': '2024-11-05', 'capabilities': {},
                              'serverInfo': {'name': 'file-input-peer', 'version': '1'}}
                elif method == 'tools/list':
                    result = {'tools': [{'name': name, 'inputSchema': {'type': 'object',
                              'properties': {'query': {'type': 'string'},
                                             'enabled': {'type': 'boolean'},
                                             'symbol': {'type': 'string'}}}}
                              for name in ('info_coin_get_coin_info', 'news_feed_search_news',
                                           'info_platformmetrics_get_cex_orderbook_depth')]}
                elif method == 'tools/call':
                    cls.calls.append(message['params'])
                    result = {'content': [{'type': 'text', 'text': '{"accepted":true}'}]}
                else:
                    result = {}
                data = json.dumps({'jsonrpc': '2.0', 'id': message.get('id'),
                                   'result': result}).encode()
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.send_header('MCP-Session-Id', 'file-input-fixture')
                self.send_header('Content-Length', str(len(data)))
                self.end_headers()
                self.wfile.write(data)

        cls.peer = http.server.HTTPServer(('127.0.0.1', 0), Peer)
        cls.thread = threading.Thread(target=cls.peer.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.peer.shutdown()
        cls.peer.server_close()
        cls.thread.join(3)
        if cls.thread.is_alive():
            raise RuntimeError('MCP peer did not stop')

    def invoke(self, root, backend, file, extra=()):
        name = 'info_coin_get_coin_info' if backend == 'info' else 'news_feed_search_news'
        return self.run_cli(root, [backend, 'invoke', '--name', name], file, extra)

    def run_cli(self, root, command, file, extra=()):
        # Each invocation has isolated config/cache; only existing endpoint env
        # settings are used, so no new product configuration is introduced.
        env = {key: value for key, value in os.environ.items()
               if key != 'HOME' and not key.startswith('GATE_')}
        url = f'http://127.0.0.1:{self.peer.server_port}'
        env.update(HOME=str(root / 'home'), GATE_INTEL_INFO_MCP_URL=url,
                   GATE_INTEL_NEWS_MCP_URL=url, NO_PROXY='127.0.0.1,localhost')
        before = len(self.calls)
        result = subprocess.run([self.binary, '--format', 'json', *command,
                                 '--args-file', str(file), *extra],
                                cwd=root / 'work', env=env, capture_output=True,
                                text=True, timeout=15)
        return result, self.calls[before:]

    def test_authorized_file_forms(self):
        with tempfile.TemporaryDirectory(prefix='intel-file-e2e-') as tmp:
            root = Path(tmp)
            (root / 'work').mkdir()
            (root / 'work/home').mkdir()
            # HOME is an alias of a directory physically inside cwd. Its path
            # spelling must not interfere with the real-root containment check.
            (root / 'home').symlink_to(root / 'work/home', target_is_directory=True)
            payload = {'query': 'BTC', 'enabled': False}
            for folder in ('home', 'work'):
                (root / folder / 'args.json').write_text(json.dumps(payload))
            for backend in ('info', 'news'):
                for path in ('args.json', root / 'work/args.json',
                             root / 'home/args.json', '~/args.json'):
                    with self.subTest(backend=backend, path=path):
                        result, calls = self.invoke(root, backend, path)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(result.stderr, '')
                        self.assertTrue(json.loads(result.stdout)['accepted'])
                        self.assertEqual(len(calls), 1)
                        self.assertEqual(calls[0]['arguments'], payload)

    def test_invalid_file_never_calls_tool(self):
        with tempfile.TemporaryDirectory(prefix='intel-file-e2e-') as tmp:
            root = Path(tmp)
            (root / 'home').mkdir()
            (root / 'work').mkdir()
            (root / 'work/bad.json').write_text('sensitive-fixture-marker: invalid')
            (root / 'work/large.json').write_text('{"query":"' + 'a' * (1 << 20) + '"}')
            (root / 'work/array.json').write_text('["sensitive-fixture-marker"]')
            (root / 'work/dangling.json').symlink_to(root / 'work/missing.json')
            (root / 'work/loop.json').symlink_to('loop.json')
            for backend in ('info', 'news'):
                for path, extra in (('bad.json', ()), ('large.json', ()), ('array.json', ()),
                                    ('dangling.json', ()), ('loop.json', ()),
                                    ('missing.json', ()), ('../outside.json', ()),
                                    ('bad.json', ('--args-json', '{"query":"BTC"}'))):
                    with self.subTest(backend=backend, path=path, extra=extra):
                        result, calls = self.invoke(root, backend, path, extra)
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(calls, [])
                        self.assertEqual(result.stdout, '')
                        self.assertNotIn('sensitive-fixture-marker', result.stderr)
                        self.assertIn('error', json.loads(result.stderr))

    def test_all_link_forms_enforce_the_same_boundary(self):
        with tempfile.TemporaryDirectory(prefix='intel-file-e2e-') as tmp:
            root = Path(tmp)
            (root / 'home').mkdir()
            (root / 'work').mkdir()
            payload = {'query': 'outside-fixture-marker', 'enabled': False}
            outside = root / 'home/outside.json'
            outside.write_text(json.dumps(payload))
            (root / 'work/escape.json').symlink_to(outside)
            (root / 'work/inside.json').write_text(json.dumps({'query': 'BTC'}))
            (root / 'work/inside-link.json').symlink_to(root / 'work/inside.json')
            (root / 'home/home-link.json').symlink_to(outside)
            (root / 'work/external-dir').symlink_to(root / 'home', target_is_directory=True)
            for backend in ('info', 'news'):
                with self.subTest(backend=backend):
                    for file in ('escape.json', root / 'work/escape.json', outside, '~/outside.json',
                                 '~/home-link.json', 'external-dir/outside.json',
                                 root / 'work/external-dir/outside.json'):
                        result, calls = self.invoke(root, backend, file)
                        self.assertNotEqual(result.returncode, 0, str(file))
                        self.assertEqual(calls, [])
                        self.assertEqual(result.stdout, '')
                        self.assertNotIn('outside-fixture-marker', result.stderr)
                        self.assertIn('must stay within working directory', result.stderr)
                    for file in ('inside-link.json', root / 'work/inside-link.json'):
                        result, calls = self.invoke(root, backend, file)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(len(calls), 1)
                        self.assertEqual(calls[0]['arguments'], {'query': 'BTC'})

    def test_external_files_are_rejected_before_parsing(self):
        with tempfile.TemporaryDirectory(prefix='intel-file-e2e-') as tmp:
            root = Path(tmp)
            (root / 'home').mkdir()
            (root / 'work').mkdir()
            # A sibling with the cwd prefix catches unsafe string-prefix checks.
            (root / 'work-sibling').mkdir()
            for file in (root / 'home/args.json', root / 'work-sibling/args.json'):
                file.write_text('{"query":"external-fixture-marker"}')
            (root / 'home/malformed.txt').write_text('external-fixture-marker: not JSON')
            for backend in ('info', 'news'):
                for file in (root / 'home/args.json', '~/args.json',
                             root / 'work-sibling/args.json', root / 'home/malformed.txt',
                             '~/malformed.txt'):
                    with self.subTest(backend=backend, file=file):
                        result, calls = self.invoke(root, backend, file)
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(calls, [])
                        self.assertEqual(result.stdout, '')
                        error = json.loads(result.stderr)['error']
                        self.assertEqual(error['label'], 'INVALID_ARGUMENTS')
                        self.assertIn('must stay within working directory', error['message'])
                        self.assertNotIn('external-fixture-marker', result.stderr)

    def test_exact_size_limit_and_flag_overlay_remain_supported(self):
        with tempfile.TemporaryDirectory(prefix='intel-file-e2e-') as tmp:
            root = Path(tmp)
            (root / 'home').mkdir()
            (root / 'work').mkdir()
            payload = {'query': 'BTC', 'enabled': False, 'padding': ''}
            encoded = json.dumps(payload, separators=(',', ':'))
            payload['padding'] = 'a' * ((1 << 20) - len(encoded))
            file = root / 'work/args.json'
            file.write_text(json.dumps(payload, separators=(',', ':')))
            self.assertEqual(file.stat().st_size, 1 << 20)
            for backend in ('info', 'news'):
                result, calls = self.invoke(root, backend, file, ('--query', 'ETH'))
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertTrue(json.loads(result.stdout)['accepted'])
                self.assertEqual(len(calls), 1)
                self.assertEqual(calls[0]['arguments'], dict(payload, query='ETH'))

    def test_original_orderbook_leaf_rejects_external_file_forms(self):
        with tempfile.TemporaryDirectory(prefix='intel-file-e2e-') as tmp:
            root = Path(tmp)
            (root / 'home').mkdir()
            (root / 'work').mkdir()
            payload = {'symbol': 'BTC_USDT', 'review_marker': 'external-fixture-marker'}
            outside = root / 'home/args.json'
            outside.write_text(json.dumps(payload))
            inside = root / 'work/args.json'
            inside.write_text(json.dumps({'symbol': 'BTC_USDT'}))
            (root / 'work/link.json').symlink_to(outside)
            command = ['info', 'platformmetrics', 'get-cex-orderbook-depth']
            result, calls = self.run_cli(root, command, inside)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(json.loads(result.stdout)['accepted'])
            self.assertEqual(len(calls), 1)
            self.assertEqual(calls[0]['arguments']['symbol'], 'BTC_USDT')
            for file in (outside, '~/args.json', 'link.json', root / 'work/link.json'):
                result, calls = self.run_cli(root, command, file)
                self.assertNotEqual(result.returncode, 0, str(file))
                self.assertEqual(calls, [])
                self.assertEqual(result.stdout, '')
                self.assertNotIn('external-fixture-marker', result.stderr)
                self.assertIn('must stay within working directory', result.stderr)


if __name__ == '__main__':
    unittest.main()
