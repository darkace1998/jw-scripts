package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestJwbBooksHelp(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("jwb-books --help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "jwb-books") {
		t.Errorf("help output missing 'jwb-books': %s", out)
	}
}
