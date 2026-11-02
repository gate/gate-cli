"""Synthetic CLI scope expectations and isolated credential fixtures."""
import contextlib
from dataclasses import dataclass
import json
import os
from pathlib import Path
import tempfile
import bot_cli


SCOPES = ('base', 'spot_trade', 'futures_trade', 'earn', 'cfd', 'stock')
ID, TYPES = bot_cli.ID, bot_cli.TYPES
SPOT_TYPES = frozenset(('spot_grid', 'margin_grid', 'infinite_grid',
                        'spot_martingale', 'smart_position'))


@dataclass(frozen=True)
class ScopeCase:
    """Independent business expectations, never derived from gateway rules."""
    name: str
    args: tuple
    method: str
    path: str
    scope: str
    response: object


def isolated_environment(home, base, inherited=None):
    """Discard host credentials and test issuer overrides before any CLI starts."""
    source = os.environ if inherited is None else inherited
    env = {key: value for key, value in source.items()
           if key != 'HOME' and not key.startswith(('GATE_', 'CLI_AGENTIC_'))}
    env.update(HOME=str(home), GATE_BASE_URL=base, NO_PROXY='127.0.0.1,localhost')
    return env


@contextlib.contextmanager
def credential_profile(base, scope, token):
    """A preissued, resource-bound fixture token avoids browser/issuer traffic."""
    with tempfile.TemporaryDirectory(prefix='agentic-scope-cli-home-') as home:
        folder = Path(home) / '.gate-cli'
        folder.mkdir(mode=0o700)
        profile = folder / 'config.yaml'
        profile.write_text(json.dumps({'oauth2': {'token': {
            'client_id': 'scope-cli-fixture', 'access_token': token,
            'token_type': 'Bearer', 'resource': base + '/api/v4',
            'expires_at': '2099-01-01T00:00:00Z', 'scopes': scope}}}))
        profile.chmod(0o600)
        yield isolated_environment(home, base)


def scope_cases():
    """Cover all twelve Bot commands, eight stop types and changed API families."""
    bot_ok = {'code': 200, 'message': 'ok', 'data': {
        'strategy_id': ID, 'status': 'running', 'strategy_type': 'smart_position'}}
    cases = []
    # Reuse the maintained CLI argument fixtures, but declare permissions here.
    for args, path in bot_cli.BotCLIProcessTest().commands():
        method = 'GET' if args[0] in ('recommend', 'running', 'detail') else 'POST'
        response = bot_ok
        if method == 'GET':
            scope = 'base'
            if args[0] == 'recommend':
                response = {'code': 200, 'message': 'ok', 'data': {
                    'scene': 'refresh', 'recommendations': []}}
            elif args[0] == 'running':
                response = {'code': 200, 'message': 'ok', 'data': {
                    'items': [], 'page': 1, 'page_size': 20, 'total': 0}}
        else:
            if '--json' in args:
                kind = json.loads(args[args.index('--json') + 1])['strategy_type']
            else:
                kind = args[args.index('--strategy-type') + 1]
            scope = 'spot_trade' if kind in SPOT_TYPES else 'futures_trade'
        cases.append(ScopeCase('bot/command/' + '/'.join(args[:2]), tuple(['bot'] + args),
                               method, path, scope, response))
    for kind in TYPES:
        args = ['bot', 'stop', '--strategy-id', ID, '--strategy-type', kind]
        if kind == 'smart_position':
            args += ['--stop-mode', 'no_close']
        cases.append(ScopeCase('bot/stop/' + kind, tuple(args), 'POST',
                               '/api/v4/bot/portfolio/stop',
                               'spot_trade' if kind in SPOT_TYPES else 'futures_trade', bot_ok))
    # Empty responses retain the SDK's actual object/array shape. Peer receipt,
    # trusted identity, exact method/path and body preservation are also asserted.
    extra = [
        ('spot/accounts', ['spot', 'account', 'list'], 'GET', '/spot/accounts', 'base', []),
        ('futures/accounts', ['futures', 'account', 'get'], 'GET', '/futures/usdt/accounts', 'base', {}),
        ('cfd/assets', ['tradfi', 'account', 'assets'], 'GET', '/tradfi/users/assets', 'base', {'code': 0, 'data': {}}),
        ('stock/assets', ['stock', 'account', 'assets'], 'GET', '/stock/users/assets', 'base', {'code': 0, 'data': {}}),
        ('unified/loans', ['unified', 'loan', 'list'], 'GET', '/unified/loans', 'base', []),
        ('earn/min-amount', ['earn', 'auto-invest', 'min-amount', '--json', '{}'], 'POST', '/earn/autoinvest/min_invest_amount', 'base', {}),
        ('spot/cancel', ['spot', 'order', 'cancel', '--id', '9007199254740993', '--pair', 'BTC_USDT'], 'DELETE', '/spot/orders/9007199254740993', 'spot_trade', {}),
        ('futures/cancel', ['futures', 'order', 'cancel', '--id', '9007199254740993'], 'DELETE', '/futures/usdt/orders/9007199254740993', 'futures_trade', {}),
        ('flash/preview', ['flash-swap', 'preview', '--sell-currency', 'BTC', '--buy-currency', 'USDT', '--sell-amount', '0.01'], 'POST', '/flash_swap/orders/preview', 'base', {}),
        # These older CLI-only operations are absent from the MCP document.
        # Preserve the existing asset -> base mapping instead of extending scope changes.
        ('flash/create', ['flash-swap', 'create', '--preview-id', 'synthetic-preview', '--sell-currency', 'BTC', '--sell-amount', '0.01', '--buy-currency', 'USDT', '--buy-amount', '1'], 'POST', '/flash_swap/orders', 'base', {}),
        ('unified/quick-repay', ['unified', 'loan', 'quick-repay', '--debt-currencies', 'USDT', '--available-currencies', 'BTC'], 'POST', '/unified/quick_repayment', 'earn', {}),
        ('unified/borrow', ['unified', 'loan', 'create', '--currency', 'USDT', '--amount', '1', '--type', 'borrow'], 'POST', '/unified/loans', 'earn', {}),
        ('margin/auto-repay', ['margin', 'auto-repay', 'set', '--status', 'on'], 'POST', '/margin/auto_repay', 'earn', {}),
        ('earn/change', ['earn', 'uni', 'change', '--json', '{"currency":"USDT","min_rate":"0.01"}'], 'PATCH', '/earn/uni/lends', 'earn', {}),
        ('cfd/transfer', ['tradfi', 'account', 'transfer', '--change', '1', '--type', 'deposit'], 'POST', '/tradfi/transactions', 'cfd', {'code': 0, 'data': {}}),
        ('stock/transfer', ['stock', 'account', 'transfer', '--change', '1', '--type', 'deposit', '--ref-id', 'scope-fixture'], 'POST', '/stock/transactions', 'base', {'code': 0, 'data': {}}),
        ('stock/cancel', ['stock', 'order', 'cancel', '--id', '9007199254740993'], 'DELETE', '/stock/orders/9007199254740993', 'stock', {'code': 0, 'data': {}}),
    ]
    cases.extend(ScopeCase(name, tuple(args), method, '/api/v4' + path, scope, response)
                 for name, args, method, path, scope, response in extra)
    return cases
