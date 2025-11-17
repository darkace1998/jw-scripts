// Package main provides the jwb-books command for downloading JW publications.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/allejok96/jwb-go/internal/books"
	"github.com/allejok96/jwb-go/internal/config"
)

func main() {
	// Command line flags
	var (
		listCategories = flag.Bool("list-categories", false, "List all available categories")
		listLanguages  = flag.Bool("list-languages", false, "List all supported languages")
		listFormats    = flag.Bool("list-formats", false, "List all supported formats")
		category       = flag.String("category", "", "Category to download (use --list-categories to see options)")
		language       = flag.String("language", "E", "Language code (use --list-languages to see options)")
		format         = flag.String("format", "pdf", "Format to download (use --list-formats to see options)")
		search         = flag.String("search", "", "Search for publications")
		outputDir      = flag.String("output", "downloads", "Output directory for downloads")
		help           = flag.Bool("help", false, "Show help information")
	)

	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Create settings
	settings := &config.Settings{
		Quiet:     0,
		RateLimit: 0,
	}

	// Create client and downloader
	client := books.NewClient(settings)
	downloader := books.NewDownloader(settings)

	// Handle list commands
	if *listLanguages {
		handleListLanguages(client)
		return
	}

	if *listFormats {
		handleListFormats(client)
		return
	}

	if *listCategories {
		handleListCategories(client, *language)
		return
	}

	if *search != "" {
		handleSearch(client, *language, *search)
		return
	}

	if *category != "" {
		handleDownloadCategory(client, downloader, *language, *category, *format, *outputDir)
		return
	}

	// Default: show help
	printHelp()
}

func printHelp() {
	fmt.Println("JW.org Book Downloader")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("A command-line tool for downloading JW.org publications in multiple languages and formats.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  jwb-books [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --list-languages      List all supported languages")
	fmt.Println("  --list-formats        List all supported formats")
	fmt.Println("  --list-categories     List all available categories")
	fmt.Println("  --language CODE       Language code (default: E for English)")
	fmt.Println("  --category NAME       Category to download")
	fmt.Println("  --format FORMAT       Format to download (default: pdf)")
	fmt.Println("  --search QUERY        Search for publications")
	fmt.Println("  --output DIR          Output directory (default: downloads)")
	fmt.Println("  --help                Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  jwb-books --list-languages")
	fmt.Println("  jwb-books --list-categories --language S")
	fmt.Println("  jwb-books --category daily-text --language E --format pdf")
	fmt.Println("  jwb-books --category bible --language S --format epub")
	fmt.Println("  jwb-books --search \"daily\" --language F")
	fmt.Println()
	fmt.Println("Supported Languages: English (E), Spanish (S), French (F), German (X), and 20+ more")
	fmt.Println("Supported Formats: PDF, EPUB, MP3, MP4, RTF, BRL")
}

func handleListLanguages(client *books.Client) {
	languages, err := client.GetSupportedLanguages()
	if err != nil {
		fmt.Printf("Error getting languages: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Supported Languages:")
	fmt.Println("===================")
	for _, lang := range languages {
		direction := ""
		if lang.Direction == "rtl" {
			direction = " (RTL)"
		}
		fmt.Printf("  %s = %s%s\n", lang.Code, lang.Name, direction)
	}
	fmt.Printf("\nTotal: %d languages supported\n", len(languages))
}

func handleListFormats(client *books.Client) {
	formats := client.GetSupportedFormats()

	fmt.Println("Supported Formats:")
	fmt.Println("==================")
	for _, format := range formats {
		var description string
		switch format {
		case books.FormatPDF:
			description = "Portable Document Format"
		case books.FormatEPUB:
			description = "Electronic Publication"
		case books.FormatMP3:
			description = "Audio (MP3)"
		case books.FormatMP4:
			description = "Video (MP4)"
		case books.FormatRTF:
			description = "Rich Text Format"
		case books.FormatBRL:
			description = "Braille"
		default:
			description = "Unknown format"
		}
		fmt.Printf("  %s - %s\n", format, description)
	}
}

func handleListCategories(client *books.Client, language string) {
	categories, err := client.GetCategories()
	if err != nil {
		fmt.Printf("Error getting categories: %v\n", err)
		os.Exit(1)
	}

	lang := getLanguageName(client, language)
	fmt.Printf("Available Categories for %s:\n", lang)
	fmt.Println("=============================")
	for _, category := range categories {
		fmt.Printf("  %s - %s\n", category.Key, category.Name)
		fmt.Printf("    %s\n", category.Description)
		fmt.Printf("    Publications: %s\n", strings.Join(category.Publications, ", "))
		fmt.Println()
	}
}

func handleSearch(client *books.Client, language, query string) {
	results, err := client.SearchBooks(language, query)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		os.Exit(1)
	}

	lang := getLanguageName(client, language)
	fmt.Printf("Search Results for \"%s\" in %s:\n", query, lang)
	fmt.Println("=====================================")

	if len(results) == 0 {
		fmt.Println("No publications found matching your search.")
		return
	}

	for i := range results {
		book := &results[i]
		fmt.Printf("  Title: %s\n", book.Title)
		fmt.Printf("  ID: %s\n", book.ID)
		if book.Description != "" {
			fmt.Printf("  Description: %s\n", book.Description)
		}
		if book.Issue != "" {
			fmt.Printf("  Issue: %s\n", book.Issue)
		}

		// List available formats
		var formats []string
		for _, file := range book.Files {
			formats = append(formats, string(file.Format))
		}
		if len(formats) > 0 {
			fmt.Printf("  Available formats: %s\n", strings.Join(formats, ", "))
		}
		fmt.Println()
	}
}

func handleDownloadCategory(client *books.Client, downloader *books.Downloader, language, categoryKey, formatStr, outputDir string) {
	// Parse format
	format := parseFormat(formatStr)
	if format == books.FormatUnknown {
		fmt.Printf("Error: Unknown format '%s'. Use --list-formats to see supported formats.\n", formatStr)
		os.Exit(1)
	}

	// Get category
	category, err := client.GetCategory(language, categoryKey)
	if err != nil {
		fmt.Printf("Error getting category '%s': %v\n", categoryKey, err)
		fmt.Println("Use --list-categories to see available categories.")
		os.Exit(1)
	}

	if len(category.Books) == 0 {
		fmt.Printf("No books found in category '%s' for language '%s'\n", categoryKey, getLanguageName(client, language))
		return
	}

	lang := getLanguageName(client, language)
	fmt.Printf("Downloading category '%s' in %s format %s to '%s'...\n",
		category.Name, lang, strings.ToUpper(formatStr), outputDir)
	fmt.Println()

	// Download the category
	err = downloader.DownloadCategory(category, format, outputDir)
	if err != nil {
		fmt.Printf("Error downloading category: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Download completed!")
}

func parseFormat(formatStr string) books.BookFormat {
	switch strings.ToLower(formatStr) {
	case "pdf":
		return books.FormatPDF
	case "epub":
		return books.FormatEPUB
	case "mp3":
		return books.FormatMP3
	case "mp4":
		return books.FormatMP4
	case "rtf":
		return books.FormatRTF
	case "brl", "braille":
		return books.FormatBRL
	default:
		return books.FormatUnknown
	}
}

func getLanguageName(client *books.Client, langCode string) string {
	languages, err := client.GetSupportedLanguages()
	if err != nil {
		return langCode
	}

	for _, lang := range languages {
		if lang.Code == langCode {
			return fmt.Sprintf("%s (%s)", lang.Name, lang.Code)
		}
	}
	return langCode
}
