package books

import (
	"os"
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
	if FormatMP3 != "mp3" {
		t.Errorf("Expected FormatMP3 to be 'mp3', got '%s'", FormatMP3)
	}
	if FormatMP4 != "mp4" {
		t.Errorf("Expected FormatMP4 to be 'mp4', got '%s'", FormatMP4)
	}
	if FormatRTF != "rtf" {
		t.Errorf("Expected FormatRTF to be 'rtf', got '%s'", FormatRTF)
	}
	if FormatBRL != "brl" {
		t.Errorf("Expected FormatBRL to be 'brl', got '%s'", FormatBRL)
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

	// Test API availability - now should return true!
	if !client.IsBookAPIAvailable() {
		t.Error("Expected IsBookAPIAvailable to return true for working API")
	}

	// Test supported formats
	formats := client.GetSupportedFormats()
	if len(formats) != 6 {
		t.Errorf("Expected 6 supported formats, got %d", len(formats))
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

	// Test supported languages
	languages, err := client.GetSupportedLanguages()
	if err != nil {
		t.Errorf("GetSupportedLanguages returned error: %v", err)
	}
	if len(languages) < 20 {
		t.Errorf("Expected at least 20 supported languages, got %d", len(languages))
	}

	// Check that English is in the list
	foundEnglish := false
	for _, lang := range languages {
		if lang.Code == "E" && lang.Name == "English" {
			foundEnglish = true
			break
		}
	}
	if !foundEnglish {
		t.Error("English language not found in supported languages")
	}

	// These should now work with the real API!
	categories, err := client.GetCategories("E")
	if err != nil {
		t.Errorf("GetCategories returned error: %v", err)
	}
	if len(categories) == 0 {
		t.Error("Expected GetCategories to return some categories")
	}

	// Test getting a specific category
	_, err = client.GetCategory("E", "bible")
	if err != nil {
		t.Errorf("GetCategory for 'bible' returned error: %v", err)
	}

	// Test getting a book (this might fail if network is down, so we don't fail the test)
	_, err = client.GetBook("E", "es25")
	if err != nil {
		t.Logf("GetBook returned error (may be expected if network issues): %v", err)
	}

	// Test search
	books, err := client.SearchBooks("E", "daily")
	if err != nil {
		t.Errorf("SearchBooks returned error: %v", err)
	}
	if len(books) == 0 {
		t.Log("SearchBooks returned no results (may be expected)")
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

	// Test progress tracking
	downloader.SetDownloadProgress(500, 1000)
	downloaded, total = downloader.GetDownloadProgress()
	if downloaded != 500 || total != 1000 {
		t.Errorf("Expected progress to be 500,1000 but got %d,%d", downloaded, total)
	}

	// Test progress reset
	downloader.ResetDownloadProgress()
	downloaded, total = downloader.GetDownloadProgress()
	if downloaded != 0 || total != 0 {
		t.Errorf("Expected progress to be 0,0 after reset but got %d,%d", downloaded, total)
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

	// Test file extension function
	extensions := map[BookFormat]string{
		FormatPDF:  "pdf",
		FormatEPUB: "epub",
		FormatMP3:  "mp3",
		FormatMP4:  "mp4",
		FormatRTF:  "rtf",
		FormatBRL:  "brl",
	}

	for format, expectedExt := range extensions {
		ext := downloader.getFileExtension(format)
		if ext != expectedExt {
			t.Errorf("Expected extension '%s' for format '%s', got '%s'", expectedExt, format, ext)
		}
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

func TestChecksumValidation(t *testing.T) {
	settings := &config.Settings{Quiet: 2}
	downloader := NewDownloader(settings)

	// Create a temporary file for testing
	tmpFile := "/tmp/test_checksum_file.txt"
	content := "Hello, World!"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpFile); removeErr != nil {
			t.Logf("Warning: failed to remove test file: %v", removeErr)
		}
	}()

	// Test with empty checksum (should pass)
	err = downloader.ValidateChecksum(tmpFile, "")
	if err != nil {
		t.Errorf("ValidateChecksum with empty checksum should pass, got error: %v", err)
	}

	// Test with correct MD5 checksum (MD5 of "Hello, World!" is 65a8e27d8879283831b664bd8b7f0ad4)
	correctChecksum := "65a8e27d8879283831b664bd8b7f0ad4"
	err = downloader.ValidateChecksum(tmpFile, correctChecksum)
	if err != nil {
		t.Errorf("ValidateChecksum with correct checksum failed: %v", err)
	}

	// Test with incorrect checksum
	incorrectChecksum := "wrongchecksum"
	err = downloader.ValidateChecksum(tmpFile, incorrectChecksum)
	if err == nil {
		t.Error("ValidateChecksum with incorrect checksum should fail")
	}

	// Test with non-existent file
	err = downloader.ValidateChecksum("/tmp/nonexistent.txt", correctChecksum)
	if err == nil {
		t.Error("ValidateChecksum with non-existent file should fail")
	}
}
