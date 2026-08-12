//go:build windows

package binary

import (
	"os"
	"syscall"
)

// HardlinkCount returns the number of directory entries pointing at the
// same file data, or 1 when unknown. On Windows this uses
// GetFileInformationByHandle (BY_HANDLE_FILE_INFORMATION.NumberOfLinks).
//
// npm's Claude Code layout hardlinks bin/claude.exe to the platform
// subpackage copy; patching must commit in place for such files so every
// link sees the patched bytes.
func HardlinkCount(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 1
	}
	defer f.Close()

	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return 1
	}
	return int(info.NumberOfLinks)
}
