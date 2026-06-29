package scanner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/VoidChecksum/void-patcher-cc/internal/patches"
)

// Regex helpers
var (
	shortIdentRE    = regexp.MustCompile(`\b[A-Za-z_$][\w$]{0,2}\b`)
	longTokenRE     = regexp.MustCompile(`[A-Za-z_$][\w$]{7,}`)
	whitespaceRunRE = regexp.MustCompile(`\s+`)
)

// collapseWS collapses all whitespace runs to a single space.
func collapseWS(s string) string {
	return strings.TrimSpace(whitespaceRunRE.ReplaceAllString(s, " "))
}

// identAgnosticPattern replaces 1-3 char identifiers with a permissive regex class.
func identAgnosticPattern(s string) string {
	return shortIdentRE.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) <= 3 {
			return `[A-Za-z_$][\w$]*`
		}
		return m
	})
}

// extractLongTokens extracts tokens > 8 chars — good anchor candidates.
// Returns deduplicated list preserving first-occurrence order.
func extractLongTokens(s string) []string {
	matches := longTokenRE.FindAllString(s, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	return result
}

// SigScanner is a signature-driven anchor locator and regex derivation engine.
type SigScanner struct {
	Text   string
	wsText *string // lazy whitespace-collapsed cache
}

// NewSigScanner creates a new SigScanner from text content.
func NewSigScanner(text string) *SigScanner {
	return &SigScanner{Text: text}
}

// getWSText returns the whitespace-collapsed text, built lazily.
func (s *SigScanner) getWSText() string {
	if s.wsText == nil {
		ws := collapseWS(s.Text)
		s.wsText = &ws
	}
	return *s.wsText
}

// FindAnchor returns the first offset where ALL anchors appear within maxDist
// bytes of the first anchor. Returns nil if not found.
func (s *SigScanner) FindAnchor(anchors []string, maxDist int) *int {
	if len(anchors) == 0 {
		return nil
	}
	idx := 0
	for {
		first := strings.Index(s.Text[idx:], anchors[0])
		if first < 0 {
			return nil
		}
		first += idx
		windowEnd := first + len(anchors[0]) + maxDist + 200
		if windowEnd > len(s.Text) {
			windowEnd = len(s.Text)
		}
		window := s.Text[first:windowEnd]
		allFound := true
		for _, a := range anchors[1:] {
			if !strings.Contains(window, a) {
				allFound = false
				break
			}
		}
		if allFound {
			return &first
		}
		idx = first + 1
	}
}

// AllOccurrences returns all offsets where anchor appears in text.
func (s *SigScanner) AllOccurrences(anchor string) []int {
	var offs []int
	idx := 0
	for {
		i := strings.Index(s.Text[idx:], anchor)
		if i < 0 {
			break
		}
		i += idx
		offs = append(offs, i)
		idx = i + 1
	}
	return offs
}

// FindAnchorFuzzy tries progressively relaxed matching for anchors.
// Returns (offset, method_name, confidence).
//
// Strategies in order:
//  1. exact       — plain substring                     (0.9)
//  2. ws_norm     — whitespace-collapsed comparison     (0.7)
//  3. ident_relax — 1-3 char idents replaced with .*?  (0.5)
func (s *SigScanner) FindAnchorFuzzy(anchors []string, maxDist int) (*int, string, float64) {
	// Strategy 1: exact
	off := s.FindAnchor(anchors, maxDist)
	if off != nil {
		return off, "anchor", 0.9
	}

	if len(anchors) == 0 {
		return nil, "none", 0.0
	}

	// Strategy 2: whitespace-normalised
	wsAnchors := make([]string, len(anchors))
	for i, a := range anchors {
		wsAnchors[i] = collapseWS(a)
	}
	wsText := s.getWSText()
	idx := 0
	for {
		wsFirst := strings.Index(wsText[idx:], wsAnchors[0])
		if wsFirst < 0 {
			break
		}
		wsFirst += idx
		winEnd := wsFirst + len(wsAnchors[0]) + maxDist + 200
		if winEnd > len(wsText) {
			winEnd = len(wsText)
		}
		win := wsText[wsFirst:winEnd]
		allFound := true
		for _, a := range wsAnchors[1:] {
			if !strings.Contains(win, a) {
				allFound = false
				break
			}
		}
		if allFound {
			// Map back to approximate raw offset
			searchStart := 0
			if wsFirst > 200 {
				searchStart = wsFirst - 200
			}
			prefix := anchors[0]
			if len(prefix) > 6 {
				prefix = prefix[:6]
			}
			rawGuess := strings.Index(s.Text[searchStart:], prefix)
			if rawGuess >= 0 {
				rawGuess += searchStart
			} else {
				rawGuess = wsFirst
			}
			return &rawGuess, "fuzzy_ws", 0.7
		}
		idx = wsFirst + 1
	}

	// Strategy 3: identifier-agnostic regex (only first anchor to bound cost)
	for _, anchor := range anchors[:1] {
		pat := identAgnosticPattern(regexp.QuoteMeta(anchor))
		re, err := regexp.Compile(pat)
		if err != nil {
			continue
		}
		m := re.FindStringIndex(s.Text)
		if m == nil {
			continue
		}
		off := m[0]
		// Verify rest of anchors within window (relaxed too)
		remainingOK := true
		for _, other := range anchors[1:] {
			opat := identAgnosticPattern(regexp.QuoteMeta(other))
			ore, err := regexp.Compile(opat)
			if err != nil {
				remainingOK = false
				break
			}
			winEnd := off + len(anchor) + maxDist + 200
			if winEnd > len(s.Text) {
				winEnd = len(s.Text)
			}
			win := s.Text[off:winEnd]
			if !ore.MatchString(win) {
				remainingOK = false
				break
			}
		}
		if remainingOK {
			return &off, "fuzzy_ident", 0.5
		}
	}

	return nil, "none", 0.0
}

// KeywordSearch extracts long tokens from anchors and searches for them.
// Returns (offset, confidence). Only reports if ALL keywords are found
// within a reasonable window.
func (s *SigScanner) KeywordSearch(anchors []string) (*int, float64) {
	var keywords []string
	for _, a := range anchors {
		keywords = append(keywords, extractLongTokens(a)...)
	}
	if len(keywords) == 0 {
		return nil, 0.0
	}

	idx := 0
	for {
		firstOff := strings.Index(s.Text[idx:], keywords[0])
		if firstOff < 0 {
			return nil, 0.0
		}
		firstOff += idx
		winStart := firstOff - 200
		if winStart < 0 {
			winStart = 0
		}
		winEnd := firstOff + 1000
		if winEnd > len(s.Text) {
			winEnd = len(s.Text)
		}
		window := s.Text[winStart:winEnd]
		allFound := true
		for _, kw := range keywords[1:] {
			if !strings.Contains(window, kw) {
				allFound = false
				break
			}
		}
		if allFound {
			return &firstOff, 0.3
		}
		idx = firstOff + 1
	}
}

// minifyNames replaces short identifiers with a permissive regex class.
func minifyNames(s string) string {
	return shortIdentRE.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) <= 3 {
			return `[A-Za-z_$][\w$]*`
		}
		return m
	})
}

// DeriveRegex generates an escaped, minifier-tolerant regex around an anchor.
// Returns empty string if anchor not found.
func (s *SigScanner) DeriveRegex(anchor string, before, after int, softmin bool) string {
	i := strings.Index(s.Text, anchor)
	if i < 0 {
		return ""
	}
	start := i - before
	if start < 0 {
		start = 0
	}
	end := i + len(anchor) + after
	if end > len(s.Text) {
		end = len(s.Text)
	}
	ctx := s.Text[start:end]
	esc := regexp.QuoteMeta(ctx)
	if softmin {
		esc = minifyNames(esc)
	}
	return esc
}

// DeriveRegexWindow derives a minifier-tolerant regex from a raw offset + window.
func (s *SigScanner) DeriveRegexWindow(offset, before, after int, softmin bool) string {
	start := offset - before
	if start < 0 {
		start = 0
	}
	end := offset + after
	if end > len(s.Text) {
		end = len(s.Text)
	}
	ctx := s.Text[start:end]
	esc := regexp.QuoteMeta(ctx)
	if softmin {
		esc = minifyNames(esc)
	}
	return esc
}

// BulkMarkerHits performs a bulk marker existence check across all patches.
// Dedupes markers across patches, runs one strings.Contains per unique marker,
// then maps hits back to patch indices.
func (s *SigScanner) BulkMarkerHits(patchList []patches.Patch) map[int]bool {
	markerToIdx := make(map[string][]int)
	for i, p := range patchList {
		for _, sub := range p.Patches {
			if sub.AppliedMarker != "" {
				markerToIdx[sub.AppliedMarker] = append(markerToIdx[sub.AppliedMarker], i)
			}
		}
	}
	if len(markerToIdx) == 0 {
		return nil
	}

	hits := make(map[int]bool)
	seenPatches := make(map[int]bool)

	for marker, idxs := range markerToIdx {
		allSeen := true
		for _, i := range idxs {
			if !seenPatches[i] {
				allSeen = false
				break
			}
		}
		if allSeen {
			continue
		}
		if strings.Contains(s.Text, marker) {
			for _, i := range idxs {
				hits[i] = true
				seenPatches[i] = true
			}
		}
	}
	return hits
}

// ScanPatches runs the full multi-strategy cascade on a list of patches.
// Returns one patches.ScanRow per patch.
//
// Detection strategies (in priority order):
//  1. Bulk marker pre-sweep (applied_marker strings)
//  2. Exact anchor match (co-located within 400 bytes)
//  3. Regex match (search_regex from first sub-patch)
//  3b. Scattered anchors (all present anywhere in text)
//  4. Fuzzy whitespace-normalized anchors
//  5. Fuzzy identifier-agnostic anchors
//  6. Keyword extraction (long tokens > 8 chars)
//  7. All-optional escape hatch
//  8. No anchors/no regex → unclassified
func (s *SigScanner) ScanPatches(patchList []patches.Patch) []patches.ScanRow {
	bulkMarker := s.BulkMarkerHits(patchList)

	var out []patches.ScanRow
	for idx, p := range patchList {
		pid := p.ID
		if pid == "" {
			pid = "?"
		}

		// Retired patches — placeholder row, skip scanning.
		if p.Retired {
			out = append(out, patches.ScanRow{
				ID:      pid,
				Anchors: p.AnchorStrings,
				Status:  "retired",
				Method:  "retired",
			})
			continue
		}

		anchors := p.AnchorStrings
		var sigRegex string
		var markers []string
		subs := p.Patches

		for _, sub := range subs {
			if sigRegex == "" {
				if sub.SearchRegex != "" {
					sigRegex = sub.SearchRegex
				} else if sub.Search != "" {
					sigRegex = sub.Search
				}
			}
			if sub.AppliedMarker != "" {
				markers = append(markers, sub.AppliedMarker)
			}
		}

		// All-optional: every sub is explicitly required=false AND count==0.
		// Uses IsRequired(true) so absent "required" defaults to true (Python
		// semantics: s.get("required", True)), and EffectiveCount(1) so absent
		// "count" defaults to 1 (Python: s.get("count", 1)).
		allOptional := len(subs) > 0
		for _, sub := range subs {
			if sub.IsRequired(true) || sub.EffectiveCount(1) != 0 {
				allOptional = false
				break
			}
		}

		// --- cascade ---
		status := "drift"
		confidence := 0.0
		method := "none"
		var anchorOff *int

		// 1. applied_marker — highest priority, patch already landed
		markerHit := false
		if bulkMarker != nil {
			markerHit = bulkMarker[idx]
		} else {
			for _, m := range markers {
				if strings.Contains(s.Text, m) {
					markerHit = true
					break
				}
			}
		}
		if markerHit {
			status = "applied"
			confidence = 1.0
			method = "marker"
			if len(anchors) > 0 {
				anchorOff = s.FindAnchor(anchors, 400)
			}
		}

		// 2. exact anchor_strings (co-located within max_dist)
		regexHit := false
		if status != "applied" && status != "ok" && len(anchors) > 0 {
			anchorOff = s.FindAnchor(anchors, 400)
			if anchorOff != nil {
				status = "ok"
				confidence = 0.9
				method = "anchor"
			}
		}

		// 3. search_regex — only if anchors didn't prove "ok"
		if status != "applied" && status != "ok" && sigRegex != "" {
			re, err := regexp.Compile("(?s)" + sigRegex)
			if err == nil {
				regexHit = re.MatchString(s.Text)
			}
			if regexHit {
				status = "ok"
				confidence = 1.0
				method = "regex"
				if len(anchors) > 0 {
					anchorOff = s.FindAnchor(anchors, 400)
				}
			}
		}

		// 3b. scattered anchors: all exist anywhere in text
		if status != "applied" && status != "ok" && len(anchors) > 0 {
			allPresent := true
			for _, a := range anchors {
				if !strings.Contains(s.Text, a) {
					allPresent = false
					break
				}
			}
			if allPresent {
				off := strings.Index(s.Text, anchors[0])
				anchorOff = &off
				status = "ok"
				confidence = 0.7
				method = "scattered"
			}
		}

		// 4. fuzzy anchor
		if status != "applied" && status != "ok" && len(anchors) > 0 {
			foff, fmethod, fconf := s.FindAnchorFuzzy(anchors, 600)
			if foff != nil && fconf > 0.0 {
				anchorOff = foff
				status = "ok"
				confidence = fconf
				method = fmethod
			}
		}

		// 5. keyword extraction
		if status != "applied" && status != "ok" && len(anchors) > 0 {
			koff, kconf := s.KeywordSearch(anchors)
			if koff != nil && kconf > 0.0 {
				anchorOff = koff
				status = "ok"
				confidence = kconf
				method = "keyword"
			}
		}

		// All-optional escape hatch: drift but at least one anchor is present
		if status == "drift" && allOptional && len(anchors) > 0 {
			for _, a := range anchors {
				if strings.Contains(s.Text, a) {
					status = "applied"
					confidence = 0.6
					method = "optional"
					break
				}
			}
		}

		// 6. no anchors and no regex → unclassified
		if status == "drift" && len(anchors) == 0 && !regexHit {
			status = "unclassified"
		}

		out = append(out, patches.ScanRow{
			ID:           pid,
			Anchors:      anchors,
			AnchorOffset: anchorOff,
			RegexHit:     regexHit,
			MarkerHit:    markerHit,
			Status:       status,
			Confidence:   confidence,
			Method:       method,
			AllOptional:  allOptional,
		})
	}
	return out
}

// FormatScanReport generates a colored scan report from scan rows.
func FormatScanReport(rows []patches.ScanRow, verbose bool) string {
	const (
		G = "\033[32m"
		Y = "\033[33m"
		R = "\033[31m"
		X = "\033[0m"
	)

	var lines []string
	okCount, driftCount, appliedCount, retiredCount := 0, 0, 0, 0

	for _, row := range rows {
		switch row.Status {
		case "ok":
			okCount++
		case "drift", "unclassified":
			driftCount++
		case "applied":
			appliedCount++
		case "retired":
			retiredCount++
		}

		var statusColor, statusLabel string
		switch row.Status {
		case "ok":
			statusColor = G
			statusLabel = "ok"
		case "applied":
			statusColor = G
			statusLabel = "applied"
		case "drift":
			statusColor = R
			statusLabel = "DRIFT"
		case "retired":
			statusColor = Y
			statusLabel = "retired"
		case "unclassified":
			statusColor = Y
			statusLabel = "unclassified"
		default:
			statusColor = Y
			statusLabel = row.Status
		}

		line := fmt.Sprintf("  %s%-13s%s %s", statusColor, statusLabel, X, row.ID)
		if verbose {
			line += fmt.Sprintf("  [%s %.1f]", row.Method, row.Confidence)
			if row.AnchorOffset != nil {
				line += fmt.Sprintf("  @0x%x", *row.AnchorOffset)
			}
		}
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s%d ok%s  %s%d applied%s  %s%d drift%s",
		G, okCount, X, G, appliedCount, X, R, driftCount, X))
	if retiredCount > 0 {
		lines = append(lines, fmt.Sprintf("  %s%d retired%s", Y, retiredCount, X))
	}

	return strings.Join(lines, "\n")
}
