package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	fmt.Println("Comprehensive Subtitle Count Analysis")
	fmt.Println("Searching for exactly 625 missing subtitles...")
	fmt.Println("=" + strings.Repeat("=", 60))

	settings := &config.Settings{
		Lang:              "E",
		IncludeCategories: []string{"VideoOnDemand"},
		ExcludeCategories: []string{"VODSJJMeetings"},
		Quiet:             1,
	}

	client := api.NewClient(settings)
	data, err := client.ParseBroadcasting()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing broadcasting data: %v\n", err)
		os.Exit(1)
	}

	// Analyze all categories
	type CategoryData struct {
		Key           string `json:"key"`
		Name          string `json:"name"`
		MediaCount    int    `json:"media_count"`
		SubtitleCount int    `json:"subtitle_count"`
		IsHome        bool   `json:"is_home"`
	}

	var categories []CategoryData
	var totalWithSubtitles int

	for _, cat := range data {
		var mediaCount int
		var subtitleCount int

		for _, item := range cat.Contents {
			if media, ok := item.(*api.Media); ok {
				mediaCount++
				if media.SubtitleURL != "" {
					subtitleCount++
					totalWithSubtitles++
				}
			}
		}

		categories = append(categories, CategoryData{
			Key:           cat.Key,
			Name:          cat.Name,
			MediaCount:    mediaCount,
			SubtitleCount: subtitleCount,
			IsHome:        cat.Home,
		})
	}

	// Sort by subtitle count
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].SubtitleCount > categories[j].SubtitleCount
	})

	fmt.Printf("Total subtitle count: %d\n", totalWithSubtitles)
	fmt.Printf("User reported Linux: 2935, Windows: 2310\n")
	fmt.Printf("Difference: %d\n", 2935-2310)
	fmt.Println()

	// Analysis 1: Look for categories that could sum to ~625
	fmt.Println("ANALYSIS 1: Combination of categories that sum to ~625 subtitles")
	fmt.Println("=" + strings.Repeat("-", 60))

	// Check various combinations
	combinations := [][]string{
		// Studio categories
		{"StudioMonthlyPrograms", "StudioTalks", "StudioNewsReports", "StudioFeatured"},
		// Convention categories
		{"2025Convention", "2024Convention", "2023Convention", "2022Convention"},
		// Children categories
		{"ChildrensSongs", "ChildrenSongs", "SeriesBJFSongs", "BJF"},
		// Bible categories
		{"BibleBooks", "SeriesBibleBooks", "VODBiblePrinciples"},
		// Morning worship + others
		{"VODPgmEvtMorningWorship", "VODIntExpEndurance"},
		// Top 5 categories
		{"VODPgmEvtMorningWorship", "VODIntExpEndurance", "VODPgmEvtGilead", "VODOriginalSongs", "StudioMonthlyPrograms"},
	}

	catMap := make(map[string]CategoryData)
	for _, cat := range categories {
		catMap[cat.Key] = cat
	}

	for i, combo := range combinations {
		var totalSubs int
		var comboDetails []string

		for _, catKey := range combo {
			if cat, exists := catMap[catKey]; exists {
				totalSubs += cat.SubtitleCount
				comboDetails = append(comboDetails, fmt.Sprintf("%s(%d)", catKey, cat.SubtitleCount))
			}
		}

		fmt.Printf("Combo %d: %d subtitles - %s\n", i+1, totalSubs, strings.Join(comboDetails, " + "))
		if abs(totalSubs-625) < 50 {
			fmt.Printf("*** CLOSE MATCH! Difference: %d ***\n", abs(totalSubs-625))
		}
	}

	// Analysis 2: Look for patterns in category names
	fmt.Println("\nANALYSIS 2: Categories by pattern")
	fmt.Println("=" + strings.Repeat("-", 60))

	patterns := map[string][]CategoryData{
		"Studio":     {},
		"Convention": {},
		"Children":   {},
		"Series":     {},
		"AD (Audio)": {},
		"VODPgmEvt":  {},
		"VODIntExp":  {},
	}

	for _, cat := range categories {
		if strings.Contains(cat.Key, "Studio") {
			patterns["Studio"] = append(patterns["Studio"], cat)
		}
		if strings.Contains(cat.Key, "Convention") {
			patterns["Convention"] = append(patterns["Convention"], cat)
		}
		if strings.Contains(cat.Key, "Children") {
			patterns["Children"] = append(patterns["Children"], cat)
		}
		if strings.Contains(cat.Key, "Series") {
			patterns["Series"] = append(patterns["Series"], cat)
		}
		if strings.HasSuffix(cat.Key, "AD") {
			patterns["AD (Audio)"] = append(patterns["AD (Audio)"], cat)
		}
		if strings.HasPrefix(cat.Key, "VODPgmEvt") {
			patterns["VODPgmEvt"] = append(patterns["VODPgmEvt"], cat)
		}
		if strings.HasPrefix(cat.Key, "VODIntExp") {
			patterns["VODIntExp"] = append(patterns["VODIntExp"], cat)
		}
	}

	for pattern, cats := range patterns {
		var totalSubs int
		for _, cat := range cats {
			totalSubs += cat.SubtitleCount
		}
		fmt.Printf("%-15s: %3d categories, %4d subtitles\n", pattern, len(cats), totalSubs)

		if abs(totalSubs-625) < 100 {
			fmt.Printf("*** POTENTIAL MATCH! ***\n")
			for _, cat := range cats {
				if cat.SubtitleCount > 0 {
					fmt.Printf("    %s: %d subtitles\n", cat.Key, cat.SubtitleCount)
				}
			}
		}
	}

	// Analysis 3: Identify exactly 625 difference
	fmt.Println("\nANALYSIS 3: Finding exact 625 subtitle difference")
	fmt.Println("=" + strings.Repeat("-", 60))

	// Check if specific category sets sum to 625
	targetDiff := 625
	fmt.Printf("Looking for category combinations that sum to exactly %d...\n", targetDiff)

	// Try different approaches
	approaches := []struct {
		name       string
		categories []string
	}{
		{
			"Studio + top categories",
			[]string{"StudioMonthlyPrograms", "StudioTalks", "StudioNewsReports", "VODIntExpEndurance", "VODPgmEvtGilead", "VODOriginalSongs"},
		},
		{
			"Convention years",
			[]string{"2025Convention", "2024Convention", "2023Convention", "2022Convention", "2021Convention", "2020Convention"},
		},
		{
			"Morning Worship + Gilead + Others",
			[]string{"VODPgmEvtMorningWorship", "VODPgmEvtGilead", "VODIntExpEndurance"},
		},
	}

	for _, approach := range approaches {
		var total int
		var details []string

		for _, catKey := range approach.categories {
			if cat, exists := catMap[catKey]; exists {
				total += cat.SubtitleCount
				details = append(details, fmt.Sprintf("%s(%d)", catKey, cat.SubtitleCount))
			}
		}

		fmt.Printf("%-30s: %4d subtitles - %s\n", approach.name, total, strings.Join(details, " + "))
		if abs(total-targetDiff) < 20 {
			fmt.Printf("*** VERY CLOSE MATCH! ***\n")
		}
	}

	// Save detailed data for further analysis
	saveData := map[string]interface{}{
		"total_subtitles": totalWithSubtitles,
		"categories":      categories,
		"patterns":        patterns,
	}

	jsonData, _ := json.MarshalIndent(saveData, "", "  ")
	tmpFile, err := os.CreateTemp("", "comprehensive_analysis_*.json")
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
		fmt.Println("\nDetailed analysis saved to /tmp/comprehensive_analysis.json")
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
