package api

import (
	"testing"

	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestAssignFilenamesSameURLSharesFile(t *testing.T) {
	a := &Media{Name: "Video", URL: "https://x/v_r720P.mp4", Date: 10}
	b := &Media{Name: "Video", URL: "https://x/v_r720P.mp4", Date: 10}
	data := []*Category{{Key: "A", Contents: []interface{}{a}}, {Key: "B", Contents: []interface{}{b}}}

	AssignFilenames(&config.Settings{FriendlyFilenames: true}, data)

	if a.Filename != "Video.mp4" || b.Filename != "Video.mp4" {
		t.Errorf("same media in two categories got %q and %q, want one shared file", a.Filename, b.Filename)
	}
}

func TestAssignFilenamesStableAcrossOrder(t *testing.T) {
	older := func() *Media { return &Media{Name: "X", URL: "https://x/old.mp4", Date: 100} }
	newer := func() *Media { return &Media{Name: "X", URL: "https://x/new.mp4", Date: 200} }
	s := &config.Settings{FriendlyFilenames: true}

	o1, n1 := older(), newer()
	AssignFilenames(s, []*Category{{Contents: []interface{}{o1, n1}}})
	o2, n2 := older(), newer()
	AssignFilenames(s, []*Category{{Contents: []interface{}{n2, o2}}})

	if o1.Filename != "X.mp4" || n1.Filename != "X (1).mp4" {
		t.Errorf("got %q / %q, want the older item to keep the plain name", o1.Filename, n1.Filename)
	}
	if o1.Filename != o2.Filename || n1.Filename != n2.Filename {
		t.Errorf("names depend on API order: %q/%q vs %q/%q", o1.Filename, n1.Filename, o2.Filename, n2.Filename)
	}
}

func TestAssignFilenamesSubtitlesFollowVideo(t *testing.T) {
	noSubs := &Media{Name: "X", URL: "https://x/a.mp4", Date: 1}
	withSubs := &Media{Name: "X", URL: "https://x/b.mp4", SubtitleURL: "https://x/b.vtt", Date: 2}
	plain := &Media{Name: "Y", URL: "https://x/pk_E_01_r720P.mp4", SubtitleURL: "https://x/pk_E_01.vtt", Date: 3}

	AssignFilenames(&config.Settings{FriendlyFilenames: true}, []*Category{{Contents: []interface{}{noSubs, withSubs}}})
	AssignFilenames(&config.Settings{}, []*Category{{Contents: []interface{}{plain}}})

	if noSubs.SubtitleFilename != "" {
		t.Errorf("media without subtitles got %q", noSubs.SubtitleFilename)
	}
	if withSubs.Filename != "X (1).mp4" || withSubs.SubtitleFilename != "X (1).vtt" {
		t.Errorf("got %q with subtitle %q, want matching names", withSubs.Filename, withSubs.SubtitleFilename)
	}
	if plain.SubtitleFilename != "pk_E_01_r720P.vtt" {
		t.Errorf("subtitle = %q, want it named after the video", plain.SubtitleFilename)
	}
}

func TestAssignFilenamesReservesImportedNames(t *testing.T) {
	imported := &Media{Name: "song", Filename: "Song.mp3", LocalPath: "/import/Song.mp3"}
	indexed := &Media{Name: "song", URL: "https://x/a.mp3"}

	AssignFilenames(&config.Settings{FriendlyFilenames: true}, []*Category{{Contents: []interface{}{indexed, imported}}})

	if imported.Filename != "Song.mp3" {
		t.Errorf("imported filename changed to %q", imported.Filename)
	}
	if indexed.Filename != "song (1).mp3" {
		t.Errorf("indexed filename = %q, want it to avoid the imported name", indexed.Filename)
	}
}

func TestAssignFilenamesFallsBackToURLForUnusableTitle(t *testing.T) {
	m := &Media{Name: "..", URL: "https://x/file.mp4"}
	AssignFilenames(&config.Settings{FriendlyFilenames: true}, []*Category{{Contents: []interface{}{m}}})
	if m.Filename != "file.mp4" {
		t.Errorf("Filename = %q, want URL basename", m.Filename)
	}
}
