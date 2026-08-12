package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/target"
)

// NewUnpatchCmd restores binaries from backups (full unpatch).
func NewUnpatchCmd() *cobra.Command {
	var list bool
	var to string
	var allTargets bool
	c := &cobra.Command{
		Use:     "unpatch",
		Aliases: []string{"restore"},
		Short:   "Restore Claude Code from backup (undo patches)",
		Long: `Restore one or more Claude Code binaries from Unleash backups.

  unleash unpatch              restore every known target from its latest backup
  unleash unpatch --list       show backups / index
  unleash unpatch --to FILE    restore primary target from a specific backup
  unleash restore              alias

Backups are created automatically on every successful 'unleash patch'.
Selection (enable/disable) is unchanged — unpatch only restores binary bytes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnpatch(list, to, allTargets)
		},
	}
	c.Flags().BoolVar(&list, "list", false, "List backups and index entries")
	c.Flags().StringVar(&to, "to", "", "Restore primary target from this backup file")
	c.Flags().BoolVar(&allTargets, "all-targets", true, "Restore every discovered target that has a backup")
	return c
}

func runUnpatch(list bool, to string, allTargets bool) error {
	if list {
		return listBackups()
	}

	if to != "" {
		tgt, _ := target.FindTarget()
		if tgt == "" {
			return fmt.Errorf("claude-code not found")
		}
		return restoreFile(tgt, to)
	}

	all := target.FindAllTargets()
	if len(all) == 0 {
		return fmt.Errorf("claude-code not found")
	}

	// Prefer index-based per-target restore
	okN, failN := 0, 0
	restored := map[string]bool{}

	if allTargets {
		for _, f := range all {
			rec, found := target.LatestBackupForPath(f.Path)
			if !found {
				// fallback: newest global backup only for primary
				continue
			}
			if err := restoreFile(f.Path, rec.Backup); err != nil {
				fmt.Printf("  %sfail%s %s  %v\n", console.R, console.X, f.Path, err)
				failN++
				continue
			}
			restored[f.Path] = true
			okN++
		}
	}

	// If nothing restored via index, fall back to legacy single latest backup → primary
	if okN == 0 {
		tgt := all[0].Path
		bdir := target.BackupDir()
		entries, _ := filepath.Glob(filepath.Join(bdir, "claude.*.bak"))
		if len(entries) == 0 {
			return fmt.Errorf("no backups in %s — run 'unleash patch' first", bdir)
		}
		sort.Strings(entries)
		latest := entries[len(entries)-1]
		if err := restoreFile(tgt, latest); err != nil {
			return err
		}
		// also try same backup on hardlinked same-size targets
		for _, f := range all[1:] {
			if sameFileIdentity(tgt, f.Path) {
				_ = restoreFile(f.Path, latest)
			}
		}
		okN = 1
	}

	fmt.Printf("\n%s%d restored %s %d failed%s\n", console.B, okN, console.DOT, failN, console.X)
	if failN > 0 {
		return fmt.Errorf("unpatch partially failed")
	}
	// Clear SHA manifest so guard re-patches if desired
	_ = os.Remove(shaManifestPath())
	return nil
}

func listBackups() error {
	bdir := target.BackupDir()
	fmt.Printf("%sunleash backups%s  %s\n", console.B, console.X, bdir)

	recs, _ := target.ListBackupRecords()
	if len(recs) > 0 {
		fmt.Printf("\n  index (%d entries, newest last):\n", len(recs))
		start := 0
		if len(recs) > 20 {
			start = len(recs) - 20
			fmt.Printf("  … showing last 20\n")
		}
		for _, r := range recs[start:] {
			st := "missing"
			if _, err := os.Stat(r.Backup); err == nil {
				st = "ok"
			}
			fmt.Printf("  %s  %s  %s\n    <- %s\n", r.Time, padRight(r.SHA, 12), st, r.Path)
			fmt.Printf("    %s\n", filepath.Base(r.Backup))
		}
	}

	entries, _ := filepath.Glob(filepath.Join(bdir, "claude.*.bak"))
	sort.Strings(entries)
	fmt.Printf("\n  files: %d\n", len(entries))
	for _, e := range entries {
		info, err := os.Stat(e)
		sz := int64(0)
		if err == nil {
			sz = info.Size() / 1024 / 1024
		}
		fmt.Printf("  %s  %d MB\n", filepath.Base(e), sz)
	}
	if len(entries) == 0 && len(recs) == 0 {
		fmt.Printf("  %s(none — run unleash patch to create backups)%s\n", console.Y, console.X)
	}
	return nil
}

func restoreFile(dstPath, backupPath string) error {
	info, err := os.Stat(dstPath)
	mode := os.FileMode(0o755)
	if err == nil {
		mode = info.Mode().Perm()
	}
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %s", backupPath)
	}

	dir := filepath.Dir(dstPath)
	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.unleash-unpatch-%d", filepath.Base(dstPath), os.Getpid()))

	src, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		return err
	}
	dst.Close()
	os.Chmod(tmpPath, mode)

	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(dstPath)
		if err2 := os.Rename(tmpPath, dstPath); err2 != nil {
			// in-place overwrite when rename is denied (exe locked)
			in, rerr := os.ReadFile(tmpPath)
			if rerr != nil {
				_ = os.Remove(tmpPath)
				return fmt.Errorf("restore failed (rename: %v; read temp: %v) — close Claude Code and retry", err2, rerr)
			}
			if werr := os.WriteFile(dstPath, in, mode); werr != nil {
				_ = os.Remove(tmpPath)
				return fmt.Errorf("restore failed (rename: %v; write: %v) — close Claude Code and retry", err2, werr)
			}
			_ = os.Remove(tmpPath)
		}
	}
	fmt.Printf("  %s%s%s %s <- %s\n", console.G, console.CHECK, console.X, dstPath, filepath.Base(backupPath))
	return nil
}

func sameFileIdentity(a, b string) bool {
	// same short sha or same resolved path
	if target.SHA256Short(a) == target.SHA256Short(b) {
		return true
	}
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 == nil && err2 == nil && ra == rb {
		return true
	}
	return false
}

// Improve legacy rollback to use the same multi-target path.
func runRollbackImproved() int {
	if err := runUnpatch(false, "", true); err != nil {
		fmt.Printf("%s%s%s\n", console.R, err.Error(), console.X)
		if strings.Contains(err.Error(), "not found") {
			return 2
		}
		return 1
	}
	return 0
}
