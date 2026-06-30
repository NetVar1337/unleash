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

func TestPatchManagerSelectionCommandAppliesSettingsPatch(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	m := tuiModel{
		view: viewPatchMgr,
		allPatches: []patches.Patch{
			{
				ID:           "settings-only",
				Type:         "settings",
				Category:     "permissions",
				SettingsPath: settingsPath,
				Settings: map[string]any{
					"permissions.defaultMode": "bypassPermissions",
				},
			},
		},
		toggleSet: make(map[int]bool),
	}
	m.buildCategoryIndex()

	updated, cmd := m.handlePatchMgrKey("tab")
	if cmd != nil {
		t.Fatalf("tab returned command %T, want nil", cmd)
	}
	m = updated.(tuiModel)
	if !m.patchFocus {
		t.Fatal("patch manager did not focus patch list")
	}

	updated, cmd = m.handlePatchMgrKey(" ")
	if cmd != nil {
		t.Fatalf("space returned command %T, want nil", cmd)
	}
	m = updated.(tuiModel)
	if !m.toggleSet[0] {
		t.Fatal("space did not select patch")
	}

	updated, cmd = m.handlePatchMgrKey("enter")
	if cmd == nil {
		t.Fatal("enter returned nil command")
	}
	m = updated.(tuiModel)
	if !m.busy {
		t.Fatal("enter did not enter busy state")
	}

	msg := cmd()
	done, ok := msg.(patchDoneMsg)
	if !ok {
		t.Fatalf("apply command returned %T, want patchDoneMsg", msg)
	}
	if !done.result.OK {
		t.Fatalf("selected patch failed to apply: %s", done.result.Err)
	}
	if done.result.Applied != 1 {
		t.Fatalf("applied count = %d, want 1", done.result.Applied)
	}

	gotBytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	if got["permissions.defaultMode"] != "bypassPermissions" {
		t.Fatalf("settings value = %v, want bypassPermissions", got["permissions.defaultMode"])
	}
}

func TestPatchManagerDoesNotToggleAppliedPatch(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	want := map[string]any{"permissions.defaultMode": "bypassPermissions"}
	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	m := tuiModel{
		view:       viewPatchMgr,
		patchFocus: true,
		allPatches: []patches.Patch{{
			ID:           "settings-only",
			Type:         "settings",
			Category:     "permissions",
			SettingsPath: settingsPath,
			Settings:     want,
		}},
		toggleSet: make(map[int]bool),
	}
	m.buildCategoryIndex()

	updated, _ := m.handlePatchMgrKey(" ")
	next := updated.(tuiModel)
	if next.toggleSet[0] {
		t.Fatal("applied patch was toggled selected")
	}
}

func TestPatchSearchPersistsAfterEnterAndCanClear(t *testing.T) {
	m := tuiModel{
		view: viewPatchMgr,
		allPatches: []patches.Patch{
			{ID: "alpha", Type: "settings", Category: "permissions", Description: "first"},
			{ID: "beta", Type: "settings", Category: "permissions", Description: "second"},
		},
		toggleSet: make(map[int]bool),
	}
	m.buildCategoryIndex()
	m.searchMode = true
	m.searchInput.SetValue("beta")

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.searchMode {
		t.Fatal("enter did not leave search mode")
	}
	got := m.currentCategoryPatches()
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("filtered patches = %v, want [1]", got)
	}

	updated, _ = m.handlePatchMgrKey("c")
	m = updated.(tuiModel)
	if m.searchInput.Value() != "" {
		t.Fatalf("search value = %q, want empty", m.searchInput.Value())
	}
	got = m.currentCategoryPatches()
	if len(got) != 2 {
		t.Fatalf("cleared filter patches = %v, want two patches", got)
	}
}

func TestScanDoneDoesNotClearUnrelatedBusyState(t *testing.T) {
	m := tuiModel{busy: true, busyMsg: "Running doctor..."}
	updated, _ := m.Update(scanDoneMsg{})
	next := updated.(tuiModel)
	if !next.busy || next.busyMsg != "Running doctor..." {
		t.Fatalf("scanDone cleared unrelated busy state: busy=%v msg=%q", next.busy, next.busyMsg)
	}
}

func TestScannablePatchesFiltersToEnabledJSReplace(t *testing.T) {
	disabled := true
	noScan := false
	got := scannablePatches([]patches.Patch{
		{ID: "settings", Type: "settings"},
		{ID: "disabled", Type: "js_replace", Disabled: disabled},
		{ID: "no-scan", Type: "js_replace", ScanSignatures: &noScan},
		{ID: "enabled", Type: "js_replace"},
	})
	if len(got) != 1 || got[0].ID != "enabled" {
		t.Fatalf("scannable patches = %#v, want only enabled", got)
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
