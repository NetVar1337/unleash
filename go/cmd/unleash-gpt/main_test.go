package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpListsUnleashGPTCommands(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Unleash-GPT", "setup", "patch", "status", "verify", "install-rules", "uninstall-rules", "rollback"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q in:\n%s", want, out)
		}
	}
}

func TestBundledCodexPatchesCoverPolicyAndTelemetry(t *testing.T) {
	patches, err := loadCodexPatches()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	total := 0
	for _, p := range patches {
		ids[p.ID] = true
		total += len(p.Patches)
	}
	for _, want := range []string{
		"codex-telemetry-endpoints-localhost",
		"codex-policy-prompts-neutralized",
		"codex-approval-sandbox-errors-neutralized",
	} {
		if !ids[want] {
			t.Fatalf("missing bundled patch %q in %#v", want, ids)
		}
	}
	if total < 6 {
		t.Fatalf("bundled Codex subpatches = %d, want at least 6", total)
	}
}

func TestVerifyRequiresCodexBypassFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(codexDir, "config.toml"),
		[]byte("approval_policy = \"never\"\nsandbox_mode = \"danger-full-access\"\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := runVerify(new(bytes.Buffer)); err == nil {
		t.Fatal("verify accepted config without dangerously_bypass_approvals_and_sandbox")
	}
}
