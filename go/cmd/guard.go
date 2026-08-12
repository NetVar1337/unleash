package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewGuardCmd creates the "guard" cobra command.
func NewGuardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "guard",
		Short: "Fast SHA guard: auto-patch if CC binary changed (<100ms when current)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runGuard()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

func runGuard() int {
	all := target.FindAllTargets()
	if len(all) == 0 {
		return 0 // no CC found, nothing to guard
	}

	recorded := readSHAManifest()
	legacySHA := recorded[""]

	changed := false
	for _, f := range all {
		curSHA := target.SHA256Short(f.Path)
		if prev, ok := recorded[f.Path]; ok {
			if prev != curSHA {
				changed = true
				fmt.Printf("%sunleash guard — target changed:%s %s (%s -> %s)\n",
					console.B, console.X, f.Path, prev, curSHA)
			}
			continue
		}
		// Target not in manifest: new install, or legacy single-sha stamp.
		if legacySHA != "" && legacySHA == curSHA {
			continue
		}
		changed = true
		fmt.Printf("%sunleash guard — untracked/new target:%s %s (%s)\n",
			console.B, console.X, f.Path, curSHA)
	}

	if !changed {
		return 0 // all binaries unchanged
	}

	fmt.Printf("%sunleash guard — running autopilot...%s\n", console.B, console.X)
	for _, f := range all {
		target.Backup(f.Path, f.Kind)
	}
	rc := runAutopilot()

	// Stamp new SHA manifest regardless of outcome
	writeSHAManifest(target.FindAllTargets())

	return rc
}
