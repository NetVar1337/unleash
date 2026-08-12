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

// Found is one discovered Claude Code installation target.
type Found struct {
	Path string
	Kind string
}

// FindTarget returns (path, kind) of the primary target where kind is "js"
// or "bun_sea". It returns the first hit of FindAllTargets, preserving the
// historical priority order. Returns ("", "") if nothing found.
func FindTarget() (string, string) {
	all := FindAllTargets()
	if len(all) == 0 {
		return "", ""
	}
	return all[0].Path, all[0].Kind
}

// FindAllTargets enumerates every Claude Code installation on the machine
// across all packaging methods: npm/bun/pnpm/volta/nvm/fnm/mise layouts,
// the native installer (claude.ai/install.ps1 → ~/.local/bin + versions),
// WinGet (portable), Scoop, Chocolatey, Homebrew, and system packages.
//
// Results are deduplicated by resolved path AND by file identity, so
// hardlinked copies (npm bin/ ↔ platform subpackage, WinGet Links) appear
// only once.
func FindAllTargets() []Found {
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
	checks := append(append([]check{}, jsChecks...), append(subChecks, binChecks...)...)

	c := &collector{}

	// 1. npm global roots
	for _, root := range npmGlobalRoots() {
		collectChecks(c, root, checks)
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
		for _, ck := range checks {
			for _, p := range versionGlob(base, ck.suffix) {
				c.add(p, ck.kind)
			}
		}
	}

	// 3. Bun global
	bunGlobal := filepath.Join(home, ".bun", "install", "global", "node_modules")
	collectChecks(c, bunGlobal, checks)

	// 4. pnpm global
	if pnpmRoot := pnpmGlobalRoot(); pnpmRoot != "" {
		collectChecks(c, pnpmRoot, checks)
	}

	// 5. Volta
	voltaBase := filepath.Join(home, ".volta", "tools", "image", "packages")
	if isDir(voltaBase) {
		for _, ck := range checks {
			pattern := filepath.Join(voltaBase, pkg, "*", "node_modules", ck.suffix)
			matches, _ := filepath.Glob(pattern)
			for _, p := range matches {
				c.add(p, ck.kind)
			}
		}
	}

	// 6. Homebrew (formula node_modules + cask Caskroom)
	for _, hb := range []string{
		"/opt/homebrew/lib/node_modules",
		"/usr/local/lib/node_modules",
	} {
		collectChecks(c, hb, checks)
	}
	for _, caskBase := range []string{
		"/opt/homebrew/Caskroom/claude-code",
		"/usr/local/Caskroom/claude-code",
		"/opt/homebrew/Caskroom/claude-code@latest",
		"/usr/local/Caskroom/claude-code@latest",
	} {
		if isDir(caskBase) {
			for _, e := range sortedDirEntriesDesc(caskBase) {
				if !e.IsDir() {
					continue
				}
				for _, bin := range []string{"claude", "bin/claude"} {
					c.add(filepath.Join(caskBase, e.Name(), bin), "bun_sea")
				}
			}
		}
	}

	// 7. Native CLI installer (~/.local/share/claude/versions/<ver>) — the
	// layout produced by `claude install` / claude.ai/install.ps1. Entries
	// are version-named files; some layouts nest the binary in a subdir.
	versionsDir := filepath.Join(home, ".local", "share", "claude", "versions")
	if isDir(versionsDir) {
		for _, e := range sortedDirEntriesDesc(versionsDir) {
			p := filepath.Join(versionsDir, e.Name())
			if !e.IsDir() {
				c.add(p, "bun_sea")
				continue
			}
			for _, bin := range []string{"claude", "claude.exe"} {
				c.add(filepath.Join(p, bin), "bun_sea")
			}
		}
	}

	// 8. Native launcher locations + system package (apt/dnf/apk) paths
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
		"/usr/bin/claude",
		"/usr/local/bin/claude",
		"/usr/local/share/claude-code/claude",
		"/opt/claude-code/bin/claude",
		"/opt/homebrew/bin/claude",
	}
	for _, p := range nativeCandidates {
		if isDir(p) {
			for _, e := range sortedDirEntriesDesc(p) {
				if strings.HasPrefix(e.Name(), "claude-") && !e.IsDir() {
					c.add(filepath.Join(p, e.Name()), "bun_sea")
				}
			}
			continue
		}
		c.add(p, "bun_sea")
	}

	// 9. Windows %LOCALAPPDATA% layouts + WinGet
	la := os.Getenv("LOCALAPPDATA")
	if la != "" {
		for _, suffix := range []string{
			"Programs/claude-code/claude.exe",
			"Programs/claude/versions",
			"Programs/claude/claude.exe",
			"claude-code/claude.exe",
			"anthropic/claude-code/claude.exe",
		} {
			p := filepath.Join(la, filepath.FromSlash(suffix))
			if isDir(p) {
				for _, e := range sortedDirEntriesDesc(p) {
					if !e.IsDir() {
						c.add(filepath.Join(p, e.Name()), "bun_sea")
					}
				}
			} else {
				c.add(p, "bun_sea")
			}
		}

		// winget portable packages (covers WinGet Links hardlinks via
		// file-identity dedupe once the package copy is listed)
		wingetDir := filepath.Join(la, "Microsoft", "WinGet", "Packages")
		if isDir(wingetDir) {
			entries, _ := os.ReadDir(wingetDir)
			for _, e := range entries {
				lname := strings.ToLower(e.Name())
				if !e.IsDir() || !strings.Contains(lname, "claudecode") {
					continue
				}
				collectNamed(c, filepath.Join(wingetDir, e.Name()), map[string]bool{"claude.exe": true, "claude": true})
			}
		}
	}

	// 10. Scoop (Windows)
	scoopDir := filepath.Join(home, "scoop", "apps", "claude-code", "current")
	if isDir(scoopDir) {
		for _, name := range []string{"claude.exe", "claude"} {
			c.add(filepath.Join(scoopDir, name), "bun_sea")
		}
	}

	// 11. Chocolatey (Windows)
	chocoDir := "C:/ProgramData/chocolatey/lib/claude-code"
	if isDir(chocoDir) {
		collectNamed(c, chocoDir, map[string]bool{"claude.exe": true})
	}

	// 12. Last resort: os/exec LookPath (cross-platform which)
	for _, name := range []string{"claude", "claude.exe"} {
		if found, err := exec.LookPath(name); err == nil {
			c.add(found, "bun_sea")
		}
	}

	return c.results
}

// collector accumulates unique targets in discovery order.
type collector struct {
	results []Found
	paths   map[string]bool
	infos   []os.FileInfo
}

func (c *collector) add(path, kind string) {
	if path == "" || !fileExists(path) {
		return
	}
	if kind == "bun_sea" && fileSize(path) <= minBinSize {
		return
	}
	resolved := resolvePath(path)
	if c.paths == nil {
		c.paths = make(map[string]bool)
	}
	if c.paths[resolved] {
		return
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return
	}
	for _, kept := range c.infos {
		if os.SameFile(info, kept) {
			return // hardlink or symlink to an already-listed target
		}
	}
	c.paths[resolved] = true
	c.infos = append(c.infos, info)
	c.results = append(c.results, Found{Path: resolved, Kind: kind})
}

func collectChecks(c *collector, root string, checks []check) {
	for _, ck := range checks {
		c.add(filepath.Join(root, filepath.FromSlash(ck.suffix)), ck.kind)
	}
}

// collectNamed walks dir and adds files whose base name is in names.
func collectNamed(c *collector, dir string, names map[string]bool) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && names[d.Name()] {
			c.add(path, "bun_sea")
		}
		return nil
	})
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
		abs, err := filepath.Abs(path)
		if err != nil {
			return path
		}
		return abs
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
