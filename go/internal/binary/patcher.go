package binary

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/VoidChecksum/unleash/internal/patches"
)

// PatchResult holds the result of an in-place patching attempt.
type PatchResult struct {
	OK           bool             `json:"ok"`
	Noop         bool             `json:"noop,omitempty"`
	Err          string           `json:"err,omitempty"`
	Applied      int              `json:"applied"`
	Skipped      int              `json:"skipped"`
	PerPatch     []PerPatchResult `json:"per_patch,omitempty"`
	SkippedHeavy []string         `json:"skipped_heavy,omitempty"`
}

// PerPatchResult tracks per-patch application stats.
type PerPatchResult struct {
	ID         string `json:"id"`
	Applied    int    `json:"applied"`
	Skipped    int    `json:"skipped"`
	MaxPadding int    `json:"max_padding"`
}

// PatchBunSEAInplace patches a Bun SEA binary in-place, preserving the
// ELF/MachO/PE layout. It writes to a temp file, verifies by running
// --version, and performs an atomic rename on success.
//
// Retry logic: if verification fails and some patches needed >64 bytes of
// space-padding, those are dropped and the binary is re-patched. A second
// retry drops ALL padded patches.
func PatchBunSEAInplace(binaryPath string, patchList []patches.Patch) PatchResult {
	info, err := os.Stat(binaryPath)
	if err != nil {
		return PatchResult{Err: fmt.Sprintf("stat: %v", err)}
	}
	originalMode := info.Mode()
	originalSize := info.Size()

	sourceBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		return PatchResult{Err: fmt.Sprintf("read: %v", err)}
	}

	attempt := func(attemptPatches []patches.Patch) PatchResult {
		data := make([]byte, len(sourceBytes))
		copy(data, sourceBytes)

		bunOff, bunSize, err := FindBunSection(data)
		if err != nil {
			return PatchResult{Err: fmt.Sprintf("find .bun section: %v", err)}
		}
		bunLo := bunOff
		bunHi := bunOff + bunSize

		if !BunSectionHasValidTrailer(data, bunLo, bunHi) {
			return PatchResult{Err: "Bun trailer invalid — format change"}
		}

		effLo, effHi := FindActiveBundleBounds(data, bunLo, bunHi)

		appliedTotal := 0
		skippedTotal := 0
		var perPatch []PerPatchResult

		for _, p := range attemptPatches {
			appliedN := 0
			skippedN := 0
			maxPadding := 0

			for _, sub := range p.Patches {
				searchRegex := sub.SearchRegex
				search := sub.Search
				replace := sub.Replace
				marker := sub.AppliedMarker

				// Check if already applied via marker
				if marker != "" {
					markerBytes := []byte(marker)
					if bytes.Index(data[effLo:effHi], markerBytes) >= 0 {
						continue
					}
				}

				if searchRegex != "" {
					re, err := regexp.Compile("(?s)" + searchRegex)
					if err != nil {
						skippedN++
						continue
					}
					sectionView := make([]byte, effHi-effLo)
					copy(sectionView, data[effLo:effHi])
					loc := re.FindSubmatchIndex(sectionView)
					if loc != nil {
						mb := sectionView[loc[0]:loc[1]]
						// Expand backreferences in replacement template
						rb := re.Expand(nil, []byte(replace), sectionView, loc)

						if len(rb) > len(mb) {
							skippedN++
							continue
						}
						if len(rb) < len(mb) {
							padding := len(mb) - len(rb)
							padBytes := make([]byte, padding)
							for i := range padBytes {
								padBytes[i] = ' '
							}
							rb = append(rb, padBytes...)
							if padding > maxPadding {
								maxPadding = padding
							}
						}
						absStart := effLo + loc[0]
						copy(data[absStart:absStart+len(mb)], rb)
						appliedN++
					}
				} else if search != "" {
					sb := []byte(search)
					rb := []byte(replace)
					if len(rb) > len(sb) {
						skippedN++
						continue
					}
					if len(rb) < len(sb) {
						padding := len(sb) - len(rb)
						padBytes := make([]byte, padding)
						for i := range padBytes {
							padBytes[i] = ' '
						}
						rb = append(rb, padBytes...)
						if padding > maxPadding {
							maxPadding = padding
						}
					}
					// First occurrence only within effective region
					j := bytes.Index(data[effLo:effHi], sb)
					if j >= 0 {
						copy(data[effLo+j:effLo+j+len(sb)], rb)
						appliedN++
					}
				}
			}

			perPatch = append(perPatch, PerPatchResult{
				ID:         p.ID,
				Applied:    appliedN,
				Skipped:    skippedN,
				MaxPadding: maxPadding,
			})
			appliedTotal += appliedN
			skippedTotal += skippedN
		}

		if int64(len(data)) != originalSize {
			return PatchResult{
				Err:      fmt.Sprintf("size drift %d vs %d", len(data), originalSize),
				Skipped:  skippedTotal,
				PerPatch: perPatch,
			}
		}

		if appliedTotal == 0 {
			return PatchResult{
				OK:       true,
				Noop:     true,
				Skipped:  skippedTotal,
				PerPatch: perPatch,
			}
		}

		// Write to temp file, verify, then atomic swap.
		dir := filepath.Dir(binaryPath)
		base := filepath.Base(binaryPath)
		tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.unleashtmp-%d", base, os.Getpid()))

		err = os.WriteFile(tmpPath, data, originalMode)
		if err != nil {
			return PatchResult{
				Err:     fmt.Sprintf("write failed: %v", err),
				Skipped: skippedTotal,
			}
		}
		defer func() {
			// Cleanup on failure — if file still exists after this function,
			// it means we didn't successfully rename it.
			os.Remove(tmpPath)
		}()

		// Try to set mode (may fail on Windows)
		_ = os.Chmod(tmpPath, originalMode)

		// macOS: re-sign with ad-hoc signature
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("codesign", "--force", "--sign", "-", tmpPath)
			cmd.Stdout = nil
			cmd.Stderr = nil
			_ = runWithTimeout(cmd, 30)
		}

		// Verify by running --version
		verifyCmd := exec.Command(tmpPath, "--version")
		verifyOut, verifyErr := runCaptureWithTimeout(verifyCmd, 60)
		if verifyErr != nil || !strings.Contains(verifyOut, "Claude Code") {
			rc := -1
			if verifyCmd.ProcessState != nil {
				rc = verifyCmd.ProcessState.ExitCode()
			}
			return PatchResult{
				Err:      fmt.Sprintf("verify failed: %q rc=%d", truncate(verifyOut, 500), rc),
				Applied:  appliedTotal,
				Skipped:  skippedTotal,
				PerPatch: perPatch,
			}
		}

		// Atomic replace
		if err := os.Rename(tmpPath, binaryPath); err != nil {
			return PatchResult{
				Err:     fmt.Sprintf("rename failed: %v", err),
				Applied: appliedTotal,
				Skipped: skippedTotal,
			}
		}

		// Restore mode after rename (may fail on Windows)
		_ = os.Chmod(binaryPath, originalMode)

		return PatchResult{
			OK:       true,
			Applied:  appliedTotal,
			Skipped:  skippedTotal,
			PerPatch: perPatch,
		}
	}

	// ── First attempt: all patches ──
	result := attempt(patchList)
	if result.OK {
		return result
	}

	if !strings.Contains(result.Err, "verify failed") {
		return result
	}

	perPatch0 := result.PerPatch

	// ── Retry 1: drop patches that needed >64 bytes of padding ──
	heavyIDs := make(map[string]bool)
	for _, pr := range perPatch0 {
		if pr.MaxPadding > 64 {
			heavyIDs[pr.ID] = true
		}
	}
	if len(heavyIDs) > 0 {
		var reduced1 []patches.Patch
		for _, p := range patchList {
			if !heavyIDs[p.ID] {
				reduced1 = append(reduced1, p)
			}
		}
		if len(reduced1) > 0 {
			r1 := attempt(reduced1)
			if r1.OK {
				r1.SkippedHeavy = sortedKeys(heavyIDs)
				return r1
			}
		}
	}

	// ── Retry 2: drop ALL patches that needed any padding (>0 bytes) ──
	anyPadIDs := make(map[string]bool)
	for _, pr := range perPatch0 {
		if pr.MaxPadding > 0 {
			anyPadIDs[pr.ID] = true
		}
	}
	// extraIDs = anyPadIDs - heavyIDs
	hasExtra := false
	for id := range anyPadIDs {
		if !heavyIDs[id] {
			hasExtra = true
			break
		}
	}
	if hasExtra {
		var reduced2 []patches.Patch
		for _, p := range patchList {
			if !anyPadIDs[p.ID] {
				reduced2 = append(reduced2, p)
			}
		}
		if len(reduced2) > 0 {
			r2 := attempt(reduced2)
			if r2.OK {
				r2.SkippedHeavy = sortedKeys(anyPadIDs)
				return r2
			}
		}
	}

	return result
}

// runWithTimeout runs a command with a timeout in seconds.
func runWithTimeout(cmd *exec.Cmd, timeoutSec int) error {
	if timeoutSec <= 0 {
		return cmd.Run()
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("command timed out after %ds", timeoutSec)
	}
}

// runCaptureWithTimeout runs a command and captures combined stdout+stderr.
func runCaptureWithTimeout(cmd *exec.Cmd, timeoutSec int) (string, error) {
	if timeoutSec <= 0 {
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	type result struct {
		out []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		done <- result{out: out, err: err}
	}()
	select {
	case res := <-done:
		return string(res.out), res.err
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		res := <-done
		return string(res.out), fmt.Errorf("command timed out after %ds", timeoutSec)
	}
}

// truncate returns s truncated to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
