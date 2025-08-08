//go:build windows
// +build windows

package downloader

import (
	"syscall"
	"unsafe"
)

func getFreeDiskSpace(path string) (uint64, error) {
	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return 0, err
	}
	defer kernel32.Release()

	getDiskFreeSpaceExW, err := kernel32.FindProc("GetDiskFreeSpaceExW")
	if err != nil {
		return 0, err
	}

	var freeBytes int64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	ret, _, err := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		0,
		0,
	)

	if ret == 0 {
		return 0, err
	}

	return uint64(freeBytes), nil
}
