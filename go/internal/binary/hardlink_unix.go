//go:build !windows

package binary

import (
	"os"
	"syscall"
)

// HardlinkCount returns the number of directory entries pointing at the
// same file data (st_nlink), or 1 when unknown.
func HardlinkCount(path string) int {
	info, err := os.Stat(path)
	if err != nil {
		return 1
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Nlink > 0 {
		return int(st.Nlink)
	}
	return 1
}
