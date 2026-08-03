package metadata

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrUnsupportedFormat is returned by Embed for file formats that cannot
// carry embedded metadata tags. Callers can fall back to a JSON sidecar.
var ErrUnsupportedFormat = errors.New("embedded metadata not supported for this file format")

// Embed writes the metadata directly into the media file at path. MP3 files
// get an ID3v2.4 tag, MP4-family files get iTunes-style metadata atoms.
// Embedding is idempotent: when the file already carries exactly the tag
// that would be written, it is left untouched. The file's modification time
// is preserved because the downloader relies on it for date-based cleanup.
func Embed(path string, meta *FileMetadata) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return embedMP3(path, meta)
	case ".mp4", ".m4a", ".m4v":
		return embedMP4(path, meta)
	default:
		return ErrUnsupportedFormat
	}
}

// album returns the value used for the album tag: the category name for
// broadcasting media, or the publication code for publication files.
func (m *FileMetadata) album() string {
	if m.CategoryName != "" {
		return m.CategoryName
	}
	return m.Publication
}

// dateTag returns the published date formatted for tags (YYYY-MM-DD).
func (m *FileMetadata) dateTag() string {
	if len(m.Published) >= 10 {
		return m.Published[:10]
	}
	return m.Published
}
