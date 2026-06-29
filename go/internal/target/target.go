// Package target implements Claude Code binary discovery, SHA256 hashing, and backup.
package target

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	pkg        = "@anthropic-ai/claude-code"
	minBinSize = 1_000_000 // 1 MB — SEA binaries are >100 MB
)

// check holds a suffix path and its kind.
type check struct {
	suffix string
	kind   string
}

// FindTarget returns (path, kind) where kind is "js" or "bun_sea".
// It enumerates every known CC packaging: legacy cli.js, Linux/macOS/Windows SEA.
// Returns ("", "") if nothing found.
func FindTarget() (string, string) {
	jsChecks := []check{
		{pkg + "/cli.js", "js"},
	}
	subChecks := []check{
		{pkg + "/node_modules/@anthropic-ai/claude-code-linux-x64/claude", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-linux-arm64/claude", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-linux-x64-musl/claude", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-linux-arm64-musl/claude", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-darwin-x64/claude", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-darwin-arm64/claude", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-win32-x64/claude.exe", "bun_sea"},
		{pkg + "/node_modules/@anthropic-ai/claude-code-win32-arm64/claude.exe", "bun_sea"},
	}
	binChecks := []check{
		{pkg + "/bin/claude.exe", "bun_sea"},
		{pkg + "/bin/claude", "bun_sea"},
	}
	checks := append(jsChecks, append(subChecks, binChecks...)...)

	// 1. npm global roots
	for _, root := range npmGlobalRoots() {
		if p, k := probeChecks(root, checks); p != "" {
			return p, k
		}
	}

	home, _ := os.UserHomeDir()

	// 2. Version-managed Node installs (mise, nvm, fnm, APPDATA/npm)
	miseBase := filepath.Join(home, ".local", "share", "mise", "installs", "node")
	nvmDir := os.Getenv("NVM_DIR")
	if nvmDir == "" {
		nvmDir = filepath.Join(home, ".nvm")
	}
	nvmBase := filepath.Join(nvmDir, "versions", "node")
	fnmDir := os.Getenv("FNM_DIR")
	if fnmDir == "" {
		fnmDir = filepath.Join(home, ".local", "share", "fnm")
	}
	fnmBase := filepath.Join(fnmDir, "node-versions")

	extraBases := []string{miseBase, nvmBase, fnmBase}

	appdata := os.Getenv("APPDATA")
	if appdata != "" {
		extraBases = append(extraBases,
			filepath.Join(appdata, "npm", "node_modules"),
			filepath.Join(appdata, "npm"),
		)
	}

	for _, base := range extraBases {
		for _, c := range checks {
			candidates := versionGlob(base, c.suffix)
			for _, p := range candidates {
				if validTarget(p, c.kind) {
					return resolvePath(p), c.kind
				}
			}
		}
	}

	// 3. Bun global
	bunGlobal := filepath.Join(home, ".bun", "install", "global", "node_modules")
	if p, k := probeChecks(bunGlobal, checks); p != "" {
		return p, k
	}

	// 4. pnpm global
	pnpmRoot := pnpmGlobalRoot()
	if pnpmRoot != "" {
		if p, k := probeChecks(pnpmRoot, checks); p != "" {
			return p, k
		}
	}

	// 5. Volta
	voltaBase := filepath.Join(home, ".volta", "tools", "image", "packages")
	if isDir(voltaBase) {
		for _, c := range checks {
			pattern := filepath.Join(voltaBase, pkg, "*", "node_modules", c.suffix)
			matches, _ := filepath.Glob(pattern)
			for _, p := range matches {
				if validTarget(p, c.kind) {
					return resolvePath(p), c.kind
				}
			}
		}
	}

	// 6. Homebrew (formula node_modules + cask Caskroom)
	for _, hb := range []string{
		"/opt/homebrew/lib/node_modules",
		"/usr/local/lib/node_modules",
	} {
		if p, k := probeChecks(hb, checks); p != "" {
			return p, k
		}
	}
	// Homebrew cask: /opt/homebrew/Caskroom/claude-code/<ver>/claude or /usr/local/Caskroom/...
	for _, caskBase := range []string{
		"/opt/homebrew/Caskroom/claude-code",
		"/usr/local/Caskroom/claude-code",
		"/opt/homebrew/Caskroom/claude-code@latest",
		"/usr/local/Caskroom/claude-code@latest",
	} {
		if isDir(caskBase) {
			entries := sortedDirEntriesDesc(caskBase)
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				for _, bin := range []string{"claude", "bin/claude"} {
					p := filepath.Join(caskBase, e.Name(), bin)
					if fileExists(p) && fileSize(p) > minBinSize {
						return resolvePath(p), "bun_sea"
					}
				}
			}
		}
	}

	// 7. Native CLI installer (~/.local/share/claude/versions/<ver>)
	versionsDir := filepath.Join(home, ".local", "share", "claude", "versions")
	if isDir(versionsDir) {
		entries := sortedDirEntriesDesc(versionsDir)
		for _, e := range entries {
			p := filepath.Join(versionsDir, e.Name())
			if !e.IsDir() && fileSize(p) > minBinSize {
				return resolvePath(p), "bun_sea"
			}
		}
	}

	// 8. Native installer + system package (apt/dnf/apk) candidate paths
	nativeCandidates := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, ".local", "bin", "claude.exe"),
		filepath.Join(home, ".claude", "local", "claude"),
		filepath.Join(home, ".claude", "local", "claude.exe"),
		filepath.Join(home, ".claude", "bin", "claude"),
		filepath.Join(home, ".claude", "bin", "claude.exe"),
		filepath.Join(home, ".local", "share", "claude-code", "claude"),
		filepath.Join(home, ".local", "share", "claude-code", "claude.exe"),
		filepath.Join(home, ".local", "bin", "claude-code"),
		filepath.Join(home, ".local", "bin", "claude-code.exe"),
		// System package manager installs (apt, dnf, apk)
		"/usr/bin/claude",
		"/usr/local/bin/claude",
		"/usr/local/share/claude-code/claude",
		"/opt/claude-code/bin/claude",
		// Homebrew bin symlinks
		"/opt/homebrew/bin/claude",
	}
	for _, p := range nativeCandidates {
		if isDir(p) {
			// Directory — look for claude-* binaries inside.
			entries := sortedDirEntriesDesc(p)
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), "claude-") && !e.IsDir() {
					cp := filepath.Join(p, e.Name())
					if fileSize(cp) > minBinSize {
						return resolvePath(cp), "bun_sea"
					}
				}
			}
			continue
		}
		if fileExists(p) && fileSize(p) > minBinSize {
			return resolvePath(p), "bun_sea"
		}
	}

	// 9. Windows %LOCALAPPDATA% paths
	la := os.Getenv("LOCALAPPDATA")
	if la != "" {
		for _, suffix := range []string{
			"Programs/claude-code/claude.exe",
			"Programs/claude/versions",
			"claude-code/claude.exe",
			"anthropic/claude-code/claude.exe",
		} {
			p := filepath.Join(la, filepath.FromSlash(suffix))
			if isDir(p) {
				// e.g. Programs/claude/versions/* — pick newest binary
				entries := sortedDirEntriesDesc(p)
				for _, e := range entries {
					cp := filepath.Join(p, e.Name())
					if !e.IsDir() && fileSize(cp) > minBinSize {
						return resolvePath(cp), "bun_sea"
					}
				}
			} else if fileExists(p) && fileSize(p) > minBinSize {
				return resolvePath(p), "bun_sea"
			}
		}

		// winget
		wingetDir := filepath.Join(la, "Microsoft", "WinGet", "Packages")
		if isDir(wingetDir) {
			entries, _ := os.ReadDir(wingetDir)
			for _, e := range entries {
				if !e.IsDir() || !strings.Contains(strings.ToLower(e.Name()), "claude") {
					continue
				}
				pkgDir := filepath.Join(wingetDir, e.Name())
				if found := rglob(pkgDir, "claude.exe"); found != "" {
					return resolvePath(found), "bun_sea"
				}
			}
		}
	}

	// 10. Scoop (Windows)
	scoopDir := filepath.Join(home, "scoop", "apps", "claude-code", "current")
	if isDir(scoopDir) {
		for _, name := range []string{"claude.exe", "claude"} {
			p := filepath.Join(scoopDir, name)
			if fileExists(p) && fileSize(p) > minBinSize {
				return resolvePath(p), "bun_sea"
			}
		}
	}

	// 11. Chocolatey (Windows)
	chocoDir := "C:/ProgramData/chocolatey/lib/claude-code"
	if isDir(chocoDir) {
		if found := rglob(chocoDir, "claude.exe"); found != "" {
			return resolvePath(found), "bun_sea"
		}
	}

	// 12. Last resort: os/exec LookPath (cross-platform which)
	for _, name := range []string{"claude", "claude.exe"} {
		if found, err := exec.LookPath(name); err == nil {
			p := resolvePath(found)
			if fileExists(p) && fileSize(p) > minBinSize {
				return p, "bun_sea"
			}
		}
	}

	return "", ""
}

// SHA256Short returns the first 12 hex chars of the SHA-256 of the file at path.
func SHA256Short(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:12]
}

// BackupDir is ~/.unleash/backups/
func BackupDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".unleash", "backups")
}

// Backup copies the target binary to ~/.unleash/backups/ with a timestamp and
// SHA256 short hash. Keeps only the last 10 backups. Returns the backup path.
func Backup(target string, kind string) (string, error) {
	dir := BackupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating backup dir: %w", err)
	}

	stamp := time.Now().Format("20060102-150405")
	ext := "js.bak"
	if kind == "bun_sea" {
		ext = "exe.bak"
	}
	sha := SHA256Short(target)
	dst := filepath.Join(dir, fmt.Sprintf("claude.%s.%s.%s", stamp, sha, ext))

	// Copy file
	src, err := os.Open(target)
	if err != nil {
		return "", err
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return "", err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}

	// Prune old backups — keep last 10
	pruneBackups(dir, 10)

	return dst, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func npmGlobalRoots() []string {
	var roots []string

	// On Windows the npm shim is npm.cmd (or npm.exe), not bare npm.
	candidates := []string{"npm"}
	if runtime.GOOS == "windows" {
		candidates = []string{"npm.cmd", "npm.exe", "npm"}
	}
	for _, npm := range candidates {
		out, err := exec.Command(npm, "root", "-g").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			if s != "" {
				roots = append([]string{s}, roots...)
				break
			}
		}
	}

	home, _ := os.UserHomeDir()
	roots = append(roots,
		filepath.Join(home, ".npm-global", "lib", "node_modules"),
		filepath.Join(home, ".local", "lib", "node_modules"),
		"/usr/local/lib/node_modules",
		"/usr/lib/node_modules",
	)
	return roots
}

func versionGlob(base, suffix string) []string {
	pattern := filepath.Join(base, "*", "lib", "node_modules", filepath.FromSlash(suffix))
	matches, _ := filepath.Glob(pattern)
	return matches
}

func pnpmGlobalRoot() string {
	out, err := exec.Command("pnpm", "root", "-g").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		if s != "" {
			return s
		}
	}
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, ".local", "share", "pnpm", "global", "5", "node_modules")
	if isDir(fallback) {
		return fallback
	}
	return ""
}

func probeChecks(root string, checks []check) (string, string) {
	for _, c := range checks {
		p := filepath.Join(root, filepath.FromSlash(c.suffix))
		if validTarget(p, c.kind) {
			return resolvePath(p), c.kind
		}
	}
	return "", ""
}

func validTarget(path, kind string) bool {
	if !fileExists(path) {
		return false
	}
	if kind == "js" {
		return true
	}
	return fileSize(path) > minBinSize
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}
	return abs
}

// sortedDirEntriesDesc reads a directory and returns entries sorted by name descending.
func sortedDirEntriesDesc(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})
	return entries
}

// rglob recursively finds a file by name under a directory.
// Returns the first match with size > minBinSize, or "".
func rglob(dir, name string) string {
	var result string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !d.IsDir() && d.Name() == name && fileSize(path) > minBinSize {
			result = path
			return filepath.SkipAll
		}
		return nil
	})
	return result
}

// pruneBackups keeps only the newest n backups matching claude.*.bak.
func pruneBackups(dir string, keep int) {
	pattern := filepath.Join(dir, "claude.*.bak")
	matches, _ := filepath.Glob(pattern)
	if len(matches) <= keep {
		return
	}
	sort.Strings(matches) // oldest first (timestamp-based names)
	for _, old := range matches[:len(matches)-keep] {
		os.Remove(old)
	}
}
