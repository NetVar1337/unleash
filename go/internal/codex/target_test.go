package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeCodexVersion(t *testing.T) {
	version, ok := LooksLikeCodexVersion("codex-cli 0.142.3\n")
	if !ok || version != "0.142.3" {
		t.Fatalf("LooksLikeCodexVersion() = %q, %v", version, ok)
	}

	if _, ok := LooksLikeCodexVersion("Claude Code 2.1.195\n"); ok {
		t.Fatal("accepted Claude Code version output")
	}
}

func TestFindTargetNativeWindowsLayout(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "Programs", "OpenAI", "Codex", "bin", "codex.exe")
	writeFakeFile(t, exe, minCodexBinSize+1)

	got, ok := FindTargetWithEnv(map[string]string{
		"LOCALAPPDATA": root,
		"HOME":         root,
		"USERPROFILE":  root,
	}, nil)
	if !ok {
		t.Fatal("target not found")
	}
	if got.Path != exe {
		t.Fatalf("Path = %q, want %q", got.Path, exe)
	}
	if got.Kind != "native" {
		t.Fatalf("Kind = %q, want native", got.Kind)
	}
}

func TestFindTargetNpmOptionalPackageLayout(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "npm", "node_modules", "@openai", "codex-win32-x64", "vendor", "x86_64-pc-windows-msvc", "bin", "codex.exe")
	writeFakeFile(t, exe, minCodexBinSize+1)

	got, ok := FindTargetWithEnv(map[string]string{
		"APPDATA":      root,
		"LOCALAPPDATA": filepath.Join(root, "empty"),
		"HOME":         root,
		"USERPROFILE":  root,
	}, nil)
	if !ok {
		t.Fatal("target not found")
	}
	if got.Path != exe {
		t.Fatalf("Path = %q, want %q", got.Path, exe)
	}
	if got.Kind != "npm" {
		t.Fatalf("Kind = %q, want npm", got.Kind)
	}
}

func TestFindTargetLookPathFallback(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "codex.exe")
	writeFakeFile(t, exe, minCodexBinSize+1)

	got, ok := FindTargetWithEnv(map[string]string{"HOME": root, "USERPROFILE": root}, func(name string) (string, error) {
		if name == "codex" || name == "codex.exe" {
			return exe, nil
		}
		return "", os.ErrNotExist
	})
	if !ok {
		t.Fatal("target not found")
	}
	if got.Path != exe || got.Kind != "path" {
		t.Fatalf("got %#v", got)
	}
}

func writeFakeFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
