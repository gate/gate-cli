package oauth2

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gate/gate-cli/internal/exitcode"
)

// CallbackResult holds the authorization code received via local callback.
type CallbackResult struct {
	Code  string
	State string
	Error string
}

// CallbackServer manages a local HTTP server for receiving OAuth2 callbacks.
type CallbackServer struct {
	RedirectURI string

	listener net.Listener
	resultCh chan *CallbackResult

	mu        sync.Mutex
	closed    bool
	started   bool
	srv       *http.Server
	serveWG   sync.WaitGroup
	closeOnce sync.Once
}

// NewCallbackServer binds to the first available candidate port.
// Returns a CodedError with EnvUnsatisfied when all candidate ports are
// occupied, so the top-level mapExitCode emits exit code 5 without string
// matching.
func NewCallbackServer() (*CallbackServer, error) {
	for _, port := range CallbackPorts {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		return &CallbackServer{
			RedirectURI: fmt.Sprintf("http://127.0.0.1:%d/callback", port),
			listener:    l,
			resultCh:    make(chan *CallbackResult, 1),
		}, nil
	}
	return nil, exitcode.New(5, fmt.Errorf(
		"local callback ports exhausted (18991-18995 all occupied)\n"+
			"  Close programs using these ports, or use manual mode:\n"+
			"    gate-cli cex auth login --manual"))
}

// Close shuts down the HTTP server and releases the bound port. Safe to call
// multiple times and from multiple goroutines; only the first call takes effect.
func (s *CallbackServer) Close() {
	s.closeOnce.Do(func() {
		// 启动与关闭共享同一生命周期锁；Add必须先于Close的Wait，关闭后不可再启动。
		s.mu.Lock()
		s.closed = true
		srv := s.srv
		s.mu.Unlock()
		if srv != nil {
			shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutCtx); err != nil && s.listener != nil {
				// Shutdown can time out if the handler is mid-write. Force-close
				// the listener so Serve unblocks and serveWG drains.
				_ = s.listener.Close()
			}
		} else if s.listener != nil {
			_ = s.listener.Close()
		}
		// Wait so callers are guaranteed no Serve goroutine remains after Close.
		s.serveWG.Wait()
	})
}

// WaitForCallback starts serving and blocks until a callback is received or ctx is cancelled.
func (s *CallbackServer) WaitForCallback(ctx context.Context) (*CallbackResult, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		result := &CallbackResult{
			Code:  q.Get("code"),
			State: q.Get("state"),
			Error: q.Get("error"),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'")
		if result.Error != "" {
			if _, err := fmt.Fprintf(w, "<html><body><h2>Authorization Failed</h2><p>%s</p><p>You can close this window.</p></body></html>", html.EscapeString(result.Error)); err != nil {
				log.Print("OAuth callback page write failed")
			}
		} else {
			if _, err := fmt.Fprint(w, "<html><body><h2>Authorization Successful</h2><p>You can close this window and return to the terminal.</p></body></html>"); err != nil {
				log.Print("OAuth callback page write failed")
			}
		}

		select {
		case s.resultCh <- result:
		default:
			// Duplicate callback (browser refresh / replay). Drop silently.
		}
	})

	s.mu.Lock()
	if s.closed || s.started {
		s.mu.Unlock()
		return nil, fmt.Errorf("callback server already started or closed")
	}
	s.started = true
	s.srv = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
	srv := s.srv
	s.serveWG.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.serveWG.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[warn] callback server panicked: %v\n", r)
			}
		}()
		if err := srv.Serve(s.listener); err != nil && err != http.ErrServerClosed && !errors.Is(err, net.ErrClosed) {
			fmt.Fprintf(os.Stderr, "[warn] callback server exited: %v\n", err)
		}
	}()

	defer s.Close()

	select {
	case result := <-s.resultCh:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
