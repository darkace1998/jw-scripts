package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkDir(t *testing.T) {
	if dir, err := WorkDir(nil, "default"); err != nil || dir != "default" {
		t.Errorf("WorkDir(nil) = %q, %v", dir, err)
	}
	if dir, err := WorkDir([]string{"/data"}, "."); err != nil || dir != "/data" {
		t.Errorf("WorkDir(/data) = %q, %v", dir, err)
	}
	if _, err := WorkDir([]string{"2"}, "."); err == nil {
		t.Error("expected leftover quiet level to be rejected")
	}
}

func TestWorkDirAllowsExistingNumericDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "1"), 0o750); err != nil {
		t.Fatal(err)
	}
	if got, err := WorkDir([]string{"1"}, "."); err != nil || got != "1" {
		t.Errorf("WorkDir(1) = %q, %v", got, err)
	}
}
