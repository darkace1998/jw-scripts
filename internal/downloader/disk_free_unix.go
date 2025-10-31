//go:build !windows

// Package downloader provides file downloading functionality with rate limiting.
package downloader

import (
	"fmt"
	"syscall"
)

func getFreeDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	if stat.Bsize < 0 {
		return 0, fmt.Errorf("invalid block size: %d", stat.Bsize)
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
