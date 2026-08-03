package main

import (
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// apiReachable reports whether the live JW.org API can be reached, so
// network-dependent tests can be skipped in offline environments.
func apiReachable(t *testing.T) bool {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://data.jw-api.org/mediator/v1/languages/E/web?clientType=www")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

func TestJwbIndexHelp(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("jwb-index --help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "jwb-index") {
		t.Errorf("help output missing 'jwb-index': %s", out)
	}
}

func TestJwbIndexLanguages(t *testing.T) {
	if !apiReachable(t) {
		t.Skip("JW.org API not reachable; skipping network-dependent test")
	}
	out, err := exec.Command("go", "run", ".", "--languages").CombinedOutput()
	if err != nil {
		t.Fatalf("--languages failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "language codes") {
		t.Errorf("expected language listing, got: %s", out)
	}
}
