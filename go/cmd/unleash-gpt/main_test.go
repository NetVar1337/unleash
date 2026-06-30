package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VoidChecksum/unleash/internal/codex"
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

func TestCodexPatchDirMergesAndOverridesBundledPatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "01-extra.json"), []byte(`{"id":"extra-codex","patches":[{"search":"a","replace":"b"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "02-override.json"), []byte(`{"id":"bundled","patches":[{"search":"new","replace":"old"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	base := []codex.Patch{{ID: "bundled", Patches: []codex.SubPatch{{Search: "old", Replace: "new"}}}}
	extra, err := loadCodexPatchesFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeCodexPatches(base, extra)
	ids := map[string]codex.Patch{}
	for _, p := range merged {
		ids[p.ID] = p
	}
	if _, ok := ids["extra-codex"]; !ok {
		t.Fatalf("extra patch missing from merged set: %#v", ids)
	}
	if ids["bundled"].Patches[0].Search != "new" {
		t.Fatalf("override did not replace bundled patch: %#v", ids["bundled"])
	}
}

func TestVerifyRequiresCodexTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldFind := findCodexTarget
	findCodexTarget = func() (codex.Target, bool) { return codex.Target{}, false }
	t.Cleanup(func() { findCodexTarget = oldFind })
	if err := os.WriteFile(
		filepath.Join(codexDir, "config.toml"),
		[]byte("approval_policy = \"never\"\nsandbox_mode = \"danger-full-access\"\ndangerously_bypass_approvals_and_sandbox = true\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := runVerify(new(bytes.Buffer)); err == nil {
		t.Fatal("verify accepted valid config without a Codex target")
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
	targetPath := filepath.Join(home, ".local", "bin", "codex")
	writeLargeFile(t, targetPath, 1_000_001)

	if err := runVerify(new(bytes.Buffer)); err == nil {
		t.Fatal("verify accepted config without dangerously_bypass_approvals_and_sandbox")
	}
}

func writeLargeFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
