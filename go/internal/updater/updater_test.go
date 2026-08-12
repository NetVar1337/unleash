package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdaterUsesUnleashRepository(t *testing.T) {
	if Repo != "NetVar1337/unleash" {
		t.Fatalf("Repo = %q, want NetVar1337/unleash", Repo)
	}
}

func TestUpstreamStatusWarnsWhenSyncStateMissing(t *testing.T) {
	restoreState := useTempState(t)
	defer restoreState()

	patchDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(patchDir, "01-test.json"), []byte(`{"id":"test"}`), 0o644); err != nil {
		t.Fatalf("write patch: %v", err)
	}

	info := UpstreamStatusWithRemote(patchDir, "remote-sha")
	if !info.UpdateAvailable {
		t.Fatal("UpdateAvailable = false, want true")
	}
	if !info.StateMissing {
		t.Fatal("StateMissing = false, want true")
	}
	if info.Drift {
		t.Fatal("Drift = true with no local state, want false")
	}
	if info.LocalFiles != 1 {
		t.Fatalf("LocalFiles = %d, want 1", info.LocalFiles)
	}
}

func TestUpstreamStatusWarnsWhenRemoteDiffers(t *testing.T) {
	restoreState := useTempState(t)
	defer restoreState()
	SaveState(map[string]interface{}{"patches_commit": "local-sha"})

	info := UpstreamStatusWithRemote(t.TempDir(), "remote-sha")
	if !info.UpdateAvailable {
		t.Fatal("UpdateAvailable = false, want true")
	}
	if !info.Drift {
		t.Fatal("Drift = false, want true")
	}
	if info.StateMissing {
		t.Fatal("StateMissing = true with local state, want false")
	}
}

func TestUpstreamStatusCurrentWhenRemoteMatches(t *testing.T) {
	restoreState := useTempState(t)
	defer restoreState()
	SaveState(map[string]interface{}{"patches_commit": "same-sha"})

	info := UpstreamStatusWithRemote(t.TempDir(), "same-sha")
	if info.UpdateAvailable {
		t.Fatal("UpdateAvailable = true, want false")
	}
	if info.Drift {
		t.Fatal("Drift = true, want false")
	}
}

func useTempState(t *testing.T) func() {
	t.Helper()
	oldDir := StateDir
	oldFile := StateFile
	StateDir = t.TempDir()
	StateFile = filepath.Join(StateDir, "state.json")
	return func() {
		StateDir = oldDir
		StateFile = oldFile
	}
}
