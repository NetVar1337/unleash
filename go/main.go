package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/void-patcher-cc/cmd"
)

const version = "3.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "vpcc",
		Short: "Void Patcher for Claude Code — regex-signature patches, cli.js + Bun SEA",
		Version: version,
		Run: func(c *cobra.Command, args []string) {
			c.Help()
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
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
