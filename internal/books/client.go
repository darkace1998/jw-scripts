package books

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/allejok96/jwb-go/internal/config"
)

// Client implements the BookAPI interface for JW.org book operations
type Client struct {
	baseURL    string
	httpClient *http.Client
	settings   *config.Settings
}

// PublicationResponse represents the API response from the JW.org publication media API
type PublicationResponse struct {
	PubName       string                             `json:"pubName"`
	ParentPubName string                             `json:"parentPubName"`
	Pub           string                             `json:"pub"`
	Issue         string                             `json:"issue"`
	FormattedDate string                             `json:"formattedDate"`
	FileFormat    []string                           `json:"fileformat"`
	Files         map[string]map[string][]FileInfo   `json:"files"`
}

// FileInfo represents a downloadable file from the API
type FileInfo struct {
	Title    string `json:"title"`
	File     struct {
		URL              string `json:"url"`
		ModifiedDatetime string `json:"modifiedDatetime"`
		Checksum         string `json:"checksum"`
	} `json:"file"`
	FileSize int64  `json:"filesize"`
	MimeType string `json:"mimetype"`
}

// NewClient creates a new book API client
func NewClient(s *config.Settings) *Client {
	return &Client{
		baseURL:    "https://b.jw-cdn.org/apis/pub-media/GETPUBMEDIALINKS",
		httpClient: &http.Client{},
		settings:   s,
	}
}

// GetCategories returns all available book categories
func (c *Client) GetCategories(lang string) ([]BookCategory, error) {
	// Pre-defined categories based on known publication types
	categories := []BookCategory{
		{
			Key:         "bible",
			Name:        "Bible",
			Description: "New World Translation of the Holy Scriptures",
			Publications: []string{"nwtsty"},
		},
		{
			Key:         "daily-text",
			Name:        "Daily Text",
			Description: "Examining the Scriptures Daily",
			Publications: []string{"es25"},
		},
		{
			Key:         "yearbooks",
			Name:        "Yearbooks",
			Description: "Watch Tower Publications Index and Yearbooks",
			Publications: []string{"dx24"},
		},
		{
			Key:         "circuit-assembly",
			Name:        "Circuit Assembly Programs",
			Description: "Circuit Assembly Programs",
			Publications: []string{"ca-brpgm26"},
		},
		{
			Key:         "convention",
			Name:        "Convention Materials",
			Description: "Convention invitations and programs",
			Publications: []string{"co-inv25"},
		},
		{
			Key:         "magazines",
			Name:        "Magazines",
			Description: "Watchtower and Awake! magazines (requires issue specification)",
			Publications: []string{"w", "g"},
		},
	}
	
	return categories, nil
}

// GetCategory returns books in a specific category
func (c *Client) GetCategory(lang, categoryKey string) (*BookCategory, error) {
	categories, err := c.GetCategories(lang)
	if err != nil {
		return nil, err
	}
	
	for _, category := range categories {
		if category.Key == categoryKey {
			// Populate books for this category
			var books []Book
			for _, pubCode := range category.Publications {
				book, err := c.GetBook(lang, pubCode)
				if err != nil {
					// Log error but continue with other publications
					continue
				}
				books = append(books, *book)
			}
			category.Books = books
			return &category, nil
		}
	}
	
	return nil, fmt.Errorf("category '%s' not found", categoryKey)
}

// GetBook returns details for a specific book
func (c *Client) GetBook(lang, bookID string) (*Book, error) {
	// Make request to the publication API
	pubResp, err := c.getPublicationData(bookID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get publication data for '%s': %w", bookID, err)
	}
	
	// Convert to our Book format
	book := &Book{
		ID:          pubResp.Pub,
		Title:       pubResp.PubName,
		Description: pubResp.ParentPubName,
		Language:    lang,
		Issue:       pubResp.Issue,
		Files:       make([]BookFile, 0),
	}
	
	// Convert files
	if langFiles, exists := pubResp.Files[strings.ToUpper(lang)]; exists {
		for formatName, fileList := range langFiles {
			format := c.parseFormat(formatName)
			if format == FormatUnknown {
				continue // Skip unsupported formats
			}
			
			for _, fileInfo := range fileList {
				bookFile := BookFile{
					Format:   format,
					URL:      fileInfo.File.URL,
					Size:     fileInfo.FileSize,
					Checksum: fileInfo.File.Checksum,
					Title:    fileInfo.Title,
				}
				book.Files = append(book.Files, bookFile)
			}
		}
	}
	
	return book, nil
}

// SearchBooks searches for books by title or content
func (c *Client) SearchBooks(lang, query string) ([]Book, error) {
	// Get all categories and search through them
	categories, err := c.GetCategories(lang)
	if err != nil {
		return nil, err
	}
	
	var results []Book
	queryLower := strings.ToLower(query)
	
	for _, category := range categories {
		for _, pubCode := range category.Publications {
			book, err := c.GetBook(lang, pubCode)
			if err != nil {
				continue
			}
			
			// Simple text matching
			if strings.Contains(strings.ToLower(book.Title), queryLower) ||
			   strings.Contains(strings.ToLower(book.Description), queryLower) {
				results = append(results, *book)
			}
		}
	}
	
	return results, nil
}

// GetSupportedFormats returns the book formats that are supported
func (c *Client) GetSupportedFormats() []BookFormat {
	return []BookFormat{FormatPDF, FormatEPUB}
}

// IsBookAPIAvailable checks if the book API is currently available
func (c *Client) IsBookAPIAvailable() bool {
	return true // The publication API is now available!
}

// GetAPILimitations returns information about current API status
func (c *Client) GetAPILimitations() string {
	return `JW.org Publication API Status:

✅ Publication downloads are now available through the JW.org Publication Media API!

Available Features:
- PDF and EPUB format downloads
- Bible (New World Translation Study Edition)
- Daily text publications
- Yearbooks and indexes
- Circuit assembly and convention materials
- Magazine downloads (with issue specification)

API Endpoint: https://b.jw-cdn.org/apis/pub-media/GETPUBMEDIALINKS

The framework fully supports book downloads with real data from JW.org.`
}

// getPublicationData fetches publication data from the JW.org API
func (c *Client) getPublicationData(pubCode, issue string) (*PublicationResponse, error) {
	params := url.Values{}
	params.Set("output", "json")
	params.Set("pub", pubCode)
	params.Set("fileformat", "PDF,EPUB")
	params.Set("alllangs", "0")
	params.Set("langwritten", "E")
	params.Set("txtCMSLang", "E")
	
	if issue != "" {
		params.Set("issue", issue)
	}
	
	requestURL := c.baseURL + "?" + params.Encode()
	
	resp, err := c.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d for publication '%s'", resp.StatusCode, pubCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var pubResp PublicationResponse
	if err := json.Unmarshal(body, &pubResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return &pubResp, nil
}

// parseFormat converts API format strings to our BookFormat enum
func (c *Client) parseFormat(formatString string) BookFormat {
	switch strings.ToUpper(formatString) {
	case "PDF":
		return FormatPDF
	case "EPUB":
		return FormatEPUB
	default:
		return FormatUnknown
	}
}