package books

import (
	"crypto/md5" // #nosec G501 - MD5 used for file integrity verification, not cryptographic security
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/downloader"
	"github.com/darkace1998/jw-scripts/internal/metadata"
)

// Downloader implements the BookDownloader interface
type Downloader struct {
	settings *config.Settings
}

// NewDownloader creates a new book downloader
func NewDownloader(s *config.Settings) *Downloader {
	return &Downloader{
		settings: s,
	}
}

// DownloadBook downloads all files of a book in the specified format.
// Publications can consist of multiple files in the same format (for example
// audio books with one MP3 per chapter), so every matching file is fetched.
func (d *Downloader) DownloadBook(book *Book, format BookFormat, outputDir string) error {
	if book == nil {
		return fmt.Errorf("book cannot be nil")
	}

	// Find all files with the requested format
	var targetFiles []*BookFile
	for i := range book.Files {
		if book.Files[i].Format == format {
			targetFiles = append(targetFiles, &book.Files[i])
		}
	}

	if len(targetFiles) == 0 {
		return fmt.Errorf("book '%s' does not have a file in %s format", book.Title, format)
	}

	// Create output directory if it doesn't exist
	if err := downloader.MkdirAll(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var errs []error
	for i, targetFile := range targetFiles {
		if err := d.downloadBookFile(book, targetFile, format, outputDir, i); err != nil {
			errs = append(errs, fmt.Errorf("'%s' file %d/%d: %w", book.Title, i+1, len(targetFiles), err))
			if d.settings.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "Failed: %v\n", errs[len(errs)-1])
			}
		}
	}

	return errors.Join(errs...)
}

// downloadBookFile downloads a single file of a book, validates its size and
// checksum when available and optionally writes an NFO metadata file.
func (d *Downloader) downloadBookFile(book *Book, targetFile *BookFile, format BookFormat, outputDir string, index int) error {
	filename := targetFile.Filename
	if !hasBaseName(filename) {
		// Generate a filename if the API did not provide a usable one
		title := api.FormatFilename(strings.ReplaceAll(book.Title, "—", "-"), true)
		if title == "" {
			title = api.FormatFilename(book.ID, true)
		}
		if title == "" {
			title = "publication"
		}
		if index > 0 {
			title = fmt.Sprintf("%s (%d)", title, index+1)
		}
		filename = fmt.Sprintf("%s.%s", title, d.getFileExtension(format))
	}
	outputPath := filepath.Join(outputDir, filename)

	// Skip files that are already fully downloaded
	if fi, err := os.Stat(outputPath); err == nil && d.isComplete(targetFile, outputPath, fi.Size()) {
		if d.settings.Quiet < 1 {
			fmt.Printf("Already downloaded: %s\n", outputPath)
		}
		return d.writeMetadataIfEnabled(book, targetFile, outputDir, filename)
	}

	if d.settings.Quiet < 1 {
		fmt.Printf("Downloading: %s -> %s\n", book.Title, outputPath)
	}

	// Download to a temporary file so an interrupted or corrupt download
	// never looks complete; an existing partial file is resumed.
	tmpPath := outputPath + ".part"
	_, statErr := os.Stat(tmpPath)
	resume := statErr == nil
	if err := downloader.DownloadFile(targetFile.URL, tmpPath, resume, d.settings.RateLimit, downloader.ShowProgress(d.settings.Quiet)); err != nil {
		return err
	}

	if err := d.verify(targetFile, tmpPath); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && d.settings.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "Failed to remove corrupt file %s: %v\n", tmpPath, removeErr)
		}
		return err
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return err
	}

	return d.writeMetadataIfEnabled(book, targetFile, outputDir, filename)
}

// hasBaseName reports whether a filename has a real name besides its
// extension (not just dots or spaces, which would hide or break the file).
func hasBaseName(name string) bool {
	return strings.Trim(strings.TrimSuffix(name, filepath.Ext(name)), ". ") != ""
}

// isComplete reports whether an existing file matches the expected download.
// Without a known size, the checksum decides; without either, the file is
// downloaded again.
func (d *Downloader) isComplete(targetFile *BookFile, path string, size int64) bool {
	if targetFile.Size > 0 {
		return size == targetFile.Size
	}
	if targetFile.Checksum != "" {
		return d.ValidateChecksum(path, targetFile.Checksum) == nil
	}
	return false
}

// verify checks a downloaded file against the size and checksum reported by
// the API.
func (d *Downloader) verify(targetFile *BookFile, path string) error {
	if targetFile.Size > 0 {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fi.Size() != targetFile.Size {
			return fmt.Errorf("size mismatch: expected %d bytes, got %d", targetFile.Size, fi.Size())
		}
	}
	return d.ValidateChecksum(path, targetFile.Checksum)
}

// writeMetadataIfEnabled writes an NFO metadata file next to a downloaded
// book file when metadata generation is enabled.
func (d *Downloader) writeMetadataIfEnabled(book *Book, targetFile *BookFile, outputDir, filename string) error {
	if !d.settings.WriteMetadata {
		return nil
	}

	title := targetFile.Title
	if title == "" {
		title = book.Title
	}
	meta := &metadata.FileMetadata{
		Title:       title,
		Filename:    filename,
		Language:    book.Language,
		URL:         targetFile.URL,
		Description: book.Description,
		Format:      string(targetFile.Format),
		Publication: book.ID,
		Issue:       book.Issue,
	}

	if err := metadata.Write(outputDir, filename, meta); err != nil {
		return fmt.Errorf("failed to write metadata for %s: %w", filename, err)
	}
	return nil
}

// DownloadCategory downloads all books in a category. It returns an error if
// any book failed to download.
func (d *Downloader) DownloadCategory(category *BookCategory, format BookFormat, outputDir string) error {
	if category == nil {
		return fmt.Errorf("category cannot be nil")
	}

	if len(category.Books) == 0 {
		if d.settings.Quiet < 1 {
			fmt.Printf("No books found in category: %s\n", category.Name)
		}
		return nil
	}

	// Create category subdirectory
	categoryDir := filepath.Join(outputDir, category.Key)
	if err := downloader.MkdirAll(categoryDir); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	successCount := 0
	errorCount := 0

	for i := range category.Books {
		book := &category.Books[i]

		if d.settings.Quiet < 2 {
			fmt.Printf("[%d/%d] ", i+1, len(category.Books))
		}

		if err := d.DownloadBook(book, format, categoryDir); err != nil {
			errorCount++
			if d.settings.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "Failed to download '%s': %v\n", book.Title, err)
			}
		} else {
			successCount++
		}
	}

	if d.settings.Quiet < 1 {
		fmt.Printf("Category '%s' download complete: %d successful, %d failed\n",
			category.Name, successCount, errorCount)
	}

	if errorCount > 0 {
		return fmt.Errorf("%d of %d publications in category '%s' failed to download", errorCount, len(category.Books), category.Name)
	}
	return nil
}

// ValidateChecksum validates the checksum of a downloaded file
func (d *Downloader) ValidateChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // No checksum to validate
	}

	// #nosec G304 - Path is for file checksum verification in download process
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum validation: %w", err)
	}
	defer func() { _ = file.Close() }()

	hash := md5.New() // #nosec G401 - MD5 used for file integrity verification, not cryptographic security
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	// Compare checksums case-insensitively
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	return nil
}

// getFileExtension returns the appropriate file extension for a format
func (d *Downloader) getFileExtension(format BookFormat) string {
	switch format {
	case FormatPDF:
		return "pdf"
	case FormatEPUB:
		return "epub"
	case FormatMP3:
		return "mp3"
	case FormatMP4:
		return "mp4"
	case FormatRTF:
		return "rtf"
	case FormatBRL:
		return "brl"
	default:
		return string(format)
	}
}
