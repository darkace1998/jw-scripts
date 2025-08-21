package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	fmt.Println("Detailed Category and Subtitle Analysis")
	fmt.Println("=" + strings.Repeat("=", 50))

	settings := &config.Settings{
		Lang:              "E", // Default language
		IncludeCategories: []string{"VideoOnDemand"}, // Default category  
		ExcludeCategories: []string{"VODSJJMeetings"}, // Default exclude
		Quiet:             1,
	}

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Language: %s\n", settings.Lang)
	fmt.Printf("  Include Categories: %v\n", settings.IncludeCategories)
	fmt.Printf("  Exclude Categories: %v\n", settings.ExcludeCategories)
	fmt.Printf("  Filter Categories: %v\n", settings.FilterCategories)
	fmt.Printf("  MinDate: %d\n", settings.MinDate)
	fmt.Println()

	client := api.NewClient(settings)
	data, err := client.ParseBroadcasting()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing broadcasting data: %v\n", err)
		os.Exit(1)
	}

	// Analyze by category
	categoryStats := make(map[string]CategoryInfo)
	var totalMedia int
	var totalWithSubtitles int

	for _, cat := range data {
		var mediaCount int
		var subtitleCount int
		
		for _, item := range cat.Contents {
			if media, ok := item.(*api.Media); ok {
				mediaCount++
				totalMedia++
				
				if media.SubtitleURL != "" {
					subtitleCount++
					totalWithSubtitles++
				}
			}
		}
		
		categoryStats[cat.Key] = CategoryInfo{
			Name: cat.Name,
			MediaCount: mediaCount,
			SubtitleCount: subtitleCount,
			IsHome: cat.Home,
		}
	}

	fmt.Printf("Overall Summary:\n")
	fmt.Printf("  Total Categories: %d\n", len(data))
	fmt.Printf("  Total Media Items: %d\n", totalMedia)
	fmt.Printf("  Total with Subtitles: %d\n", totalWithSubtitles)
	fmt.Printf("  Subtitle Ratio: %.2f%%\n", float64(totalWithSubtitles)/float64(totalMedia)*100)
	fmt.Println()

	// Sort categories by subtitle count for better analysis
	type CategoryResult struct {
		Key string
		Info CategoryInfo
	}
	
	var sortedCategories []CategoryResult
	for key, info := range categoryStats {
		sortedCategories = append(sortedCategories, CategoryResult{key, info})
	}
	
	sort.Slice(sortedCategories, func(i, j int) bool {
		return sortedCategories[i].Info.SubtitleCount > sortedCategories[j].Info.SubtitleCount
	})

	fmt.Printf("Top 20 Categories by Subtitle Count:\n")
	fmt.Printf("%-25s %-8s %-8s %-6s %s\n", "Key", "Media", "Subs", "Home", "Name")
	fmt.Println(strings.Repeat("-", 80))
	
	for i, cat := range sortedCategories {
		if i >= 20 {
			break
		}
		homeFlag := ""
		if cat.Info.IsHome {
			homeFlag = "HOME"
		}
		fmt.Printf("%-25s %-8d %-8d %-6s %s\n", 
			cat.Key, 
			cat.Info.MediaCount, 
			cat.Info.SubtitleCount,
			homeFlag,
			cat.Info.Name)
	}

	// Show categories with zero subtitles (might indicate filtering issues)
	fmt.Printf("\nCategories with Zero Subtitles:\n")
	zeroCount := 0
	for _, cat := range sortedCategories {
		if cat.Info.SubtitleCount == 0 && cat.Info.MediaCount > 0 {
			if zeroCount < 10 { // Show first 10
				fmt.Printf("  %s: %d media items, 0 subtitles (%s)\n", 
					cat.Key, cat.Info.MediaCount, cat.Info.Name)
			}
			zeroCount++
		}
	}
	if zeroCount > 10 {
		fmt.Printf("  ... and %d more categories with zero subtitles\n", zeroCount-10)
	}

	// Check if we can identify potential data loss
	fmt.Println("\nPotential Data Loss Analysis:")
	
	// Categories that might be excluded on Windows but not Linux
	suspiciousCategories := []string{
		"VODSJJMeetings", // This is in default exclude list
		"StudioFeatured",
		"StudioMonthlyPrograms", 
		"StudioTalks",
		"StudioNewsReports",
	}
	
	var suspiciousSubtitleCount int
	for _, catKey := range suspiciousCategories {
		if info, exists := categoryStats[catKey]; exists {
			fmt.Printf("  %s: %d subtitles (might be excluded on some platforms)\n", 
				catKey, info.SubtitleCount)
			suspiciousSubtitleCount += info.SubtitleCount
		}
	}
	
	fmt.Printf("\nIf the %d subtitles from suspicious categories were excluded,\n", suspiciousSubtitleCount)
	fmt.Printf("the count would be: %d (difference from %d: %d)\n", 
		totalWithSubtitles - suspiciousSubtitleCount, totalWithSubtitles, suspiciousSubtitleCount)
	
	fmt.Printf("\nUser reported Windows: 2310, Linux: 2935\n")
	fmt.Printf("Difference: %d subtitles missing on Windows\n", 2935-2310)
	
	if abs(suspiciousSubtitleCount - (2935-2310)) < 50 {
		fmt.Printf("*** LIKELY CAUSE: Category filtering differences! ***\n")
	}
}

type CategoryInfo struct {
	Name string
	MediaCount int
	SubtitleCount int
	IsHome bool
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}