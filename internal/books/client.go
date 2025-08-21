package books

import (
	"fmt"
	"net/http"

	"github.com/allejok96/jwb-go/internal/config"
)

// Client implements the BookAPI interface for JW.org book operations
type Client struct {
	baseURL    string
	httpClient *http.Client
	settings   *config.Settings
}

// NewClient creates a new book API client
func NewClient(s *config.Settings) *Client {
	return &Client{
		baseURL:    "https://data.jw-api.org/mediator/v1", // This would need to be updated when books API becomes available
		httpClient: &http.Client{},
		settings:   s,
	}
}

// GetCategories returns all available book categories
// NOTE: This is currently not supported by the JW.org API
func (c *Client) GetCategories(lang string) ([]BookCategory, error) {
	return nil, fmt.Errorf("book categories not available through current JW.org API - only broadcasting content is supported")
}

// GetCategory returns books in a specific category
// NOTE: This is currently not supported by the JW.org API
func (c *Client) GetCategory(lang, categoryKey string) (*BookCategory, error) {
	return nil, fmt.Errorf("book category '%s' not available through current JW.org API - only broadcasting content is supported", categoryKey)
}

// GetBook returns details for a specific book
// NOTE: This is currently not supported by the JW.org API
func (c *Client) GetBook(lang, bookID string) (*Book, error) {
	return nil, fmt.Errorf("book '%s' not available through current JW.org API - only broadcasting content is supported", bookID)
}

// SearchBooks searches for books by title or content
// NOTE: This is currently not supported by the JW.org API
func (c *Client) SearchBooks(lang, query string) ([]Book, error) {
	return nil, fmt.Errorf("book search not available through current JW.org API - only broadcasting content is supported")
}

// GetSupportedFormats returns the book formats that would be supported
func (c *Client) GetSupportedFormats() []BookFormat {
	return []BookFormat{FormatPDF, FormatEPUB}
}

// IsBookAPIAvailable checks if the book API is currently available
func (c *Client) IsBookAPIAvailable() bool {
	return false // Currently, no book API is available
}

// GetAPILimitations returns information about current API limitations
func (c *Client) GetAPILimitations() string {
	return `Current JW.org API Limitations for Book Downloads:

1. The data.jw-api.org/mediator/v1 API only provides access to JW Broadcasting content (videos and audio)
2. No publication/book endpoints are available (/publications, /books, /library, etc. all return 404)
3. Books and publications appear to be served through different systems not accessible via this API
4. The current API structure only supports media files (.mp4, .mp3) - no PDF or EPUB content

Alternative Approaches:
- Books may be available through the Watchtower Online Library (wol.jw.org) but this requires a different API
- Mobile apps may use different endpoints for publication downloads
- Web scraping of jw.org might be possible but would violate the terms of service

This framework is designed to support book downloads when/if such functionality becomes available through the API.`
}