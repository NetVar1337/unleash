package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRollbackCmd creates the "rollback" cobra command.
func NewRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rollback",
		Aliases: []string{"restore-backup"},
		Short:   "Restore from backup (same as unpatch)",
		Long:    "Alias for 'unleash unpatch' — restores every target from its latest backup.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc := RunRollback()
			if rc != 0 {
				os.Exit(rc)
			}
			return nil
		},
	}
}

// RunRollback restores backups for all targets. Returns exit code.
func RunRollback() int {
	return runRollbackImproved()
}

// RunRollbackErr wraps RunRollback as error for autoheal callback.
func RunRollbackErr() error {
	rc := RunRollback()
	if rc != 0 {
		return fmt.Errorf("rollback failed (rc=%d)", rc)
	}
	return nil
}
