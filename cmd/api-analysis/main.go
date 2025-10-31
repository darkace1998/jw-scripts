// Package main provides API analysis tools for JW Broadcasting.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	fmt.Println("API Response Analysis - Root Categories Filtering")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Test the actual API call to see what root categories are returned
	baseURL := "https://data.jw-api.org/mediator/v1"
	lang := "E"

	fmt.Printf("Making API call to: %s/categories/%s/?detailed=1\n", baseURL, lang)

	resp, err := http.Get(fmt.Sprintf("%s/categories/%s/?detailed=1", baseURL, lang))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making API call: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "API returned status: %s\n", resp.Status)
		return
	}

	var rootResp api.RootCategoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&rootResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding JSON: %v\n", err)
		return
	}

	fmt.Printf("Total categories returned by API: %d\n\n", len(rootResp.Categories))

	// Analyze the filtering logic
	majorExcludeTags := map[string]bool{
		"WebExclude":   true,
		"JWORGExclude": true,
	}

	knownUseful := map[string]bool{
		"Audio": true,
	}

	var includedCategories []string
	var excludedCategories []string

	type RootCategory struct {
		Key         string   `json:"key"`
		Type        string   `json:"type"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}

	// Convert to our local type for easier handling
	var categories []RootCategory
	for _, cat := range rootResp.Categories {
		categories = append(categories, RootCategory{
			Key:         cat.Key,
			Type:        cat.Type,
			Name:        cat.Name,
			Description: cat.Description,
			Tags:        cat.Tags,
		})
	}

	// Sort by key for consistent output
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Key < categories[j].Key
	})

	for _, cat := range categories {
		// Check if this category has major exclude tags
		hasMajorExclude := false
		for _, tag := range cat.Tags {
			if majorExcludeTags[tag] {
				hasMajorExclude = true
				break
			}
		}

		// Apply the same filtering logic as the code
		if (cat.Type == "container" || cat.Type == "ondemand") &&
			(!hasMajorExclude || knownUseful[cat.Key]) {
			includedCategories = append(includedCategories, cat.Key)
		} else {
			excludedCategories = append(excludedCategories, cat.Key)
		}
	}

	fmt.Printf("Categories INCLUDED by filtering logic: %d\n", len(includedCategories))
	fmt.Printf("Categories EXCLUDED by filtering logic: %d\n", len(excludedCategories))
	fmt.Println()

	fmt.Println("INCLUDED CATEGORIES:")
	fmt.Println(strings.Repeat("-", 60))
	for _, key := range includedCategories {
		for _, cat := range categories {
			if cat.Key == key {
				fmt.Printf("%-25s %-12s %s\n", cat.Key, cat.Type, cat.Name)
				if len(cat.Tags) > 0 {
					fmt.Printf("%-25s %-12s Tags: %s\n", "", "", strings.Join(cat.Tags, ", "))
				}
				break
			}
		}
	}

	fmt.Println("\nEXCLUDED CATEGORIES:")
	fmt.Println(strings.Repeat("-", 60))
	for _, catKey := range excludedCategories {
		for _, cat := range categories {
			if cat.Key != catKey {
				continue
			}
			reason := "Type not container/ondemand"
			if cat.Type == "container" || cat.Type == "ondemand" {
				reason = "Has exclude tags"
			}
			fmt.Printf("%-25s %-12s %s [%s]\n", cat.Key, cat.Type, cat.Name, reason)
			if len(cat.Tags) > 0 {
				fmt.Printf("%-25s %-12s Tags: %s\n", "", "", strings.Join(cat.Tags, ", "))
			}
			break
		}
	}

	// Now test what categories are actually processed by the client
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("COMPARING WITH CLIENT LOGIC")
	fmt.Println(strings.Repeat("=", 60))

	settings := &config.Settings{
		Lang:              "E",
		IncludeCategories: []string{"VideoOnDemand"},
		ExcludeCategories: []string{"VODSJJMeetings"},
		Quiet:             1,
	}

	client := api.NewClient(settings)

	// Get root categories using the client logic
	clientCategories, err := client.GetRootCategories()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting root categories via client: %v\n", err)
		return
	}

	fmt.Printf("Categories returned by client.GetRootCategories(): %d\n", len(clientCategories))
	fmt.Printf("Difference from manual filtering: %d\n", len(includedCategories)-len(clientCategories))

	// Check for differences
	manualSet := make(map[string]bool)
	for _, cat := range includedCategories {
		manualSet[cat] = true
	}

	clientSet := make(map[string]bool)
	for _, cat := range clientCategories {
		clientSet[cat] = true
	}

	fmt.Println("\nCategories in manual filtering but NOT in client:")
	for cat := range manualSet {
		if !clientSet[cat] {
			fmt.Printf("  %s\n", cat)
		}
	}

	fmt.Println("\nCategories in client but NOT in manual filtering:")
	for cat := range clientSet {
		if !manualSet[cat] {
			fmt.Printf("  %s\n", cat)
		}
	}

	// Save the raw API response for comparison
	saveData := map[string]interface{}{
		"api_response":        rootResp,
		"manual_included":     includedCategories,
		"client_returned":     clientCategories,
		"excluded_categories": excludedCategories,
	}

	jsonData, _ := json.MarshalIndent(saveData, "", "  ")
	tmpFile, err := os.CreateTemp("", "api_analysis_*.json")
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
		fmt.Printf("Warning: could not save API analysis: %v\n", err)
	} else {
		fmt.Printf("\nAPI analysis saved to %s\n", tmpFile.Name())
	}
}

// RootCategory represents a top-level category from the API response
type RootCategory struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}
