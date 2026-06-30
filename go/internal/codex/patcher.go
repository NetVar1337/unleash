package codex

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Patch struct {
	ID      string     `json:"id"`
	Patches []SubPatch `json:"patches"`
}

type SubPatch struct {
	Search        string `json:"search,omitempty"`
	Replace       string `json:"replace"`
	AppliedMarker string `json:"applied_marker,omitempty"`
	Count         int    `json:"count,omitempty"`
}

type PatchResult struct {
	DryRun     bool
	Applied    int
	Skipped    int
	BackupPath string
	Messages   []string
}

func ApplyPatches(path string, patchList []Patch, dryRun bool, home string) (PatchResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PatchResult{}, err
	}
	result := PatchResult{DryRun: dryRun}
	updated := append([]byte(nil), data...)

	for _, patch := range patchList {
		for _, sub := range patch.Patches {
			if sub.Search == "" {
				result.Skipped++
				continue
			}
			search := []byte(sub.Search)
			replace := []byte(sub.Replace)
			if len(replace) > len(search) {
				return result, fmt.Errorf("%s replacement is longer than search (%d > %d)", patch.ID, len(replace), len(search))
			}
			if sub.AppliedMarker != "" && bytes.Contains(updated, []byte(sub.AppliedMarker)) {
				result.Skipped++
				continue
			}
			padded := append([]byte(nil), replace...)
			if len(padded) < len(search) {
				padded = append(padded, bytes.Repeat([]byte(" "), len(search)-len(padded))...)
			}
			count := sub.Count
			if count <= 0 {
				count = -1
			}
			changed := replaceBytes(updated, search, padded, count)
			if changed == 0 {
				result.Skipped++
				continue
			}
			result.Applied += changed
			result.Messages = append(result.Messages, fmt.Sprintf("%s: %d replacement(s)", patch.ID, changed))
		}
	}

	if dryRun || result.Applied == 0 {
		return result, nil
	}

	backupPath, err := backupFile(path, home)
	if err != nil {
		return result, err
	}
	result.BackupPath = backupPath

	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".unleash-gpt-*")
	if err != nil {
		return result, err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(updated); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return result, err
	}
	if err := tmp.Chmod(info.Mode()); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return result, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return result, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return result, err
	}
	return result, nil
}

func replaceBytes(data []byte, search []byte, replace []byte, count int) int {
	changed := 0
	pos := 0
	for count != 0 {
		idx := bytes.Index(data[pos:], search)
		if idx < 0 {
			break
		}
		start := pos + idx
		copy(data[start:start+len(search)], replace)
		changed++
		pos = start + len(search)
		if count > 0 {
			count--
		}
	}
	return changed
}

func backupFile(path string, home string) (string, error) {
	dir := BackupDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := filepath.Base(path)
	stamp := time.Now().UTC().Format("20060102-150405")
	backup := filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+"."+stamp+"."+SHA256Short(path)+".bak")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return backup, os.WriteFile(backup, data, 0o644)
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
