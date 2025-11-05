package books

import (
	"crypto/md5" // #nosec G501 - MD5 used for file integrity verification, not cryptographic security
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/allejok96/jwb-go/internal/config"
	"github.com/allejok96/jwb-go/internal/downloader"
)

// Downloader implements the BookDownloader interface
type Downloader struct {
	settings        *config.Settings
	progressMutex   sync.Mutex
	downloadedBytes int64
	totalBytes      int64
}

// NewDownloader creates a new book downloader
func NewDownloader(s *config.Settings) *Downloader {
	return &Downloader{
		settings:        s,
		downloadedBytes: 0,
		totalBytes:      0,
	}
}

// DownloadBook downloads a book in the specified format
func (d *Downloader) DownloadBook(book *Book, format BookFormat, outputDir string) error {
	if book == nil {
		return fmt.Errorf("book cannot be nil")
	}

	// Find the file with the requested format
	var targetFile *BookFile
	for i := range book.Files {
		if book.Files[i].Format == format {
			targetFile = &book.Files[i]
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("book '%s' does not have a file in %s format", book.Title, format)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Determine output file path
	// Sanitize the filename to remove problematic characters
	safeTitle := strings.ReplaceAll(book.Title, "—", "-")
	safeTitle = strings.ReplaceAll(safeTitle, ":", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "_")

	outputPath := filepath.Join(outputDir, targetFile.Filename)
	if outputPath == filepath.Join(outputDir, "") {
		// Generate filename if not provided
		ext := d.getFileExtension(format)
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s.%s", safeTitle, ext))
	}

	// Use the existing downloader infrastructure
	if d.settings.Quiet < 1 {
		fmt.Printf("Downloading: %s -> %s\n", book.Title, outputPath)
	}

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %v", outputPath, err)
	}

	return downloader.DownloadFile(targetFile.URL, outputPath, false, d.settings.RateLimit)
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
		return fmt.Errorf("failed to create category directory: %v", err)
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
		return fmt.Errorf("failed to open file for checksum validation: %v", err)
	}
	defer func() { _ = file.Close() }()

	hash := md5.New() // #nosec G401 - MD5 used for file integrity verification, not cryptographic security
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %v", err)
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

// GetDownloadProgress returns download progress information
func (d *Downloader) GetDownloadProgress() (downloaded, total int64) {
	d.progressMutex.Lock()
	defer d.progressMutex.Unlock()
	return d.downloadedBytes, d.totalBytes
}

// SetDownloadProgress updates the download progress information
func (d *Downloader) SetDownloadProgress(downloaded, total int64) {
	d.progressMutex.Lock()
	defer d.progressMutex.Unlock()
	d.downloadedBytes = downloaded
	d.totalBytes = total
}

// ResetDownloadProgress resets the download progress to zero
func (d *Downloader) ResetDownloadProgress() {
	d.progressMutex.Lock()
	defer d.progressMutex.Unlock()
	d.downloadedBytes = 0
	d.totalBytes = 0
}
