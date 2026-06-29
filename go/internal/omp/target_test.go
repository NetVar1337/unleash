package omp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeOMPVersion(t *testing.T) {
	version, ok := LooksLikeOMPVersion("omp/16.2.5\n")
	if !ok || version != "16.2.5" {
		t.Fatalf("LooksLikeOMPVersion() = %q, %v", version, ok)
	}
	if _, ok := LooksLikeOMPVersion("codex-cli 0.142.3\n"); ok {
		t.Fatal("accepted Codex version output")
	}
}

func TestFindTargetBunGlobalPackage(t *testing.T) {
	root := t.TempDir()
	cli := filepath.Join(root, ".bun", "install", "global", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js")
	writeFakeOMPFile(t, cli, minOMPBundleSize+1)

	got, ok := FindTargetWithEnv(map[string]string{"HOME": root, "USERPROFILE": root}, nil)
	if !ok {
		t.Fatal("target not found")
	}
	if got.Path != cli || got.Kind != "bun" {
		t.Fatalf("got %#v", got)
	}
}

func TestFindTargetNpmGlobalPackage(t *testing.T) {
	root := t.TempDir()
	cli := filepath.Join(root, "npm", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js")
	writeFakeOMPFile(t, cli, minOMPBundleSize+1)

	got, ok := FindTargetWithEnv(map[string]string{"APPDATA": root, "HOME": filepath.Join(root, "home"), "USERPROFILE": filepath.Join(root, "home")}, nil)
	if !ok {
		t.Fatal("target not found")
	}
	if got.Path != cli || got.Kind != "npm" {
		t.Fatalf("got %#v", got)
	}
}

func TestTargetFromShimCustomNpmPrefix(t *testing.T) {
	root := t.TempDir()
	shim := filepath.Join(root, "bin", "omp")
	cli := filepath.Join(root, "lib", "node_modules", "@oh-my-pi", "pi-coding-agent", "dist", "cli.js")
	writeFakeOMPFile(t, shim, 1)
	writeFakeOMPFile(t, cli, minOMPBundleSize+1)

	got, ok := targetFromShim(shim)
	if !ok {
		t.Fatal("target not found")
	}
	if got != cli {
		t.Fatalf("target = %q, want %q", got, cli)
	}
}

func writeFakeOMPFile(t *testing.T, path string, size int64) {
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
