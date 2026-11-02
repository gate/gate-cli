"""Public unit checks for synthetic scope contracts and profile isolation."""
import json
from pathlib import Path
import unittest
import scope_contract as suite


class ScopeCLIContractTest(unittest.TestCase):
    """No gateway, issuer or external credentials are required."""
    def test_environment_cannot_inherit_credentials_or_issuer(self):
        source = {'HOME': '/real/home', 'PATH': '/bin', 'GATE_API_KEY': 'host-key',
                  'GATE_OAUTH_ISSUER': 'https://other.invalid',
                  'CLI_AGENTIC_TEST_ISSUER': 'https://other.invalid'}
        env = suite.isolated_environment('/fixture', 'http://127.0.0.1:1234', source)
        self.assertEqual(env['HOME'], '/fixture')
        self.assertEqual(env['GATE_BASE_URL'], 'http://127.0.0.1:1234')
        self.assertEqual(env['PATH'], '/bin')
        self.assertNotIn('GATE_API_KEY', env)
        self.assertNotIn('GATE_OAUTH_ISSUER', env)
        self.assertNotIn('CLI_AGENTIC_TEST_ISSUER', env)
        self.assertEqual(source['HOME'], '/real/home')

    def test_profile_is_resource_bound_private_and_removed(self):
        with suite.credential_profile('http://127.0.0.1:1234', 'earn', 'synthetic') as env:
            home = Path(env['HOME'])
            profile = home / '.gate-cli/config.yaml'
            token = json.loads(profile.read_text())['oauth2']['token']
            self.assertEqual(token['resource'], 'http://127.0.0.1:1234/api/v4')
            self.assertEqual(token['scopes'], 'earn')
            self.assertNotIn('refresh_token', token)
            self.assertNotIn('discovery_url', token)
            self.assertEqual(profile.stat().st_mode & 0o777, 0o600)
        self.assertFalse(home.exists())

    def test_complete_bot_and_scope_inventory(self):
        cases = suite.scope_cases()
        self.assertEqual({case.scope for case in cases}, set(suite.SCOPES))
        bot = [case for case in cases if case.args[0] == 'bot']
        self.assertEqual(len({(case.method, case.path) for case in bot}), 12)
        stops = [case for case in bot if case.name.startswith('bot/stop/')]
        self.assertEqual(len(stops), 8)
        self.assertEqual({case.args[case.args.index('--strategy-type') + 1] for case in stops},
                         set(suite.TYPES))
        for case in stops:
            kind = case.args[case.args.index('--strategy-type') + 1]
            self.assertEqual(case.scope, 'spot_trade' if kind in suite.SPOT_TYPES else 'futures_trade')

    def test_cli_only_flash_operations_preserve_existing_mapping(self):
        # The document does not authorize reclassifying or retiring these methods.
        flash = {case.name: case for case in suite.scope_cases() if case.name.startswith('flash/')}
        self.assertEqual(set(flash), {'flash/create', 'flash/preview'})
        self.assertEqual(flash['flash/create'].path, '/api/v4/flash_swap/orders')
        self.assertEqual(flash['flash/preview'].path, '/api/v4/flash_swap/orders/preview')
        self.assertTrue(all(case.scope == 'base' for case in flash.values()))


if __name__ == "__main__":
    unittest.main()
