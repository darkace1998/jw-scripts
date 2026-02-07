package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/config"
)

const (
	baseURL       = "https://data.jw-api.org/mediator/v1"
	pubMediaURL   = "https://b.jw-cdn.org/apis/pub-media/GETPUBMEDIALINKS"
	latestJWBYear = 134 // jwb-134 is 2026 (increases each year)
)

// Client is a client for the JW.ORG API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	settings   *config.Settings
}

// NewClient creates a new API client.
func NewClient(s *config.Settings) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		settings: s,
	}
}

// GetLanguages fetches the list of available languages.
func (c *Client) GetLanguages() ([]Language, error) {
	reqURL := fmt.Sprintf("%s/languages/E/web?clientType=www", c.baseURL)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get languages: %s", resp.Status)
	}

	var langResp LanguagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&langResp); err != nil {
		return nil, err
	}

	return langResp.Languages, nil
}

// GetRootCategories fetches all available root categories from the API.
func (c *Client) GetRootCategories() ([]string, error) {
	reqURL := fmt.Sprintf("%s/categories/%s/?detailed=1", c.baseURL, c.settings.Lang)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get root categories: %s", resp.Status)
	}

	var rootResp RootCategoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&rootResp); err != nil {
		return nil, err
	}

	// Filter categories that are likely to be user-accessible root categories
	// We'll use a more sophisticated approach: include categories that either
	// 1. Don't have major exclude tags, OR
	// 2. Are commonly useful categories even if they have some exclude tags
	var categories []string
	majorExcludeTags := map[string]bool{
		"WebExclude":   true,
		"JWORGExclude": true,
	}

	// Known useful categories that might have some exclude tags but are still valuable
	knownUseful := map[string]bool{
		"Audio": true,
	}

	for _, cat := range rootResp.Categories {
		// Check if this category has major exclude tags
		hasMajorExclude := false
		for _, tag := range cat.Tags {
			if majorExcludeTags[tag] {
				hasMajorExclude = true
				break
			}
		}

		// Include if it's a container/ondemand type AND either:
		// - Doesn't have major exclude tags, OR
		// - Is in the known useful list
		if (cat.Type == "container" || cat.Type == "ondemand") &&
			(!hasMajorExclude || knownUseful[cat.Key]) {
			categories = append(categories, cat.Key)
		}
	}

	return categories, nil
}

// GetCategory fetches a category by its key.
func (c *Client) GetCategory(lang, key string) (*CategoryResponse, error) {
	reqURL := fmt.Sprintf("%s/categories/%s/%s?detailed=1", c.baseURL, lang, key)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get category %s: %s", key, resp.Status)
	}

	var catResp CategoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&catResp); err != nil {
		return nil, err
	}

	return &catResp, nil
}

// GetBroadcastingMP3s fetches JW Broadcasting MP3s from the Publication Media API.
// It searches through recent JWB publication issues to find available MP3 files.
func (c *Client) GetBroadcastingMP3s() ([]*Category, error) {
	var result []*Category
	usedFilenames := make(map[string]bool)

	cat := &Category{
		Key:  "JWBroadcasting",
		Name: "JW Broadcasting (Audio)",
		Home: true,
	}

	// Search through recent JWB issues (going back about 3 years)
	// Each jwb-NNN publication contains monthly programs for that year
	startIssue := latestJWBYear
	endIssue := latestJWBYear - 10 // Go back about 10 years worth of issues

	for issue := startIssue; issue >= endIssue; issue-- {
		pubCode := fmt.Sprintf("jwb-%d", issue)

		if c.settings.Quiet < 1 {
			fmt.Fprintf(os.Stderr, "indexing: %s\n", pubCode)
		}

		files, err := c.fetchPubMediaMP3s(pubCode)
		if err != nil {
			if c.settings.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "could not fetch %s: %v\n", pubCode, err)
			}
			continue
		}

		for _, f := range files {
			// Skip audio description versions (track >= 100) unless specifically requested
			// Also skip items with "audio description" in the title (case-insensitive)
			titleLower := strings.ToLower(f.Title)
			if f.Track >= 100 || strings.Contains(titleLower, "audio description") {
				continue
			}

			media := &Media{
				URL:      f.File.URL,
				Name:     f.Title,
				MD5:      f.File.Checksum,
				Size:     f.Filesize,
				Duration: f.Duration,
			}

			// Parse date from the modified datetime
			if f.File.ModifiedDatetime != "" {
				if date, err := parsePubMediaDate(f.File.ModifiedDatetime); err == nil {
					if date.Unix() < c.settings.MinDate {
						continue
					}
					if c.settings.MaxDate > 0 && date.Unix() > c.settings.MaxDate {
						continue
					}
					media.Date = date.Unix()
				}
			}

			media.Filename = getFilename(media.URL, c.settings.SafeFilenames)
			media.FriendlyName = getFriendlyFilename(media.Name, media.URL, c.settings.SafeFilenames)

			// Ensure unique filenames
			if c.settings.FriendlyFilenames {
				media.Filename = makeUniqueFilename(media.FriendlyName, usedFilenames)
			} else {
				media.Filename = makeUniqueFilename(media.Filename, usedFilenames)
			}

			cat.Contents = append(cat.Contents, media)
		}
	}

	if len(cat.Contents) > 0 {
		result = append(result, cat)
	}

	return result, nil
}

// fetchPubMediaMP3s fetches MP3 files for a specific publication from the Publication Media API.
func (c *Client) fetchPubMediaMP3s(pubCode string) ([]PubMediaFile, error) {
	params := url.Values{}
	params.Set("output", "json")
	params.Set("pub", pubCode)
	params.Set("langwritten", c.settings.Lang)
	params.Set("alllangs", "0")
	params.Set("fileformat", "MP3")

	reqURL := pubMediaURL + "?" + params.Encode()
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get publication %s: %s", pubCode, resp.Status)
	}

	var pubResp PubMediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&pubResp); err != nil {
		return nil, err
	}

	// Get MP3 files for the requested language
	langFiles, ok := pubResp.Files[c.settings.Lang]
	if !ok {
		return nil, fmt.Errorf("no files found for language %s", c.settings.Lang)
	}

	return langFiles.MP3, nil
}

// parsePubMediaDate parses dates from the Publication Media API format.
func parsePubMediaDate(dateString string) (time.Time, error) {
	// Format: "2026-01-18 19:25:59"
	t, err := time.Parse("2006-01-02 15:04:05", dateString)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// ParseBroadcasting is the main function to parse the broadcasting data.
func (c *Client) ParseBroadcasting() ([]*Category, error) {
	queue := make([]string, len(c.settings.IncludeCategories))
	copy(queue, c.settings.IncludeCategories)

	var result []*Category

	processed := make(map[string]bool)

	// Track used filenames to prevent duplicates
	usedFilenames := make(map[string]bool)
	usedSubtitleFilenames := make(map[string]bool)

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]

		if processed[key] {
			continue
		}
		processed[key] = true

		if c.settings.Quiet < 1 {
			fmt.Fprintf(os.Stderr, "indexing: %s\n", key)
		}

		catResp, err := c.GetCategory(c.settings.Lang, key)
		if err != nil {
			// In the Python code, a 404 is not a fatal error, so we just print a message.
			if c.settings.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "could not get category %s: %v\n", key, err)
			}
			continue
		}

		cat := &Category{
			Key:  catResp.Category.Key,
			Name: catResp.Category.Name,
			Home: contains(c.settings.IncludeCategories, catResp.Category.Key),
		}
		if !c.settings.Update {
			result = append(result, cat)
		}

		for _, sub := range catResp.Category.Subcategories {
			subCat := &Category{
				Key:  sub.Key,
				Name: sub.Name,
			}
			cat.Contents = append(cat.Contents, subCat)
			if !contains(c.settings.ExcludeCategories, sub.Key) {
				queue = append(queue, sub.Key)
			}
		}

		for _, m := range catResp.Category.Media {
			if contains(c.settings.FilterCategories, m.PrimaryCategory) {
				continue
			}

			var bestFile *File

switch {
		case m.Type == "audio":
			if len(m.Files) > 0 {
				bestFile = &m.Files[0]
			}
		case c.settings.AudioOnly:
			// When audio-only mode is enabled, try to find an audio file
			bestFile = getBestAudio(m.Files)
			if bestFile == nil {
				if c.settings.Quiet < 1 {
					fmt.Fprintf(os.Stderr, "no audio files found for: %s (skipping video-only content)\n", m.Title)
				}
				continue
			}
		default:
				bestFile = getBestVideo(m.Files, c.settings.Quality, c.settings.HardSubtitles)
			}

			if bestFile == nil {
				if c.settings.Quiet < 1 {
					fmt.Fprintf(os.Stderr, "no media files found for: %s\n", m.Title)
				}
				continue
			}

			media := &Media{
				URL:         bestFile.ProgressiveDownloadURL,
				Name:        m.Title,
				MD5:         bestFile.Checksum,
				Size:        bestFile.Filesize,
				Duration:    bestFile.Duration,
				SubtitleURL: bestFile.Subtitles.URL,
			}

			if m.FirstPublished != "" {
				date, err := parseDate(m.FirstPublished)
				if err != nil {
					if c.settings.Quiet < 1 {
						fmt.Fprintf(os.Stderr, "could not get timestamp on: %s\n", m.Title)
					}
				} else {
					if date.Unix() < c.settings.MinDate {
						continue
					}
					if c.settings.MaxDate > 0 && date.Unix() > c.settings.MaxDate {
						continue
					}
					media.Date = date.Unix()
				}
			}

			media.Filename = getFilename(media.URL, c.settings.SafeFilenames)
			media.FriendlyName = getFriendlyFilename(media.Name, media.URL, c.settings.SafeFilenames)
			media.SubtitleFilename = getSubtitleFilename(media.SubtitleURL, c.settings.SafeFilenames)
			media.FriendlySubtitleFilename = getFriendlySubtitleFilename(media.Name, media.SubtitleURL, c.settings.SafeFilenames)

			// Use friendly filenames if requested and ensure uniqueness
			if c.settings.FriendlyFilenames {
				media.Filename = makeUniqueFilename(media.FriendlyName, usedFilenames)
				media.SubtitleFilename = makeUniqueFilename(media.FriendlySubtitleFilename, usedSubtitleFilenames)
			} else {
				// Even for non-friendly filenames, ensure uniqueness to prevent overwrites
				media.Filename = makeUniqueFilename(media.Filename, usedFilenames)
				media.SubtitleFilename = makeUniqueFilename(media.SubtitleFilename, usedSubtitleFilenames)
			}

			if c.settings.Update {
				var pcat *Category
				for _, r := range result {
					if r.Key == m.PrimaryCategory {
						pcat = r
						break
					}
				}
				if pcat == nil {
					pcat = &Category{
						Key:  m.PrimaryCategory,
						Home: false,
					}
					result = append(result, pcat)
				}
				pcat.Contents = append(pcat.Contents, media)
			} else {
				cat.Contents = append(cat.Contents, media)
			}
		}
	}

	return result, nil
}

func getBestVideo(files []File, quality int, subtitles bool) *File {
	var bestFile *File
	maxRank := -1

	for i := range files {
		file := &files[i]
		rank := 0
		res, _ := strconv.Atoi(strings.TrimSuffix(file.Label, "p"))
		rank += res / 10
		if res > 0 && res <= quality {
			rank += 200
		}
		if file.Subtitled == subtitles {
			rank += 100
		}

		if rank > maxRank {
			maxRank = rank
			bestFile = file
		}
	}

	return bestFile
}

// getBestAudio returns the first audio file from a list of files.
// Returns nil if no audio files are found.
func getBestAudio(files []File) *File {
	for i := range files {
		file := &files[i]
		// Check if the file is an audio file by examining the mimetype
		if strings.HasPrefix(file.Mimetype, "audio/") {
			return file
		}
	}
	return nil
}

func parseDate(dateString string) (time.Time, error) {
	// Try parsing with RFC3339 format first (includes timezone)
	if t, err := time.Parse(time.RFC3339, dateString); err == nil {
		return t.UTC(), nil
	}
	// Strip milliseconds and parse as UTC
	re := regexp.MustCompile(`\.\d+Z$`)
	dateString = re.ReplaceAllString(dateString, "")
	t, err := time.Parse("2006-01-02T15:04:05", dateString)
	if err != nil {
		return time.Time{}, err
	}
	// Return time in UTC since API timestamps are typically in UTC
	return t.UTC(), nil
}

// isWindowsReservedName checks if a filename is a Windows reserved name
func isWindowsReservedName(name string) bool {
	// Windows reserved names (case-insensitive)
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	nameUpper := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	for _, r := range reserved {
		if nameUpper == r {
			return true
		}
	}
	return false
}

func formatFilename(s string, safe bool) string {
	var forbidden string
	if safe {
		s = strings.ReplaceAll(s, `"`, "'")
		s = strings.ReplaceAll(s, ":", "-") // Use dash instead of dot for colons
		forbidden = "<>|?\\*/\x00\n"
	} else {
		forbidden = "/\x00"
	}
	result := strings.Map(func(r rune) rune {
		if strings.ContainsRune(forbidden, r) {
			return -1
		}
		return r
	}, s)

	if safe {
		// Remove trailing dots and spaces (problematic on Windows)
		result = strings.TrimRight(result, ". ")
		// Handle Windows reserved names by prefixing with underscore
		if isWindowsReservedName(result) {
			ext := filepath.Ext(result)
			nameWithoutExt := strings.TrimSuffix(result, ext)
			result = "_" + nameWithoutExt + ext
		}
	}
	return result
}

func getFilename(fileURL string, safe bool) string {
	if fileURL == "" {
		return ""
	}
	return formatFilename(filepath.Base(fileURL), safe)
}

func getSubtitleFilename(fileURL string, safe bool) string {
	if fileURL == "" {
		return ""
	}
	filename := filepath.Base(fileURL)
	ext := filepath.Ext(filename)
	// Only use the extension if it's a valid subtitle extension (.vtt)
	// Otherwise, add .vtt for subtitle files
	if ext != ".vtt" {
		filename += ".vtt"
	}
	return formatFilename(filename, safe)
}

func getFriendlyFilename(name, fileURL string, safe bool) string {
	if fileURL == "" {
		return ""
	}
	return formatFilename(name+filepath.Ext(fileURL), safe)
}

func getFriendlySubtitleFilename(name, subtitleURL string, safe bool) string {
	if subtitleURL == "" {
		return ""
	}
	ext := filepath.Ext(subtitleURL)
	// Only use the extension if it's a valid subtitle extension (.vtt)
	// Otherwise, default to .vtt for subtitle files
	if ext != ".vtt" {
		ext = ".vtt"
	}
	return formatFilename(name+ext, safe)
}

// makeUniqueFilename ensures filename is unique by appending a number if needed
func makeUniqueFilename(filename string, usedFilenames map[string]bool) string {
	if filename == "" {
		return ""
	}

	originalFilename := filename
	counter := 1

	// Keep trying until we find a unique filename
	for usedFilenames[filename] {
		ext := filepath.Ext(originalFilename)
		nameWithoutExt := strings.TrimSuffix(originalFilename, ext)
		filename = fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, ext)
		counter++
	}

	usedFilenames[filename] = true
	return filename
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
