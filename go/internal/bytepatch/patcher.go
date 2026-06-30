package bytepatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

type Result struct {
	DryRun     bool
	Applied    int
	Skipped    int
	BackupPath string
	Messages   []string
}

func ApplyToBytes(data []byte, patchList []Patch) (Result, error) {
	result := Result{}
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
			if sub.AppliedMarker != "" && bytes.Contains(data, []byte(sub.AppliedMarker)) {
				result.Skipped++
				continue
			}
			padded := append([]byte(nil), replace...)
			if len(padded) < len(search) {
				padded = append(padded, bytes.Repeat([]byte(" "), len(search)-len(padded))...)
			}
			changed := ReplaceBytes(data, search, padded, sub.Count)
			if changed == 0 {
				result.Skipped++
				continue
			}
			result.Applied += changed
			result.Messages = append(result.Messages, fmt.Sprintf("%s: %d replacement(s)", patch.ID, changed))
		}
	}
	return result, nil
}

func ApplyFile(path string, patchList []Patch, dryRun bool, backupDir, tempPattern string, backupMode os.FileMode) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	updated := append([]byte(nil), data...)
	result, err := ApplyToBytes(updated, patchList)
	if err != nil {
		return result, err
	}
	result.DryRun = dryRun
	if dryRun || result.Applied == 0 {
		return result, nil
	}
	backupPath, err := backupFile(path, backupDir, backupMode)
	if err != nil {
		return result, err
	}
	result.BackupPath = backupPath
	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), tempPattern)
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

func ReplaceBytes(data []byte, search []byte, replace []byte, count int) int {
	changed := 0
	pos := 0
	if count <= 0 {
		count = -1
	}
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

func backupFile(path, backupDir string, mode os.FileMode) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if mode == 0 {
		if info, err := os.Stat(path); err == nil {
			mode = info.Mode()
		} else {
			mode = 0o644
		}
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	sha := sha256.Sum256(data)
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	backup := filepath.Join(backupDir, fmt.Sprintf("%s.%s.%s.bak", stem, stamp, hex.EncodeToString(sha[:])[:12]))
	return backup, os.WriteFile(backup, data, mode)
}
