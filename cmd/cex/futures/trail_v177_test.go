package futures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Price reference is a caller decision. Missing or invalid input must never
// reach the trading backend or silently choose a different reference price.
func TestTrail177PriceType(t *testing.T) {
	for _, value := range []string{"omitted", "0", "-1", "4", "1", "2", "3"} {
		t.Run(value, func(t *testing.T) {
			cmd := confirmedFuturesLeaf(t)
			if value != "omitted" {
				require.NoError(t, cmd.Flags().Set("price-type", value))
			}
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, "/api/v4/futures/usdt/autoorder/v1/trail/create", r.URL.Path)
				var body map[string]json.RawMessage
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, value, string(body["price_type"]))
				assert.JSONEq(t, `"SYNTH_USDT"`, string(body["contract"]))
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"code":0,"message":"ok","data":{"id":"9007199254740993"}}`)
			}))
			t.Cleanup(peer.Close)
			t.Setenv("GATE_BASE_URL", peer.URL)
			silenceStdout(t)
			err := runFuturesTrailCreate(cmd, nil)
			if value == "1" || value == "2" || value == "3" {
				require.NoError(t, err)
				require.EqualValues(t, 1, calls.Load())
			} else {
				require.ErrorContains(t, err, "price-type")
				require.Zero(t, calls.Load())
			}
		})
	}
}

func TestTrail177RequiredFlag(t *testing.T) {
	create, _, err := trailCmd.Find([]string{"create"})
	require.NoError(t, err)
	flag := create.Flags().Lookup("price-type")
	require.NotNil(t, flag)
	require.Equal(t, "int32", flag.Value.Type())
	require.Equal(t, "0", flag.DefValue)
	require.NotEmpty(t, flag.Annotations[cobraBashCompOneRequiredFlag])
}
