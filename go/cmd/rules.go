package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/scanner"
	"github.com/VoidChecksum/unleash/internal/target"
)

const (
	unleashClaudeMDStart = "<!-- unleash:authorization:start -->"
	unleashClaudeMDEnd   = "<!-- unleash:authorization:end -->"
)

// NewInstallRulesCmd creates the "install-rules" cobra command.
func NewInstallRulesCmd() *cobra.Command {
	var noHook bool
	c := &cobra.Command{
		Use:   "install-rules",
		Short: "Deploy operator-authorization bundle (CLAUDE.md + settings + hook)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runInstallRules(noHook)
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&noHook, "no-hook", false,
		"Skip PreToolUse shell hook; rely on binary patch 05 (zero subprocess overhead)")
	return c
}

// NewUninstallRulesCmd creates the "uninstall-rules" cobra command.
func NewUninstallRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-rules",
		Short: "Remove unleash authorization rules, preserve operator content",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runUninstallRules()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

func runInstallRules(noHook bool) int {
	authSrc, err := getContribFile("rules/AUTHORIZATION.md")
	if err != nil {
		fmt.Printf("%scontrib/rules/ missing in package%s\n", console.R, console.X)
		return 2
	}

	claudeDir := filepath.Join(homeDir(), ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	os.MkdirAll(claudeDir, 0o755)
	os.MkdirAll(hooksDir, 0o755)

	skipHook := noHook
	pd := patchDir()

	// Auto-detect: if binary patch 05 is applied, .sh hook is redundant
	if !skipHook {
		tgt, kind := target.FindTarget()
		if tgt != "" {
			autoAllowPath := filepath.Join(pd, "05-auto-allow-hook.json")
			if data, err := os.ReadFile(autoAllowPath); err == nil {
				var patchJSON map[string]interface{}
				if json.Unmarshal(data, &patchJSON) == nil {
					if subs, ok := patchJSON["patches"].([]interface{}); ok {
						text, loadErr := scanner.LoadTextFromTarget(tgt, kind)
						if loadErr == nil {
							for _, sub := range subs {
								if sm, ok := sub.(map[string]interface{}); ok {
									if marker, ok := sm["applied_marker"].(string); ok && marker != "" {
										if strings.Contains(text, marker) {
											skipHook = true
											fmt.Printf("  %s%s%s binary patch 05 auto-allow detected %s skipping .sh hook (faster)\n",
												console.G, console.ARROW, console.X, console.ARROW)
											break
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Deploy AUTHORIZATION.md into every major agent instruction surface.
	block := unleashClaudeMDStart + "\n" + strings.TrimRight(string(authSrc), "\n\r") + "\n" + unleashClaudeMDEnd + "\n"
	blockRE := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(unleashClaudeMDStart) + `.*?` + regexp.QuoteMeta(unleashClaudeMDEnd) + `\n?`)
	deployManagedAuthBlock := func(dst string, label string) {
		os.MkdirAll(filepath.Dir(dst), 0o755)
		if data, err := os.ReadFile(dst); err == nil {
			existing := string(data)
			if strings.Contains(existing, unleashClaudeMDStart) {
				existing = blockRE.ReplaceAllString(existing, "")
			}
			merged := block + "\n---\n\n" + strings.TrimLeft(existing, "\n\r ")
			_ = os.WriteFile(dst, []byte(merged), 0o644)
		} else {
			_ = os.WriteFile(dst, []byte(block), 0o644)
		}
		fmt.Printf("  %s%s%s %s %s %s\n", console.G, console.CHECK, console.X, label, console.ARROW, dst)
	}

	home := homeDir()
	agentsDir := filepath.Join(home, ".agents")
	os.MkdirAll(agentsDir, 0o755)

	// Claude Code + universal AGENTS
	for _, dstName := range []string{"CLAUDE.md", "AGENTS.md"} {
		deployManagedAuthBlock(filepath.Join(claudeDir, dstName), dstName)
	}
	deployManagedAuthBlock(filepath.Join(agentsDir, "AGENTS.md"), ".agents/AGENTS.md")

	// Home-level instruction files used by many harnesses
	for _, dstName := range []string{
		"AGENTS.md", "CLAUDE.md", "CODEX.md", "GEMINI.md", "GROK.md",
		"OPENCODE.md", "CURSOR.md", "WINDSURF.md", "AIDER.md", "COPILOT.md",
		"CONTINUE.md", "ZED.md", "KILO.md", "DROID.md", "AMP.md", "OMO.md",
	} {
		deployManagedAuthBlock(filepath.Join(home, dstName), "~/"+dstName)
	}

	// Product-specific homes
	deployManagedAuthBlock(filepath.Join(home, ".codex", "AGENTS.md"), ".codex/AGENTS.md")
	deployManagedAuthBlock(filepath.Join(home, ".codex", "CODEX.md"), ".codex/CODEX.md")
	deployManagedAuthBlock(filepath.Join(home, ".omp", "agent", "AGENTS.md"), ".omp/agent/AGENTS.md")
	deployManagedAuthBlock(filepath.Join(home, ".gemini", "GEMINI.md"), ".gemini/GEMINI.md")
	deployManagedAuthBlock(filepath.Join(home, ".config", "opencode", "AGENTS.md"), ".config/opencode/AGENTS.md")
	deployManagedAuthBlock(filepath.Join(home, ".continue", "config", "AGENTS.md"), ".continue/config/AGENTS.md")

	// Cursor rules
	cursorRulesDir := filepath.Join(home, ".cursor", "rules")
	os.MkdirAll(cursorRulesDir, 0o755)
	deployManagedAuthBlock(filepath.Join(home, ".cursorrules"), "~/.cursorrules")
	deployManagedAuthBlock(filepath.Join(cursorRulesDir, "unleash-authorization.mdc"), ".cursor/rules/unleash-authorization.mdc")

	// Copilot user-level instructions template (repo copy is separate)
	deployManagedAuthBlock(filepath.Join(home, ".github", "copilot-instructions.md"), "~/.github/copilot-instructions.md")

	// Merge settings
	settingsPath := filepath.Join(claudeDir, "settings.json")
	rulesData, err := getContribFile("rules/settings-rules.json")
	if err != nil {
		fmt.Printf("  %ssettings-rules.json missing%s\n", console.R, console.X)
		return 2
	}
	var rules map[string]interface{}
	json.Unmarshal(rulesData, &rules)

	if skipHook {
		stripAutoAllowHook(rules)
	}

	var current map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if json.Unmarshal(data, &current) != nil {
			bakPath := settingsPath + ".corrupt.bak"
			os.Rename(settingsPath, bakPath)
			fmt.Printf("  %s%s existing settings.json corrupt -> backed up to %s%s\n",
				console.Y, console.WARN, filepath.Base(bakPath), console.X)
			current = make(map[string]interface{})
		}
	} else {
		current = make(map[string]interface{})
	}

	merged := DeepMerge(current, rules)
	mergedJSON, _ := json.MarshalIndent(merged, "", "  ")
	os.WriteFile(settingsPath, append(mergedJSON, '\n'), 0o644)
	fmt.Printf("  %s%s%s settings.json %s %s (merged %d keys)\n",
		console.G, console.CHECK, console.X, console.ARROW, settingsPath, len(rules))

	if skipHook {
		stripAutoAllowHook(merged)
		mergedJSON, _ = json.MarshalIndent(merged, "", "  ")
		os.WriteFile(settingsPath, append(mergedJSON, '\n'), 0o644)

		hookDst := filepath.Join(hooksDir, "vpcc-auto-allow.sh")
		if _, err := os.Stat(hookDst); err == nil {
			os.Remove(hookDst)
			fmt.Printf("  %s%s%s removed redundant .sh hook (binary patch handles allow)\n",
				console.G, console.CHECK, console.X)
		} else {
			fmt.Printf("  %s%s%s no .sh hook installed (zero subprocess overhead per tool call)\n",
				console.G, console.CHECK, console.X)
		}
	} else {
		hookSrc, err := getContribFile("rules/hooks/vpcc-auto-allow.sh")
		if err == nil {
			hookDst := filepath.Join(hooksDir, "vpcc-auto-allow.sh")
			os.WriteFile(hookDst, hookSrc, 0o755)
			fmt.Printf("  %s%s%s hook %s %s\n", console.G, console.CHECK, console.X, console.ARROW, hookDst)
		}
	}

	fmt.Printf("\n%s%s authorization rules installed%s\n", console.G, console.CHECK, console.X)
	fmt.Printf("  revert: unleash uninstall-rules\n")
	return 0
}

func runUninstallRules() int {
	home := homeDir()
	claudeDir := filepath.Join(home, ".claude")
	removed := 0
	blockRE := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(unleashClaudeMDStart) + `.*?` + regexp.QuoteMeta(unleashClaudeMDEnd) + `\n?`)
	leadingSeparator := regexp.MustCompile(`\A\s*---\s*\n`)

	stripPath := func(p string) {
		data, err := os.ReadFile(p)
		if err != nil {
			return
		}
		s := string(data)
		if !strings.Contains(s, unleashClaudeMDStart) {
			return
		}
		s = blockRE.ReplaceAllString(s, "")
		s = leadingSeparator.ReplaceAllString(s, "")
		_ = os.WriteFile(p, []byte(strings.TrimLeft(s, "\n\r ")), 0o644)
		fmt.Printf("  %s%s%s stripped unleash block from %s\n", console.G, console.CHECK, console.X, p)
		removed++
	}

	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		stripPath(filepath.Join(claudeDir, name))
	}
	stripPath(filepath.Join(home, ".agents", "AGENTS.md"))
	for _, name := range []string{
		"AGENTS.md", "CLAUDE.md", "CODEX.md", "GEMINI.md", "GROK.md",
		"OPENCODE.md", "CURSOR.md", "WINDSURF.md", "AIDER.md", "COPILOT.md",
		"CONTINUE.md", "ZED.md", "KILO.md", "DROID.md", "AMP.md", "OMO.md",
		".cursorrules",
	} {
		stripPath(filepath.Join(home, name))
	}
	stripPath(filepath.Join(home, ".codex", "AGENTS.md"))
	stripPath(filepath.Join(home, ".codex", "CODEX.md"))
	stripPath(filepath.Join(home, ".omp", "agent", "AGENTS.md"))
	stripPath(filepath.Join(home, ".gemini", "GEMINI.md"))
	stripPath(filepath.Join(home, ".config", "opencode", "AGENTS.md"))
	stripPath(filepath.Join(home, ".continue", "config", "AGENTS.md"))
	stripPath(filepath.Join(home, ".cursor", "rules", "unleash-authorization.mdc"))
	stripPath(filepath.Join(home, ".github", "copilot-instructions.md"))

	hook := filepath.Join(claudeDir, "hooks", "vpcc-auto-allow.sh")
	if _, err := os.Stat(hook); err == nil {
		os.Remove(hook)
		fmt.Printf("  %s%s%s removed %s\n", console.G, console.CHECK, console.X, hook)
		removed++
	}

	// Prune unleash keys from settings
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var cur map[string]interface{}
		if json.Unmarshal(data, &cur) == nil {
			keysToRemove := []string{
				"_unleash_header",
				"skipDangerousModePermissionPrompt",
				"dangerouslyDisableSandbox",
				"isBypassPermissionsModeAvailable",
				"bypassPermissionsWorkspaceTrustCheck",
				"autoUpdaterStatus",
			}
			for _, k := range keysToRemove {
				delete(cur, k)
			}

			// Remove env keys from embedded rules
			rulesData, err := getContribFile("rules/settings-rules.json")
			if err == nil {
				var rules map[string]interface{}
				json.Unmarshal(rulesData, &rules)
				if rulesEnv, ok := rules["env"].(map[string]interface{}); ok {
					if curEnv, ok := cur["env"].(map[string]interface{}); ok {
						for k := range rulesEnv {
							delete(curEnv, k)
						}
						if len(curEnv) == 0 {
							delete(cur, "env")
						}
					}
				}
			}

			stripAutoAllowHook(cur)

			outJSON, _ := json.MarshalIndent(cur, "", "  ")
			os.WriteFile(settingsPath, append(outJSON, '\n'), 0o644)
			fmt.Printf("  %s%s%s pruned unleash keys from settings.json\n",
				console.G, console.CHECK, console.X)
			removed++
		}
	}

	if removed == 0 {
		fmt.Printf("%snothing to remove%s\n", console.Y, console.X)
	} else {
		fmt.Printf("\n%s%s authorization rules removed (%d item(s))%s\n",
			console.G, console.CHECK, removed, console.X)
	}
	return 0
}

// stripAutoAllowHook removes vpcc-auto-allow entries from hooks.PreToolUse
func stripAutoAllowHook(settings map[string]interface{}) {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return
	}
	pre, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		return
	}
	var filtered []interface{}
	for _, h := range pre {
		hJSON, _ := json.Marshal(h)
		if !strings.Contains(string(hJSON), "vpcc-auto-allow") {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) > 0 {
		hooks["PreToolUse"] = filtered
	} else {
		delete(hooks, "PreToolUse")
		if len(hooks) == 0 {
			delete(settings, "hooks")
		}
	}
}
