//go:build !linux && !darwin && !windows

package downloader

import (
	"errors"
	"fmt"
	"runtime"
)

// getFreeDiskSpace is not implemented on this platform; --free reports an
// error instead of silently doing nothing.
func getFreeDiskSpace(_ string) (uint64, error) {
	return 0, fmt.Errorf("checking free disk space on %s: %w", runtime.GOOS, errors.ErrUnsupported)
}
