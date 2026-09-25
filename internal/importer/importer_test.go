package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestScanImportsMediaFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"song.mp3", "video.MP4", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.mp4"), 0o750); err != nil {
		t.Fatal(err)
	}

	cats, err := Scan(&config.Settings{ImportDir: dir, Quiet: 2})
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(cats) != 1 || len(cats[0].Contents) != 2 {
		t.Fatalf("expected one category with 2 files, got %+v", cats)
	}
	for _, item := range cats[0].Contents {
		m := item.(*api.Media)
		if m.LocalPath == "" || !filepath.IsAbs(m.LocalPath) {
			t.Errorf("expected absolute LocalPath, got %q", m.LocalPath)
		}
		if m.FriendlyName != m.Filename || m.Size != 4 {
			t.Errorf("unexpected media: %+v", m)
		}
	}
}

func TestScanEmptyAndMissingDir(t *testing.T) {
	cats, err := Scan(&config.Settings{ImportDir: t.TempDir(), Quiet: 2})
	if err != nil || cats != nil {
		t.Errorf("expected nil result for empty dir, got %v, %v", cats, err)
	}
	if _, err := Scan(&config.Settings{ImportDir: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Error("expected error for missing directory")
	}
}
