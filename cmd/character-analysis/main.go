// Package main provides a character analysis tool for JW Broadcasting media filenames.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
)

func main() {
	fmt.Println("Analyzing character handling differences that could affect Windows vs Linux...")
	fmt.Println("=" + strings.Repeat("=", 70))

	settings := &config.Settings{
		Lang:              "E",
		IncludeCategories: []string{"VideoOnDemand"},
		ExcludeCategories: []string{"VODSJJMeetings"},
		Quiet:             1, // Reduce noise
	}

	client := api.NewClient(settings)
	data, err := client.ParseBroadcasting()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing broadcasting data: %v\n", err)
		os.Exit(1)
	}

	var totalMedia int
	var problematicMedia []ProblematicItem
	characterStats := make(map[rune]int)
	var filenameCollisions []FilenameCollision

	// Track what happens with SafeFilenames=true vs false
	filenameMapSafe := make(map[string]int)
	filenameMapUnsafe := make(map[string]int)

	fmt.Println("Analyzing media items for character handling issues...")

	for _, cat := range data {
		for _, item := range cat.Contents {
			if media, ok := item.(*api.Media); ok {
				totalMedia++

				if media.SubtitleURL != "" {
					// Test filename generation with both safe and unsafe modes
					safeName := formatFilename(media.Name+".vtt", true)
					unsafeName := formatFilename(media.Name+".vtt", false)

					safeSubtitleFilename := getSubtitleFilenameSafe(media.SubtitleURL, true)
					unsafeSubtitleFilename := getSubtitleFilenameSafe(media.SubtitleURL, false)

					// Track filename collisions
					filenameMapSafe[safeName]++
					filenameMapUnsafe[unsafeName]++

					// Check for problematic characters
					problematicChars := []rune{}

					for _, r := range media.Name + media.SubtitleURL {
						characterStats[r]++

						// Check for characters that would be filtered in safe mode
						if strings.ContainsRune(`<>|?\\*/`, r) || r == '\x00' || r == '\n' {
							problematicChars = append(problematicChars, r)
						}
					}

					// Check if safe vs unsafe processing creates different results
					if safeName != unsafeName || safeSubtitleFilename != unsafeSubtitleFilename {
						problematicMedia = append(problematicMedia, ProblematicItem{
							Name:                   media.Name,
							SubtitleURL:            media.SubtitleURL,
							SafeName:               safeName,
							UnsafeName:             unsafeName,
							SafeSubtitleFilename:   safeSubtitleFilename,
							UnsafeSubtitleFilename: unsafeSubtitleFilename,
							ProblematicChars:       string(problematicChars),
						})
					}

					// Check for empty filenames after processing
					if safeName == "" || unsafeName == "" || safeSubtitleFilename == "" || unsafeSubtitleFilename == "" {
						problematicMedia = append(problematicMedia, ProblematicItem{
							Name:                   media.Name + " [EMPTY_FILENAME]",
							SubtitleURL:            media.SubtitleURL,
							SafeName:               safeName,
							UnsafeName:             unsafeName,
							SafeSubtitleFilename:   safeSubtitleFilename,
							UnsafeSubtitleFilename: unsafeSubtitleFilename,
							ProblematicChars:       "EMPTY_RESULT",
						})
					}
				}
			}
		}
	}

	// Find filename collisions
	for filename, count := range filenameMapSafe {
		if count > 1 {
			filenameCollisions = append(filenameCollisions, FilenameCollision{
				Filename: filename,
				Count:    count,
				Mode:     "safe",
			})
		}
	}

	for filename, count := range filenameMapUnsafe {
		if count > 1 {
			filenameCollisions = append(filenameCollisions, FilenameCollision{
				Filename: filename,
				Count:    count,
				Mode:     "unsafe",
			})
		}
	}

	fmt.Printf("Total media items analyzed: %d\n", totalMedia)
	fmt.Printf("Problematic items found: %d\n", len(problematicMedia))
	fmt.Printf("Filename collisions found: %d\n", len(filenameCollisions))

	if len(problematicMedia) > 0 {
		fmt.Println("\nFirst 20 problematic items:")
		for i, item := range problematicMedia {
			if i >= 20 {
				fmt.Printf("... and %d more\n", len(problematicMedia)-20)
				break
			}
			fmt.Printf("%d. %s\n", i+1, item.Name)
			fmt.Printf("   URL: %s\n", item.SubtitleURL)
			fmt.Printf("   Safe name: '%s' | Unsafe name: '%s'\n", item.SafeName, item.UnsafeName)
			fmt.Printf("   Safe subtitle: '%s' | Unsafe subtitle: '%s'\n", item.SafeSubtitleFilename, item.UnsafeSubtitleFilename)
			fmt.Printf("   Problematic chars: %s\n", item.ProblematicChars)
			fmt.Println()
		}
	}

	if len(filenameCollisions) > 0 {
		fmt.Printf("\nFirst 10 filename collisions:\n")
		for i, collision := range filenameCollisions {
			if i >= 10 {
				fmt.Printf("... and %d more\n", len(filenameCollisions)-10)
				break
			}
			fmt.Printf("%d. '%s' (%s mode): %d duplicates\n", i+1, collision.Filename, collision.Mode, collision.Count)
		}
	}

	// Check specific character frequencies that might be problematic
	fmt.Println("\nCharacters that could cause Windows/Linux differences:")
	problematicCharList := []rune{'<', '>', '|', '?', '\\', '*', '/', '\x00', '\n', '"', ':'}
	for _, char := range problematicCharList {
		if count, exists := characterStats[char]; exists && count > 0 {
			fmt.Printf("'%c' (U+%04X): %d occurrences\n", char, char, count)
		}
	}

	// Save analysis results
	analysis := map[string]interface{}{
		"total_media":         totalMedia,
		"problematic_items":   len(problematicMedia),
		"filename_collisions": len(filenameCollisions),
		"problematic_samples": problematicMedia[:minInt(len(problematicMedia), 50)],
		"collision_samples":   filenameCollisions[:minInt(len(filenameCollisions), 50)],
		"character_stats":     characterStats,
	}

	jsonData, _ := json.MarshalIndent(analysis, "", "  ")
	tmpFile, err := os.CreateTemp("", "character_analysis_*.json")
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
		fmt.Printf("Warning: could not save analysis data: %v\n", err)
	} else {
		fmt.Printf("\nAnalysis data saved to %s\n", tmpFile.Name())
	}
}

// ProblematicItem represents a media item with filesystem-incompatible characters in its name
type ProblematicItem struct {
	Name                   string `json:"name"`
	SubtitleURL            string `json:"subtitle_url"`
	SafeName               string `json:"safe_name"`
	UnsafeName             string `json:"unsafe_name"`
	SafeSubtitleFilename   string `json:"safe_subtitle_filename"`
	UnsafeSubtitleFilename string `json:"unsafe_subtitle_filename"`
	ProblematicChars       string `json:"problematic_chars"`
}

// FilenameCollision represents a filename that appears multiple times with collision statistics
type FilenameCollision struct {
	Filename string `json:"filename"`
	Count    int    `json:"count"`
	Mode     string `json:"mode"`
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Copy formatting functions for analysis
func formatFilename(s string, safe bool) string {
	if !utf8.ValidString(s) {
		return ""
	}

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

func getSubtitleFilenameSafe(url string, safe bool) string {
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
