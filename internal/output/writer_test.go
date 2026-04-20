package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestPlaylistEntry(t *testing.T) {
	entry := PlaylistEntry{
		Name:     "Test Video",
		Source:   "http://example.com/video.mp4",
		Duration: 120,
	}

	if entry.Name != "Test Video" {
		t.Errorf("Expected Name to be 'Test Video', got %s", entry.Name)
	}
	if entry.Source != "http://example.com/video.mp4" {
		t.Errorf("Expected Source to be 'http://example.com/video.mp4', got %s", entry.Source)
	}
	if entry.Duration != 120 {
		t.Errorf("Expected Duration to be 120, got %d", entry.Duration)
	}
}

func TestStdoutWriter(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	settings := &config.Settings{}
	writer := NewStdoutWriter(settings)

	entry := PlaylistEntry{
		Name:   "Test Entry",
		Source: "http://example.com/test.mp4",
	}

	writer.Add(entry)
	if err := writer.Dump(); err != nil {
		t.Errorf("Unexpected error from Dump(): %v", err)
	}

	// Restore stdout and read captured output
	_ = w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "http://example.com/test.mp4") {
		t.Errorf("Expected output to contain URL, got: %s", output)
	}
}

func TestCommandWriter(_ *testing.T) {
	settings := &config.Settings{
		Command: []string{"echo", "test"},
	}
	writer := NewCommandWriter(settings)

	entry := PlaylistEntry{
		Name:   "Test Entry",
		Source: "http://example.com/test.mp4",
	}

	writer.Add(entry)
	// Note: We don't test Dump() here as it would actually execute the command
	// This test just verifies the writer can be created and entries added
}

func TestCreateOutputUsesDefaultFilenameForTextModes(t *testing.T) {
	dir := t.TempDir()
	settings := &config.Settings{
		Mode:    "txt",
		WorkDir: dir,
		SubDir:  "jwb-E",
	}

	data := []*api.Category{
		{
			Key:  "VideoOnDemand",
			Name: "Video on Demand",
			Contents: []interface{}{
				&api.Media{
					Name: "Test Video",
					URL:  "https://example.com/test.mp4",
				},
			},
		},
	}

	if err := CreateOutput(settings, data); err != nil {
		t.Fatalf("CreateOutput() returned error: %v", err)
	}

	if settings.OutputFilename != "playlist.txt" {
		t.Fatalf("expected default output filename playlist.txt, got %q", settings.OutputFilename)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	content, err := os.ReadFile(filepath.Join(dir, "playlist.txt"))
	if err != nil {
		t.Fatalf("expected output file to be created: %v", err)
	}
	if !strings.Contains(string(content), "https://example.com/test.mp4") {
		t.Fatalf("expected output file to contain media URL, got: %s", content)
	}
}
