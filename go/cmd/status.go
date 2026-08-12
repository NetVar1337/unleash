package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/target"
	"github.com/VoidChecksum/unleash/internal/updater"
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
	all := target.FindAllTargets()
	patchList := loadAllPatches()

	fmt.Printf("%sunleash status%s\n", console.B, console.X)
	fmt.Printf("  patches : %d\n", len(patchList))

	if len(all) > 0 {
		fmt.Printf("  targets : %d installation(s) found\n", len(all))
		for i, f := range all {
			var label string
			if f.Kind == "bun_sea" {
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
			info, _ := os.Stat(f.Path)
			sizeMB := int64(0)
			if info != nil {
				sizeMB = info.Size() / 1024 / 1024
			}
			marker := " "
			if i == 0 {
				marker = "*"
			}
			fmt.Printf("  %s target : %s\n", marker, f.Path)
			fmt.Printf("    format : %s | sha256 %s | %d MB\n", label, target.SHA256Short(f.Path), sizeMB)
		}
		fmt.Printf("  (* = primary)\n")
	} else {
		fmt.Printf("  target  : %sNOT FOUND%s\n", console.R, console.X)
	}

	bdir := target.BackupDir()
	baks := countBackups(bdir)
	fmt.Printf("  backups : %d  (%s)\n", baks, bdir)

	info := updater.UpstreamStatus(patchDir())
	if warning := updateWarning(info, "unleash update"); warning != "" {
		fmt.Printf("  update  : %s\n", warning)
	} else if info.RemoteCommit != "" {
		fmt.Printf("  update  : %scurrent%s\n", console.G, console.X)
	} else {
		fmt.Printf("  update  : unreachable\n")
	}
	return nil
}
