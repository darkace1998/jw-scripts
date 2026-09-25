package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/httpx"
	"github.com/darkace1998/jw-scripts/internal/util"
)

const (
	baseURL     = "https://data.jw-api.org/mediator/v1"
	pubMediaURL = "https://b.jw-cdn.org/apis/pub-media/GETPUBMEDIALINKS"
	// jwbStartYear and jwbStartMonth mark when JW Broadcasting began (October 2014 = issue 1).
	// Issue numbers are sequential months: issue = (year-2014)*12 + month - 10 + 1
	jwbStartYear  = 2014
	jwbStartMonth = 10

	// qualityMatchBonus is the ranking bonus applied when a video resolution
	// is within the requested quality limit.
	qualityMatchBonus = 200

	// subtitleMatchBonus is the ranking bonus applied when a video's subtitle
	// state matches the requested preference.
	subtitleMatchBonus = 100

	// maxResponseSize is the maximum allowed API response body size (10 MiB).
	maxResponseSize = 10 << 20

	// jwbIssueLookback is how many monthly JW Broadcasting issues (besides
	// the current one) are searched for audio files.
	jwbIssueLookback = 36

	// jwbFetchWorkers bounds the number of concurrent Publication Media API
	// requests when indexing JW Broadcasting audio.
	jwbFetchWorkers = 4
)

// ErrNotFound is returned when the API reports that a category or
// publication does not exist (HTTP 404, or no files for the language).
var ErrNotFound = errors.New("not found")

// parseDateMillisRegex strips milliseconds from date strings for fallback parsing.
var parseDateMillisRegex = regexp.MustCompile(`\.\d+Z$`)

// Client is a client for the JW.ORG API.
type Client struct {
	baseURL     string
	pubMediaURL string
	httpClient  *http.Client
	settings    *config.Settings
}

// NewClient creates a new API client.
func NewClient(s *config.Settings) *Client {
	return &Client{
		baseURL:     baseURL,
		pubMediaURL: pubMediaURL,
		httpClient:  httpx.NewClient(30 * time.Second),
		settings:    s,
	}
}

// getJSON fetches reqURL (retrying transient failures) and decodes the JSON
// response into v. A 404 response is reported as ErrNotFound.
func (c *Client) getJSON(reqURL, what string, v interface{}) error {
	resp, err := httpx.Get(c.httpClient, reqURL)
	if err != nil {
		return fmt.Errorf("failed to get %s: %w", what, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%s: %w", what, ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get %s: %s", what, resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(v); err != nil {
		return fmt.Errorf("failed to decode %s: %w", what, err)
	}
	return nil
}

// GetLanguages fetches the list of available languages.
func (c *Client) GetLanguages() ([]Language, error) {
	reqURL := fmt.Sprintf("%s/languages/E/web?clientType=www", c.baseURL)
	var langResp LanguagesResponse
	if err := c.getJSON(reqURL, "languages", &langResp); err != nil {
		return nil, err
	}

	return langResp.Languages, nil
}

// GetRootCategories fetches all available root categories from the API.
func (c *Client) GetRootCategories() ([]string, error) {
	reqURL := fmt.Sprintf("%s/categories/%s/?detailed=1", c.baseURL, c.settings.Lang)
	var rootResp RootCategoriesResponse
	if err := c.getJSON(reqURL, "root categories", &rootResp); err != nil {
		return nil, err
	}
	return FilterRootCategories(&rootResp), nil
}

// FilterRootCategories returns the keys of the root categories that are
// useful to index: container or on-demand categories that are not excluded
// from the website (plus a few known useful exceptions).
func FilterRootCategories(rootResp *RootCategoriesResponse) []string {
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

	return categories
}

// GetCategory fetches a category by its key.
func (c *Client) GetCategory(lang, key string) (*CategoryResponse, error) {
	reqURL := fmt.Sprintf("%s/categories/%s/%s?detailed=1", c.baseURL, lang, url.PathEscape(key))
	var catResp CategoryResponse
	if err := c.getJSON(reqURL, "category "+key, &catResp); err != nil {
		return nil, err
	}

	return &catResp, nil
}

// GetBroadcastingMP3s fetches JW Broadcasting MP3s from the Publication Media API.
// It searches through recent JWB publication issues to find available MP3 files.
// Issues that fail to load are reported in the returned error, while the
// categories that could be indexed are still returned.
func (c *Client) GetBroadcastingMP3s() ([]*Category, error) {
	var result []*Category

	cat := &Category{
		Key:  "JWBroadcasting",
		Name: "JW Broadcasting (Audio)",
		Home: true,
	}

	// Compute the current JWB issue number. Issues are numbered sequentially by
	// month starting from October 2014 (issue 1). Going back 36 issues = 3 years.
	now := time.Now()
	startIssue := (now.Year()-jwbStartYear)*12 + int(now.Month()) - jwbStartMonth + 1
	var issues []int
	for issue := startIssue; issue >= startIssue-jwbIssueLookback && issue >= 1; issue-- {
		issues = append(issues, issue)
	}

	// Fetch issues concurrently, then process them in order so the result
	// does not depend on network timing.
	type fetchResult struct {
		files []PubMediaFile
		err   error
	}
	results := make([]fetchResult, len(issues))
	sem := make(chan struct{}, jwbFetchWorkers)
	var wg sync.WaitGroup
	for i, issue := range issues {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			files, err := c.fetchPubMediaMP3s(fmt.Sprintf("jwb-%d", issue))
			results[i] = fetchResult{files: files, err: err}
		}()
	}
	wg.Wait()

	var errs []error
	for i, issue := range issues {
		pubCode := fmt.Sprintf("jwb-%d", issue)
		if c.settings.Quiet < 1 {
			fmt.Fprintf(os.Stderr, "indexing: %s\n", pubCode)
		}

		files, err := results[i].files, results[i].err
		if err != nil {
			// The newest issue is often not published yet, and older ones
			// may not exist in every language; that is not an error.
			if errors.Is(err, ErrNotFound) {
				if c.settings.Quiet < 1 {
					fmt.Fprintf(os.Stderr, "not available: %s\n", pubCode)
				}
				continue
			}
			if c.settings.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "could not fetch %s: %v\n", pubCode, err)
			}
			errs = append(errs, err)
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

			cat.Contents = append(cat.Contents, media)
		}
	}

	if len(cat.Contents) > 0 {
		result = append(result, cat)
	}

	AssignFilenames(c.settings, result)
	return result, errors.Join(errs...)
}

// fetchPubMediaMP3s fetches MP3 files for a specific publication from the Publication Media API.
func (c *Client) fetchPubMediaMP3s(pubCode string) ([]PubMediaFile, error) {
	params := url.Values{}
	params.Set("output", "json")
	params.Set("pub", pubCode)
	params.Set("langwritten", c.settings.Lang)
	params.Set("alllangs", "0")
	params.Set("fileformat", "MP3")

	reqURL := c.pubMediaURL + "?" + params.Encode()
	var pubResp PubMediaResponse
	if err := c.getJSON(reqURL, "publication "+pubCode, &pubResp); err != nil {
		return nil, err
	}

	// Get MP3 files for the requested language
	langFiles, ok := pubResp.Files[c.settings.Lang]
	if !ok {
		return nil, fmt.Errorf("no files for language %s in %s: %w", c.settings.Lang, pubCode, ErrNotFound)
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
// Categories that fail to load are reported in the returned error, while
// everything that could be indexed is still returned so callers can process
// partial results.
func (c *Client) ParseBroadcasting() ([]*Category, error) {
	queue := make([]string, len(c.settings.IncludeCategories))
	copy(queue, c.settings.IncludeCategories)

	var result []*Category

	processed := make(map[string]bool)
	var errs []error

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
			if c.settings.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "could not get category %s: %v\n", key, err)
			}
			// A missing subcategory (as in the Python version) is not fatal,
			// but a failed or misspelled requested category is.
			if !errors.Is(err, ErrNotFound) || util.Contains(c.settings.IncludeCategories, key) {
				errs = append(errs, err)
			}
			continue
		}

		cat := &Category{
			Key:  catResp.Category.Key,
			Name: catResp.Category.Name,
			Home: util.Contains(c.settings.IncludeCategories, catResp.Category.Key),
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
			if !util.Contains(c.settings.ExcludeCategories, sub.Key) {
				queue = append(queue, sub.Key)
			}
		}

		for _, m := range catResp.Category.Media {
			if util.Contains(c.settings.FilterCategories, m.PrimaryCategory) {
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

	AssignFilenames(c.settings, result)
	return result, errors.Join(errs...)
}

// AssignFilenames sets the local filenames of all media in data.
//
// The same media item often appears in several categories; every occurrence
// of a URL maps to the same file so it is only downloaded once. Different
// media that would end up with the same name (compared case-insensitively,
// as on Windows and macOS) get a " (n)" suffix. Names are handed out in
// order of publication date and URL, so they stay stable when the API
// returns items in a different order or newer items with the same title are
// published. Subtitles are named after their video ("<video name>.vtt") so
// players and media servers associate them automatically.
//
// Imported media (LocalPath set) keep their filenames. The function can be
// called again on combined results and produces the same names.
func AssignFilenames(s *config.Settings, data []*Category) {
	used := make(map[string]bool)
	var all []*Media
	for _, cat := range data {
		for _, item := range cat.Contents {
			m, ok := item.(*Media)
			if !ok {
				continue
			}
			if m.LocalPath != "" {
				used[strings.ToLower(m.Filename)] = true
				continue
			}
			all = append(all, m)
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Date != all[j].Date {
			return all[i].Date < all[j].Date
		}
		if all[i].URL != all[j].URL {
			return all[i].URL < all[j].URL
		}
		return all[i].Name < all[j].Name
	})

	byURL := make(map[string]string)
	for _, m := range all {
		m.FriendlyName = getFriendlyFilename(m.Name, m.URL, s.SafeFilenames)

		if name, ok := byURL[m.URL]; ok && m.URL != "" {
			m.Filename = name
		} else {
			base := getFilename(m.URL, s.SafeFilenames)
			if s.FriendlyFilenames && hasBaseName(m.FriendlyName) {
				base = m.FriendlyName
			}
			m.Filename = makeUniqueFilename(base, used)
			byURL[m.URL] = m.Filename
		}

		switch {
		case m.SubtitleURL == "":
			m.SubtitleFilename = ""
		case m.Filename != "":
			m.SubtitleFilename = strings.TrimSuffix(m.Filename, filepath.Ext(m.Filename)) + ".vtt"
		default:
			m.SubtitleFilename = getSubtitleFilename(m.SubtitleURL, s.SafeFilenames)
		}
	}
}

// hasBaseName reports whether a filename has a real name besides its
// extension (not just dots or spaces, which would hide or break the file).
func hasBaseName(name string) bool {
	return strings.Trim(strings.TrimSuffix(name, filepath.Ext(name)), ". ") != ""
}

func getBestVideo(files []File, quality int, subtitles bool) *File {
	var bestFile *File
	maxRank := -1

	for i := range files {
		file := &files[i]
		rank := 0
		res, err := strconv.Atoi(strings.TrimSuffix(file.Label, "p"))
		if err != nil {
			// Non-numeric label; treat resolution as 0 (lowest priority)
			res = 0
		}
		rank += res / 10
		if res > 0 && res <= quality {
			rank += qualityMatchBonus
		}
		if file.Subtitled == subtitles {
			rank += subtitleMatchBonus
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
	dateString = parseDateMillisRegex.ReplaceAllString(dateString, "")
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

// FormatFilename turns s into a name that is safe to use as a single path
// element: path separators and control characters are always removed, and
// in safe mode characters and names that Windows rejects are replaced too.
// Leading dots are removed so names never hide files or refer to "." or
// ".."; an empty string means no usable name.
func FormatFilename(s string, safe bool) string {
	return formatFilename(s, safe)
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
		if strings.ContainsRune(forbidden, r) || unicode.IsControl(r) {
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
	// Leading dots would hide the file (and "." / ".." are not filenames)
	return strings.TrimLeft(result, ". ")
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
	// Clean the title on its own first, so a title that is unusable by
	// itself (e.g. "..") cannot merge with the extension into a name.
	cleaned := formatFilename(name, safe)
	if cleaned == "" {
		return ""
	}
	return formatFilename(cleaned+filepath.Ext(fileURL), safe)
}

// makeUniqueFilename ensures filename is unique by appending a number if
// needed. Names are compared case-insensitively (usedFilenames holds
// lower-case keys) because Windows and macOS filesystems are case-insensitive.
func makeUniqueFilename(filename string, usedFilenames map[string]bool) string {
	if filename == "" {
		return ""
	}

	originalFilename := filename
	counter := 1

	// Keep trying until we find a unique filename
	for usedFilenames[strings.ToLower(filename)] {
		ext := filepath.Ext(originalFilename)
		nameWithoutExt := strings.TrimSuffix(originalFilename, ext)
		filename = fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, ext)
		counter++
	}

	usedFilenames[strings.ToLower(filename)] = true
	return filename
}
