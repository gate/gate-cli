package contracts

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqualJSON(t *testing.T) {
	// Adjacent int64 values must remain distinct above JavaScript's safe range.
	for _, tc := range []struct {
		a, b  string
		equal bool
	}{
		{`{"id":9007199254740993,"n":"1.00000001"}`, `{"n":"1.00000001","id":9007199254740993}`, true},
		{`9007199254740993`, `9007199254740992`, false},
		{`"1.00"`, `"1"`, false}, {`[]`, `null`, false}, {`{}`, `{}`, true},
		{`{`, `{}`, false}, {`{}`, `{`, false}, {`{} {}`, `{}`, false}, {`{}`, `{} x`, false},
	} {
		t.Run(tc.a+tc.b, func(t *testing.T) { require.Equal(t, tc.equal, EqualJSON([]byte(tc.a), []byte(tc.b))) })
	}
}

func TestAuthenticationOracle(t *testing.T) {
	r := httptest.NewRequest("POST", "http://127.0.0.1/api/v4/orders?text=t-test%20value", strings.NewReader(`{"amount":"0.00000001"}`))
	raw := []byte(`{"amount":"0.00000001"}`)
	require.NoError(t, CheckAuth(r, raw, "anonymous"))
	require.Error(t, CheckAuth(r, raw, "unknown"))
	r.Header.Set("Authorization", "Bearer fixture-at")
	require.NoError(t, CheckAuth(r, raw, "oauth"))
	require.Error(t, CheckAuth(r, raw, "anonymous"))
	r.Header.Set("KEY", "synthetic-key")
	require.Error(t, CheckAuth(r, raw, "oauth"))
	r.Header.Del("Authorization")
	r.Header.Set("Timestamp", "1793577600")
	digest := sha512.Sum512(raw)
	mac := hmac.New(sha512.New, []byte("synthetic-secret"))
	_, err := fmt.Fprintf(mac, "POST\n/api/v4/orders\ntext=t-test value\n%s\n1793577600", hex.EncodeToString(digest[:]))
	require.NoError(t, err)
	r.Header.Set("SIGN", hex.EncodeToString(mac.Sum(nil)))
	require.NoError(t, CheckAuth(r, raw, "hmac"))
	require.Error(t, CheckAuth(r, []byte(`{"amount":"1"}`), "hmac"))
	r.URL.RawQuery = "text=different"
	require.Error(t, CheckAuth(r, raw, "hmac"))
	r.Header.Set("Authorization", "Bearer fixture-at")
	require.Error(t, CheckAuth(r, raw, "hmac"))
}

func TestFiniteHTTPPeer(t *testing.T) {
	for _, fault := range []string{"success", "401", "403", "429", "503", "malformed", "disconnect"} {
		t.Run(fault, func(t *testing.T) {
			c := Case{Name: "test", Exchanges: []Exchange{{Method: "GET", Path: "/api/v4/fixture", Query: "limit=1", Body: []byte(`null`), Reply: []byte(`{"id":"9007199254740993"}`)}}}
			peer := NewPeer(t, c, "anonymous", fault)
			resp, err := http.Get(peer.URL + "/api/v4/fixture?limit=1")
			if fault == "disconnect" {
				require.Error(t, err)
				peer.AssertCalls(1)
				return
			}
			require.NoError(t, err)
			defer func() { require.NoError(t, resp.Body.Close()) }()
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			switch fault {
			case "success":
				require.JSONEq(t, string(c.Exchanges[0].Reply), string(raw))
			case "malformed":
				require.Equal(t, `{"incomplete":`, string(raw))
			default:
				require.Equal(t, fault, fmt.Sprint(resp.StatusCode))
				require.Contains(t, string(raw), "CONTRACT_FAILURE")
			}
			peer.AssertCalls(1)
		})
	}
}

func TestBusinessContractFixtureIntegrity(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Cases(t) {
		require.False(t, seen[c.Name], "duplicate %s", c.Name)
		seen[c.Name] = true
		require.True(t, strings.HasPrefix(c.Name, "cex "))
		require.NotEmpty(t, c.Operation)
		require.Equal(t, strings.Fields(c.Name), c.Args[:len(strings.Fields(c.Name))])
		require.NotEmpty(t, c.Exchanges)
		for _, x := range c.Exchanges {
			require.True(t, strings.HasPrefix(x.Path, "/api/v4/"))
			require.NotContains(t, x.Path, "{")
			require.True(t, EqualJSON(x.Reply, x.Reply), "invalid reply %s", c.Name)
		}
		require.True(t, EqualJSON(c.Want, c.Want), "invalid expected output %s", c.Name)
		require.NotEmpty(t, c.WantPretty, "each human projection needs a content contract: %s", c.Name)
	}
}
