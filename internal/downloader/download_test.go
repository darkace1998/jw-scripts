package downloader

import (
	"bytes"
	"crypto/md5" // #nosec G501 - test checksums match the API format
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

var content = []byte("0123456789")

// rangeServer serves content with support for range requests.
func rangeServer(t *testing.T, data []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.ServeContent(w, r, "f.mp4", time.Unix(0, 0), bytes.NewReader(data))
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

func md5Hex(b []byte) string {
	sum := md5.Sum(b) // #nosec G401 - test checksum
	return hex.EncodeToString(sum[:])
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path) // #nosec G304 - test temp dir
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDownloadMediaFinishesCompletePartFile(t *testing.T) {
	srv, _ := rangeServer(t, content)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.mp4.part"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	m := &api.Media{URL: srv.URL + "/f.mp4", Filename: "f.mp4", Size: int64(len(content))}
	if err := downloadMedia(&config.Settings{Quiet: 2}, m, dir); err != nil {
		t.Fatalf("a complete .part file must not fail with 416: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "f.mp4")); !bytes.Equal(got, content) {
		t.Errorf("got %q", got)
	}
}

func TestDownloadMediaRestartsStalePartFile(t *testing.T) {
	srv, _ := rangeServer(t, content)
	dir := t.TempDir()
	// Larger than the file on the server: the range cannot be satisfied
	if err := os.WriteFile(filepath.Join(dir, "f.mp4.part"), []byte("stale data that is too long"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := &api.Media{URL: srv.URL + "/f.mp4", Filename: "f.mp4", Size: int64(len(content))}
	if err := downloadMedia(&config.Settings{Quiet: 2}, m, dir); err != nil {
		t.Fatalf("downloadMedia() returned error: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "f.mp4")); !bytes.Equal(got, content) {
		t.Errorf("got %q", got)
	}
}

func TestDownloadMediaResumesPartialFile(t *testing.T) {
	srv, _ := rangeServer(t, content)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.mp4.part"), content[:4], 0o600); err != nil {
		t.Fatal(err)
	}

	m := &api.Media{URL: srv.URL + "/f.mp4", Filename: "f.mp4", Size: int64(len(content)), MD5: md5Hex(content), Date: 1700000000}
	if err := downloadMedia(&config.Settings{Quiet: 2, Checksums: true}, m, dir); err != nil {
		t.Fatalf("downloadMedia() returned error: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "f.mp4")); !bytes.Equal(got, content) {
		t.Errorf("got %q", got)
	}
	if fi, _ := os.Stat(filepath.Join(dir, "f.mp4")); fi.ModTime().Unix() != 1700000000 {
		t.Errorf("modification time not set from the publication date: %v", fi.ModTime())
	}
}

func TestDownloadMediaVerifiesFreshDownloads(t *testing.T) {
	srv, _ := rangeServer(t, content)

	cases := []struct {
		name  string
		media api.Media
		s     config.Settings
	}{
		{"size mismatch", api.Media{Size: 99}, config.Settings{}},
		{"checksum mismatch", api.Media{MD5: md5Hex([]byte("other"))}, config.Settings{Checksums: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			m := tc.media
			m.URL, m.Filename = srv.URL+"/f.mp4", "f.mp4"
			s := tc.s
			s.Quiet = 2
			if err := downloadMedia(&s, &m, dir); err == nil {
				t.Fatal("expected verification error")
			}
			for _, name := range []string{"f.mp4", "f.mp4.part"} {
				if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
					t.Errorf("%s should not exist after a failed verification", name)
				}
			}
		})
	}
}

func TestDownloadFileAbortsStalledTransfer(t *testing.T) {
	old := idleTimeout
	idleTimeout = 100 * time.Millisecond
	t.Cleanup(func() { idleTimeout = old })

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(release); srv.Close() })

	start := time.Now()
	err := DownloadFile(srv.URL+"/f.mp4", filepath.Join(t.TempDir(), "f"), false, 0, false)
	if err == nil || !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("expected stall error, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("stalled download took too long to abort")
	}
}

func TestDownloadAllCopiesImportedFiles(t *testing.T) {
	importDir := t.TempDir()
	src := filepath.Join(importDir, "song.mp3")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	data := []*api.Category{{Key: "imported", Contents: []interface{}{
		&api.Media{Name: "song", Filename: "song.mp3", URL: src, LocalPath: src, Size: int64(len(content))},
	}}}

	s := &config.Settings{WorkDir: dir, SubDir: "jwb-music-E", Download: true, Quiet: 2}
	if err := DownloadAll(s, data); err != nil {
		t.Fatalf("DownloadAll() returned error: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "jwb-music-E", "song.mp3")); !bytes.Equal(got, content) {
		t.Errorf("imported file not copied, got %q", got)
	}
}

func TestDownloadAllReportsFailures(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	data := []*api.Category{{Contents: []interface{}{
		&api.Media{Name: "a", Filename: "a.mp4", URL: srv.URL + "/a.mp4"},
		&api.Media{Name: "b", Filename: "b.mp4", URL: srv.URL + "/b.mp4", SubtitleURL: srv.URL + "/b.vtt", SubtitleFilename: "b.vtt"},
	}}}

	s := &config.Settings{WorkDir: dir, Download: true, DownloadSubtitles: true, Quiet: 2}
	err := DownloadAll(s, data)
	if err == nil {
		t.Fatal("expected an error when downloads fail")
	}
	for _, want := range []string{"2 of 2 downloads failed", "1 subtitle downloads failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestDownloadAllCreatesReadableDirectories(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, SubDir: "jwb-E", Quiet: 2}
	if err := DownloadAll(s, nil); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, "jwb-E"))
	if err != nil {
		t.Fatal(err)
	}
	if want := os.FileMode(0o755 &^ currentUmask()); fi.Mode().Perm() != want {
		t.Errorf("download directory mode %v is not readable by others", fi.Mode().Perm())
	}
}

func TestDiskCleanupWithoutMP4sSkipsInsteadOfFailing(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{Quiet: 2, KeepFree: 1 << 62}
	err := diskCleanup(s, dir, &api.Media{Size: 10, Date: 100})
	if diskLimitReason(err) == "" {
		t.Errorf("expected a skip reason, got %v", err)
	}
}

func TestRemoveMediaFileRemovesCompanions(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"v.mp4", "v.vtt", "v.nfo", "v.mp4.json", "other.nfo"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := removeMediaFile(dir, "v.mp4"); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "other.nfo" {
		t.Errorf("unexpected remaining files: %v", entries)
	}
}

func TestContentRangeParsing(t *testing.T) {
	if total, ok := contentRangeTotal("bytes */1234"); !ok || total != 1234 {
		t.Errorf("contentRangeTotal = %d, %v", total, ok)
	}
	if _, ok := contentRangeTotal("bytes 0-1/*"); ok {
		t.Error("unknown total should not parse")
	}
	if first, ok := contentRangeStart("bytes 100-199/1234"); !ok || first != 100 {
		t.Errorf("contentRangeStart = %d, %v", first, ok)
	}
}

func TestCheckMD5IsCaseInsensitive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	ok, err := CheckMD5(path, strings.ToUpper(md5Hex(content)))
	if err != nil || !ok {
		t.Errorf("CheckMD5 with upper-case checksum = %v, %v", ok, err)
	}
}
