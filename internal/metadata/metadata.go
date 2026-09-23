// Package metadata writes NFO sidecar files for downloaded media and
// publication files.
//
// NFO files are the Kodi-style XML metadata format that Jellyfin and Emby read
// natively (Plex reads them with an NFO agent such as XBMCnfoMoviesImporter).
// Each file gets "<name>.nfo" next to it, so media servers pick up titles,
// release dates, runtimes and categories without the media file itself being
// modified.
package metadata

import (
	"bytes"
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
)

// FileMetadata describes a single downloaded file.
type FileMetadata struct {
	Title           string
	Filename        string
	Category        string
	CategoryName    string
	Language        string
	URL             string
	Published       time.Time
	DurationSeconds float64
	Description     string
	Publication     string
	Issue           string
	Format          string
}

// nfo is the XML document written for every file. The root element is
// <movie>, which Jellyfin, Emby and Kodi use for stand-alone videos.
type nfo struct {
	XMLName     xml.Name  `xml:"movie"`
	Title       string    `xml:"title"`
	Plot        string    `xml:"plot,omitempty"`
	Runtime     int       `xml:"runtime,omitempty"`
	Premiered   string    `xml:"premiered,omitempty"`
	ReleaseDate string    `xml:"releasedate,omitempty"`
	Year        int       `xml:"year,omitempty"`
	Genres      []string  `xml:"genre,omitempty"`
	Tags        []string  `xml:"tag,omitempty"`
	Studio      string    `xml:"studio"`
	UniqueID    *uniqueID `xml:"uniqueid,omitempty"`
}

type uniqueID struct {
	Type    string `xml:"type,attr"`
	Default bool   `xml:"default,attr"`
	Value   string `xml:",chardata"`
}

// NFOPath returns the path of the NFO sidecar for the given media filename
// inside dir: the filename with its extension replaced by ".nfo", which is
// the name media servers look for.
func NFOPath(dir, filename string) string {
	return filepath.Join(dir, strings.TrimSuffix(filename, filepath.Ext(filename))+".nfo")
}

// LegacySidecarPath returns the path of the JSON sidecar written by earlier
// versions ("<filename>.json"). It is removed when an NFO file is written.
func LegacySidecarPath(dir, filename string) string {
	return filepath.Join(dir, filename+".json")
}

// Render serializes meta as an NFO document. The output is deterministic, so
// unchanged metadata produces identical bytes.
func Render(meta *FileMetadata) ([]byte, error) {
	doc := nfo{
		Title:  meta.Title,
		Plot:   meta.Description,
		Studio: "jw.org",
	}
	if doc.Title == "" {
		doc.Title = strings.TrimSuffix(meta.Filename, filepath.Ext(meta.Filename))
	}
	if meta.DurationSeconds > 0 {
		// NFO runtimes are whole minutes; never round a short clip down to 0
		doc.Runtime = int(math.Max(1, math.Round(meta.DurationSeconds/60)))
	}
	if !meta.Published.IsZero() {
		date := meta.Published.UTC().Format("2006-01-02")
		doc.Premiered = date
		doc.ReleaseDate = date
		doc.Year = meta.Published.UTC().Year()
	}
	if meta.CategoryName != "" {
		doc.Genres = append(doc.Genres, meta.CategoryName)
	}
	for _, tag := range []string{meta.Category, meta.Publication, meta.Issue, meta.Format} {
		if tag != "" {
			doc.Tags = append(doc.Tags, tag)
		}
	}
	if meta.URL != "" {
		doc.UniqueID = &uniqueID{Type: "jworg", Default: true, Value: meta.URL}
	}

	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	out.Write(body)
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// Write stores meta as an NFO file next to filename in dir. The file is only
// rewritten when its content changes, and it is replaced atomically so media
// servers never see a half-written file. A JSON sidecar left by an earlier
// version is removed.
func Write(dir, filename string, meta *FileMetadata) error {
	data, err := Render(meta)
	if err != nil {
		return err
	}

	path := NFOPath(dir, filename)
	// #nosec G304 - path is derived from the managed download directory
	if existing, err := os.ReadFile(path); err != nil || !bytes.Equal(existing, data) {
		tmp := path + ".tmp"
		// NFO files must be readable by media servers that often run as a
		// different user, so they are world-readable like the media files.
		// #nosec G306 - metadata files are intentionally world-readable
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}

	if err := os.Remove(LegacySidecarPath(dir, filename)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// FromMedia builds metadata for a broadcasting media item belonging to the
// given category. cat may be nil when the category is unknown.
func FromMedia(lang string, cat *api.Category, m *api.Media) *FileMetadata {
	meta := &FileMetadata{
		Title:           m.Name,
		Filename:        m.Filename,
		Language:        lang,
		URL:             m.URL,
		DurationSeconds: m.Duration,
	}
	if m.LocalPath != "" {
		// Imported files have no public URL to use as an identifier
		meta.URL = ""
	}
	if cat != nil {
		meta.Category = cat.Key
		meta.CategoryName = cat.Name
	}
	if m.Date > 0 {
		meta.Published = time.Unix(m.Date, 0).UTC()
	}
	return meta
}
