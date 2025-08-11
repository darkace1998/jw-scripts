//go:build windows

package downloader

import (
	"syscall"
	"unsafe"
)

var (
	kernel32            *syscall.DLL
	getDiskFreeSpaceExW *syscall.Proc
	dllLoadErr          error
)

func init() {
	kernel32, dllLoadErr = syscall.LoadDLL("kernel32.dll")
	if dllLoadErr != nil {
		return
	}
	getDiskFreeSpaceExW, dllLoadErr = kernel32.FindProc("GetDiskFreeSpaceExW")
}

func getFreeDiskSpace(path string) (uint64, error) {
	if dllLoadErr != nil {
		return 0, dllLoadErr
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
