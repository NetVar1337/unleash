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

	"github.com/VoidChecksum/unleash/internal/bytepatch"
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
// ELF/MachO/PE layout. It writes to a temp file, verifies version output and
// startup smoke behavior, then performs an atomic rename on success.
//
// Retry logic: if the full batch fails verification, recoverVerifiedSubset
// adds patches one at a time and keeps only candidates whose cumulative set
// verifies. Dropped patch IDs are reported through SkippedHeavy.
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

		applyResult := applyJSPatchesToRegion(data, effLo, effHi, attemptPatches)
		appliedTotal := applyResult.Applied
		skippedTotal := applyResult.Skipped
		perPatch := applyResult.PerPatch

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

		verifyOut, verifyErr := verifyBunSEAExecutable(tmpPath)
		if verifyErr != nil {
			return PatchResult{
				Err:      fmt.Sprintf("verify failed: %q: %v", truncate(verifyOut, 500), verifyErr),
				Applied:  appliedTotal,
				Skipped:  skippedTotal,
				PerPatch: perPatch,
			}
		}

		// Commit strategy:
		//  1) hardlinks → must write in place (rename would break siblings)
		//  2) Windows / same-size SEA → prefer in-place (rename fails if exe is locked)
		//  3) else atomic rename, falling back to in-place on access-denied
		preferInPlace := HardlinkCount(binaryPath) > 1 || runtime.GOOS == "windows"
		if preferInPlace {
			if err := commitBytesInPlace(binaryPath, data, originalMode); err != nil {
				// Last resort: try rename (may still fail if locked)
				if rerr := replaceFile(tmpPath, binaryPath); rerr != nil {
					return PatchResult{
						Err: fmt.Sprintf("commit failed (in-place: %v; replace: %v) — close Claude Code and retry",
							err, rerr),
						Applied: appliedTotal,
						Skipped: skippedTotal,
					}
				}
				// rename took the temp file
				tmpPath = ""
			} else {
				// in-place ok; drop temp
			}
			_ = os.Chmod(binaryPath, originalMode)
			return PatchResult{
				OK:       true,
				Applied:  appliedTotal,
				Skipped:  skippedTotal,
				PerPatch: perPatch,
			}
		}

		if err := replaceFile(tmpPath, binaryPath); err != nil {
			// Fall back to in-place when rename is denied (locked binary).
			if ierr := commitBytesInPlace(binaryPath, data, originalMode); ierr != nil {
				return PatchResult{
					Err: fmt.Sprintf("rename failed: %v; in-place fallback: %v — close Claude Code and retry",
						err, ierr),
					Applied: appliedTotal,
					Skipped: skippedTotal,
				}
			}
		} else {
			tmpPath = "" // consumed by rename
		}

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

	return recoverVerifiedSubset(patchList, attempt)
}

func recoverVerifiedSubset(patchList []patches.Patch, attempt func([]patches.Patch) PatchResult) PatchResult {
	var selected []patches.Patch
	dropped := make(map[string]bool)
	var last PatchResult

	// Verification failures are rare but opaque: the only safe recovery is to
	// re-run the target binary after each candidate and keep the passing set.
	for _, p := range patchList {
		candidate := append(append([]patches.Patch(nil), selected...), p)
		result := attempt(candidate)
		if result.OK {
			selected = candidate
			last = result
			continue
		}
		if strings.Contains(result.Err, "verify failed") {
			dropped[p.ID] = true
			continue
		}
		return result
	}

	if len(selected) == 0 {
		return PatchResult{Err: "verify failed for every patch"}
	}
	last.Skipped += len(dropped)
	// SkippedHeavy is legacy API naming; it now means "dropped during verify recovery".
	last.SkippedHeavy = sortedKeys(dropped)
	return last
}

// applyJSPatchesToRegion applies JS subpatches in-place within [effLo, effHi).
func applyJSPatchesToRegion(data []byte, effLo, effHi int, patchList []patches.Patch) PatchResult {
	appliedTotal := 0
	skippedTotal := 0
	var perPatch []PerPatchResult
	region := data[effLo:effHi]

	for _, p := range patchList {
		appliedN := 0
		skippedN := 0
		maxPadding := 0

		for _, sub := range p.Patches {
			if sub.AppliedMarker != "" && bytes.Contains(region, []byte(sub.AppliedMarker)) {
				continue
			}
			if sub.SearchRegex != "" {
				applied, skipped, padding := applyRegexSubPatch(region, sub)
				appliedN += applied
				skippedN += skipped
				if applied == 0 && skipped == 0 {
					skippedN++
				}
				if padding > maxPadding {
					maxPadding = padding
				}
				continue
			}
			if sub.Search == "" {
				continue
			}
			search := []byte(sub.Search)
			replace := []byte(sub.Replace)
			if len(replace) > len(search) {
				skippedN++
				continue
			}
			padding := len(search) - len(replace)
			padded := append([]byte(nil), replace...)
			if padding > 0 {
				padded = append(padded, bytes.Repeat([]byte(" "), padding)...)
				if padding > maxPadding {
					maxPadding = padding
				}
			}
			changed := bytepatch.ReplaceBytes(region, search, padded, sub.EffectiveCount(0))
			appliedN += changed
			if changed == 0 {
				skippedN++
			}
		}

		perPatch = append(perPatch, PerPatchResult{ID: p.ID, Applied: appliedN, Skipped: skippedN, MaxPadding: maxPadding})
		appliedTotal += appliedN
		skippedTotal += skippedN
	}
	return PatchResult{OK: true, Applied: appliedTotal, Skipped: skippedTotal, PerPatch: perPatch}
}

func applyRegexSubPatch(region []byte, sub patches.SubPatch) (applied int, skipped int, maxPadding int) {
	replace := bytepatch.TranslateReplace(sub.Replace)
	hasBR := bytepatch.HasBackrefs(sub.SearchRegex)
	var re *regexp.Regexp
	if !hasBR {
		var err error
		re, err = regexp.Compile("(?s)" + sub.SearchRegex)
		if err != nil {
			return 0, 1, 0
		}
	}
	count := sub.EffectiveCount(0)
	pos := 0
	if count <= 0 {
		count = -1
	}
	for count != 0 {
		var loc []int
		var matchRE *regexp.Regexp
		if hasBR {
			loc, matchRE = bytepatch.FindSubmatchBackrefs(sub.SearchRegex, region[pos:])
		} else {
			loc = re.FindSubmatchIndex(region[pos:])
			matchRE = re
		}
		if loc == nil {
			break
		}
		start := pos + loc[0]
		end := pos + loc[1]
		match := region[start:end]
		expandedLoc := append([]int(nil), loc...)
		for i := range expandedLoc {
			if expandedLoc[i] >= 0 {
				expandedLoc[i] += pos
			}
		}
		replacement := matchRE.Expand(nil, []byte(replace), region, expandedLoc)
		if len(replacement) > len(match) {
			skipped++
			pos = end
			if count > 0 {
				count--
			}
			continue
		}
		padding := len(match) - len(replacement)
		if padding > 0 {
			replacement = append(replacement, bytes.Repeat([]byte(" "), padding)...)
			if padding > maxPadding {
				maxPadding = padding
			}
		}
		copy(region[start:end], replacement)
		applied++
		pos = end
		if count > 0 {
			count--
		}
	}
	return applied, skipped, maxPadding
}

func verifyBunSEAExecutable(path string) (string, error) {
	versionCmd := exec.Command(path, "--version")
	versionOut, versionErr := runCaptureWithTimeout(versionCmd, 60)
	if versionErr != nil || !strings.Contains(versionOut, "Claude Code") {
		return versionOut, fmt.Errorf("version check failed: %v", versionErr)
	}

	startupCmd := exec.Command(path)
	startupCmd.Stdin = strings.NewReader("")
	startupOut, startupErr := runCaptureWithTimeout(startupCmd, 10)
	out := versionOut + startupOut
	if isBunCrashOutput(startupOut) {
		return out, fmt.Errorf("startup crashed")
	}
	if startupCmd.ProcessState != nil {
		rc := startupCmd.ProcessState.ExitCode()
		if rc < 0 || rc >= 128 {
			return out, fmt.Errorf("startup crashed with rc=%d", rc)
		}
	}
	if startupErr != nil && strings.Contains(startupErr.Error(), "timed out") {
		return out, startupErr
	}
	return out, nil
}

func isBunCrashOutput(out string) bool {
	return strings.Contains(out, "oh no: Bun has crashed") ||
		strings.Contains(out, "panic(main thread)") ||
		strings.Contains(out, "Segmentation fault") ||
		strings.Contains(out, "Bus error")
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

// commitBytesInPlace overwrites path with data without rename.
// Same-size Bun SEA patches keep the PE layout; works when the file is
// hardlinked or (often) when rename is blocked because the exe is mapped.
func commitBytesInPlace(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY, mode)
	if err != nil {
		// retry with truncate if exclusive write fails
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}
	}
	n, werr := f.Write(data)
	if werr == nil && n != len(data) {
		werr = fmt.Errorf("short write %d/%d", n, len(data))
	}
	// Ensure size matches when we did not truncate first
	if werr == nil {
		werr = f.Truncate(int64(len(data)))
	}
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

// replaceFile renames src onto dst, removing dst first on platforms that
// refuse non-empty destination rename (Windows).
func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Windows: destination exists / is locked → remove then rename
	_ = os.Remove(dst)
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else {
		return err
	}
}
