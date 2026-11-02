package oauth2

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Endpoints holds the resolved set of OAuth2 URLs the CLI talks to. All
// fields are populated from an RFC 8414 discovery document — either freshly
// fetched or cache-hit (see discovery.go). There is no silent fallback to
// hardcoded defaults: if discovery fails and no cache exists, InitEndpoints
// returns an error and the CLI refuses to run auth-dependent commands.
type Endpoints struct {
	BaseURL               string
	AuthorizationEndpoint string
	TokenEndpoint         string
	RegistrationEndpoint  string
	RevocationEndpoint    string
	DisplayCallbackURL    string
}

var (
	// activeEndpoints is the process-wide resolved endpoint set used by
	// client.go / BuildAuthorizeURL / BuildDCRRedirectURIs. atomic.Pointer
	// makes concurrent read (hot-path HTTP transport) safe against the
	// single startup write from InitEndpoints and any test helpers that
	// swap endpoints between cases.
	activeEndpoints atomic.Pointer[Endpoints]

	// emptyEndpoints is returned by ActiveEndpoints when no value has been
	// stored yet; never mutate it.
	emptyEndpoints = &Endpoints{}

	// oauth2YAMLWarningOnce guarantees the deprecation notice for the
	// legacy ~/.gate-cli/oauth2.yaml path is printed at most once per run.
	oauth2YAMLWarningOnce sync.Once
)

// ActiveEndpoints returns the current endpoint set used by client functions.
// Callers should not mutate the returned struct — treat it as read-only.
func ActiveEndpoints() *Endpoints {
	if p := activeEndpoints.Load(); p != nil {
		return p
	}
	return emptyEndpoints
}

// setActiveEndpointsForTest overwrites the active endpoint set; reserved for
// tests that need a deterministic DisplayCallbackURL without running a real
// discovery round-trip. Returns the previous pointer so the caller can restore
// via another call to setActiveEndpointsForTest (do NOT use activeEndpoints.Store
// directly — going through this function keeps all writes in one code path).
//
// Not safe for t.Parallel(): the Load/Store pair is not atomic, so two
// parallel tests swapping endpoints can interleave and leak state across tests.
// If parallelism is ever needed, convert this to a per-test injection model
// (e.g. pass Endpoints explicitly) instead of a global swap.
func setActiveEndpointsForTest(ep *Endpoints) *Endpoints {
	prev := activeEndpoints.Load()
	activeEndpoints.Store(ep)
	return prev
}

// InitEndpoints uses the fixed production discovery URL, resolves the RFC 8414
// metadata for that URL (cache or live fetch), and installs the derived
// endpoint set as the process-wide active set. This should be called once at
// startup — failure must be surfaced via cmd/root.go so the user sees a
// clear error rather than downstream 404s.
func InitEndpoints() error { return InitEndpointsContext(context.Background()) }

// InitEndpointsContext 将命令取消和超时传递到发现请求。
func InitEndpointsContext(ctx context.Context) error {
	ep, err := loadEndpointsContext(ctx, os.Stderr)
	if err != nil {
		return err
	}
	activeEndpoints.Store(ep)
	return nil
}

// LoadEndpoints runs the discovery + cache pipeline and returns a fully
// populated Endpoints struct. Split out from InitEndpoints for tests that
// want to assert a specific Endpoints value without mutating global state.
func LoadEndpoints(stderr io.Writer) (*Endpoints, error) {
	return loadEndpointsContext(context.Background(), stderr)
}

func loadEndpointsContext(parent context.Context, stderr io.Writer) (*Endpoints, error) {
	discoveryURL := resolveDiscoveryURL()

	// Legacy oauth2.yaml file is no longer consumed; nudge the user once so
	// custom settings don't silently get ignored.
	maybeWarnLegacyOAuthYAML(stderr)

	ctx, cancel := context.WithTimeout(parent, discoveryHTTPTimeout+2*time.Second)
	defer cancel()

	md, _, err := resolveDiscovery(ctx, discoveryURL, stderr)
	if err != nil {
		return nil, err
	}
	return endpointsFromMetadata(md), nil
}

// discoveryURLSource 仅在测试构建或包内测试中注入；普通构建固定信任正式站点，
// 不读取 YAML 或环境变量。生产初始化后不再修改，避免运行期信任源漂移。
var discoveryURLSource = func() string { return DefaultDiscoveryURL }

// CurrentDiscoveryURL 返回本进程使用的固定发现入口，用于凭据 issuer 绑定。
func CurrentDiscoveryURL() string { return resolveDiscoveryURL() }

func resolveDiscoveryURL() string { return discoveryURLSource() }

// ResolveManualRedirectURI 使用固定展示页，旧配置及发现文档均不能覆盖注册回调。
func ResolveManualRedirectURI() string { return DefaultDisplayCallbackURL }

// endpointsFromMetadata turns an RFC 8414 discovery doc into the concrete
// Endpoints struct consumed by client.go and login.go. DisplayCallbackURL 使用
// 固定的正式地址 DefaultDisplayCallbackURL（www.gate.com），与 discovery issuer
// 解耦——无论 CLI 连接哪个环境的 OAuth2 server，手动模式的展示页始终指向
// 正式域名，用户无需在 config.yaml 配置即可使用。
func endpointsFromMetadata(md *DiscoveryMetadata) *Endpoints {
	base := strings.TrimRight(md.Issuer, "/")
	return &Endpoints{
		BaseURL:               base,
		AuthorizationEndpoint: md.AuthorizationEndpoint,
		TokenEndpoint:         md.TokenEndpoint,
		RegistrationEndpoint:  md.RegistrationEndpoint,
		RevocationEndpoint:    md.RevocationEndpoint,
		DisplayCallbackURL:    DefaultDisplayCallbackURL,
	}
}

// maybeWarnLegacyOAuthYAML emits a one-time stderr notice when the deprecated
// ~/.gate-cli/oauth2.yaml file still exists. The file is no longer read; the
// message points users to the new config.yaml oauth2 section.
func maybeWarnLegacyOAuthYAML(stderr io.Writer) {
	oauth2YAMLWarningOnce.Do(func() {
		home, err := resolveHome()
		if err != nil {
			return
		}
		legacy := filepath.Join(home, ".gate-cli", "oauth2.yaml")
		if _, err := os.Stat(legacy); err != nil {
			return
		}
		log.New(stderr, "", 0).Printf(
			"Notice: %s is no longer read. OAuth2 uses fixed production endpoints; credentials are stored under ~/.gate-cli/config.yaml oauth2:. "+
				"Delete the legacy file or keep it for reference only.\n",
			legacy,
		)
	})
}
