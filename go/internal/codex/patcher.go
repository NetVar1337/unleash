package codex

import (
	"os"
	"path/filepath"

	"github.com/VoidChecksum/unleash/internal/bytepatch"
)

type Patch = bytepatch.Patch
type SubPatch = bytepatch.SubPatch
type PatchResult = bytepatch.Result

func ApplyPatches(path string, patchList []Patch, dryRun bool, home string) (PatchResult, error) {
	return bytepatch.ApplyFile(path, patchList, dryRun, BackupDir(home), "."+filepath.Base(path)+".unleash-gpt-*", 0)
}

func RestoreLatestBackup(targetPath string, home string) (string, error) {
	backup, err := latestBackup(home)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(backup)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(targetPath)
	mode := os.FileMode(0o755)
	if err == nil {
		mode = info.Mode()
	}
	return backup, os.WriteFile(targetPath, data, mode)
}
