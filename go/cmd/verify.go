package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/binary"
	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/patches"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewVerifyCmd creates the "verify" cobra command.
func NewVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Check patches are applied",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := RunVerify()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

// RunVerify checks applied markers. Returns exit code.
func RunVerify() int {
	tgt, kind := target.FindTarget()
	return runVerifyPatches(tgt, kind, loadAllPatches())
}

func runVerifyPatches(tgt, kind string, patchList []patches.Patch) int {
	if tgt == "" {
		fmt.Printf("%sclaude-code not found%s\n", console.R, console.X)
		return 2
	}

	var text string
	if kind == "bun_sea" {
		data, err := os.ReadFile(tgt)
		if err != nil {
			fmt.Printf("%sread failed: %v%s\n", console.R, err, console.X)
			return 2
		}
		bunOff, bunSize, err := binary.FindBunSection(data)
		if err != nil {
			fmt.Printf("%sELF parse failed: %v%s\n", console.R, err, console.X)
			return 2
		}
		text = string(data[bunOff : bunOff+bunSize])
	} else {
		data, err := os.ReadFile(tgt)
		if err != nil {
			fmt.Printf("%sread failed: %v%s\n", console.R, err, console.X)
			return 2
		}
		text = string(data)
	}

	requiredMissing := 0
	optionalMissing := 0
	applied := 0

	for _, p := range patchList {
		if p.Retired {
			continue
		}
		if p.Type != "js_replace" {
			continue
		}
		var hasCheckable bool
		patchApplied := false
		isRequired := false

		for _, s := range p.Patches {
			if s.AppliedMarker == "" {
				continue
			}
			hasCheckable = true
			if strings.Contains(text, s.AppliedMarker) {
				patchApplied = true
			}
			if s.IsRequired(false) {
				isRequired = true
			}
		}

		if !hasCheckable {
			continue
		}

		if patchApplied {
			applied++
			continue
		}

		if isRequired {
			fmt.Printf("%s%s%s %s\n", console.R, console.CROSS, console.X, p.ID)
			requiredMissing++
		} else {
			optionalMissing++
		}
	}

	if requiredMissing > 0 {
		fmt.Printf("%s%s %d required patch(es) missing%s\n", console.R, console.CROSS, requiredMissing, console.X)
		return 1
	}
	if optionalMissing > 0 {
		fmt.Printf("%s%s %d patch(es) verified%s  %s(%d optional not applied — run 'unleash patch' if unexpected)%s\n",
			console.G, console.CHECK, applied, console.X,
			console.Y, optionalMissing, console.X)
	} else {
		fmt.Printf("%s%s all patches verified%s\n", console.G, console.CHECK, console.X)
	}
	return 0
}
