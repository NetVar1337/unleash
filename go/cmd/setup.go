package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewSetupCmd creates the "setup" cobra command — one-shot full setup.
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "One-shot full setup: check deps + patch + rules + plugins (ponytail, caveman, karpathy, omc)",
		RunE: func(c *cobra.Command, args []string) error {
			return runSetup()
		},
	}
	return cmd
}

func runSetup() error {
	fmt.Printf("%sunleash setup — full one-shot setup%s\n\n", console.B, console.X)

	// ── Step 0: Check & install dependencies ─────────────────────────────
	fmt.Printf("  %s[0/5] checking dependencies...%s\n", console.B, console.X)
	ensureDependencies()

	// ── Step 1: Patch binary ──────────────────────────────────────────────
	fmt.Printf("\n  %s[1/5] patching Claude Code binary...%s\n", console.B, console.X)
	patchOK := true
	tgt, kind := target.FindTarget()
	if tgt == "" {
		fmt.Printf("  %sClaude Code not found — skipping binary patch%s\n", console.Y, console.X)
		patchOK = false
	} else {
		fmt.Printf("  target: %s (%s)\n", tgt, kind)
		if err := RunPatchQuiet(); err != nil {
			fmt.Printf("  %spatch failed%s\n", console.R, console.X)
			patchOK = false
		}
	}

	// ── Step 2: Install rules ─────────────────────────────────────────────
	fmt.Printf("\n  %s[2/5] installing authorization rules...%s\n", console.B, console.X)
	rulesOK := runInstallRules(true) == 0 // --no-hook

	// ── Step 3: Install plugins ───────────────────────────────────────────
	fmt.Printf("\n  %s[3/5] installing plugins...%s\n", console.B, console.X)
	pluginsInstalled, pluginsSkipped := installPlugins()

	// ── Step 4: Install guard ─────────────────────────────────────────────
	fmt.Printf("\n  %s[4/5] installing auto-patch guard...%s\n", console.B, console.X)
	guardOK := installGuardQuiet()

	// ── Step 5: Verify ───────────────────────────────────────────────────
	fmt.Printf("\n  %s[5/5] verifying installation...%s\n", console.B, console.X)
	depsOK := verifyDependencies()

	// ── Summary ───────────────────────────────────────────────────────────
	fmt.Printf("\n%sunleash setup summary%s\n", console.B, console.X)
	printSetupStatus("dependencies checked", depsOK)
	printSetupStatus("patches applied to Claude Code binary", patchOK)
	printSetupStatus("authorization rules deployed", rulesOK)
	if pluginsSkipped == 0 {
		printSetupStatus(fmt.Sprintf("plugins installed (%d)", pluginsInstalled), true)
	} else if pluginsInstalled > 0 {
		printSetupStatus(fmt.Sprintf("plugins partially installed (%d installed, %d skipped)", pluginsInstalled, pluginsSkipped), false)
	} else {
		printSetupStatus("plugins skipped (install manually from Claude Code)", false)
	}
	printSetupStatus("auto-patch guard installed", guardOK)

	if !patchOK || !rulesOK || !guardOK || !depsOK {
		return fmt.Errorf("setup incomplete")
	}
	fmt.Printf("\n%s%s unleash setup complete%s\n", console.G, console.CHECK, console.X)
	fmt.Printf("\n  restart Claude Code to activate.\n")
	return nil
}

func printSetupStatus(label string, ok bool) {
	if ok {
		fmt.Printf("  %s %s\n", console.CHECK, label)
		return
	}
	fmt.Printf("  %s %s\n", console.WARN, label)
}

// ── dependency management ───────────────────────────────────────────────────

// depStatus tracks which dependencies were found or installed.
type depStatus struct {
	name      string
	found     bool
	installed bool
	version   string
	path      string
}

// checkBin looks up a binary in PATH and gets its version via --version.
func checkBin(name string, versionArgs ...string) depStatus {
	ds := depStatus{name: name}
	p, err := exec.LookPath(name)
	if err != nil {
		// On Windows, also try without .exe suffix added and with it
		if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
			p, err = exec.LookPath(name + ".exe")
		}
		if err != nil {
			return ds
		}
	}
	ds.found = true
	ds.path = p
	if len(versionArgs) == 0 {
		versionArgs = []string{"--version"}
	}
	out, err := exec.Command(p, versionArgs...).CombinedOutput()
	if err == nil {
		v := strings.TrimSpace(string(out))
		// Take first line only
		if idx := strings.IndexByte(v, '\n'); idx > 0 {
			v = v[:idx]
		}
		ds.version = v
	}
	return ds
}

// ensureDependencies checks for required runtime deps and auto-installs missing ones.
func ensureDependencies() {
	// 1. Node.js
	node := checkBin("node", "-v")
	if node.found {
		fmt.Printf("  %s node: %s%s%s\n", console.CHECK, console.G, node.version, console.X)
	} else {
		fmt.Printf("  %s node: %snot found%s\n", console.CROSS, console.R, console.X)
		fmt.Printf("    %s attempting to install Node.js...\n", console.ARROW)
		if installNode() {
			node = checkBin("node", "-v")
			if node.found {
				fmt.Printf("    %s node installed: %s\n", console.CHECK, node.version)
			}
		} else {
			fmt.Printf("    %s install Node.js manually: https://nodejs.org%s\n", console.Y, console.X)
		}
	}

	// 2. npm
	npm := checkBin("npm", "-v")
	if npm.found {
		fmt.Printf("  %s npm: %s%s%s\n", console.CHECK, console.G, npm.version, console.X)
	} else {
		fmt.Printf("  %s npm: %snot found%s — bundled with Node.js\n", console.CROSS, console.R, console.X)
		if !node.found {
			fmt.Printf("    install Node.js first: https://nodejs.org\n")
		}
	}

	// 3. Claude Code CLI
	claudeBin := "claude"
	if runtime.GOOS == "windows" {
		claudeBin = "claude.exe"
	}
	cc := checkBin(claudeBin, "--version")
	if cc.found {
		fmt.Printf("  %s claude: %s%s%s\n", console.CHECK, console.G, cc.version, console.X)
	} else {
		fmt.Printf("  %s claude: %snot found%s\n", console.CROSS, console.R, console.X)
		if npm.found {
			fmt.Printf("    %s attempting to install Claude Code via npm...\n", console.ARROW)
			if installClaudeCode(npm.path) {
				cc = checkBin(claudeBin, "--version")
				if cc.found {
					fmt.Printf("    %s claude installed: %s\n", console.CHECK, cc.version)
				}
			} else {
				printClaudeInstallHints("    ")
			}
		} else {
			printClaudeInstallHints("    ")
		}
	}

	// 4. git (optional, used by autopilot)
	git := checkBin("git", "--version")
	if git.found {
		fmt.Printf("  %s git: %s%s%s\n", console.CHECK, console.G, git.version, console.X)
	} else {
		fmt.Printf("  %s git: %snot found (optional — needed for autopilot)%s\n", console.WARN, console.Y, console.X)
	}
}

// printClaudeInstallHints prints supported Claude Code install methods
// (native installer first — it is the upstream-recommended path).
func printClaudeInstallHints(indent string) {
	fmt.Printf("%s%s install Claude Code manually:%s\n", indent, console.Y, console.X)
	if runtime.GOOS == "windows" {
		fmt.Printf("%s  native: irm https://claude.ai/install.ps1 | iex\n", indent)
	} else {
		fmt.Printf("%s  native: curl -fsSL https://claude.ai/install.sh | bash\n", indent)
	}
	fmt.Printf("%s  winget: winget install Anthropic.ClaudeCode   (Windows)\n", indent)
	fmt.Printf("%s  npm:    npm install -g @anthropic-ai/claude-code\n", indent)
	fmt.Printf("%s  bun:    bun add -g @anthropic-ai/claude-code\n", indent)
}

// installNode attempts to install Node.js via platform package managers.
func installNode() bool {
	switch runtime.GOOS {
	case "darwin":
		// Try brew
		if _, err := exec.LookPath("brew"); err == nil {
			cmd := exec.Command("brew", "install", "node")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run() == nil
		}
	case "linux":
		// Try apt, then dnf, then pacman
		for _, pm := range []struct {
			bin  string
			args []string
		}{
			{"apt-get", []string{"install", "-y", "nodejs", "npm"}},
			{"dnf", []string{"install", "-y", "nodejs", "npm"}},
			{"pacman", []string{"-S", "--noconfirm", "nodejs", "npm"}},
			{"apk", []string{"add", "nodejs", "npm"}},
		} {
			if _, err := exec.LookPath(pm.bin); err == nil {
				// Use sudo if available and not root
				args := pm.args
				bin := pm.bin
				if os.Getuid() != 0 {
					if sudoPath, err := exec.LookPath("sudo"); err == nil {
						args = append([]string{bin}, args...)
						bin = sudoPath
					}
				}
				cmd := exec.Command(bin, args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if cmd.Run() == nil {
					return true
				}
			}
		}
	case "windows":
		// Try winget, then choco, then scoop
		for _, pm := range []struct {
			bin  string
			args []string
		}{
			{"winget", []string{"install", "--id", "OpenJS.NodeJS.LTS", "--accept-source-agreements", "--accept-package-agreements"}},
			{"choco", []string{"install", "nodejs-lts", "-y"}},
			{"scoop", []string{"install", "nodejs-lts"}},
		} {
			if _, err := exec.LookPath(pm.bin); err == nil {
				cmd := exec.Command(pm.bin, pm.args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if cmd.Run() == nil {
					return true
				}
			}
		}
	}
	return false
}

// installClaudeCode installs Claude Code via npm.
func installClaudeCode(npmPath string) bool {
	cmd := exec.Command(npmPath, "install", "-g", "@anthropic-ai/claude-code")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

// verifyDependencies prints a final dependency summary.
func verifyDependencies() bool {
	claudeBin := "claude"
	if runtime.GOOS == "windows" {
		claudeBin = "claude.exe"
	}

	checks := []struct {
		name string
		args []string
	}{
		{"node", []string{"-v"}},
		{"npm", []string{"-v"}},
		{claudeBin, []string{"--version"}},
		{"git", []string{"--version"}},
	}

	allGood := true
	for _, c := range checks {
		ds := checkBin(c.name, c.args...)
		if ds.found {
			fmt.Printf("  %s %s: %s\n", console.CHECK, c.name, ds.version)
		} else {
			if c.name == "git" {
				fmt.Printf("  %s %s: not found (optional)\n", console.WARN, c.name)
			} else {
				fmt.Printf("  %s %s: %smissing%s\n", console.CROSS, c.name, console.R, console.X)
				allGood = false
			}
		}
	}

	tgt, _ := target.FindTarget()
	if tgt != "" {
		fmt.Printf("  %s Claude Code binary: %s\n", console.CHECK, tgt)
	} else {
		fmt.Printf("  %s Claude Code binary: %snot found%s\n", console.CROSS, console.R, console.X)
		allGood = false
	}
	return allGood
}

// installPlugins installs the 4 recommended plugins via Claude Code marketplace.
func installPlugins() (int, int) {
	plugins := []struct {
		name string
		repo string
		pkg  string
	}{
		{"ponytail", "ponytail", "ponytail@ponytail"},
		{"caveman", "caveman", "caveman@caveman"},
		{"karpathy-skills", "andrej-karpathy-skills", "andrej-karpathy-skills@karpathy-skills"},
		{"oh-my-claudecode", "oh-my-claudecode", "oh-my-claudecode"},
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil && runtime.GOOS == "windows" {
		claudePath, err = exec.LookPath("claude.exe")
	}
	if err != nil {
		fmt.Printf("  %sClaude Code CLI not found in PATH — install plugins manually:%s\n", console.Y, console.X)
		for _, p := range plugins {
			fmt.Printf("    /plugin marketplace add %s\n", p.repo)
			fmt.Printf("    /plugin install %s\n", p.pkg)
		}
		return 0, len(plugins)
	}

	home, _ := os.UserHomeDir()
	pluginDir := filepath.Join(home, ".claude", "plugins")
	os.MkdirAll(pluginDir, 0o755)

	installed := 0
	skipped := 0
	for _, p := range plugins {
		fmt.Printf("  %s %s...", console.DOT, p.name)

		cmd1 := exec.Command(claudePath, "--no-interactive", "plugin", "marketplace", "add", p.repo)
		cmd1.Stdout = nil
		cmd1.Stderr = nil
		_ = cmd1.Run()

		cmd2 := exec.Command(claudePath, "--no-interactive", "plugin", "install", p.pkg)
		cmd2.Stdout = nil
		cmd2.Stderr = nil
		if err := cmd2.Run(); err != nil {
			fmt.Printf(" %sskipped (install manually: /plugin install %s)%s\n", console.Y, p.pkg, console.X)
			skipped++
		} else {
			fmt.Printf(" %sdone%s\n", console.G, console.X)
			installed++
		}
	}
	return installed, skipped
}

// installGuardQuiet installs the auto-patch guard without verbose output.
func installGuardQuiet() bool {
	vpccBin := "unleash"
	if runtime.GOOS == "windows" {
		vpccBin = "unleash.exe"
	}
	if found, err := exec.LookPath(vpccBin); err == nil {
		vpccBin = found
	} else if exe, err := os.Executable(); err == nil {
		vpccBin = exe
	}

	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("schtasks", "/Delete", "/TN", "unleash-guard", "/F").Run()
		cmd := exec.Command("schtasks", "/Create", "/TN", "unleash-guard",
			"/TR", fmt.Sprintf(`"%s" guard`, vpccBin),
			"/SC", "HOURLY", "/MO", "6", "/RL", "LIMITED", "/F")
		if err := cmd.Run(); err != nil {
			fmt.Printf("  %s Windows Task Scheduler failed: %v%s\n", console.WARN, err, console.X)
			return installWindowsStartupGuard(vpccBin)
		}
		fmt.Printf("  %s Windows Task Scheduler: unleash-guard (every 6h)%s\n", console.CHECK, console.X)
		return true
	case "darwin":
		home, _ := os.UserHomeDir()
		plistDir := filepath.Join(home, "Library", "LaunchAgents")
		if err := os.MkdirAll(plistDir, 0o755); err != nil {
			fmt.Printf("  %s macOS launchd dir failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		plist := filepath.Join(plistDir, "dev.unleash.guard.plist")
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
		if err := os.WriteFile(plist, []byte(content), 0o644); err != nil {
			fmt.Printf("  %s macOS launchd plist failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		_ = exec.Command("launchctl", "unload", plist).Run()
		if err := exec.Command("launchctl", "load", plist).Run(); err != nil {
			fmt.Printf("  %s macOS launchd load failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		fmt.Printf("  %s macOS launchd: dev.unleash.guard (every 6h)%s\n", console.CHECK, console.X)
		return true
	default:
		home, _ := os.UserHomeDir()
		unitDir := filepath.Join(home, ".config", "systemd", "user")
		if err := os.MkdirAll(unitDir, 0o755); err != nil {
			fmt.Printf("  %s systemd unit dir failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
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
		if err := os.WriteFile(filepath.Join(unitDir, "unleash-guard.service"), []byte(svc), 0o644); err != nil {
			fmt.Printf("  %s systemd service write failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		if err := os.WriteFile(filepath.Join(unitDir, "unleash-guard.timer"), []byte(tmr), 0o644); err != nil {
			fmt.Printf("  %s systemd timer write failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
			fmt.Printf("  %s systemd daemon-reload failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		if err := exec.Command("systemctl", "--user", "enable", "--now", "unleash-guard.timer").Run(); err != nil {
			fmt.Printf("  %s systemd timer enable failed: %v%s\n", console.WARN, err, console.X)
			return false
		}
		fmt.Printf("  %s systemd: unleash-guard.timer (every 6h)%s\n", console.CHECK, console.X)
		return true
	}
}

func installWindowsStartupGuard(vpccBin string) bool {
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")
	if appData == "" || localAppData == "" {
		fmt.Printf("  %s Windows startup guard skipped: APPDATA/LOCALAPPDATA missing%s\n", console.WARN, console.X)
		return false
	}

	guardDir := filepath.Join(localAppData, "unleash")
	if err := os.MkdirAll(guardDir, 0o755); err != nil {
		fmt.Printf("  %s Windows startup guard dir failed: %v%s\n", console.WARN, err, console.X)
		return false
	}
	psPath := filepath.Join(guardDir, "unleash-guard-loop.ps1")
	logPath := filepath.Join(guardDir, "guard.log")
	ps := fmt.Sprintf(`$ErrorActionPreference = 'SilentlyContinue'
$bin = %q
$log = %q
while ($true) {
  try { & $bin guard *>> $log } catch {}
  Start-Sleep -Seconds 21600
}
`, vpccBin, logPath)
	if err := os.WriteFile(psPath, []byte(ps), 0o644); err != nil {
		fmt.Printf("  %s Windows startup guard script failed: %v%s\n", console.WARN, err, console.X)
		return false
	}

	startupDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	if err := os.MkdirAll(startupDir, 0o755); err != nil {
		fmt.Printf("  %s Windows startup dir failed: %v%s\n", console.WARN, err, console.X)
		return false
	}
	cmdPath := filepath.Join(startupDir, "unleash-guard.cmd")
	cmd := fmt.Sprintf("@echo off\r\nstart \"\" /min powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File %q\r\n", psPath)
	if err := os.WriteFile(cmdPath, []byte(cmd), 0o644); err != nil {
		fmt.Printf("  %s Windows startup guard launcher failed: %v%s\n", console.WARN, err, console.X)
		return false
	}

	fmt.Printf("  %s Windows Startup fallback: unleash-guard.cmd (runs guard every 6h after login)%s\n", console.CHECK, console.X)
	return true
}
