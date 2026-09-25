package httpx

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetRetriesServerErrors(t *testing.T) {
	RetryDelay = time.Millisecond
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("unexpected User-Agent %q", r.Header.Get("User-Agent"))
		}
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp, err := Get(NewClient(5*time.Second), srv.URL)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || calls.Load() != 3 {
		t.Errorf("status %d after %d calls, want 200 after 3", resp.StatusCode, calls.Load())
	}
}

func TestGetDoesNotRetryClientErrors(t *testing.T) {
	RetryDelay = time.Millisecond
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	resp, err := Get(NewClient(5*time.Second), srv.URL)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound || calls.Load() != 1 {
		t.Errorf("status %d after %d calls, want 404 after 1", resp.StatusCode, calls.Load())
	}
}

func TestGetGivesUpOnNetworkErrors(t *testing.T) {
	RetryDelay = time.Millisecond
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	if _, err := Get(NewClient(time.Second), url); err == nil {
		t.Error("expected an error for an unreachable server")
	}
}
