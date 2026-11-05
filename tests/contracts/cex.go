// Package contracts contains test-only HTTP contracts shared by handler unit
// tests and actual-binary E2E tests. Production commands never import it.
package contracts

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
)

//go:embed cex.json
var cexJSON []byte

// Exchange fixes the complete HTTP contract, including prerequisite reads.
type Exchange struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Query  string          `json:"query,omitempty"`
	Body   json.RawMessage `json:"body"`
	Reply  json.RawMessage `json:"reply"`
}

// Case separates a frozen wire/output regression from the stateful scenarios.
// IDs are decoded with UseNumber so assertions themselves cannot round int64s.
type Case struct {
	Name      string `json:"name"`
	Operation string `json:"operation"`
	Public    bool   `json:"public"`
	// Anonymous is the CLI admission contract; Public describes the HTTP API.
	// Some historical unified commands explicitly require local authorization.
	Anonymous  bool            `json:"anonymous"`
	Args       []string        `json:"args"`
	Exchanges  []Exchange      `json:"exchanges"`
	Want       json.RawMessage `json:"want"`
	WantPretty string          `json:"want_pretty"`
	Void       bool            `json:"void,omitempty"`
}

func Cases(t testing.TB) []Case {
	t.Helper()
	var cases []Case
	if err := json.Unmarshal(cexJSON, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 397 {
		t.Fatalf("business contracts=%d, want 397", len(cases))
	}
	return cases
}

// Peer is a strict finite script. Extra calls fail instead of receiving a
// permissive default response that could hide retries or incorrect routing.
type Peer struct {
	URL         string
	t           testing.TB
	server      *httptest.Server
	c           Case
	auth, fault string
	mu          sync.Mutex
	calls       int
}

func NewPeer(t testing.TB, c Case, auth, fault string) *Peer {
	t.Helper()
	p := &Peer{t: t, c: c, auth: auth, fault: fault}
	p.server = httptest.NewServer(http.HandlerFunc(p.serve))
	p.URL = p.server.URL
	t.Cleanup(p.server.Close)
	return p
}

func (p *Peer) Calls() int { p.mu.Lock(); defer p.mu.Unlock(); return p.calls }

func (p *Peer) AssertCalls(want int) {
	p.t.Helper()
	if got := p.Calls(); got != want {
		p.t.Errorf("%s requests=%d, want %d", p.c.Name, got, want)
	}
}

func (p *Peer) serve(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	i := p.calls
	p.calls++
	p.mu.Unlock()
	if i >= len(p.c.Exchanges) {
		p.t.Errorf("unexpected extra request %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected extra request", 500)
		return
	}
	want := p.c.Exchanges[i]
	raw, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil || len(raw) > 65536 {
		p.t.Errorf("bounded request body: %v bytes=%d", err, len(raw))
	}
	if r.Method != want.Method || r.URL.Path != want.Path {
		p.t.Errorf("request=%s %s, want %s %s", r.Method, r.URL.Path, want.Method, want.Path)
	}
	query, err := url.ParseQuery(want.Query)
	if err != nil || !reflect.DeepEqual(query, r.URL.Query()) {
		p.t.Errorf("query=%q, want %q (fixture error=%v)", r.URL.RawQuery, want.Query, err)
	}
	if len(raw) == 0 {
		if len(want.Body) != 0 && string(want.Body) != "null" {
			p.t.Errorf("missing request body: want %s", want.Body)
		}
	} else if !EqualJSON(raw, want.Body) {
		p.t.Errorf("body=%s, want %s", raw, want.Body)
	}
	if err := CheckAuth(r, raw, p.auth); err != nil {
		p.t.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Gate-Trace-Id", "contract-trace")
	terminal := i == len(p.c.Exchanges)-1
	if terminal && p.fault == "disconnect" {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			p.t.Error(err)
			return
		}
		if err := conn.Close(); err != nil {
			p.t.Error(err)
		}
		return
	}
	var status int
	if terminal {
		switch p.fault {
		case "401":
			status = 401
		case "403":
			status = 403
		case "429":
			status = 429
		case "503":
			status = 503
		}
	}
	if status != 0 {
		w.WriteHeader(status)
		_, err = io.WriteString(w, `{"label":"CONTRACT_FAILURE","message":"synthetic dependency failure"}`)
	} else if terminal && p.fault == "malformed" {
		_, err = io.WriteString(w, `{"incomplete":`)
	} else {
		_, err = w.Write(want.Reply)
	}
	if err != nil {
		p.t.Errorf("write fixture response: %v", err)
	}
}

// CheckAuth verifies the actual signed bytes, not merely the presence of a
// signature. Synthetic credentials are the only credentials accepted here.
func CheckAuth(r *http.Request, raw []byte, auth string) error {
	switch auth {
	case "anonymous":
		for _, k := range []string{"Authorization", "KEY", "SIGN", "Timestamp"} {
			if r.Header.Get(k) != "" {
				return fmt.Errorf("anonymous request contains %s", k)
			}
		}
	case "oauth":
		if r.Header.Get("Authorization") != "Bearer fixture-at" || r.Header.Get("KEY") != "" || r.Header.Get("SIGN") != "" {
			return fmt.Errorf("lost or mixed OAuth credentials")
		}
	case "hmac":
		decoded, err := url.QueryUnescape(r.URL.RawQuery)
		if err != nil {
			return err
		}
		digest := sha512.Sum512(raw)
		canonical := strings.Join([]string{r.Method, r.URL.Path, decoded, hex.EncodeToString(digest[:]), r.Header.Get("Timestamp")}, "\n")
		mac := hmac.New(sha512.New, []byte("synthetic-secret"))
		_, _ = mac.Write([]byte(canonical))
		if r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("Authorization") != "" || r.Header.Get("Timestamp") == "" || !hmac.Equal([]byte(r.Header.Get("SIGN")), []byte(hex.EncodeToString(mac.Sum(nil)))) {
			return fmt.Errorf("invalid HMAC or mixed credentials")
		}
	default:
		return fmt.Errorf("unknown fixture authentication %q", auth)
	}
	return nil
}

// EqualJSON preserves number spelling and rejects trailing JSON or junk.
func EqualJSON(a, b []byte) bool {
	decode := func(raw []byte) (any, error) {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return nil, fmt.Errorf("trailing JSON: %v", err)
		}
		return v, nil
	}
	x, err := decode(a)
	if err != nil {
		return false
	}
	y, err := decode(b)
	return err == nil && reflect.DeepEqual(x, y)
}
