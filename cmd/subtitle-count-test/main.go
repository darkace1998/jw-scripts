package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	fmt.Println("Simulating Windows vs Linux subtitle count difference...")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Test with both SafeFilenames=true (Windows-like) and false (Linux-like)
	for _, safeFilenames := range []bool{false, true} {
		platform := "Linux-like"
		if safeFilenames {
			platform = "Windows-like"
		}

		fmt.Printf("\n=== Testing %s (SafeFilenames=%t) ===\n", platform, safeFilenames)

		settings := &config.Settings{
			Lang:              "E",
			IncludeCategories: []string{"VideoOnDemand"},
			ExcludeCategories: []string{"VODSJJMeetings"},
			SafeFilenames:     safeFilenames,
			Quiet:             1,
		}

		client := api.NewClient(settings)
		data, err := client.ParseBroadcasting()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing broadcasting data: %v\n", err)
			continue
		}

		// Simulate the download queue building process
		var mediaList []*api.Media
		for _, cat := range data {
			for _, item := range cat.Contents {
				if media, ok := item.(*api.Media); ok {
					mediaList = append(mediaList, media)
				}
			}
		}

		// Track unique filenames (simulating the behavior in downloader)
		var subtitleQueue []*api.Media
		seenFilenames := make(map[string]bool)
		var skippedDueToCollision int
		var emptyFilenames int

		for _, media := range mediaList {
			if media.SubtitleURL == "" {
				continue
			}
			// This mimics the logic in downloadAllSubtitles
			filename := media.SubtitleFilename

			if filename == "" {
				emptyFilenames++
				continue // Skip if filename generation failed
			}

			if seenFilenames[filename] {
				skippedDueToCollision++
				continue // Skip if we've seen this filename before
			}

			seenFilenames[filename] = true
			subtitleQueue = append(subtitleQueue, media)
		}

		fmt.Printf("Total media items: %d\n", len(mediaList))
		fmt.Printf("Media with subtitle URLs: %d\n", countWithSubtitles(mediaList))
		fmt.Printf("Media with empty subtitle filenames: %d\n", emptyFilenames)
		fmt.Printf("Media skipped due to filename collisions: %d\n", skippedDueToCollision)
		fmt.Printf("Final subtitle download queue: %d\n", len(subtitleQueue))
		fmt.Printf("Unique subtitle filenames: %d\n", len(seenFilenames))

		if emptyFilenames > 0 || skippedDueToCollision > 0 {
			fmt.Printf("*** POTENTIAL ISSUE: %d subtitles would be lost! ***\n", emptyFilenames+skippedDueToCollision)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("If the counts differ significantly between Windows-like and Linux-like,")
	fmt.Println("this explains the discrepancy the user reported!")
}

func countWithSubtitles(mediaList []*api.Media) int {
	count := 0
	for _, media := range mediaList {
		if media.SubtitleURL != "" {
			count++
		}
	}
	return count
}
