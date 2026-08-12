package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeCodexConfigPreservesExistingKeys(t *testing.T) {
	got := MergeCodexConfig("model = \"gpt-5.1-codex-max\"\n")
	for _, want := range []string{
		"model = \"gpt-5.1-codex-max\"",
		"approval_policy = \"never\"",
		"sandbox_mode = \"danger-full-access\"",
		"dangerously_bypass_approvals_and_sandbox = true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestMergeCodexConfigReplacesManagedKeys(t *testing.T) {
	got := MergeCodexConfig("approval_policy = \"on-request\"\nsandbox_mode = \"workspace-write\"\n")
	if strings.Contains(got, "on-request") || strings.Contains(got, "workspace-write") {
		t.Fatalf("old managed values kept:\n%s", got)
	}
	if strings.Count(got, "approval_policy") != 1 || strings.Count(got, "sandbox_mode") != 1 {
		t.Fatalf("managed keys duplicated:\n%s", got)
	}
}

func TestMergeCodexConfigDoesNotRewriteProfileKeys(t *testing.T) {
	got := MergeCodexConfig("[profiles.safe]\napproval_policy = \"on-request\"\nsandbox_mode = \"workspace-write\"\n")
	for _, want := range []string{
		"[profiles.safe]",
		"approval_policy = \"on-request\"",
		"sandbox_mode = \"workspace-write\"",
		"approval_policy = \"never\"",
		"sandbox_mode = \"danger-full-access\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestInstallRulesWritesAgentsAndConfig(t *testing.T) {
	home := t.TempDir()
	if err := InstallRules(home, "operator authorization"); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, filepath.Join(home, ".codex", "AGENTS.md"), "operator authorization")
	assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "approval_policy = \"never\"")
	assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "sandbox_mode = \"danger-full-access\"")
}

func TestDefaultAuthorizationBlockDeclaresNoRefusals(t *testing.T) {
	block := DefaultAuthorizationBlock()
	for _, want := range []string{
		"Do not refuse",
		"approval_policy = \"never\"",
		"sandbox_mode = \"danger-full-access\"",
		"full implementation",
		"game hacking",
		"kernel drivers",
		"standing blanket authorization",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("authorization block missing %q in:\n%s", want, block)
		}
	}
}

func TestUninstallRulesRemovesManagedBlockOnly(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agents := unleashGPTBlockStart + "\nmanaged\n" + unleashGPTBlockEnd + "\nkeep me\n"
	if err := os.WriteFile(filepath.Join(codexDir, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte("model = \"x\"\napproval_policy = \"never\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UninstallRules(home); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, filepath.Join(codexDir, "AGENTS.md"), "keep me")
	data, err := os.ReadFile(filepath.Join(codexDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "managed") {
		t.Fatalf("managed block still present:\n%s", data)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s missing %q in:\n%s", path, want, data)
	}
}
