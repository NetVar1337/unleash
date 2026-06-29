package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/VoidChecksum/unleash/internal/patches"
)

func TestRunPatchAsyncAppliesSettingsWithoutBinaryTarget(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	patchList := []patches.Patch{
		{
			ID:           "settings-only",
			Type:         "settings",
			SettingsPath: settingsPath,
			Settings: map[string]any{
				"permissions.defaultMode": "bypassPermissions",
			},
		},
	}

	msg := runPatchAsync("", patchList)()
	done, ok := msg.(patchDoneMsg)
	if !ok {
		t.Fatalf("runPatchAsync returned %T, want patchDoneMsg", msg)
	}
	if !done.result.OK {
		t.Fatalf("settings-only apply failed: %s", done.result.Err)
	}
	if done.result.Applied != 1 {
		t.Fatalf("applied count = %d, want 1", done.result.Applied)
	}

	data, err := json.MarshalIndent(map[string]any{
		"permissions.defaultMode": "bypassPermissions",
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal expected settings: %v", err)
	}
	gotBytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if string(gotBytes) != string(data) {
		t.Fatalf("settings file = %q, want %q", string(gotBytes), string(data))
	}
}

func TestPatchStatusReportsAppliedForSettingsPatch(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	want := map[string]any{
		"permissions.defaultMode": "bypassPermissions",
	}
	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	m := tuiModel{
		allPatches: []patches.Patch{
			{
				ID:           "settings-only",
				Type:         "settings",
				SettingsPath: settingsPath,
				Settings:     want,
			},
		},
	}

	if got := m.patchStatus(0); got != "APPLIED" {
		t.Fatalf("patchStatus = %q, want APPLIED", got)
	}
}

func TestApplyToggledAllowsSettingsPatchWithoutBinaryTarget(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	m := tuiModel{
		allPatches: []patches.Patch{
			{
				ID:           "settings-only",
				Type:         "settings",
				SettingsPath: settingsPath,
				Settings: map[string]any{
					"permissions.defaultMode": "bypassPermissions",
				},
			},
		},
		toggleSet: map[int]bool{0: true},
	}

	updated, cmd := m.applyToggled()
	if cmd == nil {
		t.Fatalf("applyToggled returned nil command")
	}
	next := updated.(tuiModel)
	if !next.busy {
		t.Fatalf("applyToggled did not enter busy state")
	}
}
