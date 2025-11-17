package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/config"
)

const (
	baseURL = "https://data.jw-api.org/mediator/v1"
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
		baseURL:    baseURL,
		httpClient: &http.Client{},
		settings:   s,
	}
}

// GetLanguages fetches the list of available languages.
func (c *Client) GetLanguages() ([]Language, error) {
	url := fmt.Sprintf("%s/languages/E/web?clientType=www", c.baseURL)
	resp, err := c.httpClient.Get(url)
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
	url := fmt.Sprintf("%s/categories/%s/?detailed=1", c.baseURL, c.settings.Lang)
	resp, err := c.httpClient.Get(url)
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
	url := fmt.Sprintf("%s/categories/%s/%s?detailed=1", c.baseURL, lang, key)
	resp, err := c.httpClient.Get(url)
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

			if m.Type == "audio" {
				if len(m.Files) > 0 {
					bestFile = &m.Files[0]
				}
			} else {
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

func parseDate(dateString string) (time.Time, error) {
	re := regexp.MustCompile(`\.\d+Z$`)
	dateString = re.ReplaceAllString(dateString, "")
	return time.Parse("2006-01-02T15:04:05", dateString)
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
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(forbidden, r) {
			return -1
		}
		return r
	}, s)
}

func getFilename(url string, safe bool) string {
	if url == "" {
		return ""
	}
	return formatFilename(filepath.Base(url), safe)
}

func getSubtitleFilename(url string, safe bool) string {
	if url == "" {
		return ""
	}
	filename := filepath.Base(url)
	ext := filepath.Ext(filename)
	// Only use the extension if it's a valid subtitle extension (.vtt)
	// Otherwise, add .vtt for subtitle files
	if ext != ".vtt" {
		filename += ".vtt"
	}
	return formatFilename(filename, safe)
}

func getFriendlyFilename(name, url string, safe bool) string {
	if url == "" {
		return ""
	}
	return formatFilename(name+filepath.Ext(url), safe)
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
