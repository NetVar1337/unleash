package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/patches"
	"github.com/VoidChecksum/void-patcher-cc/internal/scanner"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
	"github.com/VoidChecksum/void-patcher-cc/internal/updater"
)

// NewSelfUpdateCmd creates the "self-update" cobra command.
func NewSelfUpdateCmd() *cobra.Command {
	var dryRun, force, noReapply bool
	c := &cobra.Command{
		Use:   "self-update",
		Short: "Pull latest patches/*.json from GitHub and re-apply",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runSelfUpdate(dryRun, force, noReapply)
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without writing")
	c.Flags().BoolVarP(&force, "force", "f", false, "Sync even if commit hashes match")
	c.Flags().BoolVar(&noReapply, "no-reapply", false, "Skip re-applying after sync")
	return c
}

// NewAutohealCmd creates the "autoheal" cobra command.
func NewAutohealCmd() *cobra.Command {
	var force, quiet bool
	c := &cobra.Command{
		Use:   "autoheal",
		Short: "Detect Claude Code drift; self-update + re-patch if broken",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runAutoheal(force, quiet)
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "Run checks even if CC sha unchanged")
	c.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress output")
	return c
}

// NewCheckUpdatesCmd creates the "check-updates" cobra command.
func NewCheckUpdatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-updates",
		Short: "Show if remote patches differ from local",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runCheckUpdates()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

// NewUpgradeCmd creates the "upgrade" cobra command.
func NewUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "All-in-one: self-update + autoheal + verify + warm cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runUpgrade()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

// NewUpdateCmd creates the "update" cobra command.
func NewUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Full self-update: upgrade unleash + sync patches + re-patch binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := runUpdate()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

func runAutoheal(force, quiet bool) int {
	pd := patchDir()
	cb := updater.AutohealCallbacks{
		FindTarget:  target.FindTarget,
		SHA256Short: target.SHA256Short,
		CmdVerify:   RunVerify,
		CmdPatch:    RunPatchRC,
		CmdRollback: RunRollbackErr,
	}
	return updater.Autoheal(cb, pd, force, quiet)
}

func runSelfUpdate(dryRun, force, noReapply bool) int {
	pd := patchDir()
	fmt.Printf("%sunleash self-update%s  <- %s@%s\n",
		console.B, console.X, updater.Repo, updater.Branch)

	remote := updater.RemoteHeadSHA("patches")
	if remote == "" {
		fmt.Printf("%scould not reach GitHub API%s\n", console.R, console.X)
		return 2
	}

	state := updater.LoadState()
	local := stateString(state, "patches_commit")
	fmt.Printf("  local  : %s\n", coalesce(local, "(unknown)"))
	fmt.Printf("  remote : %s\n", remote)

	if local == remote && !force {
		fmt.Printf("%s%s already up to date%s\n", console.G, console.CHECK, console.X)
		return 0
	}
	if dryRun {
		fmt.Printf("%sdry-run: would sync%s\n", console.Y, console.X)
		return 0
	}

	// Ensure patch dir exists
	os.MkdirAll(pd, 0o755)

	changed, shaOrErr := updater.SyncPatches(pd, remote)
	if changed < 0 {
		fmt.Printf("%s%s sync failed — %s%s\n", console.R, console.CROSS, shaOrErr, console.X)
		return 2
	}
	shaShort := shaOrErr
	if len(shaShort) > 7 {
		shaShort = shaShort[:7]
	}
	fmt.Printf("%s%s synced%s  %d file(s) updated @ %s\n",
		console.G, console.CHECK, console.X, changed, shaShort)

	if changed > 0 && !noReapply {
		fmt.Printf("\n%sre-applying patches%s\n", console.B, console.X)
		if err := RunPatchQuiet(); err != nil {
			return 1
		}
	}
	return 0
}

func runCheckUpdates() int {
	pd := patchDir()
	info := updater.UpstreamStatus(pd)

	fmt.Printf("%sunleash check-updates%s\n", console.B, console.X)
	fmt.Printf("  local commit  : %s\n", coalesce(info.LocalCommit, "(unknown)"))
	fmt.Printf("  remote commit : %s\n", coalesce(info.RemoteCommit, "(unreachable)"))
	fmt.Printf("  local files   : %d\n", info.LocalFiles)

	if info.Drift {
		fmt.Printf("%s%s update available — run 'unleash self-update'%s\n",
			console.Y, console.WARN, console.X)
		return 1
	}
	if info.LocalCommit == "" && info.RemoteCommit != "" {
		fmt.Printf("%s%s no sync state — run 'unleash self-update' to pin current%s\n",
			console.Y, console.WARN, console.X)
		return 1
	}
	if info.RemoteCommit != "" {
		fmt.Printf("%s%s up to date%s\n", console.G, console.CHECK, console.X)
	}
	return 0
}

func runUpgrade() int {
	fmt.Printf("%sunleash upgrade — full pipeline%s\n", console.B, console.X)

	// Step 1: self-update
	fmt.Printf("  step 1/4: self-update patches\n")
	rcSU := runSelfUpdate(false, false, false)
	if rcSU != 0 && rcSU != 1 {
		fmt.Printf("  %sself-update returned rc=%d, continuing%s\n", console.Y, rcSU, console.X)
	}

	// Step 2: autoheal
	fmt.Printf("\n  step 2/4: autoheal (force)\n")
	rcAH := runAutoheal(true, false)
	if rcAH == 3 {
		fmt.Printf("  %sautoheal failed — aborting upgrade%s\n", console.R, console.X)
		return 3
	}

	// Step 3: verify
	fmt.Printf("\n  step 3/4: verify\n")
	rcV := RunVerify()
	if rcV != 0 {
		fmt.Printf("  %sverify failed — manual intervention required%s\n", console.R, console.X)
		return 3
	}

	// Step 4: warm scan cache
	fmt.Printf("\n  step 4/4: warm scan cache\n")
	tgt, kind := target.FindTarget()
	pd := patchDir()
	if tgt != "" {
		text, err := scanner.LoadTextFromTarget(tgt, kind)
		if err == nil {
			patchData, _ := patches.LoadPatchesForScan(pd, true)
			if patchData == nil {
				patchData, _ = patches.LoadPatchesForScanFromEmbed(true)
			}
			rows := scanner.NewSigScanner(text).ScanPatches(patchData)
			scanner.SaveCachedRows(tgt, pd, rows)
			drift := 0
			for _, r := range rows {
				if r.Status == "drift" {
					drift++
				}
			}
			fmt.Printf("  cache warmed: %d patches, %d drift\n", len(rows), drift)
		} else {
			fmt.Printf("  %scache warm skipped: %v%s\n", console.Y, err, console.X)
		}
	}

	fmt.Printf("\n%s%s upgrade complete%s\n", console.G, console.CHECK, console.X)
	return 0
}

func runUpdate() int {
	fmt.Printf("%sunleash update%s\n", console.B, console.X)
	fmt.Printf("  current : v1.0.0\n")
	fmt.Printf("  install : go (embedded)\n")

	// Step 1: check remote version (best effort)
	fmt.Printf("\n  %s[1/4] checking for updates...%s\n", console.B, console.X)
	// Go binary version checks are best-effort via patch commit comparison
	remoteCommit := updater.RemoteHeadSHA("patches")
	if remoteCommit != "" {
		fmt.Printf("  remote patches : %s\n", remoteCommit[:minInt(7, len(remoteCommit))])
	} else {
		fmt.Printf("  %s%s could not check remote — updating anyway%s\n",
			console.Y, console.WARN, console.X)
	}

	// Step 2: Go binary cannot self-update tool
	fmt.Printf("\n  %s[2/4] skipping tool update (go binary — rebuild to update)%s\n",
		console.B, console.X)

	// Step 3: sync patches
	fmt.Printf("\n  %s[3/4] syncing patches...%s\n", console.B, console.X)
	pd := patchDir()
	os.MkdirAll(pd, 0o755)
	if remoteCommit != "" {
		state := updater.LoadState()
		localSHA := stateString(state, "patches_commit")
		if localSHA == remoteCommit {
			fmt.Printf("  %s%s patches already at latest (%s)%s\n",
				console.G, console.CHECK, remoteCommit[:minInt(7, len(remoteCommit))], console.X)
		} else {
			changed, shaOrErr := updater.SyncPatches(pd, remoteCommit)
			if changed >= 0 {
				shaShort := shaOrErr
				if len(shaShort) > 7 {
					shaShort = shaShort[:7]
				}
				fmt.Printf("  %s%s %d patch file(s) synced @ %s%s\n",
					console.G, console.CHECK, changed, shaShort, console.X)
			} else {
				fmt.Printf("  %s%s patch sync failed: %s%s\n",
					console.R, console.CROSS, shaOrErr, console.X)
			}
		}
	} else {
		fmt.Printf("  %s%s could not reach GitHub — using local patches%s\n",
			console.Y, console.WARN, console.X)
	}

	// Step 4: re-apply patches
	fmt.Printf("\n  %s[4/4] patching Claude Code binary...%s\n", console.B, console.X)
	tgt, _ := target.FindTarget()
	rcPatch := 0
	if tgt != "" {
		if err := RunPatchQuiet(); err != nil {
			rcPatch = 1
		}
		if rcPatch == 0 {
			stampPath := filepath.Join(unleashDir(), "last_patched_sha")
			os.MkdirAll(filepath.Dir(stampPath), 0o755)
			os.WriteFile(stampPath, []byte(target.SHA256Short(tgt)), 0o644)
		}
	} else {
		fmt.Printf("  %s%s Claude Code not found — skipping binary patch%s\n",
			console.Y, console.WARN, console.X)
	}

	// Summary
	fmt.Printf("\n%s%s%s\n", console.B, strings.Repeat("─", 40), console.X)
	if rcPatch == 0 {
		fmt.Printf("  %s%s unleash update complete%s\n", console.G, console.CHECK, console.X)
	} else {
		fmt.Printf("  %s%s update complete with warnings (rc=%d)%s\n",
			console.Y, console.WARN, rcPatch, console.X)
	}

	return rcPatch
}
