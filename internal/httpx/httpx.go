// Package httpx provides the HTTP clients shared by all commands: a common
// User-Agent, connection timeouts that catch stalled servers, and retries for
// transient API failures.
package httpx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// UserAgent identifies this tool to the JW.org servers.
const UserAgent = "jw-scripts (+https://github.com/darkace1998/jw-scripts)"

// maxAttempts is the number of times Get tries a request before giving up.
const maxAttempts = 3

// RetryDelay is the base delay between retries; it doubles on every attempt.
// Tests may shorten it.
var RetryDelay = time.Second

// uaTransport sets the User-Agent header on every outgoing request.
type uaTransport struct {
	base http.RoundTripper
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", UserAgent)
	}
	return t.base.RoundTrip(req)
}

// newTransport returns a transport whose connection setup and response
// headers are bounded by timeouts, so a server that stops answering cannot
// hang a run forever. Reading the body is not bounded here; long downloads
// guard against stalls with an idle timeout instead.
func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	t.TLSHandshakeTimeout = 15 * time.Second
	t.ResponseHeaderTimeout = 60 * time.Second
	return t
}

// NewClient returns an HTTP client with the shared transport. timeout bounds
// the whole request including the body; zero means no overall limit (used
// for large downloads).
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &uaTransport{base: newTransport()},
	}
}

// retryable reports whether a response status is worth retrying.
func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

// Get performs a GET request and retries it on network errors, HTTP 429 and
// 5xx responses. Any other response is returned to the caller, which must
// check the status code and close the body.
func Get(client *http.Client, url string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(RetryDelay << (attempt - 1))
		}
		// #nosec G107 - callers build URLs from fixed JW.org API endpoints
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		if retryable(resp.StatusCode) && attempt < maxAttempts-1 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("server returned %s", resp.Status)
			continue
		}
		return resp, nil
	}
	return nil, errors.Join(fmt.Errorf("request failed after %d attempts", maxAttempts), lastErr)
}
