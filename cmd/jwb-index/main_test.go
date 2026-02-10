package main

import (
	"os/exec"
	"strings"
	"testing"
)

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
	out, err := exec.Command("go", "run", ".", "--languages").CombinedOutput()
	if err != nil {
		t.Fatalf("--languages failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "language codes") {
		t.Errorf("expected language listing, got: %s", out)
	}
}
