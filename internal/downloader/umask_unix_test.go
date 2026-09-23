//go:build !windows

package downloader

import "syscall"

// currentUmask returns the process umask without changing it.
func currentUmask() uint32 {
	mask := syscall.Umask(0)
	syscall.Umask(mask)
	return uint32(mask) // #nosec G115 - umask is a small non-negative value
}
