package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
)

func TestSidecarPath(t *testing.T) {
	got := SidecarPath(filepath.Join("some", "dir"), "video.mp4")
	want := filepath.Join("some", "dir", "video.mp4.json")
	if got != want {
		t.Errorf("SidecarPath() = %q, want %q", got, want)
	}
}

func TestWriteCreatesValidJSON(t *testing.T) {
	dir := t.TempDir()
	meta := &FileMetadata{
		Title:    "Test Video",
		Filename: "video.mp4",
		URL:      "https://example.com/video.mp4",
	}

	if err := Write(dir, "video.mp4", meta); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	data, err := os.ReadFile(filepath.Join(dir, "video.mp4.json"))
	if err != nil {
		t.Fatalf("sidecar file not created: %v", err)
	}

	var parsed FileMetadata
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("sidecar is not valid JSON: %v", err)
	}
	if parsed.Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", parsed.Title)
	}
	if parsed.Source != "jw.org" {
		t.Errorf("expected default source 'jw.org', got %q", parsed.Source)
	}
	if parsed.GeneratedAt == "" {
		t.Error("expected generatedAt to be set")
	}
}

func TestWriteOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, "a.mp3", &FileMetadata{Title: "old"}); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if err := Write(dir, "a.mp3", &FileMetadata{Title: "new"}); err != nil {
		t.Fatalf("second Write() returned error: %v", err)
	}

	data, err := os.ReadFile(SidecarPath(dir, "a.mp3"))
	if err != nil {
		t.Fatalf("sidecar file not readable: %v", err)
	}
	var parsed FileMetadata
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("sidecar is not valid JSON: %v", err)
	}
	if parsed.Title != "new" {
		t.Errorf("expected overwritten title 'new', got %q", parsed.Title)
	}
}

func TestFromMedia(t *testing.T) {
	cat := &api.Category{Key: "VideoOnDemand", Name: "Video on Demand"}
	m := &api.Media{
		Name:             "Test",
		Filename:         "test.mp4",
		URL:              "https://example.com/test.mp4",
		Date:             1700000000,
		Duration:         12.5,
		Size:             1024,
		MD5:              "abc123",
		SubtitleURL:      "https://example.com/test.vtt",
		SubtitleFilename: "test.vtt",
	}

	meta := FromMedia("E", cat, m)

	if meta.Title != "Test" || meta.Filename != "test.mp4" {
		t.Errorf("unexpected title/filename: %q/%q", meta.Title, meta.Filename)
	}
	if meta.Category != "VideoOnDemand" || meta.CategoryName != "Video on Demand" {
		t.Errorf("unexpected category: %q/%q", meta.Category, meta.CategoryName)
	}
	if meta.Language != "E" {
		t.Errorf("unexpected language: %q", meta.Language)
	}
	if meta.Published != "2023-11-14T22:13:20Z" {
		t.Errorf("unexpected published date: %q", meta.Published)
	}
	if meta.DurationSeconds != 12.5 || meta.SizeBytes != 1024 || meta.ChecksumMD5 != "abc123" {
		t.Errorf("unexpected file details: %v", meta)
	}
	if meta.SubtitleURL == "" || meta.SubtitleFilename == "" {
		t.Error("expected subtitle info to be carried over")
	}
}

func TestFromMediaNilCategoryAndNoDate(t *testing.T) {
	m := &api.Media{Name: "Test", Filename: "test.mp4"}
	meta := FromMedia("E", nil, m)
	if meta.Category != "" || meta.CategoryName != "" {
		t.Error("expected empty category for nil category")
	}
	if meta.Published != "" {
		t.Error("expected empty published date when media has no date")
	}
}
