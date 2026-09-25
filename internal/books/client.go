package books

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/httpx"
)

// ErrNotFound is returned when a publication is not available in the
// requested language.
var ErrNotFound = errors.New("publication not found")

// now returns the current time; tests replace it to get stable
// year-dependent publication codes.
var now = time.Now

// magazineLookbackMonths is how far back the latest magazine issue is
// searched when no --issue is given.
const magazineLookbackMonths = 12

// Client implements the BookAPI interface for JW.org book operations
type Client struct {
	baseURL    string
	httpClient *http.Client
	settings   *config.Settings
}

// PublicationResponse represents the API response from the JW.org publication media API
type PublicationResponse struct {
	PubName       string                           `json:"pubName"`
	ParentPubName string                           `json:"parentPubName"`
	Pub           string                           `json:"pub"`
	Issue         string                           `json:"issue"`
	FormattedDate string                           `json:"formattedDate"`
	FileFormat    []string                         `json:"fileformat"`
	Files         map[string]map[string][]FileInfo `json:"files"`
}

// FileInfo represents a downloadable file from the API
type FileInfo struct {
	Title string `json:"title"`
	File  struct {
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
		httpClient: httpx.NewClient(30 * time.Second),
		settings:   s,
	}
}

// GetSupportedLanguages returns all supported languages
func (c *Client) GetSupportedLanguages() ([]Language, error) {
	// Based on API discovery, these are the confirmed supported languages
	languages := []Language{
		{Code: "E", Name: "English", Direction: "ltr", Locale: "en"},
		{Code: "S", Name: "español", Direction: "ltr", Locale: "es"},
		{Code: "F", Name: "Français", Direction: "ltr", Locale: "fr"},
		{Code: "T", Name: "Português (Brasil)", Direction: "ltr", Locale: "pt-BR"},
		{Code: "X", Name: "Deutsch", Direction: "ltr", Locale: "de"},
		{Code: "P", Name: "polski", Direction: "ltr", Locale: "pl"},
		{Code: "Z", Name: "Svenska", Direction: "ltr", Locale: "sv"},
		{Code: "J", Name: "日本語", Direction: "ltr", Locale: "ja"},
		{Code: "K", Name: "українська", Direction: "ltr", Locale: "uk"},
		{Code: "I", Name: "Italiano", Direction: "ltr", Locale: "it"},
		{Code: "U", Name: "русский", Direction: "ltr", Locale: "ru"},
		{Code: "R", Name: "Արեւմտահայերէն", Direction: "ltr", Locale: "hy"},
		{Code: "Q", Name: "עברית", Direction: "rtl", Locale: "he"},
		{Code: "V", Name: "slovenčina", Direction: "ltr", Locale: "sk"},
		{Code: "W", Name: "Cymraeg", Direction: "ltr", Locale: "cy"},
		{Code: "H", Name: "magyar", Direction: "ltr", Locale: "hu"},
		{Code: "N", Name: "Norsk", Direction: "ltr", Locale: "no"},
		{Code: "A", Name: "العربية", Direction: "rtl", Locale: "ar"},
		{Code: "B", Name: "čeština", Direction: "ltr", Locale: "cs"},
		{Code: "C", Name: "hrvatski", Direction: "ltr", Locale: "hr"},
		{Code: "D", Name: "Dansk", Direction: "ltr", Locale: "da"},
		{Code: "G", Name: "Ελληνική", Direction: "ltr", Locale: "el"},
		{Code: "L", Name: "lietuvių", Direction: "ltr", Locale: "lt"},
		{Code: "M", Name: "Română", Direction: "ltr", Locale: "ro"},
		{Code: "O", Name: "Nederlands", Direction: "ltr", Locale: "nl"},
	}

	return languages, nil
}

// yearlyCodes returns publication codes built from prefix and a two-digit
// year, one for every year offset relative to the current year, in order.
func yearlyCodes(prefix string, offsets ...int) []string {
	year := now().Year()
	codes := make([]string, 0, len(offsets))
	for _, off := range offsets {
		codes = append(codes, fmt.Sprintf("%s%02d", prefix, (year+off)%100))
	}
	return codes
}

// GetCategories returns all available book categories.
//
// Many publications are released yearly with the year in their code (e.g.
// es26 for the 2026 daily text). Their codes are computed from the current
// date, and older editions are used as a fallback until the new one is out.
func (c *Client) GetCategories() ([]BookCategory, error) {
	categories := []BookCategory{
		{
			Key:         "bible",
			Name:        "Bible",
			Description: "New World Translation of the Holy Scriptures",
			candidates:  [][]string{{"nwtsty"}},
		},
		{
			Key:         "daily-text",
			Name:        "Daily Text",
			Description: "Examining the Scriptures Daily (current year)",
			candidates:  [][]string{yearlyCodes("es", 0, -1)},
		},
		{
			Key:         "yearbooks",
			Name:        "Yearbooks",
			Description: "Watch Tower Publications Index and Yearbooks (latest edition)",
			candidates:  [][]string{yearlyCodes("dx", 0, -1, -2)},
		},
		{
			Key:         "circuit-assembly",
			Name:        "Circuit Assembly Programs",
			Description: "Circuit Assembly Programs (current service year)",
			candidates:  [][]string{yearlyCodes("ca-brpgm", 1, 0)},
		},
		{
			Key:         "convention",
			Name:        "Convention Materials",
			Description: "Convention invitations and programs (current year)",
			candidates:  [][]string{yearlyCodes("co-inv", 0, -1)},
		},
		{
			Key:         "magazines",
			Name:        "Magazines",
			Description: "Watchtower and Awake! magazines (latest issue, or the one given with --issue)",
			candidates:  [][]string{{"w"}, {"g"}},
			magazines:   true,
		},
	}

	for i := range categories {
		for _, codes := range categories[i].candidates {
			categories[i].Publications = append(categories[i].Publications, codes[0])
		}
	}

	return categories, nil
}

// GetCategory returns the books in a specific category. Publications that
// cannot be found or fetched are reported in the returned error, while the
// books that were found are still returned.
func (c *Client) GetCategory(lang, categoryKey string) (*BookCategory, error) {
	categories, err := c.GetCategories()
	if err != nil {
		return nil, err
	}

	for i := range categories {
		category := &categories[i]
		if category.Key != categoryKey {
			continue
		}
		var errs []error
		for _, codes := range category.candidates {
			book, err := c.resolveBook(lang, codes, category.magazines)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			category.Books = append(category.Books, *book)
		}
		return category, errors.Join(errs...)
	}

	return nil, fmt.Errorf("category '%s' not found", categoryKey)
}

// resolveBook returns the first available publication among codes. For
// magazines it returns the issue given in the settings, or the latest issue
// published in the last year.
func (c *Client) resolveBook(lang string, codes []string, magazine bool) (*Book, error) {
	if magazine {
		code := codes[0]
		if c.settings.Issue != "" {
			return c.GetBookIssue(lang, code, c.settings.Issue)
		}
		t := now()
		month := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < magazineLookbackMonths; i++ {
			book, err := c.GetBookIssue(lang, code, month.AddDate(0, -i, 0).Format("200601"))
			if err == nil {
				return book, nil
			}
			if !errors.Is(err, ErrNotFound) {
				return nil, err
			}
		}
		return nil, fmt.Errorf("no issue of '%s' found in the last %d months: %w", code, magazineLookbackMonths, ErrNotFound)
	}

	for _, code := range codes {
		book, err := c.GetBook(lang, code)
		if err == nil {
			return book, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("none of the publications %s is available in language %s: %w", strings.Join(codes, ", "), lang, ErrNotFound)
}

// GetBook returns details for a specific book
func (c *Client) GetBook(lang, bookID string) (*Book, error) {
	return c.GetBookIssue(lang, bookID, "")
}

// GetBookIssue returns details for a specific issue of a publication. An
// empty issue fetches the publication itself.
func (c *Client) GetBookIssue(lang, bookID, issue string) (*Book, error) {
	// Make request to the publication API
	pubResp, err := c.getPublicationDataForLanguage(bookID, issue, lang)
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
				// Extract filename from URL since the API doesn't provide one
				// directly. It must be a single safe path element on every OS.
				if u, err := url.Parse(fileInfo.File.URL); err == nil {
					bookFile.Filename = api.FormatFilename(filepath.Base(path.Base(u.Path)), true)
				}
				book.Files = append(book.Files, bookFile)
			}
		}
	}

	if len(book.Files) == 0 {
		return nil, fmt.Errorf("publication '%s' has no files in language %s: %w", bookID, lang, ErrNotFound)
	}

	return book, nil
}

// SearchBooks searches for books by title or content
func (c *Client) SearchBooks(lang, query string) ([]Book, error) {
	// Get all categories and search through them
	categories, err := c.GetCategories()
	if err != nil {
		return nil, err
	}

	var results []Book
	var errs []error
	queryLower := strings.ToLower(query)

	for i := range categories {
		category := &categories[i]
		// Check if query matches the category name or key
		categoryMatch := strings.Contains(strings.ToLower(category.Name), queryLower) ||
			strings.Contains(strings.ToLower(category.Key), queryLower) ||
			strings.Contains(strings.ToLower(category.Description), queryLower)

		for _, codes := range category.candidates {
			book, err := c.resolveBook(lang, codes, category.magazines)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					errs = append(errs, err)
				}
				continue
			}

			// Match on book title, description, or parent category
			if categoryMatch ||
				strings.Contains(strings.ToLower(book.Title), queryLower) ||
				strings.Contains(strings.ToLower(book.Description), queryLower) {
				results = append(results, *book)
			}
		}
	}

	// Report failures only when they may have hidden results
	if len(results) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return results, nil
}

// GetSupportedFormats returns the book formats that are supported
func (c *Client) GetSupportedFormats() []BookFormat {
	return []BookFormat{FormatPDF, FormatEPUB, FormatMP3, FormatMP4, FormatRTF, FormatBRL}
}

// IsBookAPIAvailable checks if the book API is currently available
func (c *Client) IsBookAPIAvailable() bool {
	// Test with a known publication to verify the API endpoint is reachable
	params := url.Values{}
	params.Set("output", "json")
	params.Set("pub", "nwtsty")
	params.Set("fileformat", "PDF")
	params.Set("alllangs", "0")
	params.Set("langwritten", "E")
	requestURL := c.baseURL + "?" + params.Encode()

	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return false
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), http.NoBody)
	if err != nil {
		return false
	}
	// #nosec G704 - URL scheme is validated above to only allow http/https; base URL is a fixed JW.org endpoint
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
}

// GetAPILimitations returns information about current API status
func (c *Client) GetAPILimitations() string {
	return `JW.org Publication API Status:

✅ Publication downloads are now available through the JW.org Publication Media API!

Available Features:
- Multiple formats: PDF, EPUB, MP3, MP4, RTF, BRL
- 25+ languages including major world languages
- Bible (New World Translation Study Edition)
- Daily text for the current year
- Latest publications index
- Current circuit assembly and convention materials
- Latest magazine issues (or a specific one with --issue YYYYMM)

Supported Languages:
- English, Spanish, French, Portuguese, German, Polish, Swedish
- Japanese, Ukrainian, Italian, Russian, Armenian, Hebrew
- Slovak, Welsh, Hungarian, Norwegian, Arabic, Czech
- Croatian, Danish, Greek, Lithuanian, Romanian, Dutch

API Endpoint: https://b.jw-cdn.org/apis/pub-media/GETPUBMEDIALINKS

The framework fully supports book downloads with real data from JW.org.`
}

// getPublicationDataForLanguage fetches publication data for a specific language
func (c *Client) getPublicationDataForLanguage(pubCode, issue, lang string) (*PublicationResponse, error) {
	params := url.Values{}
	params.Set("output", "json")
	params.Set("pub", pubCode)
	params.Set("fileformat", "PDF,EPUB,MP3,MP4,RTF,BRL")
	params.Set("alllangs", "0")
	params.Set("langwritten", lang)
	params.Set("txtCMSLang", lang)

	if issue != "" {
		params.Set("issue", issue)
	}

	requestURL := c.baseURL + "?" + params.Encode()

	resp, err := httpx.Get(c.httpClient, requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("publication '%s': %w", pubCode, ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d for publication '%s'", resp.StatusCode, pubCode)
	}

	// Limit response body to 10 MiB to prevent excessive memory use
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
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
	case "MP3":
		return FormatMP3
	case "MP4":
		return FormatMP4
	case "RTF":
		return FormatRTF
	case "BRL":
		return FormatBRL
	default:
		return FormatUnknown
	}
}
