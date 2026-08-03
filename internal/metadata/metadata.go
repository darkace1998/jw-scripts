// Package metadata generates JSON sidecar metadata files for downloaded media
// and publication files.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
)

// FileMetadata describes a single downloaded file. It is serialized as a JSON
// sidecar file stored next to the file it describes.
type FileMetadata struct {
	Title            string  `json:"title"`
	Filename         string  `json:"filename"`
	Category         string  `json:"category,omitempty"`
	CategoryName     string  `json:"categoryName,omitempty"`
	Language         string  `json:"language,omitempty"`
	URL              string  `json:"url,omitempty"`
	Published        string  `json:"published,omitempty"`
	DurationSeconds  float64 `json:"durationSeconds,omitempty"`
	SizeBytes        int64   `json:"sizeBytes,omitempty"`
	ChecksumMD5      string  `json:"checksumMd5,omitempty"`
	SubtitleURL      string  `json:"subtitleUrl,omitempty"`
	SubtitleFilename string  `json:"subtitleFilename,omitempty"`
	Format           string  `json:"format,omitempty"`
	Publication      string  `json:"publication,omitempty"`
	Issue            string  `json:"issue,omitempty"`
	Source           string  `json:"source"`
	GeneratedAt      string  `json:"generatedAt"`
}

// SidecarPath returns the path of the metadata sidecar file for the given
// media filename inside dir.
func SidecarPath(dir, filename string) string {
	return filepath.Join(dir, filename+".json")
}

// Write serializes meta and stores it as a sidecar file next to filename in
// dir. Existing sidecar files are overwritten so metadata stays up to date.
func Write(dir, filename string, meta *FileMetadata) error {
	if meta.Source == "" {
		meta.Source = "jw.org"
	}
	if meta.GeneratedAt == "" {
		meta.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(SidecarPath(dir, filename), data, 0o600)
}

// FromMedia builds metadata for a broadcasting media item belonging to the
// given category. cat may be nil when the category is unknown.
func FromMedia(lang string, cat *api.Category, m *api.Media) *FileMetadata {
	meta := &FileMetadata{
		Title:            m.Name,
		Filename:         m.Filename,
		Language:         lang,
		URL:              m.URL,
		DurationSeconds:  m.Duration,
		SizeBytes:        m.Size,
		ChecksumMD5:      m.MD5,
		SubtitleURL:      m.SubtitleURL,
		SubtitleFilename: m.SubtitleFilename,
	}
	if cat != nil {
		meta.Category = cat.Key
		meta.CategoryName = cat.Name
	}
	if m.Date > 0 {
		meta.Published = time.Unix(m.Date, 0).UTC().Format(time.RFC3339)
	}
	return meta
}
