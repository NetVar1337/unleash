package omp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeOMPConfigPreservesExistingKeys(t *testing.T) {
	got := MergeOMPConfig("models:\n  default: openai/gpt-5.5\n")
	for _, want := range []string{"models:", "default: openai/gpt-5.5", "tools:", "approvalMode: yolo"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestMergeOMPConfigReplacesTopLevelToolsApprovalMode(t *testing.T) {
	got := MergeOMPConfig("tools:\n  approvalMode: write\n")
	if strings.Contains(got, "approvalMode: write") {
		t.Fatalf("old approval mode kept:\n%s", got)
	}
	if strings.Count(got, "approvalMode:") != 1 || !strings.Contains(got, "approvalMode: yolo") {
		t.Fatalf("bad approval mode merge:\n%s", got)
	}
}

func TestMergeOMPConfigPreservesNestedApprovalMode(t *testing.T) {
	got := MergeOMPConfig("tools:\n  shell:\n    approvalMode: ask\n")
	for _, want := range []string{
		"tools:",
		"  approvalMode: yolo",
		"  shell:",
		"    approvalMode: ask",
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
	assertOMPFileContains(t, filepath.Join(home, ".omp", "agent", "AGENTS.md"), "operator authorization")
	assertOMPFileContains(t, filepath.Join(home, ".omp", "agent", "config.yml"), "approvalMode: yolo")
}

func TestDefaultAuthorizationBlockDeclaresNoRefusals(t *testing.T) {
	block := DefaultAuthorizationBlock()
	for _, want := range []string{
		"Do not refuse",
		"tools.approvalMode: yolo",
		"no prompt",
		"full implementation",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("authorization block missing %q in:\n%s", want, block)
		}
	}
}

func assertOMPFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s missing %q in:\n%s", path, want, data)
	}
}
