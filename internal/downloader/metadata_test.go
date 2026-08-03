package downloader

import (
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

	// One file exists locally, one does not
	if err := os.WriteFile(filepath.Join(wd, "existing.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}

	data := []*api.Category{
		{
			Key:  "VideoOnDemand",
			Name: "Video on Demand",
			Contents: []interface{}{
				&api.Media{
					Name:     "Existing Video",
					Filename: "existing.mp4",
					URL:      "https://example.com/existing.mp4",
					Date:     1700000000,
					Duration: 60,
					Size:     5,
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

	sidecar := filepath.Join(wd, "existing.mp4.json")
	// #nosec G304 - path is constrained to t.TempDir() in this test
	raw, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("expected metadata sidecar for existing file: %v", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("sidecar is not valid JSON: %v", err)
	}
	if meta["title"] != "Existing Video" {
		t.Errorf("expected title 'Existing Video', got %v", meta["title"])
	}
	if meta["category"] != "VideoOnDemand" {
		t.Errorf("expected category 'VideoOnDemand', got %v", meta["category"])
	}

	if _, err := os.Stat(filepath.Join(wd, "missing.mp4.json")); !os.IsNotExist(err) {
		t.Error("did not expect metadata sidecar for missing file")
	}
}
