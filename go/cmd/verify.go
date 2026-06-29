package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/binary"
	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
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

	for _, p := range loadAllPatches() {
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

	// Systemic failure: more patches unapplied than applied
	systemic := optionalMissing > applied

	if requiredMissing > 0 {
		fmt.Printf("\n%s%d required patch(es) not applied%s\n", console.R, requiredMissing, console.X)
		return 1
	}

	if systemic {
		fmt.Printf("%s%s %d patch(es) not applied, %d applied%s  — run %sunleash patch%s to apply\n",
			console.R, console.CROSS, optionalMissing, applied, console.X, console.B, console.X)
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
