package omp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const minOMPBundleSize int64 = 1_000_000

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
	for _, c := range ompCandidates(env) {
		if validOMPTarget(c.path) {
			return Target{Path: filepath.Clean(c.path), Kind: c.kind}, true
		}
	}
	for _, p := range ompWingetPaths(env) {
		if validOMPTarget(p) {
			return Target{Path: filepath.Clean(p), Kind: "winget"}, true
		}
	}
	if lookPath != nil {
		for _, name := range []string{"omp", "omp.exe"} {
			p, err := lookPath(name)
			if err != nil {
				continue
			}
			// Standalone installs (WindowsApps, winget links, native bin)
			// resolve straight to the real binary.
			if validOMPTarget(p) {
				return Target{Path: filepath.Clean(p), Kind: "path"}, true
			}
			if target, ok := targetFromShim(p); ok {
				return Target{Path: target, Kind: "path"}, true
			}
		}
	}
	return Target{}, false
}

// ompWingetPaths enumerates omp binaries under WinGet package dirs.
func ompWingetPaths(env map[string]string) []string {
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
		lname := strings.ToLower(e.Name())
		if !e.IsDir() || !(strings.Contains(lname, "omp") || strings.Contains(lname, "oh-my-pi")) {
			continue
		}
		base := filepath.Join(pkgBase, e.Name())
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && (d.Name() == "omp.exe" || d.Name() == "omp") {
				out = append(out, path)
			}
			return nil
		})
	}
	return out
}

func LooksLikeOMPVersion(output string) (string, bool) {
	output = strings.TrimSpace(output)
	const prefix = "omp/"
	if !strings.HasPrefix(output, prefix) {
		return "", false
	}
	version := strings.TrimSpace(strings.TrimPrefix(output, prefix))
	return version, version != ""
}

func DetectVersion() (string, bool) {
	out, err := exec.Command("omp", "--version").CombinedOutput()
	if err != nil {
		return "", false
	}
	return LooksLikeOMPVersion(string(out))
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
	return filepath.Join(home, ".unleash-omp")
}

func BackupDir(home string) string { return filepath.Join(StateDir(home), "backups") }

func ConfigDir(home string) string {
	if home == "" {
		if override := os.Getenv("PI_CONFIG_DIR"); override != "" && filepath.IsAbs(override) {
			return override
		}
		home = homeDir()
	}
	return filepath.Join(home, ".omp")
}

func AgentDir(home string) string { return filepath.Join(ConfigDir(home), "agent") }

type candidate struct{ path, kind string }

func ompCandidates(env map[string]string) []candidate {
	home := firstNonEmpty(env["HOME"], env["USERPROFILE"])
	appdata := env["APPDATA"]
	localAppData := env["LOCALAPPDATA"]
	var out []candidate
	if home != "" {
		out = append(out,
			candidate{filepath.Join(home, ".bun", "install", "global", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"), "bun"},
			candidate{filepath.Join(home, ".local", "share", "mise", "installs", "npm-oh-my-pi-pi-coding-agent", "dist", "cli.js"), "mise"},
			// Native / standalone installs ship a Bun-SEA executable.
			candidate{filepath.Join(home, ".local", "bin", "omp.exe"), "standalone"},
			candidate{filepath.Join(home, ".local", "bin", "omp"), "standalone"},
			candidate{filepath.Join(home, ".omp", "bin", "omp.exe"), "standalone"},
			candidate{filepath.Join(home, ".omp", "bin", "omp"), "standalone"},
		)
	}
	if appdata != "" {
		out = append(out, candidate{filepath.Join(appdata, "npm", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"), "npm"})
	}
	if localAppData != "" {
		out = append(out,
			candidate{filepath.Join(localAppData, "Microsoft", "WindowsApps", "omp.exe"), "standalone"},
			candidate{filepath.Join(localAppData, "Programs", "omp", "omp.exe"), "standalone"},
		)
	}
	out = append(out,
		candidate{filepath.Join("/usr", "local", "lib", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"), "npm"},
		candidate{filepath.Join("/opt", "homebrew", "lib", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"), "homebrew"},
		candidate{filepath.Join("/usr", "local", "bin", "omp"), "standalone"},
		candidate{filepath.Join("/usr", "bin", "omp"), "standalone"},
		candidate{filepath.Join("/opt", "homebrew", "bin", "omp"), "standalone"},
	)
	return out
}

func targetFromShim(shim string) (string, bool) {
	shim = filepath.Clean(shim)
	candidates := []string{
		filepath.Join(filepath.Dir(filepath.Dir(shim)), "install", "global", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"),
		filepath.Join(filepath.Dir(filepath.Dir(shim)), "lib", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"),
		filepath.Join(filepath.Dir(shim), "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js"),
	}
	for _, c := range candidates {
		if validOMPTarget(c) {
			return c, true
		}
	}
	return "", false
}

func validOMPTarget(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > minOMPBundleSize
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
