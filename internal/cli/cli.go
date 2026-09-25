// Package cli contains helpers shared by the command-line programs.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/downloader"
	"github.com/darkace1998/jw-scripts/internal/importer"
	"github.com/darkace1998/jw-scripts/internal/output"
)

// WorkDir returns the working directory from the positional arguments (at
// most one), or def when none is given.
//
// --quiet used to take a numeric value ("-q 2"). It is now a counter ("-qq"
// or "--quiet=2"), so a leftover "-q 2" leaves "2" behind as a positional
// argument. A small number that is not an existing directory is rejected
// instead of silently being used as the download directory.
func WorkDir(args []string, def string) (string, error) {
	if len(args) == 0 {
		return def, nil
	}
	dir := args[0]
	if n, err := strconv.Atoi(dir); err == nil && n >= 0 && n <= 9 {
		if _, statErr := os.Stat(dir); statErr != nil {
			return "", fmt.Errorf("unexpected argument %q: --quiet no longer takes a separate value; use -q, -qq or --quiet=%d", dir, n)
		}
	}
	return dir, nil
}

// Process indexes the media, imports local files, downloads and writes the
// output. Failures in one step do not prevent the others from running on
// whatever could be indexed; all failures are returned together so the
// command exits with an error.
func Process(s *config.Settings, index func() ([]*api.Category, error)) error {
	var errs []error

	data, indexErr := index()
	if indexErr != nil {
		errs = append(errs, fmt.Errorf("indexing was incomplete: %w", indexErr))
	}

	// Offline import: scan the import directory for media files and add them to the data
	if s.ImportDir != "" {
		importedData, err := importer.Scan(s)
		if err != nil {
			return errors.Join(append(errs, fmt.Errorf("offline import failed: %w", err))...)
		}
		data = append(data, importedData...)
		// Keep imported filenames and resolve clashes with indexed media
		api.AssignFilenames(s, data)
	}

	if s.Download || s.DownloadSubtitles {
		if err := downloader.DownloadAll(s, data); err != nil {
			errs = append(errs, err)
		}
	}

	if s.Mode != "" {
		// An incomplete index would drop entries from a playlist that is
		// rewritten from scratch, so keep the existing file in that case.
		if indexErr != nil && output.WritesPlaylistFile(s.Mode) && !s.Append {
			if s.Quiet < 2 {
				fmt.Fprintln(os.Stderr, "not writing playlist because indexing was incomplete")
			}
		} else if err := output.CreateOutput(s, data); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
