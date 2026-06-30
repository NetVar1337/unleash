package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VoidChecksum/unleash/internal/patches"
)

func TestRunVerifyPatchesFailsWhenRequiredMarkerMissing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(target, []byte("unpatched bundle"), 0o755); err != nil {
		t.Fatal(err)
	}
	required := true
	if rc := runVerifyPatches(target, "js", []patches.Patch{{
		ID:      "required-patch",
		Type:    "js_replace",
		Patches: []patches.SubPatch{{AppliedMarker: "patched-marker", Required: &required}},
	}}); rc == 0 {
		t.Fatal("verify returned success with required marker missing")
	}
}

func TestRunVerifyPatchesAllowsOptionalMarkerMissing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(target, []byte("unpatched bundle"), 0o755); err != nil {
		t.Fatal(err)
	}
	if rc := runVerifyPatches(target, "js", []patches.Patch{{
		ID:      "optional-patch",
		Type:    "js_replace",
		Patches: []patches.SubPatch{{AppliedMarker: "patched-marker"}},
	}}); rc != 0 {
		t.Fatalf("verify rc = %d, want success for optional missing marker", rc)
	}
}
