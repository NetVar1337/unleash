package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/codex"
)

var version = "dev"

//go:embed codex-patches/*.json
var embeddedCodexPatches embed.FS

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
			t, ok := codex.FindTarget()
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
			if t, ok := codex.FindTarget(); ok {
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
			t, ok := codex.FindTarget()
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
	t, ok := codex.FindTarget()
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
	if t, ok := codex.FindTarget(); ok {
		fmt.Fprintf(out, "target: ok (%s)\n", t.Path)
	} else {
		fmt.Fprintln(out, "target: not found")
	}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(out, "config: missing")
		return nil
	}
	text := string(data)
	if containsAll(text, []string{"approval_policy = \"never\"", "sandbox_mode = \"danger-full-access\""}) {
		fmt.Fprintln(out, "config: ok")
		return nil
	}
	return fmt.Errorf("config missing Unleash-GPT approval/sandbox settings")
}

func loadCodexPatches() ([]codex.Patch, error) {
	entries, err := embeddedCodexPatches.ReadDir("codex-patches")
	if err != nil {
		return nil, err
	}
	var out []codex.Patch
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := embeddedCodexPatches.ReadFile("codex-patches/" + e.Name())
		if err != nil {
			return nil, err
		}
		var p codex.Patch
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func containsAll(text string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}
