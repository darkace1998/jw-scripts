package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestJwbOfflineHelp(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("jwb-offline --help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "jwb-offline") {
		t.Errorf("help output missing 'jwb-offline': %s", out)
	}
}
