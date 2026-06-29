package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/cmd"
)

var version = "1.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "unleash",
		Short: "Unleash — binary patcher for Claude Code",
		Version: version,
		Run: func(c *cobra.Command, args []string) {
			// Default: launch TUI when no subcommand given
			p := cmd.NewTuiProgram()
			if _, err := p.Run(); err != nil {
				c.Help()
			}
		},
	}

	rootCmd.AddCommand(
		cmd.NewPatchCmd(),
		cmd.NewVerifyCmd(),
		cmd.NewRollbackCmd(),
		cmd.NewStatusCmd(),
		cmd.NewListCmd(),
		cmd.NewSelfUpdateCmd(),
		cmd.NewAutohealCmd(),
		cmd.NewCheckUpdatesCmd(),
		cmd.NewInstallPreloadCmd(),
		cmd.NewUninstallPreloadCmd(),
		cmd.NewInstallRulesCmd(),
		cmd.NewUninstallRulesCmd(),
		cmd.NewScanCmd(),
		cmd.NewDoctorCmd(),
		cmd.NewWatchCmd(),
		cmd.NewUpgradeCmd(),
		cmd.NewBenchCmd(),
		cmd.NewGuardCmd(),
		cmd.NewInstallGuardCmd(),
		cmd.NewUninstallGuardCmd(),
		cmd.NewAutopilotCmd(),
		cmd.NewDashboardCmd(),
		cmd.NewTuiCmd(),
		cmd.NewUpdateCmd(),
		cmd.NewSetupCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
