package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewWatchCmd creates the "watch" cobra command.
func NewWatchCmd() *cobra.Command {
	var interval int
	c := &cobra.Command{
		Use:   "watch",
		Short: "Daemon: poll target, autoheal on change",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(interval)
		},
	}
	c.Flags().IntVarP(&interval, "interval", "i", 10, "Poll interval seconds (default 10)")
	return c
}

func runWatch(interval int) error {
	tgt, kind := target.FindTarget()
	if tgt == "" {
		fmt.Printf("%sclaude-code not found%s\n", console.R, console.X)
		os.Exit(2)
		return nil
	}

	dim := "\033[90m"
	fmt.Printf("%sunleash watch — polling every %ds%s\n", console.B, interval, console.X)
	fmt.Printf("  target : %s\n", tgt)

	lastSHA := target.SHA256Short(tgt)
	info, _ := os.Stat(tgt)
	var lastMtime time.Time
	if info != nil {
		lastMtime = info.ModTime()
	}
	ccVer := detectCCVersion(tgt)
	fmt.Printf("  version: %s\n", ccVer)
	fmt.Printf("  sha    : %s\n", lastSHA)
	fmt.Printf("  %swatching for changes... (Ctrl-C to stop)%s\n", dim, console.X)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Printf("\n%swatch stopped%s\n", console.B, console.X)
			return nil
		case <-ticker.C:
			tgt, kind = target.FindTarget()
			if tgt == "" {
				continue
			}
			fi, err := os.Stat(tgt)
			if err != nil {
				continue
			}
			if fi.ModTime().Equal(lastMtime) {
				continue
			}
			curSHA := target.SHA256Short(tgt)
			if curSHA == lastSHA {
				lastMtime = fi.ModTime()
				continue
			}

			ts := time.Now().Format("15:04:05")
			newVer := detectCCVersion(tgt)
			fmt.Printf("\n%s[%s] CC updated: %s (%s) %s %s (%s)%s\n",
				console.Y, ts, ccVer, lastSHA, console.ARROW, newVer, curSHA, console.X)
			target.Backup(tgt, kind)

			rc := runAutopilot()
			fmt.Printf("  autopilot rc=%d\n", rc)

			// Update stamp
			stampPath := filepath.Join(unleashDir(), "last_patched_sha")
			os.MkdirAll(filepath.Dir(stampPath), 0o755)
			os.WriteFile(stampPath, []byte(target.SHA256Short(tgt)), 0o644)

			lastSHA = target.SHA256Short(tgt)
			fi2, _ := os.Stat(tgt)
			if fi2 != nil {
				lastMtime = fi2.ModTime()
			}
			ccVer = newVer
			fmt.Printf("  %swatching for changes...%s\n", dim, console.X)
		}
	}
}
