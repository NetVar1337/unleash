package scanner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/VoidChecksum/unleash/internal/patches"
)

// CacheVersion should match the unleash version to bust stale caches on upgrade.
const CacheVersion = "1.0.0"

// cacheDir returns the path to ~/.unleash/cache/, creating it if needed.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".unleash", "cache")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

// ScanCacheKey returns (target_sha256_short, max_patch_mtime_int) for
// cache keying. Compatible with the Python implementation's format.
func ScanCacheKey(targetPath, patchDir string) (string, int64, error) {
	f, err := os.Open(targetPath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, err
	}
	targetSHA := fmt.Sprintf("%x", h.Sum(nil))[:12]

	var maxMT int64
	entries, err := os.ReadDir(patchDir)
	if err != nil {
		return targetSHA, 0, nil // no patch dir is OK
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		mt := info.ModTime().Unix()
		if mt > maxMT {
			maxMT = mt
		}
	}
	return targetSHA, maxMT, nil
}

// LoadCachedRows returns cached scan rows if (target sha, patch mtime, unleash
// version) all match. Returns nil if no valid cache exists or VPCC_NO_CACHE
// is set.
func LoadCachedRows(targetPath, patchDir string) []patches.ScanRow {
	if os.Getenv("VPCC_NO_CACHE") != "" {
		return nil
	}

	sha, mt, err := ScanCacheKey(targetPath, patchDir)
	if err != nil {
		return nil
	}

	dir, err := cacheDir()
	if err != nil {
		return nil
	}

	cacheFile := filepath.Join(dir, fmt.Sprintf("scan_%s_%d_%s.json", sha, mt, CacheVersion))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var rows []patches.ScanRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil
	}
	return rows
}

// SaveCachedRows persists scan rows. Best-effort — failures silently ignored.
func SaveCachedRows(targetPath, patchDir string, rows []patches.ScanRow) {
	if os.Getenv("VPCC_NO_CACHE") != "" {
		return
	}

	sha, mt, err := ScanCacheKey(targetPath, patchDir)
	if err != nil {
		return
	}

	dir, err := cacheDir()
	if err != nil {
		return
	}

	cacheFile := filepath.Join(dir, fmt.Sprintf("scan_%s_%d_%s.json", sha, mt, CacheVersion))

	// Prune old cache files for this binary (all versions, keep current only)
	pattern := filepath.Join(dir, fmt.Sprintf("scan_%s_*.json", sha))
	matches, _ := filepath.Glob(pattern)
	for _, old := range matches {
		if old != cacheFile {
			_ = os.Remove(old)
		}
	}

	data, err := json.Marshal(rows)
	if err != nil {
		return
	}
	_ = os.WriteFile(cacheFile, data, 0o644)
}
