package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewSetupCmd creates the "setup" cobra command — one-shot full setup.
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "One-shot full setup: patch + rules + plugins (ponytail, caveman, karpathy, omc)",
		RunE: func(c *cobra.Command, args []string) error {
			return runSetup()
		},
	}
	return cmd
}

func runSetup() error {
	fmt.Printf("%sunleash setup — full one-shot setup%s\n\n", console.B, console.X)

	// ── Step 1: Patch binary ──────────────────────────────────────────────
	fmt.Printf("  %s[1/4] patching Claude Code binary...%s\n", console.B, console.X)
	tgt, kind := target.FindTarget()
	if tgt == "" {
		fmt.Printf("  %sClaude Code not found — skipping binary patch%s\n", console.Y, console.X)
	} else {
		fmt.Printf("  target: %s (%s)\n", tgt, kind)
		err := RunPatchQuiet()
		if err != nil {
			fmt.Printf("  %spatch failed — continuing with rules%s\n", console.R, console.X)
		}
	}

	// ── Step 2: Install rules ─────────────────────────────────────────────
	fmt.Printf("\n  %s[2/4] installing authorization rules...%s\n", console.B, console.X)
	runInstallRules(true) // --no-hook

	// ── Step 3: Install plugins ───────────────────────────────────────────
	fmt.Printf("\n  %s[3/4] installing plugins...%s\n", console.B, console.X)
	installPlugins()

	// ── Step 4: Install guard ─────────────────────────────────────────────
	fmt.Printf("\n  %s[4/4] installing auto-patch guard...%s\n", console.B, console.X)
	installGuardQuiet()

	// ── Summary ───────────────────────────────────────────────────────────
	fmt.Printf("\n%s%s unleash setup complete%s\n", console.G, console.CHECK, console.X)
	fmt.Printf("  %s patches applied to Claude Code binary\n", console.CHECK)
	fmt.Printf("  %s authorization rules deployed\n", console.CHECK)
	fmt.Printf("  %s plugins installed (ponytail, caveman, karpathy, omc)\n", console.CHECK)
	fmt.Printf("  %s auto-patch guard installed\n", console.CHECK)
	fmt.Printf("\n  restart Claude Code to activate.\n")
	return nil
}

// installPlugins installs the 4 recommended plugins via Claude Code marketplace.
func installPlugins() {
	plugins := []struct {
		name string
		repo string
		pkg  string
	}{
		{"ponytail", "DietrichGebert/ponytail", "ponytail@ponytail"},
		{"caveman", "JuliusBrussee/caveman", "caveman@caveman"},
		{"karpathy-skills", "multica-ai/andrej-karpathy-skills", "andrej-karpathy-skills@karpathy-skills"},
		{"oh-my-claudecode", "Yeachan-Heo/oh-my-claudecode", "oh-my-claudecode"},
	}

	claudeBin := "claude"
	if runtime.GOOS == "windows" {
		claudeBin = "claude.exe"
	}

	// Check if claude is available
	claudePath, err := exec.LookPath(claudeBin)
	if err != nil {
		fmt.Printf("  %sClaude Code CLI not found in PATH — install plugins manually:%s\n", console.Y, console.X)
		for _, p := range plugins {
			fmt.Printf("    /plugin marketplace add %s\n", p.repo)
			fmt.Printf("    /plugin install %s\n", p.pkg)
		}
		return
	}

	// Ensure plugin config dir exists
	home, _ := os.UserHomeDir()
	pluginDir := filepath.Join(home, ".claude", "plugins")
	os.MkdirAll(pluginDir, 0o755)

	for _, p := range plugins {
		fmt.Printf("  %s %s...", console.DOT, p.name)

		// Add marketplace
		cmd1 := exec.Command(claudePath, "--no-interactive", "plugin", "marketplace", "add", p.repo)
		cmd1.Stdout = nil
		cmd1.Stderr = nil
		cmd1.Run() // ignore errors — marketplace may already be added

		// Install plugin
		cmd2 := exec.Command(claudePath, "--no-interactive", "plugin", "install", p.pkg)
		cmd2.Stdout = nil
		cmd2.Stderr = nil
		err := cmd2.Run()
		if err != nil {
			fmt.Printf(" %sskipped (install manually: /plugin install %s)%s\n", console.Y, p.pkg, console.X)
		} else {
			fmt.Printf(" %sdone%s\n", console.G, console.X)
		}
	}
}

// installGuardQuiet installs the auto-patch guard without verbose output.
func installGuardQuiet() {
	vpccBin := "unleash"
	if runtime.GOOS == "windows" {
		vpccBin = "unleash.exe"
	}
	if found, err := exec.LookPath(vpccBin); err == nil {
		vpccBin = found
	}

	switch runtime.GOOS {
	case "windows":
		exec.Command("schtasks", "/Delete", "/TN", "unleash-guard", "/F").Run()
		cmd := exec.Command("schtasks", "/Create", "/TN", "unleash-guard",
			"/TR", fmt.Sprintf(`"%s" guard`, vpccBin),
			"/SC", "HOURLY", "/MO", "6", "/RL", "LIMITED", "/F")
		cmd.Run()
		fmt.Printf("  %s Windows Task Scheduler: unleash-guard (every 6h)%s\n", console.CHECK, console.X)
	case "darwin":
		plistDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
		os.MkdirAll(plistDir, 0o755)
		plist := filepath.Join(plistDir, "dev.unleash.guard.plist")
		home, _ := os.UserHomeDir()
		content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>dev.unleash.guard</string>
    <key>ProgramArguments</key><array><string>%s</string><string>guard</string></array>
    <key>StartInterval</key><integer>21600</integer>
    <key>RunAtLoad</key><true/>
    <key>StandardOutPath</key><string>%s/.unleash/guard.log</string>
    <key>StandardErrorPath</key><string>%s/.unleash/guard.log</string>
</dict>
</plist>`, vpccBin, home, home)
		os.WriteFile(plist, []byte(content), 0o644)
		exec.Command("launchctl", "unload", plist).Run()
		exec.Command("launchctl", "load", plist).Run()
		fmt.Printf("  %s macOS launchd: dev.unleash.guard (every 6h)%s\n", console.CHECK, console.X)
	default:
		home, _ := os.UserHomeDir()
		unitDir := filepath.Join(home, ".config", "systemd", "user")
		os.MkdirAll(unitDir, 0o755)
		svc := fmt.Sprintf(`[Unit]
Description=unleash guard — auto-patch Claude Code on update
[Service]
Type=oneshot
ExecStart=%s guard
[Install]
WantedBy=default.target
`, vpccBin)
		tmr := `[Unit]
Description=Run unleash guard periodically
[Timer]
OnBootSec=2min
OnUnitActiveSec=6h
Persistent=true
[Install]
WantedBy=timers.target
`
		os.WriteFile(filepath.Join(unitDir, "unleash-guard.service"), []byte(svc), 0o644)
		os.WriteFile(filepath.Join(unitDir, "unleash-guard.timer"), []byte(tmr), 0o644)
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		exec.Command("systemctl", "--user", "enable", "--now", "unleash-guard.timer").Run()
		fmt.Printf("  %s systemd: unleash-guard.timer (every 6h)%s\n", console.CHECK, console.X)
	}
}
