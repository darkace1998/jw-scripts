package downloader

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestDownloadAllWritesNFOForExistingFiles(t *testing.T) {
	dir := t.TempDir()
	subDir := "jwb-E"
	wd := filepath.Join(dir, subDir)
	if err := os.MkdirAll(wd, 0o750); err != nil {
		t.Fatal(err)
	}

	original := []byte("\xff\xfbAUDIO")
	// #nosec G306 - the test checks that permissions are left unchanged
	if err := os.WriteFile(filepath.Join(wd, "song.mp3"), original, 0o644); err != nil {
		t.Fatal(err)
	}
	// JSON sidecar left behind by an earlier version
	if err := os.WriteFile(filepath.Join(wd, "song.mp3.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	data := []*api.Category{
		{
			Key:  "VideoOnDemand",
			Name: "Video on Demand",
			Contents: []interface{}{
				&api.Media{
					Name:     "Song Title",
					Filename: "song.mp3",
					URL:      "https://example.com/song.mp3",
					Date:     1700000000,
					Duration: 60,
				},
				&api.Media{
					Name:     "Missing Video",
					Filename: "missing.mp4",
					URL:      "https://example.com/missing.mp4",
				},
			},
		},
	}

	s := &config.Settings{WorkDir: dir, SubDir: subDir, Lang: "E", Quiet: 2, WriteMetadata: true}
	if err := DownloadAll(s, data); err != nil {
		t.Fatalf("DownloadAll() returned error: %v", err)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	nfo, err := os.ReadFile(filepath.Join(wd, "song.nfo"))
	if err != nil {
		t.Fatalf("expected NFO file: %v", err)
	}
	for _, want := range []string{"<movie>", "<title>Song Title</title>", "<genre>Video on Demand</genre>", "<premiered>2023-11-14</premiered>"} {
		if !strings.Contains(string(nfo), want) {
			t.Errorf("NFO missing %s:\n%s", want, nfo)
		}
	}

	// The media file itself must not be modified
	// #nosec G304 - path is constrained to t.TempDir() in this test
	if content, _ := os.ReadFile(filepath.Join(wd, "song.mp3")); !bytes.Equal(content, original) {
		t.Error("media file was modified")
	}
	if fi, _ := os.Stat(filepath.Join(wd, "song.mp3")); fi.Mode().Perm() != 0o644 {
		t.Errorf("media file permissions changed to %v", fi.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(wd, "song.mp3.json")); !os.IsNotExist(err) {
		t.Error("expected legacy JSON sidecar to be removed")
	}
	if _, err := os.Stat(filepath.Join(wd, "missing.nfo")); !os.IsNotExist(err) {
		t.Error("did not expect metadata for a missing file")
	}
}
