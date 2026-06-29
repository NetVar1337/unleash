package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
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
	tgt, kind := target.FindTarget()
	if tgt == "" {
		return 0 // no CC found, nothing to guard
	}

	stampPath := filepath.Join(unleashDir(), "last_patched_sha")
	curSHA := target.SHA256Short(tgt)

	data, err := os.ReadFile(stampPath)
	if err == nil {
		if string(data) == curSHA {
			return 0 // binary unchanged
		}
	}

	// Binary changed — run full autopilot
	fmt.Printf("%sunleash guard — CC binary changed (%s), running autopilot...%s\n",
		console.B, curSHA, console.X)
	target.Backup(tgt, kind)
	rc := runAutopilot()

	// Stamp new SHA regardless of outcome
	os.MkdirAll(filepath.Dir(stampPath), 0o755)
	os.WriteFile(stampPath, []byte(target.SHA256Short(tgt)), 0o644)

	return rc
}
