package bytepatch

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestApplyToBytesHonorsCount(t *testing.T) {
	data := []byte("aa aa aa")
	res, err := ApplyToBytes(data, []Patch{{ID: "limited", Patches: []SubPatch{{Search: "aa", Replace: "bb", Count: 1}}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "bb aa aa" {
		t.Fatalf("patched data = %q, want first occurrence only", got)
	}
	if res.Applied != 1 || res.Skipped != 0 {
		t.Fatalf("result = applied %d skipped %d, want 1/0", res.Applied, res.Skipped)
	}
}

func TestApplyToBytesZeroCountReplacesAll(t *testing.T) {
	data := []byte("aa aa aa")
	res, err := ApplyToBytes(data, []Patch{{ID: "all", Patches: []SubPatch{{Search: "aa", Replace: "b", Count: 0}}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "b  b  b " {
		t.Fatalf("patched data = %q, want all occurrences padded in place", got)
	}
	if res.Applied != 3 {
		t.Fatalf("applied = %d, want 3", res.Applied)
	}
}

func TestApplyFilePreservesBackupMode(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := ApplyFile(path, []Patch{{ID: "mode", Patches: []SubPatch{{Search: "old", Replace: "new"}}}}, false, filepath.Join(home, "backups"), ".tool-*", 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if res.BackupPath == "" {
		t.Fatal("missing backup path")
	}
	info, err := os.Stat(res.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("NTFS does not preserve Unix permission bits")
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("backup mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestApplyFileBackupNameSortsAfterOldTimestampFormat(t *testing.T) {
	backupDir := filepath.Join(t.TempDir(), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldName := "tool.20000101-000000.abc123.bak"
	if err := os.WriteFile(filepath.Join(backupDir, oldName), []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := ApplyFile(path, []Patch{{ID: "backup-sort", Patches: []SubPatch{{Search: "old", Replace: "new"}}}}, false, backupDir, ".tool-*", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(res.BackupPath) <= oldName {
		t.Fatalf("new backup %q should sort after old backup %q", filepath.Base(res.BackupPath), oldName)
	}
}
