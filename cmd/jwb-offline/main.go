// Package main provides the jwb-offline command for playing downloaded JW videos.
package main

import (
	"fmt"
	"os"

	"github.com/allejok96/jwb-go/internal/config"
	"github.com/allejok96/jwb-go/internal/player"
	"github.com/spf13/cobra"
)

var settings = &config.Settings{}
var replaySec int
var playerCmd []string

var rootCmd = &cobra.Command{
	Use:   "jwb-offline [DIR]",
	Short: "Shuffle and play videos in DIR",
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
	rootCmd.Flags().IntVar(&replaySec, "replay-sec", 30, "seconds to replay after a restart")
	rootCmd.Flags().StringSliceVar(&playerCmd, "cmd", []string{"omxplayer", "--pos", "{}", "--no-osd"}, "video player command")
	rootCmd.Flags().IntVarP(&settings.Quiet, "quiet", "q", 0, "less info, can be used multiple times")
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
