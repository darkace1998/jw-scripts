package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	fmt.Println("JW.org Media Content Analysis")
	fmt.Println("=============================")

	settings := &config.Settings{
		Lang:              "E",
		IncludeCategories: []string{"VideoOnDemand", "Audio"},
		ExcludeCategories: []string{},
		Quiet:             0,
	}

	client := api.NewClient(settings)

	// Get some categories and analyze their media content
	categoriesToCheck := []string{
		"VideoOnDemand", "Audio", "VODBible", "VODOurOrganization",
		"VODChildren", "VODFamily", "AudioOriginalSongs",
	}

	for _, catKey := range categoriesToCheck {
		fmt.Printf("\n=== Analyzing Category: %s ===\n", catKey)
		
		catResp, err := client.GetCategory("E", catKey)
		if err != nil {
			fmt.Printf("Error getting category %s: %v\n", catKey, err)
			continue
		}

		fmt.Printf("Category Name: %s\n", catResp.Category.Name)
		fmt.Printf("Subcategories: %d\n", len(catResp.Category.Subcategories))
		fmt.Printf("Media Items: %d\n", len(catResp.Category.Media))

		// Analyze media types and file extensions
		mediaTypes := make(map[string]int)
		fileExtensions := make(map[string]int)
		
		for i, media := range catResp.Category.Media {
			if i < 10 { // Only show first 10 for brevity
				fmt.Printf("  Media %d: %s (Type: %s)\n", i+1, media.Title, media.Type)
				
				for j, file := range media.Files {
					if j < 2 { // Only show first 2 files per media
						url := file.ProgressiveDownloadURL
						ext := getExtensionFromURL(url)
						fmt.Printf("    File %d: %s (Size: %d bytes)\n", j+1, ext, file.Filesize)
						fileExtensions[ext]++
					}
				}
			}
			
			mediaTypes[media.Type]++
		}

		fmt.Printf("Media Types Summary:\n")
		for mediaType, count := range mediaTypes {
			fmt.Printf("  %s: %d\n", mediaType, count)
		}

		fmt.Printf("File Extensions Summary:\n")
		for ext, count := range fileExtensions {
			fmt.Printf("  %s: %d\n", ext, count)
		}

		// Look for any non-video/audio files
		for _, media := range catResp.Category.Media {
			for _, file := range media.Files {
				ext := getExtensionFromURL(file.ProgressiveDownloadURL)
				if ext != ".mp4" && ext != ".mp3" && ext != ".m4v" && ext != ".m4a" && ext != "" {
					fmt.Printf("  INTERESTING: Found %s file: %s\n", ext, media.Title)
				}
			}
		}
	}

	// Also check if there are any hidden/special categories we missed
	fmt.Printf("\n=== Checking All Root Categories for Publication Clues ===\n")
	
	baseURL := "https://data.jw-api.org/mediator/v1"
	resp, err := http.Get(fmt.Sprintf("%s/categories/E/?detailed=1", baseURL))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var rootResp api.RootCategoriesResponse
	json.NewDecoder(resp.Body).Decode(&rootResp)

	fmt.Printf("All categories (including excluded ones):\n")
	for _, cat := range rootResp.Categories {
		// Look for keywords that might indicate publications
		keywords := []string{"publication", "book", "library", "watchtower", "awake", "study", "literature"}
		hasKeyword := false
		catLower := strings.ToLower(cat.Key + " " + cat.Name + " " + cat.Description)
		
		for _, keyword := range keywords {
			if strings.Contains(catLower, keyword) {
				hasKeyword = true
				break
			}
		}
		
		if hasKeyword {
			fmt.Printf("  POTENTIAL: %s (%s) - %s\n", cat.Key, cat.Name, cat.Description)
			fmt.Printf("    Tags: %s\n", strings.Join(cat.Tags, ", "))
		} else {
			fmt.Printf("  %s (%s)\n", cat.Key, cat.Name)
		}
	}
}

func getExtensionFromURL(url string) string {
	if url == "" {
		return ""
	}
	
	// Remove query parameters
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	
	// Get the last part after the last dot
	if idx := strings.LastIndex(url, "."); idx != -1 {
		return url[idx:]
	}
	
	return ""
}