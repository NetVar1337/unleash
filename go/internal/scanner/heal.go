package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/VoidChecksum/void-patcher-cc/internal/binary"
	"github.com/VoidChecksum/void-patcher-cc/internal/patches"
)

// HealDetail records what happened to a single patch during healing.
type HealDetail struct {
	ID     string `json:"id"`
	Action string `json:"action"` // "healed", "failed"
	Reason string `json:"reason,omitempty"`
	Method string `json:"method,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// HealResult holds the outcome of an auto-heal run.
type HealResult struct {
	Healed  int          `json:"healed"`
	Skipped int          `json:"skipped"`
	Failed  int          `json:"failed"`
	Details []HealDetail `json:"details"`
}

// AutoHealDrift regenerates search_regex for patches whose anchors resolve but
// regex doesn't match. It rewrites patch JSON files in-place.
//
// Enhanced: also tries fuzzy anchor matching when exact anchors fail, and
// updates stale anchor_strings in-place.
func AutoHealDrift(text string, patchDir string, verbose bool) HealResult {
	sc := NewSigScanner(text)
	healed := 0
	skipped := 0
	failed := 0
	var details []HealDetail

	patchList, err := patches.LoadPatchesForScan(patchDir, true)
	if err != nil {
		return HealResult{Failed: 1, Details: []HealDetail{{Action: "failed", Reason: err.Error()}}}
	}

	for _, p := range patchList {
		anchors := p.AnchorStrings
		if len(anchors) == 0 {
			skipped++
			continue
		}

		subs := p.Patches
		if len(subs) == 0 {
			skipped++
			continue
		}
		sub := subs[0]
		sigRegex := sub.SearchRegex
		if sigRegex == "" {
			skipped++
			continue
		}

		// Check if regex already matches
		regexHit := false
		re, err := regexp.Compile("(?s)" + sigRegex)
		if err == nil {
			regexHit = re.MatchString(text)
		}
		if regexHit {
			skipped++
			continue
		}

		// Check applied marker — if already applied, skip healing
		markerApplied := false
		for _, s := range subs {
			if s.AppliedMarker != "" && strings.Contains(text, s.AppliedMarker) {
				markerApplied = true
				break
			}
		}
		if markerApplied {
			skipped++
			continue
		}

		// Try exact anchor match first
		anchorOff := sc.FindAnchor(anchors, 400)
		anchorMethod := "exact"

		// If exact anchors fail, try fuzzy
		if anchorOff == nil {
			foff, fmethod, fconf := sc.FindAnchorFuzzy(anchors, 600)
			if foff != nil && fconf > 0.0 {
				anchorOff = foff
				anchorMethod = fmethod

				// Update anchor_strings to actual text found near the fuzzy offset.
				newAnchors := make([]string, len(anchors))
				for ai, oldA := range anchors {
					tokens := extractLongTokens(oldA)
					// Sort longest-first for best specificity
					sort.Slice(tokens, func(i, j int) bool {
						return len(tokens[i]) > len(tokens[j])
					})
					windowStart := *anchorOff - 200
					if windowStart < 0 {
						windowStart = 0
					}
					windowEnd := *anchorOff + len(oldA) + 800
					if windowEnd > len(text) {
						windowEnd = len(text)
					}
					window := text[windowStart:windowEnd]
					bestAnchor := oldA
					for _, tok := range tokens {
						tokPos := strings.Index(window, tok)
						if tokPos >= 0 {
							// Extract context: 30 chars before token, token, 30 after
							ctxStart := tokPos - 30
							if ctxStart < 0 {
								ctxStart = 0
							}
							ctxEnd := tokPos + len(tok) + 30
							if ctxEnd > len(window) {
								ctxEnd = len(window)
							}
							bestAnchor = strings.TrimSpace(window[ctxStart:ctxEnd])
							break
						}
					}
					newAnchors[ai] = bestAnchor
				}
				anchors = newAnchors
				p.AnchorStrings = newAnchors
			}
		}

		if anchorOff == nil {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: "anchors not found"})
			continue
		}

		// Derive new regex from the context window around the anchor
		newRegex := sc.DeriveRegexWindow(*anchorOff, 80, 150, true)
		if newRegex == "" {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: "derive_regex returned empty"})
			continue
		}

		// Validate: derived regex must actually match, and replacement must fit.
		newRE, err := regexp.Compile("(?s)" + newRegex)
		if err != nil {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: "derived regex does not compile"})
			continue
		}
		m := newRE.FindString(text)
		if m == "" {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: "derived regex does not match"})
			continue
		}

		replaceStr := sub.Replace
		if len(replaceStr) > len(m) {
			failed++
			if verbose {
				fmt.Printf("  SKIP %s: replacement (%d) longer than match (%d)\n", p.ID, len(replaceStr), len(m))
			}
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: "replacement longer than match"})
			continue
		}

		// Update the patch: set new search_regex, keep replace as-is
		p.Patches[0].SearchRegex = newRegex

		// Write back to file
		filePath := p.File
		if filePath == "" {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: "no file path"})
			continue
		}

		// Re-read the original JSON, update fields, and write back.
		// This preserves any fields we don't model in our Go struct.
		rawData, err := os.ReadFile(filePath)
		if err != nil {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: fmt.Sprintf("read file: %v", err)})
			continue
		}

		var rawObj map[string]any
		if err := json.Unmarshal(rawData, &rawObj); err != nil {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: fmt.Sprintf("parse json: %v", err)})
			continue
		}

		// Update search_regex in first sub-patch
		if rawPatches, ok := rawObj["patches"].([]any); ok && len(rawPatches) > 0 {
			if firstSub, ok := rawPatches[0].(map[string]any); ok {
				firstSub["search_regex"] = newRegex
			}
		}

		// Update anchor_strings if they changed (fuzzy heal)
		if anchorMethod != "exact" {
			rawObj["anchor_strings"] = anchors
		}

		outData, err := json.MarshalIndent(rawObj, "", "  ")
		if err != nil {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: fmt.Sprintf("marshal json: %v", err)})
			continue
		}
		outData = append(outData, '\n')

		if err := os.WriteFile(filePath, outData, 0o644); err != nil {
			failed++
			details = append(details, HealDetail{ID: p.ID, Action: "failed", Reason: fmt.Sprintf("write file: %v", err)})
			continue
		}

		if verbose {
			fmt.Printf("  healed %s @ 0x%x (via %s)\n", p.ID, *anchorOff, anchorMethod)
		}
		healed++
		details = append(details, HealDetail{ID: p.ID, Action: "healed", Method: anchorMethod, Offset: *anchorOff})
	}

	return HealResult{Healed: healed, Skipped: skipped, Failed: failed, Details: details}
}

// LoadTextFromTarget extracts patchable text from a cli.js or Bun SEA binary.
// kind is "js" for plain JS file or "bun" for Bun SEA binary.
func LoadTextFromTarget(targetPath, kind string) (string, error) {
	if kind == "js" {
		data, err := os.ReadFile(targetPath)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", targetPath, err)
		}
		return string(data), nil
	}

	// Bun SEA binary — extract .bun section
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("reading binary %s: %w", targetPath, err)
	}

	off, size, err := binary.FindBunSection(data)
	if err != nil {
		return "", fmt.Errorf("finding .bun section: %w", err)
	}

	if off+size > len(data) {
		return "", fmt.Errorf(".bun section bounds exceed file size")
	}

	return string(data[off : off+size]), nil
}
