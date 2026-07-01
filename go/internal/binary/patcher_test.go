package binary

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/VoidChecksum/unleash/internal/patches"
)

func TestRunWithTimeoutKillsSlowCommand(t *testing.T) {
	start := time.Now()
	err := runWithTimeout(exec.Command("sleep", "2"), 1)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("runWithTimeout error = %v, want timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Fatalf("runWithTimeout elapsed = %s, want timeout near 1s", elapsed)
	}
}

func TestRunCaptureWithTimeoutReturnsOutput(t *testing.T) {
	out, err := runCaptureWithTimeout(exec.Command("printf", "ok"), 1)
	if err != nil {
		t.Fatalf("runCaptureWithTimeout error = %v", err)
	}
	if out != "ok" {
		t.Fatalf("output = %q, want ok", out)
	}
}

func TestApplyJSPatchesToRegionHonorsCount(t *testing.T) {
	data := []byte("xx aa aa aa yy")
	res := applyJSPatchesToRegion(data, 3, 11, []patches.Patch{
		{
			ID:      "limited",
			Patches: []patches.SubPatch{{Search: "aa", Replace: "bb", Count: intPtr(1)}},
		},
	})
	if got := string(data); got != "xx bb aa aa yy" {
		t.Fatalf("patched data = %q, want only first match in region", got)
	}
	if res.Applied != 1 || res.Skipped != 0 {
		t.Fatalf("result = applied %d skipped %d, want 1/0", res.Applied, res.Skipped)
	}
}

func TestApplyJSPatchesToRegionZeroCountReplacesAll(t *testing.T) {
	data := []byte("aa aa aa")
	res := applyJSPatchesToRegion(data, 0, len(data), []patches.Patch{
		{
			ID:      "all",
			Patches: []patches.SubPatch{{Search: "aa", Replace: "b", Count: intPtr(0)}},
		},
	})
	if got := string(data); got != "b  b  b " {
		t.Fatalf("patched data = %q, want all matches padded", got)
	}
	if res.Applied != 3 {
		t.Fatalf("applied = %d, want 3", res.Applied)
	}

}
func TestApplyJSPatchesToRegionRegexZeroCountReplacesAll(t *testing.T) {
	data := []byte("aa aa aa")
	res := applyJSPatchesToRegion(data, 0, len(data), []patches.Patch{
		{
			ID:      "regex-all",
			Patches: []patches.SubPatch{{SearchRegex: "aa", Replace: "b", Count: intPtr(0)}},
		},
	})
	if got := string(data); got != "b  b  b " {
		t.Fatalf("patched data = %q, want all regex matches padded", got)
	}
	if res.Applied != 3 {
		t.Fatalf("applied = %d, want 3", res.Applied)
	}
}

func TestRecoverVerifiedSubsetKeepsSafePaddedPatch(t *testing.T) {
	patchList := []patches.Patch{
		{ID: "exact"},
		{ID: "safe-padded"},
		{ID: "bad-padded"},
	}
	try := func(candidate []patches.Patch) PatchResult {
		for _, p := range candidate {
			if p.ID == "bad-padded" {
				return PatchResult{Err: "verify failed"}
			}
		}
		return PatchResult{OK: true, Applied: len(candidate)}
	}

	res := recoverVerifiedSubset(patchList, try)
	if !res.OK {
		t.Fatalf("recoverVerifiedSubset failed: %s", res.Err)
	}
	if res.Applied != 2 {
		t.Fatalf("applied = %d, want safe exact + padded patches", res.Applied)
	}
	if res.Skipped != 1 {
		t.Fatalf("skipped = %d, want only bad-padded", res.Skipped)
	}
	if len(res.SkippedHeavy) != 1 || res.SkippedHeavy[0] != "bad-padded" {
		t.Fatalf("skipped = %v, want only bad-padded", res.SkippedHeavy)
	}
}

func TestApplyJSPatchesToRegionCountsMissingSearchAsSkipped(t *testing.T) {
	data := []byte("current code")
	res := applyJSPatchesToRegion(data, 0, len(data), []patches.Patch{
		{
			ID: "missing",
			Patches: []patches.SubPatch{
				{Search: "old code", Replace: "new code"},
				{SearchRegex: "oldRegex\\(\\)", Replace: "newRegex()"},
			},
		},
	})
	if res.Applied != 0 || res.Skipped != 2 {
		t.Fatalf("result = applied %d skipped %d, want 0/2", res.Applied, res.Skipped)
	}
	if len(res.PerPatch) != 1 || res.PerPatch[0].Skipped != 2 {
		t.Fatalf("per-patch result = %#v, want two skipped subpatches", res.PerPatch)
	}
}
func intPtr(v int) *int { return &v }
