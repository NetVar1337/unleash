package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
)

// NewStatusCmd creates the "status" cobra command.
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show install state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	tgt, kind := target.FindTarget()
	patchList := loadAllPatches()

	fmt.Printf("%sunleash status%s\n", console.B, console.X)
	fmt.Printf("  patches : %d\n", len(patchList))

	if tgt != "" {
		var label string
		if kind == "bun_sea" {
			switch runtime.GOOS {
			case "darwin":
				label = "Bun SEA (Mach-O)"
			case "windows":
				label = "Bun SEA (PE)"
			default:
				label = "Bun SEA (ELF)"
			}
		} else {
			label = "cli.js (JS)"
		}
		info, _ := os.Stat(tgt)
		sizeMB := int64(0)
		if info != nil {
			sizeMB = info.Size() / 1024 / 1024
		}
		fmt.Printf("  target  : %s\n", tgt)
		fmt.Printf("  format  : %s\n", label)
		fmt.Printf("  sha256  : %s\n", target.SHA256Short(tgt))
		fmt.Printf("  size    : %d MB\n", sizeMB)
	} else {
		fmt.Printf("  target  : %sNOT FOUND%s\n", console.R, console.X)
	}

	bdir := target.BackupDir()
	baks := countBackups(bdir)
	fmt.Printf("  backups : %d  (%s)\n", baks, bdir)
	return nil
}
