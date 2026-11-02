# Common/CEX and Intel tests share the historical Go packages and agentic_e2e suite.
# Keep builds bounded on development machines; GO can select an installed Go 1.24.
GO ?= go
# A caller's make -j must not multiply whole suites beyond the per-binary limit.
.NOTPARALLEL:
export GOMAXPROCS ?= 2
export GOMEMLIMIT ?= 768MiB

CLI_UNIT_PACKAGES := ./cmd ./cmd/auth ./cmd/config ./cmd/doctor ./cmd/migrate ./cmd/preflight ./cmd/cex/... \
	./internal/agentcmd ./internal/agentfeature ./internal/client ./internal/cmdhint ./internal/cmdindex \
	./internal/cmdutil ./internal/config ./internal/configcmd ./internal/exitcode ./internal/migration \
	./internal/oauth2 ./internal/operations ./internal/output ./internal/terminalio ./internal/useragent \
	./internal/version ./tests/contracts

# GitHub history adds public Info/News behavior: run its existing unit suites
# in both standard and Agent gates, rather than relying on a one-off go test ./....
CLI_UNIT_PACKAGES += ./cmd/info ./cmd/news ./internal/intelcmd ./internal/intelfacade \
	./internal/mcpclient ./internal/mcpspec ./internal/toolargs ./internal/toolconfig \
	./internal/toolrender ./internal/toolschema

.PHONY: test-cli test-cli-unit test-cli-e2e-local test-cli-auth test-cli-e2e test-cli-check test-cli-agent

test-cli: test-cli-unit test-cli-e2e-local test-cli-auth test-cli-agent

test-cli-unit:
	$(GO) test -p 1 -parallel 2 -count=1 $(CLI_UNIT_PACKAGES)

test-cli-e2e-local:
	$(GO) test -p 1 -parallel 2 -count=1 -tags agentic_e2e -timeout=8m ./tests/agentic

test-cli-auth:
	$(GO) test -p 1 -parallel 2 -count=1 -tags agentic_e2e -timeout=5m ./tests/agentic -run '^TestAuth'

test-cli-agent:
	$(GO) test -p 1 -parallel 2 -count=1 -tags agent $(CLI_UNIT_PACKAGES)
	$(GO) test -p 1 -parallel 2 -count=1 -tags agentic_e2e,agent -timeout=8m ./tests/agentic

# Complete public CLI/SDK suite with synthetic remote HTTP peers.
test-cli-e2e:
	$(GO) test -p 1 -parallel 2 -count=1 -tags agentic_e2e -timeout=12m ./tests/agentic

test-cli-check:
	$(GO) build -p 1 ./cmd/... ./internal/...
	$(GO) vet -p 1 $(CLI_UNIT_PACKAGES)

# Focused defect regression gate; these tests also run in the ordinary suites.
# The historical tag is retained for callers of this existing make target.
.PHONY: test-cli-adversarial test-cli-adversarial-unit test-cli-adversarial-e2e
test-cli-adversarial: test-cli-adversarial-unit test-cli-adversarial-e2e

test-cli-adversarial-unit:
	$(GO) test -p 1 -parallel 2 -count=1 -tags cli_adversarial -run '^TestAdversarial' ./cmd

test-cli-adversarial-e2e:
	$(GO) test -p 1 -parallel 2 -count=1 -tags agentic_e2e,cli_adversarial -timeout=8m -run '^TestAdversarial' ./tests/agentic
