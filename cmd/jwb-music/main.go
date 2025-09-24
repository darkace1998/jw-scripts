package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
	"github.com/allejok96/jwb-go/internal/downloader"
	"github.com/allejok96/jwb-go/internal/output"
	"github.com/spf13/cobra"
)

var settings = &config.Settings{}

// musicCategories defines all the music-related categories available for download
var musicCategories = []string{
	"AudioOriginalSongs",
	"SJJMeetings",
	"SJJChorus",
	"SJJInstrumental",
	"AudioChildrenSongs",
	"KingdomMelodies",
}

var rootCmd = &cobra.Command{
	Use:   "jwb-music",
	Short: "Download all music files from jw.org",
	Long: `jwb-music is a specialized tool for downloading all music files from jw.org.

It downloads from all music-related categories including:
- Original Songs
- "Sing Out Joyfully" (Meetings, Vocals, Instrumental)
- Children's Songs
- Kingdom Melodies

By default, it downloads all available music files. Use flags to customize the behavior.`,
	Run: func(cmd *cobra.Command, args []string) {
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
	rootCmd.Flags().StringSliceVarP(&settings.IncludeCategories, "category", "c", musicCategories, "comma separated list of music categories to include")
	rootCmd.Flags().BoolVar(&settings.ListCategories, "list-categories", false, "list all available music categories")
	rootCmd.Flags().BoolVar(&settings.Checksums, "checksum", false, "validate MD5 checksums")
	rootCmd.Flags().BoolVarP(&settings.Download, "download", "d", true, "download music files (enabled by default)")
	rootCmd.Flags().StringSliceVar(&settings.ExcludeCategories, "exclude", []string{}, "comma separated list of categories to skip")
	rootCmd.Flags().BoolVar(&settings.OverwriteBad, "fix-broken", false, "check existing files and re-download them if they are broken")
	rootCmd.Flags().Int64Var(&settings.KeepFree, "free", 0, "disk space in MiB to keep free")
	rootCmd.Flags().BoolVarP(&settings.FriendlyFilenames, "friendly", "H", false, "save downloads with human readable names")
	rootCmd.Flags().StringVar(&settings.ImportDir, "import", "", "import of music files from this directory (offline)")
	rootCmd.Flags().StringVarP(&settings.Lang, "lang", "l", "E", "language code")
	rootCmd.Flags().BoolVarP(&settings.ListLanguages, "languages", "L", false, "display a list of valid language codes")
	rootCmd.Flags().Float64VarP(&settings.RateLimit, "limit-rate", "R", 25.0, "maximum download rate, in megabytes/s")
	rootCmd.Flags().StringVarP(&settings.Mode, "mode", "m", "", "output mode (filesystem, html, m3u, run, stdout, txt)")
	rootCmd.Flags().BoolVar(&settings.Warning, "no-warning", true, "do not warn when space limit seems wrong")
	rootCmd.Flags().IntVarP(&settings.Quiet, "quiet", "q", 0, "less info, can be used multiple times")
	rootCmd.Flags().BoolVar(&settings.SafeFilenames, "safe-filenames", runtime.GOOS == "windows", "use filesystem-safe filenames (automatically enabled on Windows)")
	rootCmd.Flags().Int64Var(&settings.MinDate, "since", 0, "only index music newer than this date (YYYY-MM-DD)")
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
		return nil
	}

	if s.Mode == "" && !s.Download && s.ImportDir == "" {
		return fmt.Errorf("please use --mode or --download (download is enabled by default)")
	}

	if s.Update {
		s.Append = true
		if s.Sort == "" {
			s.Sort = "newest"
		}
	}

	if s.WorkDir == "" {
		s.WorkDir = "./music"
	}
	if !strings.HasPrefix(s.Mode, "stdout") {
		s.SubDir = "jwb-music-" + s.Lang
	}

	// TODO: Implement offline import

	data, err := client.ParseBroadcasting()
	if err != nil {
		return err
	}

	if s.Download {
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
