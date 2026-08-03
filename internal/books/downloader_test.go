package books

import (
	"crypto/md5" // #nosec G501 - MD5 used for test checksums matching the API format
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/config"
)

func newTestServer(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(content))
	}))
	t.Cleanup(server.Close)
	return server
}

func md5Of(s string) string {
	sum := md5.Sum([]byte(s)) // #nosec G401 - test checksum
	return hex.EncodeToString(sum[:])
}

func TestDownloadBookFetchesAllFilesOfFormat(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"/track1.mp3": "audio-one",
		"/track2.mp3": "audio-two",
	})

	book := &Book{
		ID:       "test-pub",
		Title:    "Test Publication",
		Language: "E",
		Files: []BookFile{
			{Format: FormatMP3, URL: server.URL + "/track1.mp3", Filename: "track1.mp3", Checksum: md5Of("audio-one"), Size: int64(len("audio-one"))},
			{Format: FormatMP3, URL: server.URL + "/track2.mp3", Filename: "track2.mp3", Checksum: md5Of("audio-two"), Size: int64(len("audio-two"))},
			{Format: FormatPDF, URL: server.URL + "/book.pdf", Filename: "book.pdf"},
		},
	}

	dir := t.TempDir()
	d := NewDownloader(&config.Settings{Quiet: 2})

	if err := d.DownloadBook(book, FormatMP3, dir); err != nil {
		t.Fatalf("DownloadBook() returned error: %v", err)
	}

	for _, name := range []string{"track1.mp3", "track2.mp3"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to be downloaded: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "book.pdf")); !os.IsNotExist(err) {
		t.Error("did not expect PDF file to be downloaded in MP3 mode")
	}
}

func TestDownloadBookChecksumMismatchRemovesFile(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"/bad.mp3": "actual-content",
	})

	book := &Book{
		ID:       "test-pub",
		Title:    "Test Publication",
		Language: "E",
		Files: []BookFile{
			{Format: FormatMP3, URL: server.URL + "/bad.mp3", Filename: "bad.mp3", Checksum: md5Of("expected-content")},
		},
	}

	dir := t.TempDir()
	d := NewDownloader(&config.Settings{Quiet: 2})

	if err := d.DownloadBook(book, FormatMP3, dir); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if _, err := os.Stat(filepath.Join(dir, "bad.mp3")); !os.IsNotExist(err) {
		t.Error("expected corrupt file to be removed")
	}
}

func TestDownloadBookWritesMetadata(t *testing.T) {
	server := newTestServer(t, map[string]string{
		"/pub.pdf": "pdf-bytes",
	})

	book := &Book{
		ID:       "es25",
		Title:    "Daily Text",
		Language: "E",
		Issue:    "202601",
		Files: []BookFile{
			{Format: FormatPDF, URL: server.URL + "/pub.pdf", Filename: "pub.pdf", Title: "Daily Text 2026", Size: int64(len("pdf-bytes"))},
		},
	}

	dir := t.TempDir()
	d := NewDownloader(&config.Settings{Quiet: 2, WriteMetadata: true})

	if err := d.DownloadBook(book, FormatPDF, dir); err != nil {
		t.Fatalf("DownloadBook() returned error: %v", err)
	}

	// #nosec G304 - path is constrained to t.TempDir() in this test
	data, err := os.ReadFile(filepath.Join(dir, "pub.pdf.json"))
	if err != nil {
		t.Fatalf("expected metadata sidecar to be written: %v", err)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("metadata sidecar is not valid JSON: %v", err)
	}
	for key, want := range map[string]string{
		"title":       "Daily Text 2026",
		"filename":    "pub.pdf",
		"publication": "es25",
		"issue":       "202601",
		"format":      "pdf",
		"language":    "E",
	} {
		if got, _ := meta[key].(string); got != want {
			t.Errorf("metadata %s: expected %q, got %q", key, want, got)
		}
	}
}

func TestDownloadBookSkipsCompleteFiles(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = fmt.Fprint(w, "content")
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "done.pdf"), []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		ID:    "test",
		Title: "Test",
		Files: []BookFile{
			{Format: FormatPDF, URL: server.URL + "/done.pdf", Filename: "done.pdf", Size: int64(len("content"))},
		},
	}

	d := NewDownloader(&config.Settings{Quiet: 2})
	if err := d.DownloadBook(book, FormatPDF, dir); err != nil {
		t.Fatalf("DownloadBook() returned error: %v", err)
	}
	if requests != 0 {
		t.Errorf("expected already-complete file to be skipped, but %d requests were made", requests)
	}
}
