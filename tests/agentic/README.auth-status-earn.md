# Authorization status and Earn error regression

These tests run through the existing Go packages and real CLI subprocess suite. Only API peers and credentials are synthetic; the SDK, parsing, resource binding, stderr and exit codes remain real. No financial writes are sent to a live account.

```sh
# Affected unit tests and historical regressions.
GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -p 1 ./cmd/auth ./internal/output ./internal/oauth2 ./internal/client ./cmd/cex -count=1
# Local process tests; no live issuer or account is required.
GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -p 1 -tags agentic_e2e ./tests/agentic -run '^(TestDevReadMatrixSafety|TestAuthStatusIssuerJSON|TestEarnMissingRouteIsFailure|TestLaunchRetirement)$' -count=1
```

Assertions cover local authorization status without exposing tokens, missing-route failures, bounded read-only command selection and retired commands. Local authorization status does not confirm remote consent, account eligibility or service availability. Tests use a temporary HOME and must not read or rotate the user's real credentials.
