package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/patches"
	"github.com/VoidChecksum/unleash/internal/scanner"
	"github.com/VoidChecksum/unleash/internal/target"
	"github.com/VoidChecksum/unleash/internal/updater"
)

// NewListCmd creates the "list" cobra command.
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List patches in catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, p := range loadAllPatches() {
				fmt.Printf("  %s  %s\n", padRight(p.ID, 40), p.Description)
			}
			return nil
		},
	}
}

// NewBenchCmd creates the "bench" cobra command.
func NewBenchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bench",
		Short: "Microbenchmark: sha256, text load, scan cold/cached",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBench()
		},
	}
}

// NewInstallGuardCmd creates the "install-guard" cobra command.
func NewInstallGuardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-guard",
		Short: "Install platform-specific auto-patch scheduler (Task Scheduler / launchd / systemd)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runInstallGuard()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

// NewUninstallGuardCmd creates the "uninstall-guard" cobra command.
func NewUninstallGuardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-guard",
		Short: "Remove auto-patch scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstallGuard()
		},
	}
}

// NewInstallPreloadCmd creates the "install-preload" cobra command.
func NewInstallPreloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-preload",
		Short: "Deploy runtime monkey-patch preload hook (100% survival layer)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallPreload()
		},
	}
}

// NewUninstallPreloadCmd creates the "uninstall-preload" cobra command.
func NewUninstallPreloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-preload",
		Short: "Remove runtime preload hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstallPreload()
		},
	}
}

// NewDashboardCmd creates the "dashboard" cobra command.
func NewDashboardCmd() *cobra.Command {
	var interval int
	c := &cobra.Command{
		Use:   "dashboard",
		Short: "Real-time status display (version, SHA, patches, guard, drift)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashboard(interval)
		},
	}
	c.Flags().IntVarP(&interval, "interval", "i", 5, "Refresh interval seconds (default 5)")
	return c
}

func runBench() error {
	tgt, kind := target.FindTarget()
	if tgt == "" {
		fmt.Printf("%sclaude-code not found%s\n", console.R, console.X)
		os.Exit(2)
		return nil
	}

	info, _ := os.Stat(tgt)
	sizeMB := int64(0)
	if info != nil {
		sizeMB = info.Size() / 1024 / 1024
	}
	fmt.Printf("%sunleash bench — %s (%d MB)%s\n", console.B, filepath.Base(tgt), sizeMB, console.X)

	pd := patchDir()

	// SHA256 benchmark
	t0 := time.Now()
	target.SHA256Short(tgt)
	fmt.Printf("  sha256             : %6.1f ms\n", float64(time.Since(t0).Microseconds())/1000)

	// Text load benchmark
	t0 = time.Now()
	text, err := scanner.LoadTextFromTarget(tgt, kind)
	tLoad := float64(time.Since(t0).Microseconds()) / 1000
	if err != nil {
		fmt.Printf("  text load+decode   : failed (%v)\n", err)
		return nil
	}
	fmt.Printf("  text load+decode   : %6.1f ms  (%d MB)\n", tLoad, len(text)/1024/1024)

	// Scan benchmark (cold)
	patchData, _ := patches.LoadPatchesForScan(pd, true)
	if patchData == nil {
		patchData, _ = patches.LoadPatchesForScanFromEmbed(true)
	}
	t0 = time.Now()
	rows := scanner.NewSigScanner(text).ScanPatches(patchData)
	tScan := float64(time.Since(t0).Microseconds()) / 1000
	fmt.Printf("  scan_patches       : %6.1f ms  (%d patches)\n", tScan, len(rows))

	// Save cache and benchmark cache hit
	scanner.SaveCachedRows(tgt, pd, rows)
	t0 = time.Now()
	scanner.LoadCachedRows(tgt, pd)
	tCache := float64(time.Since(t0).Microseconds()) / 1000
	fmt.Printf("  cache hit          : %6.1f ms  (key: sha256 + patch mtime)\n", tCache)

	fmt.Printf("\n  cold total         : %6.1f ms\n", tLoad+tScan)
	fmt.Printf("  warm total         : %6.1f ms\n", tCache)
	divisor := tCache
	if divisor < 0.01 {
		divisor = 0.01
	}
	fmt.Printf("  speedup            : %6.1fx\n", (tLoad+tScan)/divisor)
	return nil
}

func runInstallGuard() int {
	unleashBin := findUnleashBin()

	switch runtime.GOOS {
	case "windows":
		taskName := "unleash-autoheal"
		exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()
		createCmd := exec.Command("schtasks", "/Create", "/TN", taskName,
			"/TR", fmt.Sprintf(`"%s" guard`, unleashBin),
			"/SC", "HOURLY", "/MO", "6",
			"/RL", "LIMITED", "/F")
		out, err := createCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("%sTask Scheduler failed: %s%s\n", console.R, strings.TrimSpace(string(out)), console.X)
			return 1
		}
		exec.Command("schtasks", "/Change", "/TN", taskName, "/ENABLE").Run()
		fmt.Printf("%s%s Windows Task Scheduler: %s (every 6h + logon)%s\n",
			console.G, console.CHECK, taskName, console.X)

	case "darwin":
		plistDir := filepath.Join(homeDir(), "Library", "LaunchAgents")
		os.MkdirAll(plistDir, 0o755)
		plistPath := filepath.Join(plistDir, "cc.voidchecksum.unleash-guard.plist")
		home := homeDir()
		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>cc.voidchecksum.unleash-guard</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>guard</string>
    </array>
    <key>StartInterval</key>
    <integer>21600</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/.unleash/guard.log</string>
    <key>StandardErrorPath</key>
    <string>%s/.unleash/guard.log</string>
    <key>Nice</key>
    <integer>10</integer>
</dict>
</plist>
`, unleashBin, home, home)
		os.WriteFile(plistPath, []byte(plistContent), 0o644)
		exec.Command("launchctl", "unload", plistPath).Run()
		exec.Command("launchctl", "load", plistPath).Run()
		fmt.Printf("%s%s macOS launchd: %s (every 6h + login)%s\n",
			console.G, console.CHECK, filepath.Base(plistPath), console.X)

	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(homeDir(), ".config")
		}
		unitDir := filepath.Join(xdgConfig, "systemd", "user")
		os.MkdirAll(unitDir, 0o755)

		svc := filepath.Join(unitDir, "unleash-guard.service")
		os.WriteFile(svc, []byte(fmt.Sprintf(`[Unit]
Description=unleash guard — auto-patch Claude Code on update
Documentation=https://github.com/VoidChecksum/unleash
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s guard
Nice=10

[Install]
WantedBy=default.target
`, unleashBin)), 0o644)

		tmr := filepath.Join(unitDir, "unleash-guard.timer")
		os.WriteFile(tmr, []byte(`[Unit]
Description=Run unleash guard periodically
Documentation=https://github.com/VoidChecksum/unleash

[Timer]
OnBootSec=2min
OnUnitActiveSec=6h
RandomizedDelaySec=15min
Persistent=true
Unit=unleash-guard.service

[Install]
WantedBy=timers.target
`), 0o644)

		exec.Command("systemctl", "--user", "daemon-reload").Run()
		exec.Command("systemctl", "--user", "enable", "--now", "unleash-guard.timer").Run()
		fmt.Printf("%s%s systemd --user: unleash-guard.timer (every 6h + boot)%s\n",
			console.G, console.CHECK, console.X)
	}

	fmt.Printf("  unleash guard runs automatically — Claude Code updates are patched within minutes\n")
	return 0
}

func runUninstallGuard() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("schtasks", "/Delete", "/TN", "unleash-autoheal", "/F")
		if err := cmd.Run(); err == nil {
			fmt.Printf("%s%s removed Windows scheduled task%s\n", console.G, console.CHECK, console.X)
		} else {
			fmt.Printf("%sno task found%s\n", console.Y, console.X)
		}

	case "darwin":
		plist := filepath.Join(homeDir(), "Library", "LaunchAgents", "cc.voidchecksum.unleash-guard.plist")
		if _, err := os.Stat(plist); err == nil {
			exec.Command("launchctl", "unload", plist).Run()
			os.Remove(plist)
			fmt.Printf("%s%s removed macOS launchd agent%s\n", console.G, console.CHECK, console.X)
		} else {
			fmt.Printf("%sno launchd agent found%s\n", console.Y, console.X)
		}

	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(homeDir(), ".config")
		}
		unitDir := filepath.Join(xdgConfig, "systemd", "user")
		exec.Command("systemctl", "--user", "disable", "--now", "unleash-guard.timer").Run()
		for _, name := range []string{"unleash-guard.service", "unleash-guard.timer"} {
			os.Remove(filepath.Join(unitDir, name))
		}
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		fmt.Printf("%s%s removed systemd timer%s\n", console.G, console.CHECK, console.X)
	}

	// Remove stamp file
	os.Remove(filepath.Join(unleashDir(), "last_patched_sha"))
	return nil
}

func runInstallPreload() error {
	src, err := getContribFile("preload/claude-preload.js")
	if err != nil {
		fmt.Printf("%ssource missing: contrib/preload/claude-preload.js%s\n", console.R, console.X)
		os.Exit(2)
		return nil
	}

	dstDir := preloadDirPath()
	os.MkdirAll(dstDir, 0o755)
	dst := filepath.Join(dstDir, "claude-preload.js")
	os.WriteFile(dst, src, 0o644)

	fmt.Printf("%s%s installed preload%s  claude-preload.js %s %s\n",
		console.G, console.CHECK, console.X, console.ARROW, dst)
	fmt.Printf("  wrapper will auto-load via BUN_OPTIONS=--preload on next run\n")
	return nil
}

func runUninstallPreload() error {
	dst := filepath.Join(preloadDirPath(), "claude-preload.js")
	if _, err := os.Stat(dst); err == nil {
		os.Remove(dst)
		fmt.Printf("%s%s removed%s %s\n", console.G, console.CHECK, console.X, dst)
	} else {
		fmt.Printf("%snot installed%s\n", console.Y, console.X)
	}
	return nil
}

func runDashboard(interval int) error {
	clearScreen := "\033[2J\033[H"
	dim := "\033[90m"

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	render := func() {
		tgt, kind := target.FindTarget()
		ts := time.Now().Format("2006-01-02 15:04:05")
		pd := patchDir()

		var lines []string
		lines = append(lines, fmt.Sprintf("%s%sunleash dashboard%s  %s%s%s",
			clearScreen, console.B, console.X, dim, ts, console.X))
		lines = append(lines, fmt.Sprintf("%s%s%s", dim, strings.Repeat("─", 60), console.X))

		if tgt == "" {
			lines = append(lines, fmt.Sprintf("  %sclaude-code not found%s", console.R, console.X))
		} else {
			curSHA := target.SHA256Short(tgt)
			ccVer := detectCCVersion(tgt)
			fi, _ := os.Stat(tgt)
			sizeMB := int64(0)
			if fi != nil {
				sizeMB = fi.Size() / 1024 / 1024
			}

			formatStr := "Bun SEA"
			if kind != "bun_sea" {
				formatStr = "cli.js"
			}

			lines = append(lines, fmt.Sprintf("  %starget%s   : %s", console.B, console.X, tgt))
			lines = append(lines, fmt.Sprintf("  %sversion%s  : %s", console.B, console.X, ccVer))
			lines = append(lines, fmt.Sprintf("  %ssha256%s   : %s", console.B, console.X, curSHA))
			lines = append(lines, fmt.Sprintf("  %ssize%s     : %d MB", console.B, console.X, sizeMB))
			lines = append(lines, fmt.Sprintf("  %sformat%s   : %s", console.B, console.X, formatStr))

			// Guard stamp
			stampPath := filepath.Join(unleashDir(), "last_patched_sha")
			if data, err := os.ReadFile(stampPath); err == nil {
				stampSHA := strings.TrimSpace(string(data))
				stampInfo, _ := os.Stat(stampPath)
				ageStr := "?"
				if stampInfo != nil {
					ageStr = humanAge(time.Since(stampInfo.ModTime()).Seconds())
				}
				if stampSHA == curSHA {
					lines = append(lines, fmt.Sprintf("  %sguard%s    : %s%s patched (%s ago)%s",
						console.B, console.X, console.G, console.CHECK, ageStr, console.X))
				} else {
					lines = append(lines, fmt.Sprintf("  %sguard%s    : %s%s stale — binary changed since last patch%s",
						console.B, console.X, console.R, console.CROSS, console.X))
				}
			} else {
				lines = append(lines, fmt.Sprintf("  %sguard%s    : %s%s no stamp — run 'unleash patch'%s",
					console.B, console.X, console.Y, console.WARN, console.X))
			}

			// Scan cache
			cached := scanner.LoadCachedRows(tgt, pd)
			if cached != nil {
				nOK, nApplied, nDrift, nRetired := 0, 0, 0, 0
				for _, r := range cached {
					switch r.Status {
					case "ok":
						nOK++
					case "applied":
						nApplied++
					case "drift":
						nDrift++
					case "retired":
						nRetired++
					}
				}
				nTotal := nOK + nApplied + nDrift
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("  %spatches%s  : %d scanned", console.B, console.X, nTotal))

				driftStr := fmt.Sprintf("%s0 drift%s", dim, console.X)
				if nDrift > 0 {
					driftStr = fmt.Sprintf("%s%s %d drift%s", console.R, console.CROSS, nDrift, console.X)
				}
				lines = append(lines, fmt.Sprintf("    %s%s %d ok%s  %s%s %d applied%s  %s  %s%d retired%s",
					console.G, console.CHECK, nOK, console.X,
					console.G, console.CHECK, nApplied, console.X,
					driftStr,
					dim, nRetired, console.X))

				if nDrift > 0 {
					var driftIDs []string
					for _, r := range cached {
						if r.Status == "drift" {
							driftIDs = append(driftIDs, r.ID)
						}
					}
					limit := minInt(5, len(driftIDs))
					for _, did := range driftIDs[:limit] {
						lines = append(lines, fmt.Sprintf("    %s%s %s%s", console.R, console.ARROW, did, console.X))
					}
					if len(driftIDs) > 5 {
						lines = append(lines, fmt.Sprintf("    %s... and %d more%s", dim, len(driftIDs)-5, console.X))
					}
				}
			} else {
				lines = append(lines, fmt.Sprintf("\n  %spatches%s  : %s(run 'unleash scan' to populate cache)%s",
					console.B, console.X, dim, console.X))
			}

			// Backups
			bdir := target.BackupDir()
			baks := countBackups(bdir)
			lines = append(lines, fmt.Sprintf("\n  %sbackups%s  : %d in %s", console.B, console.X, baks, bdir))

			// Guard scheduler
			guardStatus := detectGuardScheduler()
			lines = append(lines, fmt.Sprintf("  %sscheduler%s: %s", console.B, console.X, guardStatus))

			// Upstream
			upInfo := updater.UpstreamStatus(pd)
			if upInfo.Drift {
				lines = append(lines, fmt.Sprintf("  %supstream%s : %sbehind — run 'unleash self-update'%s",
					console.B, console.X, console.Y, console.X))
			} else if upInfo.RemoteCommit != "" {
				lines = append(lines, fmt.Sprintf("  %supstream%s : %scurrent%s",
					console.B, console.X, console.G, console.X))
			} else {
				lines = append(lines, fmt.Sprintf("  %supstream%s : %sunreachable%s",
					console.B, console.X, dim, console.X))
			}
		}

		lines = append(lines, fmt.Sprintf("\n%srefreshing every %ds — Ctrl-C to exit%s", dim, interval, console.X))
		fmt.Println(strings.Join(lines, "\n"))
	}

	// Initial render
	render()

	for {
		select {
		case <-sigCh:
			fmt.Printf("\n%sdashboard stopped%s\n", console.B, console.X)
			return nil
		case <-ticker.C:
			render()
		}
	}
}

func humanAge(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", int(seconds))
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", int(seconds/60))
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh", int(seconds/3600))
	}
	return fmt.Sprintf("%dd", int(seconds/86400))
}

func detectGuardScheduler() string {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("schtasks", "/Query", "/TN", "unleash-autoheal", "/FO", "LIST")
		if err := cmd.Run(); err == nil {
			return fmt.Sprintf("%sWindows Task Scheduler active%s", console.G, console.X)
		}
		return fmt.Sprintf("%snot installed — run 'unleash install-guard'%s", console.Y, console.X)

	case "darwin":
		plist := filepath.Join(homeDir(), "Library", "LaunchAgents", "cc.voidchecksum.unleash-guard.plist")
		if _, err := os.Stat(plist); err == nil {
			return fmt.Sprintf("%smacOS launchd active%s", console.G, console.X)
		}
		return fmt.Sprintf("%snot installed — run 'unleash install-guard'%s", console.Y, console.X)

	default:
		cmd := exec.Command("systemctl", "--user", "is-active", "unleash-guard.timer")
		out, _ := cmd.Output()
		if strings.TrimSpace(string(out)) == "active" {
			return fmt.Sprintf("%ssystemd timer active%s", console.G, console.X)
		}
		return fmt.Sprintf("%snot installed — run 'unleash install-guard'%s", console.Y, console.X)
	}
}

func findUnleashBin() string {
	exe, err := os.Executable()
	if err == nil {
		return exe
	}
	p, err := exec.LookPath("unleash")
	if err == nil {
		return p
	}
	return "unleash"
}
