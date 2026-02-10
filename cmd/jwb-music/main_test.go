package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestJwbMusicHelp(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("jwb-music --help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "jwb-music") {
		t.Errorf("help output missing 'jwb-music': %s", out)
	}
}

func TestJwbMusicListCategories(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "--list-categories").CombinedOutput()
	if err != nil {
		t.Fatalf("--list-categories failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Available music categories") {
		t.Errorf("expected category listing, got: %s", out)
	}
}
