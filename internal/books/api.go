// Package books provides functionality for downloading JW publications.
package books

import "time"

// BookFormat represents the format of a book file
type BookFormat string

const (
	// FormatPDF represents PDF format
	FormatPDF BookFormat = "pdf"
	// FormatEPUB represents EPUB format
	FormatEPUB BookFormat = "epub"
	// FormatMP3 represents MP3 audio format
	FormatMP3 BookFormat = "mp3"
	// FormatMP4 represents MP4 video format
	FormatMP4 BookFormat = "mp4"
	// FormatRTF represents RTF text format
	FormatRTF BookFormat = "rtf"
	// FormatBRL represents Braille format
	FormatBRL BookFormat = "brl"
	// FormatUnknown represents unsupported formats
	FormatUnknown BookFormat = "unknown"
)

// Language represents a supported language
type Language struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Direction string `json:"direction"`
	Locale    string `json:"locale"`
}

// BookCategory represents a category of books
type BookCategory struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Publications []string `json:"publications"` // Publication codes for this category
	Books        []Book   `json:"books"`
}

// Book represents a publication/book item
type Book struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Language    string     `json:"language"`
	Issue       string     `json:"issue"` // For magazines and periodicals
	Published   time.Time  `json:"published"`
	Files       []BookFile `json:"files"`
}

// BookFile represents a downloadable file for a book
type BookFile struct {
	Format   BookFormat `json:"format"`
	URL      string     `json:"url"`
	Size     int64      `json:"size"`
	Checksum string     `json:"checksum"`
	Filename string     `json:"filename"`
	Title    string     `json:"title"` // Title/description for this specific file
}

// BookAPI defines the interface for book-related API operations
type BookAPI interface {
	// GetSupportedLanguages returns all supported languages
	GetSupportedLanguages() ([]Language, error)

	// GetCategories returns all available book categories
	GetCategories(lang string) ([]BookCategory, error)

	// GetCategory returns books in a specific category
	GetCategory(lang, categoryKey string) (*BookCategory, error)

	// GetBook returns details for a specific book
	GetBook(lang, bookID string) (*Book, error)

	// SearchBooks searches for books by title or content
	SearchBooks(lang, query string) ([]Book, error)

	// GetSupportedFormats returns all supported book formats
	GetSupportedFormats() []BookFormat
}

// BookDownloader defines the interface for downloading books
type BookDownloader interface {
	// DownloadBook downloads a book in the specified format
	DownloadBook(book *Book, format BookFormat, outputDir string) error

	// DownloadCategory downloads all books in a category
	DownloadCategory(category *BookCategory, format BookFormat, outputDir string) error
}
