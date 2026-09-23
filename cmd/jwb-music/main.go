// Package main provides the jwb-music command for downloading JW music files.
package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/cli"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/output"
	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

var settings = &config.Settings{}
var sinceDate string
var noWarning bool

// musicCategories defines all the music-related categories available for download
var musicCategories = []string{
	"AudioOriginalSongs",
	"SJJMeetings",
	"SJJChorus",
	"SJJInstrumental",
	"AudioChildrenSongs",
	"KingdomMelodies",
}

// JWBroadcastingCategory is the special category for JW Broadcasting MP3s
const JWBroadcastingCategory = "JWBroadcasting"

var rootCmd = &cobra.Command{
	Use:   "jwb-music",
	Short: "Download all music and audio files from jw.org",
	Long: `jwb-music is a specialized tool for downloading all music and audio files from jw.org.

It downloads from all music-related categories including:
- Original Songs
- "Sing Out Joyfully" (Meetings, Vocals, Instrumental)
- Children's Songs
- Kingdom Melodies

You can also download JW Broadcasting monthly programs as MP3 using:
  jwb-music -c JWBroadcasting

By default, it downloads all available music files. Use flags to customize the behavior.`,
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		dir, err := cli.WorkDir(args, "./music")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		settings.WorkDir = dir
		if err := run(settings); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().BoolVar(&settings.Append, "append", false, "append to file instead of overwriting")
	rootCmd.Flags().BoolVar(&settings.AudioOnly, "audio-only", true, "download only audio (MP3) files, skip video-only content (enabled by default)")
	rootCmd.Flags().StringSliceVarP(&settings.IncludeCategories, "category", "c", musicCategories, "comma separated list of music categories to include")
	rootCmd.Flags().BoolVar(&settings.ListCategories, "list-categories", false, "list all available music categories")
	rootCmd.Flags().BoolVar(&settings.Checksums, "checksum", false, "verify MD5 checksums of downloads (and of existing files with --fix-broken)")
	rootCmd.Flags().BoolVarP(&settings.Download, "download", "d", true, "download music files (enabled by default)")
	rootCmd.Flags().StringSliceVar(&settings.ExcludeCategories, "exclude", []string{}, "comma separated list of categories to skip")
	rootCmd.Flags().BoolVar(&settings.OverwriteBad, "fix-broken", false, "check the size (and MD5 with --checksum) of existing files and re-download broken ones")
	rootCmd.Flags().Int64Var(&settings.KeepFree, "free", 0, "disk space in MiB to keep free")
	rootCmd.Flags().BoolVarP(&settings.FriendlyFilenames, "friendly", "H", false, "save downloads with human readable names")
	rootCmd.Flags().StringVar(&settings.ImportDir, "import", "", "copy music files from this directory into the library (offline import)")
	rootCmd.Flags().StringVarP(&settings.Lang, "lang", "l", "E", "language code")
	rootCmd.Flags().BoolVarP(&settings.ListLanguages, "languages", "L", false, "display a list of valid language codes")
	rootCmd.Flags().BoolVar(&settings.WriteMetadata, "metadata", false, "write an .nfo metadata file next to each download (read by Jellyfin, Emby and Kodi; Plex via an NFO agent)")
	rootCmd.Flags().Float64VarP(&settings.RateLimit, "limit-rate", "R", 25.0, "maximum download rate, in megabytes/s")
	rootCmd.Flags().StringVarP(&settings.Mode, "mode", "m", "", "output mode (filesystem, html, m3u, run, stdout, txt)")
	rootCmd.Flags().StringVarP(&settings.OutputFilename, "output", "o", "", "output filename for txt/m3u/html modes")
	rootCmd.Flags().BoolVar(&noWarning, "no-warning", false, "do not warn when the disk space limit (--free) seems wrong")
	rootCmd.Flags().CountVarP(&settings.Quiet, "quiet", "q", "less info, can be used multiple times (-q, -qq or --quiet=N)")
	rootCmd.Flags().BoolVar(&settings.SafeFilenames, "safe-filenames", runtime.GOOS == "windows", "use filesystem-safe filenames (automatically enabled on Windows)")
	rootCmd.Flags().StringVar(&sinceDate, "since", "", "only index music newer than this date (YYYY-MM-DD)")
	rootCmd.Flags().StringVar(&settings.Sort, "sort", "", "sort output (newest, oldest, name, random)")
	rootCmd.Flags().BoolVar(&settings.Update, "update", false, "update existing categories with the latest music")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(s *config.Settings) error {
	s.Warning = !noWarning

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
		// Show the music categories that will be downloaded
		fmt.Println("Available music categories:")

		for _, cat := range musicCategories {
			catResp, err := client.GetCategory(s.Lang, cat)
			if err != nil {
				if s.Quiet < 2 {
					fmt.Printf("  %s (could not fetch details)\n", cat)
				}
			} else {
				fmt.Printf("  %s (%s)\n", catResp.Category.Name, cat)
			}
		}
		// Also show the JW Broadcasting option
		fmt.Printf("  JW Broadcasting - Audio (%s)\n", JWBroadcastingCategory)
		return nil
	}

	if s.Mode == "" && !s.Download && s.ImportDir == "" {
		return fmt.Errorf("please use --mode or --download (download is enabled by default)")
	}
	if err := output.ValidateMode(s.Mode); err != nil {
		return err
	}

	if s.Update {
		s.Append = true
		if s.Sort == "" {
			s.Sort = "newest"
		}
	}

	// Parse --since date if provided
	if sinceDate != "" {
		t, err := time.Parse("2006-01-02", sinceDate)
		if err != nil {
			return fmt.Errorf("invalid --since date %q (expected YYYY-MM-DD): %w", sinceDate, err)
		}
		s.MinDate = t.Unix()
	}

	// Convert MiB to bytes for disk space calculations
	if s.KeepFree < 0 || s.KeepFree > math.MaxInt64/(1024*1024) {
		return fmt.Errorf("invalid --free value %d: must be between 0 and %d MiB", s.KeepFree, int64(math.MaxInt64/(1024*1024)))
	}
	s.KeepFree *= 1024 * 1024

	if s.WorkDir == "" {
		s.WorkDir = "./music"
	}
	if !strings.HasPrefix(s.Mode, "stdout") {
		s.SubDir = "jwb-music-" + s.Lang
	}

	return cli.Process(s, func() ([]*api.Category, error) { return indexMusic(s, client) })
}

// indexMusic indexes the requested music categories, including the special
// JW Broadcasting audio category. Failures are returned together with the
// data that could be indexed.
func indexMusic(s *config.Settings, client *api.Client) ([]*api.Category, error) {
	var data []*api.Category
	var errs []error

	hasJWBroadcasting := false
	var otherCategories []string
	for _, cat := range s.IncludeCategories {
		if cat == JWBroadcastingCategory {
			hasJWBroadcasting = true
		} else {
			otherCategories = append(otherCategories, cat)
		}
	}

	// Fetch JW Broadcasting MP3s if requested
	if hasJWBroadcasting {
		jwbData, err := client.GetBroadcastingMP3s()
		if err != nil {
			errs = append(errs, fmt.Errorf("JW Broadcasting: %w", err))
		}
		data = append(data, jwbData...)
	}

	// Fetch other categories using the standard API
	if len(otherCategories) > 0 {
		s.IncludeCategories = otherCategories
		otherData, err := client.ParseBroadcasting()
		if err != nil {
			errs = append(errs, err)
		}
		data = append(data, otherData...)
	}

	// Both sources share one download directory, so names must be unique
	// across them.
	api.AssignFilenames(s, data)
	return data, errors.Join(errs...)
}
