// Package main provides subtitle diagnostics for JW Broadcasting content.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("UTF-8 support test: %t\n", utf8.ValidString("测试汉字"))
	fmt.Println("=" + strings.Repeat("=", 50))

	settings := &config.Settings{
		Lang:              "E",
		IncludeCategories: []string{"VideoOnDemand"},
		ExcludeCategories: []string{"VODSJJMeetings"},
		Quiet:             0,
	}

	client := api.NewClient(settings)
	data, err := client.ParseBroadcasting()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing broadcasting data: %v\n", err)
		os.Exit(1)
	}

	var totalMedia int
	var mediaWithSubtitles int
	var subtitleURLs []string
	var problematicURLs []string
	var emptySubtitleURLs []string

	fmt.Println("Analyzing media items...")

	for _, cat := range data {
		for _, item := range cat.Contents {
			if media, ok := item.(*api.Media); ok {
				totalMedia++

				if media.SubtitleURL != "" {
					mediaWithSubtitles++
					subtitleURLs = append(subtitleURLs, media.SubtitleURL)

					// Check for potentially problematic URLs
					if !utf8.ValidString(media.SubtitleURL) {
						problematicURLs = append(problematicURLs, fmt.Sprintf("INVALID_UTF8: %s", media.SubtitleURL))
					}
					if strings.Contains(media.SubtitleURL, " ") {
						problematicURLs = append(problematicURLs, fmt.Sprintf("SPACE: %s", media.SubtitleURL))
					}
					if !strings.HasPrefix(media.SubtitleURL, "http") {
						problematicURLs = append(problematicURLs, fmt.Sprintf("NO_HTTP: %s", media.SubtitleURL))
					}
					if !utf8.ValidString(media.Name) {
						problematicURLs = append(problematicURLs, fmt.Sprintf("INVALID_UTF8_NAME: %s -> %s", media.Name, media.SubtitleURL))
					}
				} else {
					emptySubtitleURLs = append(emptySubtitleURLs, fmt.Sprintf("No subtitle for: %s", media.Name))
				}
			}
		}
	}

	fmt.Printf("Total media items: %d\n", totalMedia)
	fmt.Printf("Media with subtitles: %d\n", mediaWithSubtitles)
	fmt.Printf("Media without subtitles: %d\n", totalMedia-mediaWithSubtitles)
	fmt.Printf("Subtitle ratio: %.2f%%\n", float64(mediaWithSubtitles)/float64(totalMedia)*100)

	fmt.Println("\nFirst 10 subtitle URLs:")
	for i, url := range subtitleURLs {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d: %s\n", i+1, url)
	}

	fmt.Println("\nLast 10 subtitle URLs:")
	start := len(subtitleURLs) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(subtitleURLs); i++ {
		fmt.Printf("  %d: %s\n", i+1, subtitleURLs[i])
	}

	if len(problematicURLs) > 0 {
		fmt.Printf("\nProblematic URLs found: %d\n", len(problematicURLs))
		for i, url := range problematicURLs {
			if i >= 20 { // Show first 20
				fmt.Printf("  ... and %d more\n", len(problematicURLs)-20)
				break
			}
			fmt.Printf("  %s\n", url)
		}
	} else {
		fmt.Println("\nNo problematic URLs detected")
	}

	if len(emptySubtitleURLs) > 10 {
		fmt.Printf("\nFirst 10 items without subtitles:\n")
		for i := 0; i < 10; i++ {
			fmt.Printf("  %s\n", emptySubtitleURLs[i])
		}
		fmt.Printf("  ... and %d more without subtitles\n", len(emptySubtitleURLs)-10)
	}

	// Test filename generation for some examples
	fmt.Println("\nTesting filename generation:")
	testURLs := []string{
		"https://example.com/subtitle.vtt",
		"https://example.com/subtitle123",
		"https://example.com/subtitle.txt",
		"https://example.com/sub 测试.vtt",
		"",
	}

	for _, url := range testURLs {
		if url == "" {
			continue
		}
		filename := getSubtitleFilename(url, true)
		friendlyName := getFriendlySubtitleFilename("Test Media 测试", url, true)
		fmt.Printf("  URL: %s\n", url)
		fmt.Printf("    Filename: %s\n", filename)
		fmt.Printf("    Friendly: %s\n", friendlyName)
		fmt.Printf("    Valid UTF-8: %t\n", utf8.ValidString(filename))
		fmt.Println()
	}

	// Save raw data for comparison
	fmt.Println("Saving diagnostic data...")
	saveData := map[string]interface{}{
		"platform":             fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		"go_version":           runtime.Version(),
		"total_media":          totalMedia,
		"media_with_subtitles": mediaWithSubtitles,
		"subtitle_urls":        subtitleURLs,
		"problematic_urls":     problematicURLs,
		"sample_empty_items":   emptySubtitleURLs[:minInt(len(emptySubtitleURLs), 50)],
	}

	jsonData, _ := json.MarshalIndent(saveData, "", "  ")
	tmpFile, err := os.CreateTemp("", "subtitle_diagnostic_*.json")
	if err != nil {
		fmt.Printf("Warning: could not create temp file: %v\n", err)
		return
	}
	defer func() {
		if closeErr := tmpFile.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close temp file: %v\n", closeErr)
		}
	}()

	err = os.WriteFile(tmpFile.Name(), jsonData, 0o600)
	if err != nil {
		fmt.Printf("Warning: could not save diagnostic data: %v\n", err)
	} else {
		fmt.Printf("Diagnostic data saved to %s\n", tmpFile.Name())
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Copy the functions from the actual code for testing
func formatFilename(s string, safe bool) string {
	var forbidden string
	if safe {
		s = strings.ReplaceAll(s, `"`, "'")
		s = strings.ReplaceAll(s, ":", "-")
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

func getSubtitleFilename(url string, safe bool) string {
	if url == "" {
		return ""
	}
	filename := filepath.Base(url)
	ext := filepath.Ext(filename)
	if ext != ".vtt" {
		filename += ".vtt"
	}
	return formatFilename(filename, safe)
}

func getFriendlySubtitleFilename(name, subtitleURL string, safe bool) string {
	if subtitleURL == "" {
		return ""
	}
	ext := filepath.Ext(subtitleURL)
	if ext != ".vtt" {
		ext = ".vtt"
	}
	return formatFilename(name+ext, safe)
}
