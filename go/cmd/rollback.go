package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/internal/console"
	"github.com/VoidChecksum/void-patcher-cc/internal/target"
)

// NewRollbackCmd creates the "rollback" cobra command.
func NewRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "Restore from most recent backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := RunRollback()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

// RunRollback restores the most recent backup. Returns exit code.
func RunRollback() int {
	tgt, _ := target.FindTarget()
	if tgt == "" {
		fmt.Printf("%sclaude-code not found%s\n", console.R, console.X)
		return 2
	}

	bdir := target.BackupDir()
	entries, err := filepath.Glob(filepath.Join(bdir, "claude.*.bak"))
	if err != nil || len(entries) == 0 {
		fmt.Printf("%sno backups in %s%s\n", console.R, bdir, console.X)
		return 1
	}
	sort.Strings(entries)
	latest := entries[len(entries)-1]

	// Get original mode
	info, err := os.Stat(tgt)
	mode := os.FileMode(0o755)
	if err == nil {
		mode = info.Mode().Perm()
	}

	// Copy to temp, then atomic replace
	dir := filepath.Dir(tgt)
	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.vpcc-rollback-%d", filepath.Base(tgt), os.Getpid()))

	src, err := os.Open(latest)
	if err != nil {
		fmt.Printf("%srollback read failed: %v%s\n", console.R, err, console.X)
		return 2
	}
	defer src.Close()

	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		fmt.Printf("%srollback write failed: %v%s\n", console.R, err, console.X)
		return 2
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		fmt.Printf("%srollback copy failed: %v%s\n", console.R, err, console.X)
		return 2
	}
	dst.Close()

	os.Chmod(tmpPath, mode)
	if err := os.Rename(tmpPath, tgt); err != nil {
		os.Remove(tmpPath)
		fmt.Printf("%srollback rename failed: %v%s\n", console.R, err, console.X)
		return 2
	}

	fmt.Printf("%s%s restored%s %s <- %s\n", console.G, console.CHECK, console.X, tgt, filepath.Base(latest))
	return 0
}

// RunRollbackErr wraps RunRollback as error for autoheal callback.
func RunRollbackErr() error {
	rc := RunRollback()
	if rc != 0 {
		return fmt.Errorf("rollback failed (rc=%d)", rc)
	}
	return nil
}
