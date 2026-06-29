package omp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPatchesDryRunDoesNotWrite(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "cli.js")
	writeOMPTextFile(t, target, "https://qa.omp.sh/v1/grievances")

	res, err := ApplyPatches(target, []Patch{{ID: "telemetry", Patches: []SubPatch{{Search: "https://qa.omp.sh/v1/grievances", Replace: "http://127.0.0.1:9/xxxxxxxxxxx"}}}}, true, home)
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied != 1 || !res.DryRun {
		t.Fatalf("unexpected result: %#v", res)
	}
	assertOMPFileContains(t, target, "https://qa.omp.sh/v1/grievances")
}

func TestApplyPatchesWritesBackup(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "cli.js")
	writeOMPTextFile(t, target, "mode=write")

	res, err := ApplyPatches(target, []Patch{{ID: "approval", Patches: []SubPatch{{Search: "write", Replace: "yolo"}}}}, false, home)
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied != 1 || res.BackupPath == "" {
		t.Fatalf("unexpected result: %#v", res)
	}
	assertOMPFileContains(t, target, "mode=yolo")
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Fatalf("backup not created: %v", err)
	}
}

func TestApplyPatchesRejectsLongerReplacement(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "cli.js")
	writeOMPTextFile(t, target, "mode=yolo")

	_, err := ApplyPatches(target, []Patch{{ID: "bad", Patches: []SubPatch{{Search: "yolo", Replace: "always-ask"}}}}, false, home)
	if err == nil {
		t.Fatal("expected longer replacement error")
	}
	assertOMPFileContains(t, target, "mode=yolo")
}

func writeOMPTextFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
