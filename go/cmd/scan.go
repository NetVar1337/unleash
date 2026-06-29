package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/patches"
	"github.com/VoidChecksum/unleash/internal/scanner"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewScanCmd creates the "scan" cobra command.
func NewScanCmd() *cobra.Command {
	var verbose bool
	var exportPatch string
	var autoHeal bool

	c := &cobra.Command{
		Use:   "scan",
		Short: "Signature-based offset discovery (survives regex drift)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runScan(verbose, exportPatch, autoHeal)
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	c.Flags().StringVar(&exportPatch, "export-patch", "", "Regenerate regex for patch ID from anchor strings")
	c.Flags().BoolVar(&autoHeal, "auto-heal", false, "Rewrite drifted regexes in patches/*.json from anchor windows")
	return c
}

func runScan(verbose bool, exportPatch string, autoHeal bool) int {
	tgt, kind := target.FindTarget()
	if tgt == "" {
		fmt.Printf("%sclaude-code not found%s\n", console.R, console.X)
		return 2
	}

	pd := patchDir()
	needsText := autoHeal || exportPatch != ""
	var rows []patches.ScanRow
	var text string
	var sc *scanner.SigScanner

	if !needsText {
		rows = scanner.LoadCachedRows(tgt, pd)
	}
	if rows == nil {
		var err error
		text, err = scanner.LoadTextFromTarget(tgt, kind)
		if err != nil {
			fmt.Printf("%sextract failed: %v%s\n", console.R, err, console.X)
			return 2
		}
		patchData, _ := patches.LoadPatchesForScan(pd, true)
		if patchData == nil {
			patchData, _ = patches.LoadPatchesForScanFromEmbed(true)
		}
		sc = scanner.NewSigScanner(text)
		rows = sc.ScanPatches(patchData)
		scanner.SaveCachedRows(tgt, pd, rows)
	}

	fmt.Printf("%sunleash scan — %d js_replace patches%s\n", console.B, len(rows), console.X)
	fmt.Printf("  target : %s\n", tgt)
	if text != "" {
		if kind == "js" {
			fmt.Printf("  format : cli.js\n")
		} else {
			fmt.Printf("  format : .bun section (%d bytes)\n", len(text))
		}
	} else {
		if kind == "js" {
			fmt.Printf("  format : cli.js (cached)\n")
		} else {
			fmt.Printf("  format : Bun SEA (cached)\n")
		}
	}
	fmt.Printf("  sha    : %s\n", target.SHA256Short(tgt))
	fmt.Println()
	fmt.Println(scanner.FormatScanReport(rows, verbose))

	if autoHeal {
		res := scanner.AutoHealDrift(text, pd, verbose)
		fmt.Printf("\n%sauto-heal:%s  %s%d healed%s  %d skipped  %s%d failed%s\n",
			console.B, console.X,
			console.G, res.Healed, console.X,
			res.Skipped,
			console.R, res.Failed, console.X)
	}

	if exportPatch != "" {
		var match *patches.ScanRow
		for i := range rows {
			if rows[i].ID == exportPatch {
				match = &rows[i]
				break
			}
		}
		if match == nil {
			fmt.Printf("\n%sno patch id '%s'%s\n", console.R, exportPatch, console.X)
			return 1
		}
		if len(match.Anchors) == 0 {
			fmt.Printf("\n%spatch has no anchor_strings — cannot regenerate%s\n", console.R, console.X)
			return 1
		}
		if sc == nil {
			fmt.Printf("\n%sneed live scanner for export (no cache)%s\n", console.R, console.X)
			return 1
		}
		regex := sc.DeriveRegex(match.Anchors[0], 200, 200, true)
		fmt.Printf("\n%sregenerated regex for %s:%s\n", console.B, exportPatch, console.X)
		fmt.Printf("  %s\n", regex)
	}

	for _, r := range rows {
		if r.Status == "drift" {
			return 1
		}
	}
	return 0
}
