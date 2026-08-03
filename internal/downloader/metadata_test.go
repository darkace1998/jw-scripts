package downloader

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestDownloadAllWritesMetadataForExistingFiles(t *testing.T) {
	dir := t.TempDir()
	subDir := "jwb-E"
	wd := filepath.Join(dir, subDir)
	if err := os.MkdirAll(wd, 0o750); err != nil {
		t.Fatal(err)
	}

	// An MP3 gets metadata embedded; a corrupt MP4 falls back to a JSON
	// sidecar; a missing file gets nothing.
	if err := os.WriteFile(filepath.Join(wd, "song.mp3"), []byte("\xff\xfbAUDIO"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wd, "broken.mp4"), []byte("not a real mp4"), 0o600); err != nil {
		t.Fatal(err)
	}

	data := []*api.Category{
		{
			Key:  "VideoOnDemand",
			Name: "Video on Demand",
			Contents: []interface{}{
				&api.Media{
					Name:     "Embedded Song",
					Filename: "song.mp3",
					URL:      "https://example.com/song.mp3",
					Date:     1700000000,
					Duration: 60,
				},
				&api.Media{
					Name:     "Broken Video",
					Filename: "broken.mp4",
					URL:      "https://example.com/broken.mp4",
				},
				&api.Media{
					Name:     "Missing Video",
					Filename: "missing.mp4",
					URL:      "https://example.com/missing.mp4",
				},
			},
		},
	}

	s := &config.Settings{
		WorkDir:       dir,
		SubDir:        subDir,
		Lang:          "E",
		Quiet:         2,
		WriteMetadata: true,
	}

	if err := DownloadAll(s, data); err != nil {
		t.Fatalf("DownloadAll() returned error: %v", err)
	}

	// MP3: embedded tag, no sidecar
	// #nosec G304 - path is constrained to t.TempDir() in this test
	mp3, err := os.ReadFile(filepath.Join(wd, "song.mp3"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(mp3, []byte("ID3")) {
		t.Error("expected MP3 to start with an embedded ID3 tag")
	}
	if !bytes.Contains(mp3, []byte("Embedded Song")) {
		t.Error("expected embedded title in MP3 tag")
	}
	if _, err := os.Stat(filepath.Join(wd, "song.mp3.json")); !os.IsNotExist(err) {
		t.Error("did not expect a sidecar for a successfully embedded MP3")
	}

	// Corrupt MP4: sidecar fallback
	// #nosec G304 - path is constrained to t.TempDir() in this test
	raw, err := os.ReadFile(filepath.Join(wd, "broken.mp4.json"))
	if err != nil {
		t.Fatalf("expected sidecar fallback for corrupt MP4: %v", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("sidecar is not valid JSON: %v", err)
	}
	if meta["title"] != "Broken Video" {
		t.Errorf("expected title 'Broken Video', got %v", meta["title"])
	}
	if meta["category"] != "VideoOnDemand" {
		t.Errorf("expected category 'VideoOnDemand', got %v", meta["category"])
	}

	if _, err := os.Stat(filepath.Join(wd, "missing.mp4.json")); !os.IsNotExist(err) {
		t.Error("did not expect metadata for a missing file")
	}
}

func TestDownloadAllRemovesStaleSidecarAfterEmbedding(t *testing.T) {
	dir := t.TempDir()
	subDir := "jwb-E"
	wd := filepath.Join(dir, subDir)
	if err := os.MkdirAll(wd, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wd, "song.mp3"), []byte("\xff\xfbAUDIO"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Sidecar left behind by an earlier version that wrote JSON files
	if err := os.WriteFile(filepath.Join(wd, "song.mp3.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	data := []*api.Category{
		{
			Key:  "Audio",
			Name: "Audio",
			Contents: []interface{}{
				&api.Media{Name: "Song", Filename: "song.mp3", URL: "https://example.com/song.mp3"},
			},
		},
	}

	s := &config.Settings{
		WorkDir:       dir,
		SubDir:        subDir,
		Lang:          "E",
		Quiet:         2,
		WriteMetadata: true,
	}

	if err := DownloadAll(s, data); err != nil {
		t.Fatalf("DownloadAll() returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(wd, "song.mp3.json")); !os.IsNotExist(err) {
		t.Error("expected stale sidecar to be removed after successful embedding")
	}
}

func TestCheckMediaToleratesEmbeddedMetadataGrowth(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}

	media := &api.Media{Name: "Video", Filename: "video.mp4", Size: 5, MD5: "doesnotmatch"}

	// Without embedded metadata, a size mismatch marks the file as broken
	s := &config.Settings{OverwriteBad: true, Checksums: true, Quiet: 2}
	if checkMedia(s, media, dir) {
		t.Error("expected size mismatch to mark file as broken without --metadata")
	}

	// With embedded metadata, larger-than-expected files are considered
	// complete and the checksum check is skipped
	s.WriteMetadata = true
	if !checkMedia(s, media, dir) {
		t.Error("expected grown file to be accepted with --metadata")
	}

	// A file smaller than the original download is still broken
	media.Size = 100
	if checkMedia(s, media, dir) {
		t.Error("expected too-small file to be marked broken even with --metadata")
	}
}
