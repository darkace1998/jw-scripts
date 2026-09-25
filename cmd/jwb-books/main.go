// Package main provides the jwb-books command for downloading JW publications.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/darkace1998/jw-scripts/internal/books"
	"github.com/darkace1998/jw-scripts/internal/config"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// issuePattern matches magazine issues in YYYYMM format.
var issuePattern = regexp.MustCompile(`^\d{6}$`)

// fail prints an error message to stderr and exits with status 1.
func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

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
		issue          = flag.String("issue", "", "Magazine issue to download (YYYYMM, default: latest)")
		writeMetadata  = flag.Bool("metadata", false, "Write an .nfo metadata file next to each download")
		quiet          = flag.Int("quiet", 0, "Less output: 1 hides progress, 2 hides everything but errors")
		rateLimit      = flag.Float64("limit-rate", 0, "Maximum download rate in megabytes/s (0 = unlimited)")
		showVersion    = flag.Bool("version", false, "Show the version")
		help           = flag.Bool("help", false, "Show help information")
	)

	flag.Usage = printHelp
	flag.Parse()

	if *help {
		printHelp()
		return
	}
	if *showVersion {
		fmt.Println("jwb-books version", version)
		return
	}
	if flag.NArg() > 0 {
		fail("unexpected argument %q (use --help for usage)", flag.Arg(0))
	}
	if *issue != "" && !issuePattern.MatchString(*issue) {
		fail("invalid --issue %q (expected YYYYMM, e.g. 202601)", *issue)
	}

	// Create settings
	settings := &config.Settings{
		Quiet:         *quiet,
		RateLimit:     *rateLimit,
		WriteMetadata: *writeMetadata,
		Issue:         *issue,
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
	os.Exit(2)
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
	fmt.Println("  --issue YYYYMM        Magazine issue to download (default: latest)")
	fmt.Println("  --metadata            Write an .nfo metadata file next to each download")
	fmt.Println("  --quiet N             Less output (1: no progress, 2: errors only)")
	fmt.Println("  --limit-rate MB       Maximum download rate in megabytes/s")
	fmt.Println("  --version             Show the version")
	fmt.Println("  --help                Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  jwb-books --list-languages")
	fmt.Println("  jwb-books --list-categories --language S")
	fmt.Println("  jwb-books --category daily-text --language E --format pdf")
	fmt.Println("  jwb-books --category bible --language S --format epub")
	fmt.Println("  jwb-books --category magazines --issue 202601 --format epub")
	fmt.Println("  jwb-books --search \"daily\" --language F")
	fmt.Println()
	fmt.Println("Supported Languages: English (E), Spanish (S), French (F), German (X), and 20+ more")
	fmt.Println("Supported Formats: PDF, EPUB, MP3, MP4, RTF, BRL")
}

func handleListLanguages(client *books.Client) {
	languages, err := client.GetSupportedLanguages()
	if err != nil {
		fail("getting languages: %v", err)
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
		fail("getting categories: %v", err)
	}

	lang := getLanguageName(client, language)
	fmt.Printf("Available Categories for %s:\n", lang)
	fmt.Println("=============================")
	for i := range categories {
		category := &categories[i]
		fmt.Printf("  %s - %s\n", category.Key, category.Name)
		fmt.Printf("    %s\n", category.Description)
		fmt.Printf("    Publications: %s\n", strings.Join(category.Publications, ", "))
		fmt.Println()
	}
}

func handleSearch(client *books.Client, language, query string) {
	results, err := client.SearchBooks(language, query)
	if err != nil {
		fail("searching: %v", err)
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

		// List available formats (deduplicated)
		seen := make(map[string]bool)
		var formats []string
		for _, file := range book.Files {
			f := string(file.Format)
			if !seen[f] {
				seen[f] = true
				formats = append(formats, f)
			}
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
		fail("unknown format '%s'. Use --list-formats to see supported formats.", formatStr)
	}

	// Get category. Publications that could not be found are reported, but
	// the ones that were found are still downloaded.
	category, lookupErr := client.GetCategory(language, categoryKey)
	if category == nil {
		fail("getting category '%s': %v\nUse --list-categories to see available categories.", categoryKey, lookupErr)
	}
	if lookupErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: some publications are unavailable: %v\n", lookupErr)
	}

	if len(category.Books) == 0 {
		fail("no publications found in category '%s' for language '%s'", categoryKey, getLanguageName(client, language))
	}

	lang := getLanguageName(client, language)
	fmt.Printf("Downloading category '%s' in %s format %s to '%s'...\n",
		category.Name, lang, strings.ToUpper(formatStr), outputDir)
	fmt.Println()

	// Download the category
	if err := downloader.DownloadCategory(category, format, outputDir); err != nil {
		fail("downloading category: %v", err)
	}
	if lookupErr != nil {
		fail("not all publications in category '%s' were available", categoryKey)
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
