package metadata

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
)

func TestNFOPath(t *testing.T) {
	got := NFOPath(filepath.Join("some", "dir"), "video.name.mp4")
	want := filepath.Join("some", "dir", "video.name.nfo")
	if got != want {
		t.Errorf("NFOPath() = %q, want %q", got, want)
	}
}

func TestRenderProducesMovieNFO(t *testing.T) {
	data, err := Render(&FileMetadata{
		Title:           "Title <with> & special \"chars\"",
		Filename:        "video.mp4",
		Category:        "VODStudio",
		CategoryName:    "Studio",
		URL:             "https://example.com/video.mp4",
		Published:       time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC),
		DurationSeconds: 150,
	})
	if err != nil {
		t.Fatalf("Render() returned error: %v", err)
	}
	text := string(data)
	if !strings.HasPrefix(text, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`) {
		t.Errorf("missing XML declaration:\n%s", text)
	}

	var parsed nfo
	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("NFO is not valid XML: %v\n%s", err, text)
	}
	if parsed.XMLName.Local != "movie" {
		t.Errorf("root element = %q, want movie", parsed.XMLName.Local)
	}
	if parsed.Title != "Title <with> & special \"chars\"" {
		t.Errorf("title = %q", parsed.Title)
	}
	if parsed.Premiered != "2023-11-14" || parsed.ReleaseDate != "2023-11-14" || parsed.Year != 2023 {
		t.Errorf("unexpected dates: %q %q %d", parsed.Premiered, parsed.ReleaseDate, parsed.Year)
	}
	if parsed.Runtime != 3 {
		t.Errorf("runtime = %d, want 3 minutes", parsed.Runtime)
	}
	if len(parsed.Genres) != 1 || parsed.Genres[0] != "Studio" {
		t.Errorf("genres = %v", parsed.Genres)
	}
	if parsed.UniqueID == nil || parsed.UniqueID.Value != "https://example.com/video.mp4" {
		t.Errorf("uniqueid = %+v", parsed.UniqueID)
	}
}

func TestRenderShortClipHasOneMinuteRuntime(t *testing.T) {
	data, err := Render(&FileMetadata{Title: "Clip", DurationSeconds: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<runtime>1</runtime>") {
		t.Errorf("expected 1 minute runtime:\n%s", data)
	}
}

func TestWriteCreatesReadableNFOAndRemovesLegacySidecar(t *testing.T) {
	dir := t.TempDir()
	legacy := LegacySidecarPath(dir, "video.mp4")
	if err := os.WriteFile(legacy, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Write(dir, "video.mp4", &FileMetadata{Title: "Test Video", Filename: "video.mp4"}); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	path := filepath.Join(dir, "video.nfo")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("NFO not created: %v", err)
	}
	if fi.Mode().Perm()&0o044 != 0o044 {
		t.Errorf("NFO mode %v is not readable by group/others", fi.Mode().Perm())
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("expected legacy JSON sidecar to be removed")
	}
}

func TestWriteSkipsUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	meta := &FileMetadata{Title: "Same", Filename: "a.mp3"}
	if err := Write(dir, "a.mp3", meta); err != nil {
		t.Fatal(err)
	}
	path := NFOPath(dir, "a.mp3")
	old := time.Unix(1000, 0)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	if err := Write(dir, "a.mp3", meta); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	if !fi.ModTime().Equal(old) {
		t.Error("expected unchanged NFO not to be rewritten")
	}

	if err := Write(dir, "a.mp3", &FileMetadata{Title: "Changed", Filename: "a.mp3"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path) // #nosec G304 - test temp dir
	if !strings.Contains(string(data), "Changed") {
		t.Error("expected changed metadata to be written")
	}
}

func TestFromMedia(t *testing.T) {
	cat := &api.Category{Key: "VideoOnDemand", Name: "Video on Demand"}
	m := &api.Media{
		Name:     "Test",
		Filename: "test.mp4",
		URL:      "https://example.com/test.mp4",
		Date:     1700000000,
		Duration: 12.5,
	}

	meta := FromMedia("E", cat, m)

	if meta.Title != "Test" || meta.Filename != "test.mp4" {
		t.Errorf("unexpected title/filename: %q/%q", meta.Title, meta.Filename)
	}
	if meta.Category != "VideoOnDemand" || meta.CategoryName != "Video on Demand" {
		t.Errorf("unexpected category: %q/%q", meta.Category, meta.CategoryName)
	}
	if !meta.Published.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("unexpected published date: %v", meta.Published)
	}
	if meta.DurationSeconds != 12.5 || meta.URL != m.URL {
		t.Errorf("unexpected details: %+v", meta)
	}
}

func TestFromMediaImportedAndNilCategory(t *testing.T) {
	m := &api.Media{Name: "Test", Filename: "test.mp4", URL: "/local/test.mp4", LocalPath: "/local/test.mp4"}
	meta := FromMedia("E", nil, m)
	if meta.Category != "" || meta.CategoryName != "" {
		t.Error("expected empty category for nil category")
	}
	if !meta.Published.IsZero() {
		t.Error("expected no published date when media has no date")
	}
	if meta.URL != "" {
		t.Error("expected no URL for imported media")
	}
}
