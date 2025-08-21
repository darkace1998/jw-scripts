package books

import (
	"testing"
	"time"

	"github.com/allejok96/jwb-go/internal/config"
)

func TestBookFormats(t *testing.T) {
	if FormatPDF != "pdf" {
		t.Errorf("Expected FormatPDF to be 'pdf', got '%s'", FormatPDF)
	}
	if FormatEPUB != "epub" {
		t.Errorf("Expected FormatEPUB to be 'epub', got '%s'", FormatEPUB)
	}
}

func TestClient(t *testing.T) {
	settings := &config.Settings{
		Lang: "E",
	}
	
	client := NewClient(settings)
	
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	// Test API availability
	if client.IsBookAPIAvailable() {
		t.Error("Expected IsBookAPIAvailable to return false for current API")
	}
	
	// Test supported formats
	formats := client.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 supported formats, got %d", len(formats))
	}
	
	// Test limitations message
	limitations := client.GetAPILimitations()
	if limitations == "" {
		t.Error("GetAPILimitations returned empty string")
	}
}

func TestClientMethods(t *testing.T) {
	settings := &config.Settings{Lang: "E"}
	client := NewClient(settings)
	
	// All these should return errors since API is not available
	_, err := client.GetCategories("E")
	if err == nil {
		t.Error("Expected GetCategories to return error")
	}
	
	_, err = client.GetCategory("E", "test")
	if err == nil {
		t.Error("Expected GetCategory to return error")
	}
	
	_, err = client.GetBook("E", "test")
	if err == nil {
		t.Error("Expected GetBook to return error")
	}
	
	_, err = client.SearchBooks("E", "test")
	if err == nil {
		t.Error("Expected SearchBooks to return error")
	}
}

func TestDownloader(t *testing.T) {
	settings := &config.Settings{
		Quiet: 2, // Suppress output during tests
	}
	
	downloader := NewDownloader(settings)
	if downloader == nil {
		t.Fatal("NewDownloader returned nil")
	}
	
	// Test download progress
	downloaded, total := downloader.GetDownloadProgress()
	if downloaded != 0 || total != 0 {
		t.Errorf("Expected progress to be 0,0 but got %d,%d", downloaded, total)
	}
}

func TestDownloadBook(t *testing.T) {
	settings := &config.Settings{Quiet: 2}
	downloader := NewDownloader(settings)
	
	// Test with nil book
	err := downloader.DownloadBook(nil, FormatPDF, "/tmp")
	if err == nil {
		t.Error("Expected DownloadBook with nil book to return error")
	}
	
	// Test with book that has no files in requested format
	book := &Book{
		Title: "Test Book",
		Files: []BookFile{
			{Format: FormatEPUB, URL: "test.epub"},
		},
	}
	
	err = downloader.DownloadBook(book, FormatPDF, "/tmp")
	if err == nil {
		t.Error("Expected DownloadBook to return error when format not available")
	}
}

func TestDownloadCategory(t *testing.T) {
	settings := &config.Settings{Quiet: 2}
	downloader := NewDownloader(settings)
	
	// Test with nil category
	err := downloader.DownloadCategory(nil, FormatPDF, "/tmp")
	if err == nil {
		t.Error("Expected DownloadCategory with nil category to return error")
	}
	
	// Test with empty category
	category := &BookCategory{
		Key:   "test",
		Name:  "Test Category",
		Books: []Book{},
	}
	
	err = downloader.DownloadCategory(category, FormatPDF, "/tmp")
	if err != nil {
		t.Errorf("Expected DownloadCategory with empty category to succeed, got error: %v", err)
	}
}

func TestBookModels(t *testing.T) {
	// Test Book model
	book := Book{
		ID:          "test-id",
		Title:       "Test Book", 
		Description: "A test book",
		Category:    "test-category",
		Language:    "E",
		Published:   time.Now(),
		Files: []BookFile{
			{
				Format:   FormatPDF,
				URL:      "https://example.com/book.pdf",
				Size:     1024000,
				Checksum: "abc123",
				Filename: "test-book.pdf",
			},
		},
	}
	
	if book.ID != "test-id" {
		t.Errorf("Expected book ID 'test-id', got '%s'", book.ID)
	}
	
	if len(book.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(book.Files))
	}
	
	if book.Files[0].Format != FormatPDF {
		t.Errorf("Expected PDF format, got '%s'", book.Files[0].Format)
	}
}

func TestBookCategory(t *testing.T) {
	category := BookCategory{
		Key:         "bible-study",
		Name:        "Bible Study",
		Description: "Books for Bible study",
		Books: []Book{
			{ID: "book1", Title: "Book 1"},
			{ID: "book2", Title: "Book 2"},
		},
	}
	
	if len(category.Books) != 2 {
		t.Errorf("Expected 2 books in category, got %d", len(category.Books))
	}
	
	if category.Key != "bible-study" {
		t.Errorf("Expected category key 'bible-study', got '%s'", category.Key)
	}
}