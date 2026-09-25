package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestFilesystemSanitizesNamesAndContinues(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{Quiet: 2, WorkDir: dir, SubDir: "jwb-E", Mode: "filesystem"}
	data := []*api.Category{
		{Key: "A", Name: "Music / Songs", Home: true, Contents: []interface{}{&api.Category{Key: "Sub", Name: "../escape"}}},
		{Key: "B", Name: "..", Home: true},
		{Key: "C", Name: "Other", Home: true},
	}

	if err := CreateOutput(s, data); err != nil {
		t.Fatalf("CreateOutput() returned error: %v", err)
	}
	for _, link := range []string{"Music  Songs", "B", "Other", filepath.Join("jwb-E", "A", "escape")} {
		if _, err := os.Lstat(filepath.Join(dir, link)); err != nil {
			t.Errorf("expected symlink %s: %v", link, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(dir), "escape")); err == nil {
		t.Error("symlink escaped the work directory")
	}
}

func TestM3UEntriesStayOnOneLine(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{Quiet: 2, WorkDir: dir, Mode: "m3u", OutputFilename: "p.m3u"}
	data := []*api.Category{{Key: "A", Contents: []interface{}{
		&api.Media{Name: "Title\nhttp://evil/x", URL: "http://ok/y"},
	}}}

	if err := CreateOutput(s, data); err != nil {
		t.Fatal(err)
	}
	// #nosec G304 - test temp dir
	b, _ := os.ReadFile(filepath.Join(dir, "p.m3u"))
	want := "#EXTM3U\n#EXTINF:0, Title http://evil/x\nhttp://ok/y\n"
	if string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
}

func TestPlaylistPathsAreRelativeToPlaylist(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "jwb-E"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lists"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "jwb-E", "v.mp4"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s := &config.Settings{Quiet: 2, WorkDir: dir, SubDir: "jwb-E", Mode: "txt", OutputFilename: filepath.Join("lists", "p.txt")}
	data := []*api.Category{{Key: "A", Contents: []interface{}{&api.Media{Name: "V", Filename: "v.mp4", URL: "http://x/v.mp4"}}}}

	if err := CreateOutput(s, data); err != nil {
		t.Fatal(err)
	}
	// #nosec G304 - test temp dir
	b, _ := os.ReadFile(filepath.Join(dir, "lists", "p.txt"))
	if want := filepath.Join("..", "jwb-E", "v.mp4") + "\n"; string(b) != want {
		t.Errorf("got %q, want %q", b, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "lists", "p.txt.tmp")); !os.IsNotExist(err) {
		t.Error("temporary playlist file left behind")
	}
}

func TestValidateMode(t *testing.T) {
	for _, mode := range []string{"", "filesystem", "stdout", "run", "txt", "m3u", "html", "txtmulti", "m3utree"} {
		if err := ValidateMode(mode); err != nil {
			t.Errorf("ValidateMode(%q) = %v", mode, err)
		}
	}
	for _, mode := range []string{"m3uu", "file", "htmlx"} {
		if err := ValidateMode(mode); err == nil || !strings.Contains(err.Error(), "unknown mode") {
			t.Errorf("ValidateMode(%q) = %v, want error", mode, err)
		}
	}
}

func TestCommandWriterWithoutCommand(t *testing.T) {
	w := NewCommandWriter(&config.Settings{})
	w.Add(PlaylistEntry{Source: "x"})
	if err := w.Dump(); err == nil {
		t.Error("expected error without a command")
	}
}
