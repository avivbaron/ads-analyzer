package analysis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetcher_OK(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("google.com, x, DIRECT\n"))
	}))
	defer ts.Close()
}

// TestFetcher_HTTPFallbackAnd404 verifies that the fetcher can retrieve
// ads.txt over HTTP when HTTP fallback is enabled, and returns a body.
// PASS: non-empty body returned without error.
// FAIL: error or empty body.
func TestFetcher_HTTPFallbackAnd404(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ads.txt" {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("appnexus.com, y, DIRECT\n"))
	}))
	defer httpSrv.Close()
	host := httpSrv.URL[len("http://"):]
	f := NewHTTPFetcher(2*time.Second, true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b, err := f.GetAdsTxt(ctx, host)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("empty body")
	}
}

// TestFetcher_NotFound ensures 404 from origin propagates as error.
// PASS: error is returned for 404 server.
// FAIL: no error.
func TestFetcher_NotFound(t *testing.T) {
	httpSrv := httptest.NewServer(http.NotFoundHandler())
	defer httpSrv.Close()
	host := httpSrv.URL[len("http://"):]
	f := NewHTTPFetcher(2*time.Second, false)
	ctx := context.Background()
	_, err := f.GetAdsTxt(ctx, host)
	if err == nil {
		t.Fatalf("want error for 404")
	}
}

// TestFetcher_Timeout verifies client timeout is enforced.
// PASS: request errors due to timeout.
// FAIL: request unexpectedly succeeds.
func TestFetcher_Timeout(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte("google.com, x, DIRECT\n"))
	}))
	defer httpSrv.Close()
	host := httpSrv.URL[len("http://"):]
	f := NewHTTPFetcher(50*time.Millisecond, true)
	ctx := context.Background()
	_, err := f.GetAdsTxt(ctx, host)
	if err == nil {
		t.Fatalf("want timeout error")
	}
}
