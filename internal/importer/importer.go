// Package importer turns media files from a local directory (--import) into
// media items that the download and output pipeline can process.
package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

// mediaExts are the file extensions that are imported.
var mediaExts = map[string]bool{
	".mp4": true, ".mp3": true, ".m4a": true,
	".aac": true, ".ogg": true, ".wav": true,
}

// Scan reads the import directory and returns its media files as a single
// "imported" category. Imported media are copied into the download
// directory by the downloader. It returns nil when no media files are found.
func Scan(s *config.Settings) ([]*api.Category, error) {
	entries, err := os.ReadDir(s.ImportDir)
	if err != nil {
		return nil, fmt.Errorf("could not read import directory: %w", err)
	}

	cat := &api.Category{
		Key:  "imported",
		Name: "Imported Media",
		Home: true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !mediaExts[ext] {
			continue
		}

		fullPath, err := filepath.Abs(filepath.Join(s.ImportDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not resolve path for %s: %v\n", entry.Name(), err)
			continue
		}

		info, err := entry.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get file info for %s: %v\n", entry.Name(), err)
			continue
		}

		cat.Contents = append(cat.Contents, &api.Media{
			URL:       fullPath,
			LocalPath: fullPath,
			Name:      strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			Filename:  entry.Name(),
			// FriendlyName is used as the symlink name in filesystem mode, so
			// it must always be set, not only when --friendly is enabled.
			FriendlyName: entry.Name(),
			Size:         info.Size(),
			Date:         info.ModTime().Unix(),
		})
	}

	if len(cat.Contents) == 0 {
		return nil, nil
	}

	if s.Quiet < 1 {
		fmt.Fprintf(os.Stderr, "imported %d files from %s\n", len(cat.Contents), s.ImportDir)
	}

	return []*api.Category{cat}, nil
}
