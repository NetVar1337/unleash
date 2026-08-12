package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	vpccembed "github.com/VoidChecksum/unleash/embed"
	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/patches"
	"github.com/VoidChecksum/unleash/internal/updater"
)

// ToolVersion is the unleash tool version, injected from main via SetVersion.
var ToolVersion = "1.1.0"

// SetVersion overrides ToolVersion (called from main with the build version).
func SetVersion(v string) {
	if v != "" && v != "dev" {
		ToolVersion = v
	}
}

// ── string helpers ──────────────────────────────────────────────────────────

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func shortSHA(s string) string {
	if len(s) <= 7 {
		return s
	}
	return s[:7]
}

func updateWarning(info updater.UpstreamInfo, command string) string {
	if info.RemoteCommit == "" {
		return ""
	}
	if info.StateMissing {
		return fmt.Sprintf("%s%s update status unknown — run '%s' to sync%s",
			console.Y, console.WARN, command, console.X)
	}
	if info.UpdateAvailable {
		return fmt.Sprintf("%s%s update available: %s %s %s — run '%s'%s",
			console.Y, console.WARN, shortSHA(info.LocalCommit), console.ARROW,
			shortSHA(info.RemoteCommit), command, console.X)
	}
	return ""
}

func hexInt(v int) string {
	return fmt.Sprintf("0x%x", v)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── platform helpers ────────────────────────────────────────────────────────

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(homeDir(), path[2:])
	}
	if path == "~" {
		return homeDir()
	}
	return os.ExpandEnv(path)
}

func unleashDir() string {
	return filepath.Join(homeDir(), ".unleash")
}

// patchDir returns the override patch directory (~/.unleash/patches/),
// falling back to a tmp dir if needed. The loader checks override dir first.
func patchDir() string {
	return filepath.Join(unleashDir(), "patches")
}

func preloadDirPath() string {
	if isWindows() {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homeDir(), "AppData", "Local")
		}
		return filepath.Join(localAppData, "void-patcher")
	}
	return filepath.Join(homeDir(), ".local", "share", "void-patcher")
}

// ── patch loader wrappers ───────────────────────────────────────────────────

// loadAllPatches loads patches from the override dir first, then embedded.
func loadAllPatches() []patches.Patch {
	dir := patchDir()
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		if p, err := patches.LoadPatches(dir); err == nil && len(p) > 0 {
			return p
		}
	}
	p, _ := patches.LoadPatchesFromEmbed()
	return p
}

// countRetired counts retired patch JSON files in a directory.
func countRetired(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			if patches.IsRetired(filepath.Join(dir, e.Name())) {
				n++
			}
		}
	}
	return n
}

// countBackups counts backup files in a directory matching claude.*.bak.
func countBackups(dir string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, "claude.*.bak"))
	return len(matches)
}

// ── embedded contrib file access ────────────────────────────────────────────

// getContribFile reads a file from the embedded contrib/ filesystem.
func getContribFile(relPath string) ([]byte, error) {
	return fs.ReadFile(vpccembed.EmbeddedContrib, filepath.ToSlash(filepath.Join("contrib", relPath)))
}

// ── CC version detection ────────────────────────────────────────────────────

func detectCCVersion(targetPath string) string {
	cmd := exec.Command(targetPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return filepath.Base(targetPath)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
			return strings.Fields(line)[0]
		}
	}
	return filepath.Base(targetPath)
}

// ── JSON helpers ────────────────────────────────────────────────────────────

func jsonEqual(a, b interface{}) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

// setNested writes val into m at a dotted path (e.g. "env.FOO"), creating
// intermediate maps. Returns true when the value actually changed.
func setNested(m map[string]interface{}, dotted string, val interface{}) bool {
	parts := strings.Split(dotted, ".")
	cur := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			cur[p] = next
		}
		cur = next
	}
	leaf := parts[len(parts)-1]
	if existing, ok := cur[leaf]; ok && jsonEqual(existing, val) {
		return false
	}
	cur[leaf] = val
	return true
}

// unsetNested removes the value at a dotted path. Returns true when something
// was removed.
func unsetNested(m map[string]interface{}, dotted string) bool {
	parts := strings.Split(dotted, ".")
	cur := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			return false
		}
		cur = next
	}
	leaf := parts[len(parts)-1]
	if _, ok := cur[leaf]; !ok {
		return false
	}
	delete(cur, leaf)
	return true
}

// nestedEqual reports whether the value stored at dotted path equals val.
func nestedEqual(m map[string]interface{}, dotted string, val interface{}) bool {
	parts := strings.Split(dotted, ".")
	var cur interface{} = m
	for _, p := range parts {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return false
		}
		cur, ok = mm[p]
		if !ok {
			return false
		}
	}
	return jsonEqual(cur, val)
}

// stateString extracts a string value from a state map.
func stateString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
