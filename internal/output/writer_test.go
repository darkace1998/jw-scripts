package output

import (
	"os"
	"strings"
	"testing"

	"github.com/allejok96/jwb-go/internal/config"
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

func TestCommandWriter(t *testing.T) {
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
