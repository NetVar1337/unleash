package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/patches"
	"github.com/VoidChecksum/void-patcher-cc/internal/scanner"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
	"github.com/VoidChecksum/void-patcher-cc/internal/updater"
)

// NewDoctorCmd creates the "doctor" cobra command.
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Full health report: target, patches, sig drift, upstream",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runDoctor()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

func runDoctor() int {
	tgt, kind := target.FindTarget()
	patchList := loadAllPatches()
	pd := patchDir()
	nRetired := countRetired(pd)

	fmt.Printf("%sunleash doctor%s\n", console.B, console.X)
	fmt.Printf("  version    : 1.0.0\n")
	fmt.Printf("  patches    : %d\n", len(patchList))

	if tgt == "" {
		fmt.Printf("  target     : %sNOT FOUND%s\n", console.R, console.X)
		return 2
	}

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

	fmt.Printf("  target     : %s\n", tgt)
	fmt.Printf("  format     : %s\n", label)
	fmt.Printf("  sha256     : %s\n", target.SHA256Short(tgt))
	fmt.Printf("  size       : %d MB\n", sizeMB)

	// Sig drift
	rows := scanner.LoadCachedRows(tgt, pd)
	if rows == nil {
		text, err := scanner.LoadTextFromTarget(tgt, kind)
		if err == nil {
			patchData, _ := patches.LoadPatchesForScan(pd, true)
			if patchData == nil {
				patchData, _ = patches.LoadPatchesForScanFromEmbed(true)
			}
			rows = scanner.NewSigScanner(text).ScanPatches(patchData)
			scanner.SaveCachedRows(tgt, pd, rows)
		}
	}

	if rows != nil {
		var drift, nometa []string
		for _, r := range rows {
			if r.Status == "drift" {
				drift = append(drift, r.ID)
			}
			if r.Status == "unclassified" {
				nometa = append(nometa, r.ID)
			}
		}
		if len(drift) > 0 {
			ids := strings.Join(drift, ", ")
			if len(drift) > 3 {
				ids = strings.Join(drift[:3], ", ") + "..."
			}
			fmt.Printf("  %ssig drift  : %d patches %s %s%s\n", console.R, len(drift), console.DOT, ids, console.X)
		} else {
			fmt.Printf("  %ssig drift  : 0 (all anchors locatable)%s\n", console.G, console.X)
		}
		if len(nometa) > 0 {
			fmt.Printf("  %ssig nometa : %d patches (pre-v2.1.114, no anchor_strings yet)%s\n",
				console.Y, len(nometa), console.X)
		}
	} else {
		fmt.Printf("  %ssig scan   : failed%s\n", console.R, console.X)
	}
	fmt.Printf("  retired    : %d patches\n", nRetired)

	// Applied markers
	rcV := RunVerify()
	applied := "all"
	if rcV != 0 {
		applied = "partial/none"
	}
	fmt.Printf("  applied    : %s\n", applied)

	// Backups
	bdir := target.BackupDir()
	baks := countBackups(bdir)
	fmt.Printf("  backups    : %d in %s\n", baks, bdir)

	// Upstream
	info2 := updater.UpstreamStatus(pd)
	if info2.Drift {
		fmt.Printf("  %supstream   : behind — run 'unleash self-update'%s\n", console.Y, console.X)
	} else if info2.RemoteCommit != "" {
		fmt.Printf("  %supstream   : current%s\n", console.G, console.X)
	} else {
		fmt.Printf("  upstream   : unreachable\n")
	}

	return 0
}
