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
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var errs []error
	for i, targetFile := range targetFiles {
		if err := d.downloadBookFile(book, targetFile, format, outputDir, i); err != nil {
			errs = append(errs, fmt.Errorf("'%s' file %d/%d: %w", book.Title, i+1, len(targetFiles), err))
			if d.settings.Quiet < 2 {
				fmt.Printf("Failed: %v\n", errs[len(errs)-1])
			}
		}
	}

	return errors.Join(errs...)
}

// downloadBookFile downloads a single file of a book, validates its checksum
// when available and optionally writes a metadata sidecar file.
func (d *Downloader) downloadBookFile(book *Book, targetFile *BookFile, format BookFormat, outputDir string, index int) error {
	filename := targetFile.Filename
	if filename == "" {
		// Generate a filename if the API did not provide one
		safeTitle := strings.ReplaceAll(book.Title, "—", "-")
		safeTitle = strings.ReplaceAll(safeTitle, ":", "_")
		safeTitle = strings.ReplaceAll(safeTitle, "/", "_")
		safeTitle = strings.ReplaceAll(safeTitle, "\\", "_")
		if index > 0 {
			safeTitle = fmt.Sprintf("%s (%d)", safeTitle, index+1)
		}
		filename = fmt.Sprintf("%s.%s", safeTitle, d.getFileExtension(format))
	}
	outputPath := filepath.Join(outputDir, filename)

	// Skip files that are already fully downloaded
	if fi, err := os.Stat(outputPath); err == nil && targetFile.Size > 0 && fi.Size() == targetFile.Size {
		if d.settings.Quiet < 1 {
			fmt.Printf("Already downloaded: %s\n", outputPath)
		}
		return d.writeMetadataIfEnabled(book, targetFile, outputDir, filename)
	}

	if d.settings.Quiet < 1 {
		fmt.Printf("Downloading: %s -> %s\n", book.Title, outputPath)
	}

	if err := downloader.DownloadFile(targetFile.URL, outputPath, false, d.settings.RateLimit); err != nil {
		return err
	}

	if targetFile.Checksum != "" {
		if err := d.ValidateChecksum(outputPath, targetFile.Checksum); err != nil {
			if removeErr := os.Remove(outputPath); removeErr != nil && d.settings.Quiet < 2 {
				fmt.Printf("Failed to remove corrupt file %s: %v\n", outputPath, removeErr)
			}
			return err
		}
	}

	return d.writeMetadataIfEnabled(book, targetFile, outputDir, filename)
}

// writeMetadataIfEnabled writes a JSON metadata sidecar for a downloaded
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
		SizeBytes:   targetFile.Size,
		ChecksumMD5: targetFile.Checksum,
		Format:      string(targetFile.Format),
		Publication: book.ID,
		Issue:       book.Issue,
	}
	if err := metadata.Write(outputDir, filename, meta); err != nil {
		return fmt.Errorf("failed to write metadata for %s: %w", filename, err)
	}
	return nil
}

// DownloadCategory downloads all books in a category
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
	if err := os.MkdirAll(categoryDir, 0o750); err != nil {
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
				fmt.Printf("Failed to download '%s': %v\n", book.Title, err)
			}
		} else {
			successCount++
		}
	}

	if d.settings.Quiet < 1 {
		fmt.Printf("Category '%s' download complete: %d successful, %d failed\n",
			category.Name, successCount, errorCount)
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
