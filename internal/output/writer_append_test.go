package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func makeData(urls ...string) []*api.Category {
	cat := &api.Category{Key: "VideoOnDemand", Name: "Video on Demand"}
	for i, u := range urls {
		cat.Contents = append(cat.Contents, &api.Media{
			Name: "Video " + string(rune('A'+i)),
			URL:  u,
		})
	}
	return []*api.Category{cat}
}

func TestTxtWriterAppendKeepsAndDeduplicatesEntries(t *testing.T) {
	dir := t.TempDir()
	settings := &config.Settings{
		Mode:           "txt",
		WorkDir:        dir,
		OutputFilename: "playlist.txt",
	}

	if err := CreateOutput(settings, makeData("https://example.com/a.mp4", "https://example.com/b.mp4")); err != nil {
		t.Fatalf("first CreateOutput() returned error: %v", err)
	}

	// Second run in append mode with one duplicate and one new entry
	settings.Append = true
	if err := CreateOutput(settings, makeData("https://example.com/b.mp4", "https://example.com/c.mp4")); err != nil {
		t.Fatalf("second CreateOutput() returned error: %v", err)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	content, err := os.ReadFile(filepath.Join(dir, "playlist.txt"))
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}

	got := strings.Fields(string(content))
	want := []string{
		"https://example.com/a.mp4",
		"https://example.com/b.mp4",
		"https://example.com/c.mp4",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d entries after append, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestM3uWriterAppendKeepsSingleHeader(t *testing.T) {
	dir := t.TempDir()
	settings := &config.Settings{
		Mode:           "m3u",
		WorkDir:        dir,
		OutputFilename: "playlist.m3u",
	}

	if err := CreateOutput(settings, makeData("https://example.com/a.mp4")); err != nil {
		t.Fatalf("first CreateOutput() returned error: %v", err)
	}

	settings.Append = true
	if err := CreateOutput(settings, makeData("https://example.com/b.mp4")); err != nil {
		t.Fatalf("second CreateOutput() returned error: %v", err)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	content, err := os.ReadFile(filepath.Join(dir, "playlist.m3u"))
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}

	text := string(content)
	if strings.Count(text, "#EXTM3U") != 1 {
		t.Errorf("expected exactly one #EXTM3U header, got: %s", text)
	}
	if !strings.Contains(text, "https://example.com/a.mp4") || !strings.Contains(text, "https://example.com/b.mp4") {
		t.Errorf("expected both entries in appended playlist, got: %s", text)
	}
}

func TestHTMLWriterAppendDeduplicatesEntries(t *testing.T) {
	dir := t.TempDir()
	settings := &config.Settings{
		Mode:           "html",
		WorkDir:        dir,
		OutputFilename: "playlist.html",
	}

	if err := CreateOutput(settings, makeData("https://example.com/a.mp4")); err != nil {
		t.Fatalf("first CreateOutput() returned error: %v", err)
	}

	settings.Append = true
	if err := CreateOutput(settings, makeData("https://example.com/a.mp4", "https://example.com/b.mp4")); err != nil {
		t.Fatalf("second CreateOutput() returned error: %v", err)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	content, err := os.ReadFile(filepath.Join(dir, "playlist.html"))
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}

	text := string(content)
	if strings.Count(text, "https://example.com/a.mp4") != 1 {
		t.Errorf("expected duplicate entry to be skipped, got: %s", text)
	}
	if !strings.Contains(text, "https://example.com/b.mp4") {
		t.Errorf("expected new entry to be appended, got: %s", text)
	}
	if strings.Count(text, "<!DOCTYPE html>") != 1 || strings.Count(text, "</body></html>") != 1 {
		t.Errorf("expected valid single HTML document, got: %s", text)
	}
}

func TestCleanSymlinksRemovesStaleLinks(t *testing.T) {
	dir := t.TempDir()
	settings := &config.Settings{
		Mode:             "filesystem",
		WorkDir:          dir,
		SubDir:           "jwb-E",
		CleanAllSymlinks: true,
	}

	// Simulate a stale symlink from a previous run
	dataDir := filepath.Join(dir, "jwb-E")
	catDir := filepath.Join(dataDir, "OldCategory")
	if err := os.MkdirAll(catDir, 0o750); err != nil {
		t.Fatal(err)
	}
	staleLink := filepath.Join(catDir, "Old Video.mp4")
	if err := os.Symlink(filepath.Join(dataDir, "gone.mp4"), staleLink); err != nil {
		t.Skipf("cannot create symlinks in this environment: %v", err)
	}
	staleHomeLink := filepath.Join(dir, "Old Category Name")
	if err := os.Symlink(catDir, staleHomeLink); err != nil {
		t.Skipf("cannot create symlinks in this environment: %v", err)
	}

	if err := CreateOutput(settings, []*api.Category{{Key: "VideoOnDemand", Name: "Video on Demand"}}); err != nil {
		t.Fatalf("CreateOutput() returned error: %v", err)
	}

	if _, err := os.Lstat(staleLink); !os.IsNotExist(err) {
		t.Error("expected stale symlink in data dir to be removed")
	}
	if _, err := os.Lstat(staleHomeLink); !os.IsNotExist(err) {
		t.Error("expected stale home symlink in work dir to be removed")
	}
}
