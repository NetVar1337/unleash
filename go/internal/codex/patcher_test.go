package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatchesDryRunDoesNotWrite(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "codex.bin")
	writeTextFile(t, target, "hello on-request world")

	res, err := ApplyPatches(target, []Patch{{ID: "approval", Patches: []SubPatch{{Search: "on-request", Replace: "never"}}}}, true, home)
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied != 1 || !res.DryRun {
		t.Fatalf("unexpected result: %#v", res)
	}
	assertFileContains(t, target, "on-request")
}

func TestApplyPatchesPadsShorterReplacement(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "codex.bin")
	writeTextFile(t, target, "approval=on-request;")

	res, err := ApplyPatches(target, []Patch{{ID: "approval", Patches: []SubPatch{{Search: "on-request", Replace: "never"}}}}, false, home)
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied != 1 || res.BackupPath == "" {
		t.Fatalf("unexpected result: %#v", res)
	}
	assertFileContains(t, target, "approval=never     ;")
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Fatalf("backup not created: %v", err)
	}
}

func TestApplyPatchesRejectsLongerReplacementWithoutWriting(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "codex.bin")
	writeTextFile(t, target, "mode=never")

	_, err := ApplyPatches(target, []Patch{{ID: "bad", Patches: []SubPatch{{Search: "never", Replace: "danger-full-access"}}}}, false, home)
	if err == nil {
		t.Fatal("expected longer replacement error")
	}
	assertFileContains(t, target, "mode=never")
}

func TestApplyPatchesHonorsCount(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "codex.bin")
	writeTextFile(t, target, "on-request on-request")

	res, err := ApplyPatches(target, []Patch{{ID: "approval", Patches: []SubPatch{{Search: "on-request", Replace: "never", Count: 1}}}}, false, home)
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied != 1 {
		t.Fatalf("Applied = %d, want 1", res.Applied)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "never") != 1 || strings.Count(string(data), "on-request") != 1 {
		t.Fatalf("bad count replacement: %q", data)
	}
}

func writeTextFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
