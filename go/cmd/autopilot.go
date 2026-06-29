package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/patches"
	"github.com/VoidChecksum/void-patcher-cc/internal/scanner"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
)

// NewAutopilotCmd creates the "autopilot" cobra command.
func NewAutopilotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "autopilot",
		Short: "Full pipeline: scan → auto-heal → patch → git commit/push → GitHub issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runAutopilot()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

func runAutopilot() int {
	tgt, kind := target.FindTarget()
	if tgt == "" {
		fmt.Printf("%sclaude-code not found%s\n", console.R, console.X)
		return 2
	}

	ts := time.Now().Format("15:04:05")
	curSHA := target.SHA256Short(tgt)
	pd := patchDir()

	fmt.Printf("%sunleash autopilot — %s%s\n", console.B, ts, console.X)
	fmt.Printf("  target : %s\n", tgt)
	fmt.Printf("  sha    : %s\n", curSHA)

	// ── 1. Scan for drift ─────────────────────────────────────────────────
	fmt.Printf("\n  %s[1/5] scanning...%s\n", console.B, console.X)
	text, err := scanner.LoadTextFromTarget(tgt, kind)
	if err != nil {
		fmt.Printf("  %sextract failed: %v%s\n", console.R, err, console.X)
		return 2
	}
	patchData, _ := patches.LoadPatchesForScan(pd, true)
	if patchData == nil {
		patchData, _ = patches.LoadPatchesForScanFromEmbed(true)
	}
	sc := scanner.NewSigScanner(text)
	rows := sc.ScanPatches(patchData)

	nOK, nApplied, nDrift, nRetired := 0, 0, 0, 0
	for _, r := range rows {
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
	fmt.Printf("  %s%d ok%s  %d applied  %s%d drift%s  %d retired\n",
		console.G, nOK, console.X,
		nApplied,
		console.R, nDrift, console.X,
		nRetired)

	var healResult *scanner.HealResult
	if nDrift == 0 {
		fmt.Printf("  %sno drift — skipping heal%s\n", console.G, console.X)
	} else {
		// ── 2. Auto-heal drifted patches ──────────────────────────────────
		fmt.Printf("\n  %s[2/5] auto-healing %d drifted patches...%s\n", console.B, nDrift, console.X)
		hr := scanner.AutoHealDrift(text, pd, true)
		healResult = &hr
		fmt.Printf("  %s%d healed%s  %s%d failed%s\n",
			console.G, healResult.Healed, console.X,
			console.R, healResult.Failed, console.X)
	}

	// ── 3. Apply patches ──────────────────────────────────────────────────
	fmt.Printf("\n  %s[3/5] patching binary...%s\n", console.B, console.X)
	rcPatchErr := RunPatchQuiet()
	rcPatch := 0
	if rcPatchErr != nil {
		rcPatch = 1
	}

	// ── 4. Git commit + push ────────────────────────────────────────────
	fmt.Printf("\n  %s[4/5] checking git...%s\n", console.B, console.X)
	gitPushed := false
	repoRoot := findGitRoot(pd)
	if repoRoot != "" {
		diffCmd := exec.Command("git", "diff", "--quiet", "--", "patches/")
		diffCmd.Dir = repoRoot
		hasChanges := diffCmd.Run() != nil

		untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "patches/")
		untrackedCmd.Dir = repoRoot
		untrackedOut, _ := untrackedCmd.Output()
		hasNew := len(strings.TrimSpace(string(untrackedOut))) > 0

		if hasChanges || hasNew {
			ccVer := detectCCVersion(tgt)
			healed := 0
			if healResult != nil {
				healed = healResult.Healed
			}
			msg := fmt.Sprintf("fix(patches): auto-heal for Claude Code %s\n\n"+
				"Binary SHA: %s\n"+
				"Healed: %d patches\n"+
				"Scan: %d ok, %d applied, %d drift\n\n"+
				"[autopilot]", ccVer, curSHA, healed, nOK, nApplied, nDrift)

			addCmd := exec.Command("git", "add", "patches/")
			addCmd.Dir = repoRoot
			addCmd.Run()

			commitCmd := exec.Command("git", "commit", "-m", msg)
			commitCmd.Dir = repoRoot
			commitOut, commitErr := commitCmd.Output()
			if commitErr == nil {
				lines := strings.Split(strings.TrimSpace(string(commitOut)), "\n")
				if len(lines) > 0 {
					fmt.Printf("  %s%s committed:%s %s\n", console.G, console.CHECK, console.X, lines[0])
				}

				pushCmd := exec.Command("git", "push", "origin", "HEAD")
				pushCmd.Dir = repoRoot
				if pushErr := pushCmd.Run(); pushErr == nil {
					fmt.Printf("  %s%s pushed to origin%s\n", console.G, console.CHECK, console.X)
					gitPushed = true
				} else {
					fmt.Printf("  %s%s push failed%s\n", console.Y, console.WARN, console.X)
				}
			} else {
				fmt.Printf("  %snothing to commit%s\n", console.Y, console.X)
			}
		} else {
			fmt.Printf("  %sno patch changes to commit%s\n", console.G, console.X)
		}
	} else {
		fmt.Printf("  %snot a git repo — skip commit/push%s\n", console.Y, console.X)
	}

	// ── 5. Create GitHub issue for unfixable drift ──────────────────────
	var unfixed []patches.ScanRow
	if nDrift > 0 {
		patchData2, _ := patches.LoadPatchesForScan(pd, true)
		if patchData2 == nil {
			patchData2, _ = patches.LoadPatchesForScanFromEmbed(true)
		}
		rows2 := scanner.NewSigScanner(text).ScanPatches(patchData2)
		for _, r := range rows2 {
			if r.Status == "drift" {
				unfixed = append(unfixed, r)
			}
		}
	}

	fmt.Printf("\n  %s[5/5] issue check...%s\n", console.B, console.X)
	if len(unfixed) > 0 {
		ccVer := detectCCVersion(tgt)
		issueTitle := fmt.Sprintf("Signature drift: %d patches broken on CC %s", len(unfixed), ccVer)
		issueBody := fmt.Sprintf("## Autopilot drift report\n\n"+
			"- **Claude Code version**: `%s`\n"+
			"- **Binary SHA**: `%s`\n"+
			"- **Unfixed patches**: %d\n\n"+
			"| Patch | Anchors |\n|---|---|\n", ccVer, curSHA, len(unfixed))
		for _, u := range unfixed {
			var anchorStrs []string
			limit := minInt(2, len(u.Anchors))
			for _, a := range u.Anchors[:limit] {
				s := a
				if len(s) > 40 {
					s = s[:40]
				}
				anchorStrs = append(anchorStrs, fmt.Sprintf("`%s`", s))
			}
			anchors := "none"
			if len(anchorStrs) > 0 {
				anchors = strings.Join(anchorStrs, ", ")
			}
			issueBody += fmt.Sprintf("| `%s` | %s |\n", u.ID, anchors)
		}
		issueBody += fmt.Sprintf("\n### Action needed\n"+
			"RE the binary at `%s` and update `search_regex` + `anchor_strings` "+
			"in the affected `patches/*.json` files.\n\n"+
			"*Generated by `unleash autopilot`*", tgt)

		ghCmd := exec.Command("gh", "issue", "create",
			"--title", issueTitle,
			"--body", issueBody,
			"--label", "signature-drift")
		if repoRoot != "" {
			ghCmd.Dir = repoRoot
		}
		ghOut, ghErr := ghCmd.Output()
		if ghErr == nil {
			fmt.Printf("  %s%s issue created:%s %s\n",
				console.G, console.CHECK, console.X, strings.TrimSpace(string(ghOut)))
		} else {
			fmt.Printf("  %s%s gh issue failed (install 'gh' CLI for auto-issues)%s\n",
				console.Y, console.WARN, console.X)
		}
	} else {
		fmt.Printf("  %sno unfixed drift — no issue needed%s\n", console.G, console.X)
	}

	// ── Summary ───────────────────────────────────────────────────────────
	fmt.Printf("\n%s%s%s\n", console.B, strings.Repeat("─", 50), console.X)
	statusIcon := console.CHECK
	color := console.G
	if len(unfixed) > 0 {
		statusIcon = console.WARN
		color = console.Y
	} else if rcPatch != 0 {
		statusIcon = console.CROSS
		color = console.R
	}
	fmt.Printf("  %s%s autopilot complete%s\n", color, statusIcon, console.X)
	if rcPatch == 0 {
		fmt.Printf("  patches : applied\n")
	} else {
		fmt.Printf("  patches : failed\n")
	}
	if nDrift > 0 && healResult != nil {
		fmt.Printf("  healed  : %d/%d\n", healResult.Healed, nDrift)
	}
	if gitPushed {
		fmt.Printf("  git     : committed + pushed\n")
	}
	if len(unfixed) > 0 {
		fmt.Printf("  %sunfixed : %d patches need manual RE%s\n", console.R, len(unfixed), console.X)
	}

	if len(unfixed) > 0 {
		return 1
	}
	if rcPatch != 0 {
		return 2
	}
	return 0
}

func findGitRoot(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
