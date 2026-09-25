// Package main provides the jwb-offline command for playing downloaded JW videos.
package main

import (
	"fmt"
	"os"

	"github.com/darkace1998/jw-scripts/internal/cli"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/player"
	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

var settings = &config.Settings{}
var replaySec int
var playerCmd []string

var rootCmd = &cobra.Command{
	Use:     "jwb-offline [DIR]",
	Short:   "Shuffle and play videos in DIR",
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		dir, err := cli.WorkDir(args, ".")
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
	rootCmd.Flags().IntVar(&replaySec, "replay-sec", 30, "seconds to replay after a restart")
	rootCmd.Flags().StringSliceVar(&playerCmd, "cmd", []string{"mpv", "--start", "{}", "--no-terminal"}, "video player command")
	rootCmd.Flags().CountVarP(&settings.Quiet, "quiet", "q", "less info, can be used multiple times (-q, -qq or --quiet=N)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(s *config.Settings) error {
	if s.WorkDir == "" {
		s.WorkDir = "."
	}

	vm := player.NewVideoManager(s)
	vm.SetCmd(playerCmd)
	vm.SetReplay(replaySec)

	return vm.Run()
}
