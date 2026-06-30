package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

	msg := runPatchAsync("", "", patchList)()
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

func TestRunPatchAsyncSkipsJSPatchesForPlainJSTarget(t *testing.T) {
	msg := runPatchAsync("cli.js", "js", []patches.Patch{
		{
			ID:   "js-only",
			Type: "js_replace",
		},
	})()
	done, ok := msg.(patchDoneMsg)
	if !ok {
		t.Fatalf("runPatchAsync returned %T, want patchDoneMsg", msg)
	}
	if !done.result.OK {
		t.Fatalf("js target skip failed: %s", done.result.Err)
	}
	if done.result.Applied != 0 || done.result.Skipped != 1 {
		t.Fatalf("result = applied %d skipped %d, want applied 0 skipped 1", done.result.Applied, done.result.Skipped)
	}
}

func TestScanViewportReceivesRealKeyMessages(t *testing.T) {
	rows := make([]patches.ScanRow, 20)
	for i := range rows {
		rows[i] = patches.ScanRow{
			ID:         "patch",
			Status:     "ok",
			Confidence: 1,
			Method:     "anchor",
		}
	}
	m := tuiModel{
		width:    100,
		height:   12,
		view:     viewScan,
		scanDone: true,
		scanRows: rows,
	}
	m.resizeViewports()
	m.buildScanViewport()

	updated, _ := m.handleScanKey(tea.KeyMsg{Type: tea.KeyDown})
	next := updated.(tuiModel)
	if next.scanViewport.YOffset == 0 {
		t.Fatalf("scan viewport did not scroll on KeyDown")
	}
}
