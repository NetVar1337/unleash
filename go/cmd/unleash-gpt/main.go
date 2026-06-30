package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/codex"
	"github.com/VoidChecksum/unleash/internal/productcli"
)

var version = "dev"

//go:embed codex-patches/*.json
var embeddedCodexPatches embed.FS

var findCodexTarget = codex.FindTarget

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "unleash-gpt",
		Short:   "Unleash-GPT — Codex CLI operator setup",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.AddCommand(
		newSetupCmd(),
		newPatchCmd(),
		newStatusCmd(),
		newVerifyCmd(),
		newInstallRulesCmd(),
		newUninstallRulesCmd(),
		newRollbackCmd(),
	)
	return root
}

func newSetupCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Patch Codex and install Unleash-GPT rules/config",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "unleash-gpt setup")
			t, ok := findCodexTarget()
			if !ok {
				fmt.Fprintln(out, "Codex CLI not found. Install with: npm install -g @openai/codex")
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
			if err := codex.InstallRules("", codex.DefaultAuthorizationBlock()); err != nil {
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
		Short: "Apply Codex byte patches",
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
		Short: "Show Codex target and Unleash-GPT state",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if t, ok := findCodexTarget(); ok {
				version, _ := codex.DetectVersion(t.Path)
				fmt.Fprintf(out, "target: %s\nkind: %s\nsha: %s\n", t.Path, t.Kind, codex.SHA256Short(t.Path))
				if version != "" {
					fmt.Fprintf(out, "version: %s\n", version)
				}
			} else {
				fmt.Fprintln(out, "target: not found")
			}
			fmt.Fprintf(out, "state: %s\n", codex.StateDir(""))
			return nil
		},
	}
}

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify Codex target and Unleash-GPT config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(cmd.OutOrStdout())
		},
	}
}

func newInstallRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-rules",
		Short: "Install Codex operator rules/config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := codex.InstallRules("", codex.DefaultAuthorizationBlock()); err != nil {
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
		Short: "Remove Unleash-GPT managed rules block",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := codex.UninstallRules(""); err != nil {
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
		Short: "Restore the newest Codex backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, ok := findCodexTarget()
			if !ok {
				return fmt.Errorf("Codex target not found")
			}
			backup, err := codex.RestoreLatestBackup(t.Path, "")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "restored: %s\n", backup)
			return nil
		},
	}
}

func runPatch(out interface{ Write([]byte) (int, error) }, dryRun bool) error {
	t, ok := findCodexTarget()
	if !ok {
		fmt.Fprintln(out, "target: not found")
		return nil
	}
	patches, err := loadCodexPatches()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "target: %s\n", t.Path)
	if len(patches) == 0 {
		fmt.Fprintln(out, "patches: none bundled; Codex full-access behavior is configured through install-rules")
		return nil
	}
	res, err := codex.ApplyPatches(t.Path, patches, dryRun, "")
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
	t, ok := findCodexTarget()
	if !ok {
		fmt.Fprintln(out, "target: not found")
		return fmt.Errorf("Codex target not found")
	}
	fmt.Fprintf(out, "target: ok (%s)\n", t.Path)
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(out, "config: missing")
		return fmt.Errorf("Codex config missing")
	}
	text := string(data)
	if containsAll(text, []string{"approval_policy = \"never\"", "sandbox_mode = \"danger-full-access\"", "dangerously_bypass_approvals_and_sandbox = true"}) {
		fmt.Fprintln(out, "config: ok")
		return nil
	}
	return fmt.Errorf("config missing Unleash-GPT approval/sandbox/bypass settings")
}

func loadCodexPatches() ([]codex.Patch, error) {
	patches, err := loadCodexPatchesFromEmbedded()
	if err != nil {
		return nil, err
	}
	overrideDir := filepath.Join(codex.StateDir(""), "patches")
	extra, err := loadCodexPatchesFromDir(overrideDir)
	if err != nil {
		return nil, err
	}
	return mergeCodexPatches(patches, extra), nil
}

func loadCodexPatchesFromEmbedded() ([]codex.Patch, error) {
	return productcli.LoadPatchesFromFS(embeddedCodexPatches, "codex-patches")
}

func loadCodexPatchesFromDir(dir string) ([]codex.Patch, error) {
	return productcli.LoadPatchesFromDir(dir)
}

func mergeCodexPatches(base, extra []codex.Patch) []codex.Patch {
	return productcli.MergePatches(base, extra)
}

func containsAll(text string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}
