package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/omp"
	"github.com/VoidChecksum/unleash/internal/productcli"
)

var version = "dev"

//go:embed omp-patches/*.json
var embeddedOMPPatches embed.FS

var findOMPTarget = omp.FindTarget

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "unleash-omp",
		Short:   "Unleash-OMP — Oh-My-Pi operator setup",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.AddCommand(newSetupCmd(), newPatchCmd(), newStatusCmd(), newVerifyCmd(), newInstallRulesCmd(), newUninstallRulesCmd(), newRollbackCmd())
	return root
}

func newSetupCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Patch OMP and install Unleash-OMP rules/config",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "unleash-omp setup")
			t, ok := findOMPTarget()
			if !ok {
				fmt.Fprintln(out, "OMP not found. Install with: bun install -g @oh-my-pi/pi-coding-agent")
			} else {
				fmt.Fprintf(out, "target: %s (%s)\n", t.Path, t.Kind)
				if err := runPatch(out, dryRun); err != nil {
					return err
				}
			}
			if dryRun {
				fmt.Fprintln(out, "dry-run: rules/config not written")
				return nil
			}
			if err := omp.InstallRules("", omp.DefaultAuthorizationBlock()); err != nil {
				return err
			}
			fmt.Fprintln(out, "rules: installed")
			return runVerify(out)
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without writing")
	return cmd
}

func newPatchCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Apply OMP byte patches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPatch(cmd.OutOrStdout(), dryRun)
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without writing")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show OMP target and Unleash-OMP state",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if t, ok := findOMPTarget(); ok {
				version, _ := omp.DetectVersion()
				fmt.Fprintf(out, "target: %s\nkind: %s\nsha: %s\n", t.Path, t.Kind, omp.SHA256Short(t.Path))
				if version != "" {
					fmt.Fprintf(out, "version: %s\n", version)
				}
			} else {
				fmt.Fprintln(out, "target: not found")
			}
			fmt.Fprintf(out, "state: %s\n", omp.StateDir(""))
			return nil
		},
	}
}

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify OMP target and Unleash-OMP config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(cmd.OutOrStdout())
		},
	}
}

func newInstallRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-rules",
		Short: "Install OMP operator rules/config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := omp.InstallRules("", omp.DefaultAuthorizationBlock()); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "rules: installed")
			return nil
		},
	}
}

func newUninstallRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-rules",
		Short: "Remove Unleash-OMP managed rules block",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := omp.UninstallRules(""); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "rules: uninstalled")
			return nil
		},
	}
}

func newRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "Restore the newest OMP backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, ok := findOMPTarget()
			if !ok {
				return fmt.Errorf("OMP target not found")
			}
			backup, err := omp.RestoreLatestBackup(t.Path, "")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "restored: %s\n", backup)
			return nil
		},
	}
}

func runPatch(out interface{ Write([]byte) (int, error) }, dryRun bool) error {
	t, ok := findOMPTarget()
	if !ok {
		fmt.Fprintln(out, "target: not found")
		return nil
	}
	patches, err := loadOMPPatches()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "target: %s\n", t.Path)
	res, err := omp.ApplyPatches(t.Path, patches, dryRun, "")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "applied: %d\nskipped: %d\n", res.Applied, res.Skipped)
	if res.BackupPath != "" {
		fmt.Fprintf(out, "backup: %s\n", res.BackupPath)
	}
	return nil
}

func runVerify(out interface{ Write([]byte) (int, error) }) error {
	t, ok := findOMPTarget()
	if !ok {
		fmt.Fprintln(out, "target: not found")
		return fmt.Errorf("OMP target not found")
	}
	fmt.Fprintf(out, "target: ok (%s)\n", t.Path)
	configPath := filepath.Join(omp.AgentDir(""), "config.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(out, "config: missing")
		return fmt.Errorf("OMP config missing")
	}
	if strings.Contains(string(data), "approvalMode: yolo") {
		fmt.Fprintln(out, "config: ok")
		return nil
	}
	return fmt.Errorf("config missing Unleash-OMP approval mode")
}

func loadOMPPatches() ([]omp.Patch, error) {
	patches, err := loadOMPPatchesFromEmbedded()
	if err != nil {
		return nil, err
	}
	overrideDir := filepath.Join(omp.StateDir(""), "patches")
	extra, err := loadOMPPatchesFromDir(overrideDir)
	if err != nil {
		return nil, err
	}
	return mergeOMPPatches(patches, extra), nil
}

func loadOMPPatchesFromEmbedded() ([]omp.Patch, error) {
	return productcli.LoadPatchesFromFS(embeddedOMPPatches, "omp-patches")
}

func loadOMPPatchesFromDir(dir string) ([]omp.Patch, error) {
	return productcli.LoadPatchesFromDir(dir)
}

func mergeOMPPatches(base, extra []omp.Patch) []omp.Patch {
	return productcli.MergePatches(base, extra)
}
