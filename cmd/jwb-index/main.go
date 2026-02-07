// Package main provides the jwb-index command for downloading JW Broadcasting media.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/downloader"
	"github.com/darkace1998/jw-scripts/internal/output"
	"github.com/spf13/cobra"
)

var settings = &config.Settings{}

var rootCmd = &cobra.Command{
	Use:   "jwb-index",
	Short: "Index or download media from jw.org",
	Run: func(_ *cobra.Command, args []string) {
		if len(args) > 0 {
			settings.WorkDir = args[0]
		}
		if err := run(settings); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().BoolVar(&settings.Append, "append", false, "append to file instead of overwriting")
	rootCmd.Flags().BoolVar(&settings.AudioOnly, "audio-only", false, "download only audio (MP3) files, skip video-only content")
	rootCmd.Flags().StringSliceVarP(&settings.IncludeCategories, "category", "c", []string{"VideoOnDemand"}, "comma separated list of categories to index (use --list-categories-all to see available categories)")
	rootCmd.Flags().BoolVar(&settings.ListCategories, "list-categories-all", false, "list all available root categories")
	rootCmd.Flags().BoolVar(&settings.Checksums, "checksum", false, "validate MD5 checksums")
	rootCmd.Flags().BoolVar(&settings.CleanAllSymlinks, "clean-symlinks", false, "remove all old symlinks (mode=filesystem)")
	rootCmd.Flags().StringSliceVar(&settings.Command, "command", []string{}, "command to execute in run mode")
	rootCmd.Flags().BoolVarP(&settings.Download, "download", "d", false, "download media files")
	rootCmd.Flags().BoolVar(&settings.DownloadSubtitles, "download-subtitles", false, "download VTT subtitle files")
	rootCmd.Flags().StringSliceVar(&settings.ExcludeCategories, "exclude", []string{"VODSJJMeetings"}, "comma separated list of categories to skip")
	rootCmd.Flags().BoolVar(&settings.OverwriteBad, "fix-broken", false, "check existing files and re-download them if they are broken")
	rootCmd.Flags().Int64Var(&settings.KeepFree, "free", 0, "disk space in MiB to keep free")
	rootCmd.Flags().BoolVarP(&settings.FriendlyFilenames, "friendly", "H", false, "save downloads with human readable names")
	rootCmd.Flags().BoolVar(&settings.HardSubtitles, "hard-subtitles", false, "prefer videos with hard-coded subtitles")
	rootCmd.Flags().StringVar(&settings.ImportDir, "import", "", "import of media files from this directory (offline)")
	rootCmd.Flags().StringVarP(&settings.Lang, "lang", "l", "E", "language code")
	rootCmd.Flags().BoolVarP(&settings.ListLanguages, "languages", "L", false, "display a list of valid language codes")
	rootCmd.Flags().BoolVarP(&settings.Latest, "latest", "D", false, "fetch subtitles and videos from the past 31 days up to today (31-day window ending today)")
	rootCmd.Flags().Float64VarP(&settings.RateLimit, "limit-rate", "R", 25.0, "maximum download rate, in megabytes/s")
	rootCmd.Flags().StringVarP(&settings.PrintCategory, "list-categories", "C", "", "print a list of (sub) category names")
	rootCmd.Flags().StringVarP(&settings.Mode, "mode", "m", "", "output mode (filesystem, html, m3u, run, stdout, txt)")
	rootCmd.Flags().BoolVar(&settings.Warning, "no-warning", true, "do not warn when space limit seems wrong")
	rootCmd.Flags().IntVarP(&settings.Quality, "quality", "Q", 720, "maximum video quality")
	rootCmd.Flags().IntVarP(&settings.Quiet, "quiet", "q", 0, "less info, can be used multiple times")
	rootCmd.Flags().BoolVar(&settings.SafeFilenames, "safe-filenames", runtime.GOOS == "windows", "use filesystem-safe filenames (automatically enabled on Windows)")
	rootCmd.Flags().Int64Var(&settings.MinDate, "since", 0, "only index media newer than this date (YYYY-MM-DD)")
	rootCmd.Flags().StringVar(&settings.Sort, "sort", "", "sort output (newest, oldest, name, random)")
	rootCmd.Flags().BoolVar(&settings.Update, "update", false, "update existing categories with the latest videos")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(s *config.Settings) error {
	client := api.NewClient(s)

	if s.ListLanguages {
		langs, err := client.GetLanguages()
		if err != nil {
			return err
		}
		fmt.Println("language codes:")
		for _, l := range langs {
			fmt.Printf("%3s  %s\n", l.Code, l.Name)
		}
		return nil
	}

	if s.ListCategories {
		rootCategories, err := client.GetRootCategories()
		if err != nil {
			return fmt.Errorf("failed to get root categories: %v", err)
		}

		fmt.Println("Available root categories:")
		for _, cat := range rootCategories {
			catResp, err := client.GetCategory(s.Lang, cat)
			if err != nil {
				if s.Quiet < 2 {
					fmt.Printf("  %s (could not fetch details)\n", cat)
				}
			} else {
				fmt.Printf("  %s (%s)\n", catResp.Category.Name, cat)
			}
		}
		return nil
	}

	if s.PrintCategory != "" {
		catResp, err := client.GetCategory(s.Lang, s.PrintCategory)
		if err != nil {
			return fmt.Errorf("failed to get category %s: %v", s.PrintCategory, err)
		}

		fmt.Printf("Category: %s (%s)\n", catResp.Category.Name, catResp.Category.Key)
		if len(catResp.Category.Subcategories) > 0 {
			fmt.Println("Subcategories:")
			for _, sub := range catResp.Category.Subcategories {
				fmt.Printf("  %s (%s)\n", sub.Name, sub.Key)
			}
		} else {
			fmt.Println("No subcategories found.")
		}
		return nil
	}

	if s.Mode == "" && !s.Download && !s.DownloadSubtitles && s.ImportDir == "" {
		return fmt.Errorf("please use --mode or --download")
	}

	if s.Update {
		s.Append = true
		s.Latest = true
		if s.Sort == "" {
			s.Sort = "newest"
		}
	}

	if s.Latest {
		// Set date range for 31-day window: from today back to 31 days ago when --latest flag is used
		now := time.Now()
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		thirtyOneDaysAgo := startOfToday.AddDate(0, 0, -31)
		endOfToday := startOfToday.AddDate(0, 0, 1).Add(-time.Nanosecond) // End of today

		s.MinDate = thirtyOneDaysAgo.Unix()
		s.MaxDate = endOfToday.Unix()

		if s.Quiet < 1 {
			fmt.Fprintf(os.Stderr, "filtering to content from %s through %s (past 31 days)\n",
				thirtyOneDaysAgo.Format("2006-01-02"), endOfToday.Format("2006-01-02"))
		}
	}

	if s.Mode == "run" {
		if len(s.Command) == 0 {
			return fmt.Errorf("run mode requires a command to be specified")
		}
		// Run mode is handled by the output.CreateOutput function
		// which will use the CommandWriter to execute the configured command
	}

	if s.WorkDir == "" {
		s.WorkDir = "."
	}
	if !strings.HasPrefix(s.Mode, "stdout") {
		s.SubDir = "jwb-" + s.Lang
	}

	// TODO: Implement offline import

	data, err := client.ParseBroadcasting()
	if err != nil {
		return err
	}

	if s.Download || s.DownloadSubtitles {
		if err := downloader.DownloadAll(s, data); err != nil {
			return err
		}
	}

	if s.Mode != "" {
		if err := output.CreateOutput(s, data); err != nil {
			return err
		}
	}

	return nil
}
