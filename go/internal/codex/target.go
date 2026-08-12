package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const minCodexBinSize int64 = 1_000_000

type Target struct {
	Path    string
	Kind    string
	Version string
}

type lookPathFunc func(string) (string, error)

func FindTarget() (Target, bool) {
	env := map[string]string{
		"APPDATA":      os.Getenv("APPDATA"),
		"LOCALAPPDATA": os.Getenv("LOCALAPPDATA"),
		"HOME":         os.Getenv("HOME"),
		"USERPROFILE":  os.Getenv("USERPROFILE"),
	}
	return FindTargetWithEnv(env, exec.LookPath)
}

func FindTargetWithEnv(env map[string]string, lookPath lookPathFunc) (Target, bool) {
	for _, c := range codexCandidates(env) {
		if validCodexTarget(c.path) {
			return Target{Path: resolvePath(c.path), Kind: c.kind}, true
		}
	}

	for _, p := range codexWingetPaths(env) {
		if validCodexTarget(p) {
			return Target{Path: resolvePath(p), Kind: "winget"}, true
		}
	}

	if lookPath != nil {
		for _, name := range []string{"codex", "codex.exe"} {
			p, err := lookPath(name)
			if err == nil && validCodexTarget(p) {
				return Target{Path: resolvePath(p), Kind: "path"}, true
			}
		}
	}

	return Target{}, false
}

// codexWingetPaths enumerates codex binaries under WinGet package dirs
// (portable installs register there with source-suffixed directory names).
func codexWingetPaths(env map[string]string) []string {
	la := env["LOCALAPPDATA"]
	if la == "" {
		return nil
	}
	pkgBase := filepath.Join(la, "Microsoft", "WinGet", "Packages")
	entries, err := os.ReadDir(pkgBase)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || !strings.Contains(strings.ToLower(e.Name()), "codex") {
			continue
		}
		base := filepath.Join(pkgBase, e.Name())
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && (d.Name() == "codex.exe" || d.Name() == "codex") {
				out = append(out, path)
			}
			return nil
		})
	}
	return out
}

func LooksLikeCodexVersion(output string) (string, bool) {
	output = strings.TrimSpace(output)
	const prefix = "codex-cli "
	if !strings.HasPrefix(output, prefix) {
		return "", false
	}
	version := strings.TrimSpace(strings.TrimPrefix(output, prefix))
	if version == "" {
		return "", false
	}
	return version, true
}

func DetectVersion(path string) (string, bool) {
	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return "", false
	}
	return LooksLikeCodexVersion(string(out))
}

func SHA256Short(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

func StateDir(home string) string {
	if home == "" {
		home = homeDir()
	}
	return filepath.Join(home, ".unleash-gpt")
}

func BackupDir(home string) string {
	return filepath.Join(StateDir(home), "backups")
}

type candidate struct {
	path string
	kind string
}

func codexCandidates(env map[string]string) []candidate {
	home := firstNonEmpty(env["HOME"], env["USERPROFILE"])
	appdata := env["APPDATA"]
	localappdata := env["LOCALAPPDATA"]
	var out []candidate

	if localappdata != "" {
		out = append(out,
			candidate{filepath.Join(localappdata, "Programs", "OpenAI", "Codex", "bin", "codex.exe"), "native"},
			candidate{filepath.Join(localappdata, "Programs", "Codex", "bin", "codex.exe"), "native"},
		)
	}
	if home != "" {
		out = append(out,
			candidate{filepath.Join(home, ".local", "bin", exeName()), "native"},
			candidate{filepath.Join(home, ".codex", "bin", exeName()), "native"},
			candidate{filepath.Join(home, "scoop", "apps", "codex", "current", "codex.exe"), "scoop"},
		)
	}
	out = append(out,
		candidate{filepath.Join("/usr", "local", "bin", "codex"), "native"},
		candidate{filepath.Join("/usr", "bin", "codex"), "native"},
		candidate{filepath.Join("/opt", "homebrew", "bin", "codex"), "homebrew"},
	)

	roots := npmRoots(home, appdata)
	exes := []string{exeName()}
	if exes[0] != "codex.exe" {
		exes = append(exes, "codex.exe")
	}
	for _, root := range roots {
		for _, pkg := range platformPackages() {
			for _, exe := range exes {
				out = append(out, candidate{filepath.Join(root, "@openai", pkg.name, "vendor", pkg.triple, "bin", exe), "npm"})
				// npm nests the platform package under the wrapper package's
				// own node_modules on some installs.
				out = append(out, candidate{filepath.Join(root, "@openai", "codex", "node_modules", "@openai", pkg.name, "vendor", pkg.triple, "bin", exe), "npm"})
			}
		}
		for _, exe := range exes {
			out = append(out, candidate{filepath.Join(root, "@openai", "codex", "vendor", targetTriple(), "bin", exe), "npm"})
		}
	}

	return out
}

func npmRoots(home, appdata string) []string {
	var roots []string
	if appdata != "" {
		roots = append(roots, filepath.Join(appdata, "npm", "node_modules"))
	}
	if home != "" {
		roots = append(roots,
			filepath.Join(home, ".bun", "install", "global", "node_modules"),
			filepath.Join(home, ".local", "share", "pnpm", "global", "5", "node_modules"),
			filepath.Join(home, ".volta", "tools", "image", "packages"),
		)
	}
	return roots
}

type platformPackage struct{ name, triple string }

func platformPackages() []platformPackage {
	pkgs := []platformPackage{
		{"codex-linux-x64", "x86_64-unknown-linux-musl"},
		{"codex-linux-arm64", "aarch64-unknown-linux-musl"},
		{"codex-darwin-x64", "x86_64-apple-darwin"},
		{"codex-darwin-arm64", "aarch64-apple-darwin"},
		{"codex-win32-x64", "x86_64-pc-windows-msvc"},
		{"codex-win32-arm64", "aarch64-pc-windows-msvc"},
	}
	current := targetTriple()
	if current == "" {
		return pkgs
	}
	ordered := make([]platformPackage, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.triple == current {
			ordered = append(ordered, pkg)
			break
		}
	}
	for _, pkg := range pkgs {
		if pkg.triple != current {
			ordered = append(ordered, pkg)
		}
	}
	return ordered
}

func targetTriple() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "x86_64-unknown-linux-musl"
	case "linux/arm64":
		return "aarch64-unknown-linux-musl"
	case "darwin/amd64":
		return "x86_64-apple-darwin"
	case "darwin/arm64":
		return "aarch64-apple-darwin"
	case "windows/amd64":
		return "x86_64-pc-windows-msvc"
	case "windows/arm64":
		return "aarch64-pc-windows-msvc"
	default:
		return ""
	}
}

func exeName() string {
	if runtime.GOOS == "windows" {
		return "codex.exe"
	}
	return "codex"
}

func validCodexTarget(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > minCodexBinSize
}

func resolvePath(path string) string {
	return filepath.Clean(path)
}

func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func latestBackup(home string) (string, error) {
	entries, err := os.ReadDir(BackupDir(home))
	if err != nil {
		return "", err
	}
	var newest os.DirEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".bak") {
			continue
		}
		if newest == nil || e.Name() > newest.Name() {
			newest = e
		}
	}
	if newest == nil {
		return "", errors.New("no backups found")
	}
	return filepath.Join(BackupDir(home), newest.Name()), nil
}
