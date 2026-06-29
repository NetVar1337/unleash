package scanner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/VoidChecksum/unleash/internal/patches"
)

// Regex helpers
var (
	shortIdentRE    = regexp.MustCompile(`\b[A-Za-z_$][\w$]{0,2}\b`)
	longTokenRE     = regexp.MustCompile(`[A-Za-z_$][\w$]{7,}`)
	whitespaceRunRE = regexp.MustCompile(`\s+`)
)

// jsKeywords are JavaScript keywords to preserve exactly in enhanced regex derivation.
var jsKeywords = map[string]bool{
	"function": true, "return": true, "if": true, "else": true,
	"var": true, "let": true, "const": true, "new": true,
	"this": true, "typeof": true, "instanceof": true, "void": true,
	"delete": true, "throw": true, "try": true, "catch": true,
	"finally": true, "switch": true, "case": true, "break": true,
	"continue": true, "for": true, "while": true, "do": true,
	"in": true, "of": true, "class": true, "extends": true,
	"super": true, "import": true, "export": true, "default": true,
	"yield": true, "async": true, "await": true, "null": true,
	"true": true, "false": true, "undefined": true,
}

// commonAnchors are very common JS tokens that reduce anchor confidence.
var commonAnchors = []string{
	"function", "return", "this.", "var ", "const ", "let ",
	"null", "undefined", "true", "false", ".length",
	".prototype", ".call(", ".apply(", ".bind(",
	"toString", "valueOf", "hasOwnProperty",
}

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

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)
	for i := range prev {
		prev[i] = i
	}
	for j := 1; j <= len(b); j++ {
		curr[0] = j
		for i := 1; i <= len(a); i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[i] + 1
			ins := curr[i-1] + 1
			sub := prev[i-1] + cost
			best := del
			if ins < best {
				best = ins
			}
			if sub < best {
				best = sub
			}
			curr[i] = best
		}
		prev, curr = curr, prev
	}
	return prev[len(a)]
}

// structuralFingerprint extracts structural delimiters from s as a fingerprint.
func structuralFingerprint(s string) string {
	var fp []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{', '}', '(', ')', '[', ']', ';', ':', '=', '!', ',':
			fp = append(fp, s[i])
		}
	}
	return string(fp)
}

// isSubsequence checks if sub appears as a subsequence of seq.
func isSubsequence(sub, seq string) bool {
	si := 0
	for i := 0; i < len(seq) && si < len(sub); i++ {
		if seq[i] == sub[si] {
			si++
		}
	}
	return si == len(sub)
}

// isIdentStart returns true if c can start a JS identifier.
func isIdentStart(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' || c == '$'
}

// isIdentCont returns true if c can continue a JS identifier.
func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// buildEnhancedPattern generates a minifier-resilient regex from raw JS source:
//   - 1-4 char identifiers → [A-Za-z_$][\w$]*
//   - string literals preserved exactly
//   - JS keywords preserved exactly
//   - whitespace collapsed to \s*
//   - structural tokens preserved (escaped)
func buildEnhancedPattern(src string) string {
	var out strings.Builder
	i := 0
	for i < len(src) {
		c := src[i]
		// String literals — preserve exactly
		if c == '"' || c == '\'' || c == '`' {
			quote := c
			out.WriteString(regexp.QuoteMeta(string(c)))
			i++
			for i < len(src) && src[i] != quote {
				if src[i] == '\\' && i+1 < len(src) {
					out.WriteString(regexp.QuoteMeta(src[i : i+2]))
					i += 2
				} else {
					out.WriteString(regexp.QuoteMeta(string(src[i])))
					i++
				}
			}
			if i < len(src) {
				out.WriteString(regexp.QuoteMeta(string(src[i])))
				i++
			}
			continue
		}
		// Whitespace — collapse to \s*
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			out.WriteString(`\s*`)
			for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
				i++
			}
			continue
		}
		// Identifiers / keywords
		if isIdentStart(c) {
			j := i + 1
			for j < len(src) && isIdentCont(src[j]) {
				j++
			}
			word := src[i:j]
			if jsKeywords[word] {
				out.WriteString(regexp.QuoteMeta(word))
			} else if len(word) <= 4 {
				out.WriteString(`[A-Za-z_$][\w$]*`)
			} else {
				out.WriteString(regexp.QuoteMeta(word))
			}
			i = j
			continue
		}
		// Structural / other chars — escape for regex safety
		out.WriteString(regexp.QuoteMeta(string(c)))
		i++
	}
	return out.String()
}

// AdjustConfidence modifies base confidence based on anchor quality:
//   - more anchors → higher confidence (up to +0.1)
//   - longer anchors → higher confidence (+0.05)
//   - very common patterns → penalized (-0.1)
func AdjustConfidence(base float64, anchors []string) float64 {
	if len(anchors) == 0 {
		return base
	}
	conf := base
	// Anchor count bonus
	if len(anchors) >= 3 {
		conf += 0.1
	} else if len(anchors) >= 2 {
		conf += 0.05
	}
	// Anchor specificity: average length
	totalLen := 0
	for _, a := range anchors {
		totalLen += len(a)
	}
	avgLen := totalLen / len(anchors)
	if avgLen > 40 {
		conf += 0.05
	} else if avgLen < 15 {
		conf -= 0.05
	}
	// Penalize when all anchors contain very common patterns
	commonCount := 0
	for _, a := range anchors {
		for _, cp := range commonAnchors {
			if strings.Contains(a, cp) {
				commonCount++
				break
			}
		}
	}
	if commonCount == len(anchors) {
		conf -= 0.1
	}
	if conf > 1.0 {
		conf = 1.0
	}
	if conf < 0.05 {
		conf = 0.05
	}
	return conf
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
//  2b. ngram      — n-gram cluster voting               (0.3–0.7)
//  3. ident_relax — 1-3 char idents replaced with .*?  (0.5)
//  4. levenshtein — edit distance ≤ 3                   (0.35–0.5)
//  5. structural  — delimiter fingerprint match         (0.35)
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

	// Strategy 2b: n-gram cluster matching
	if noff, nconf := s.FindAnchorNgram(anchors, maxDist); noff != nil && nconf > 0.2 {
		return noff, "ngram", nconf
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

	// Strategy 4: Levenshtein edit distance (within distance 3)
	if loff, lconf := s.FindAnchorLevenshtein(anchors, maxDist); loff != nil && lconf > 0.2 {
		return loff, "levenshtein", lconf
	}

	// Strategy 5: structural context matching (delimiter fingerprint)
	if soff, sconf := s.FindAnchorStructural(anchors, maxDist); soff != nil && sconf > 0.2 {
		return soff, "structural", sconf
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

// FindAnchorNgram uses n-gram indexing for fast approximate anchor matching.
// Extracts 6-char n-grams from anchors, finds position clusters in text via
// bucket voting. Returns (offset, confidence) or (nil, 0).
func (s *SigScanner) FindAnchorNgram(anchors []string, maxDist int) (*int, float64) {
	if len(anchors) == 0 {
		return nil, 0.0
	}
	const gramSize = 6
	const gramStep = 3
	seen := make(map[string]bool)
	var grams []string
	for _, a := range anchors {
		for i := 0; i+gramSize <= len(a); i += gramStep {
			g := a[i : i+gramSize]
			if !seen[g] {
				seen[g] = true
				grams = append(grams, g)
			}
		}
	}
	if len(grams) < 2 {
		return nil, 0.0
	}
	// Vote for 256-byte buckets where n-grams appear
	const bucketSize = 256
	votes := make(map[int]int)
	for _, g := range grams {
		idx := 0
		for hits := 0; hits < 50; hits++ {
			pos := strings.Index(s.Text[idx:], g)
			if pos < 0 {
				break
			}
			pos += idx
			votes[pos/bucketSize]++
			idx = pos + 1
		}
	}
	if len(votes) == 0 {
		return nil, 0.0
	}
	bestBucket, bestVotes := 0, 0
	for b, v := range votes {
		if v > bestVotes {
			bestVotes = v
			bestBucket = b
		}
	}
	threshold := len(grams) * 3 / 10
	if threshold < 3 {
		threshold = 3
	}
	if bestVotes < threshold {
		return nil, 0.0
	}
	conf := float64(bestVotes) / float64(len(grams))
	if conf > 0.7 {
		conf = 0.7
	}
	off := bestBucket * bucketSize
	return &off, conf
}

// FindAnchorLevenshtein uses edit distance to find approximate anchor matches.
// Extracts long tokens as search seeds, then verifies surrounding windows
// have Levenshtein distance ≤ 3 from the anchor.
func (s *SigScanner) FindAnchorLevenshtein(anchors []string, maxDist int) (*int, float64) {
	if len(anchors) == 0 {
		return nil, 0.0
	}
	anchor := anchors[0]
	if len(anchor) < 10 {
		return nil, 0.0 // too short for meaningful edit distance
	}
	tokens := extractLongTokens(anchor)
	if len(tokens) == 0 {
		return nil, 0.0
	}
	bestDist := 4  // > max allowed distance
	bestOff := -1
	anchorLen := len(anchor)
	// Cap anchor length for Levenshtein to keep cost bounded
	cmpAnchor := anchor
	if len(cmpAnchor) > 120 {
		cmpAnchor = cmpAnchor[:120]
		anchorLen = 120
	}
	for _, tok := range tokens {
		if len(tok) < 6 {
			continue
		}
		idx := 0
		for attempts := 0; attempts < 20; attempts++ {
			pos := strings.Index(s.Text[idx:], tok)
			if pos < 0 {
				break
			}
			pos += idx
			// Try windows around the token position
			for _, delta := range []int{0, anchorLen / 4, anchorLen / 2} {
				winStart := pos - delta
				if winStart < 0 {
					winStart = 0
				}
				winEnd := winStart + anchorLen + 4
				if winEnd > len(s.Text) {
					winEnd = len(s.Text)
				}
				candidate := s.Text[winStart:winEnd]
				if len(candidate) > anchorLen+4 {
					candidate = candidate[:anchorLen+4]
				}
				dist := levenshtein(cmpAnchor, candidate)
				if dist < bestDist {
					bestDist = dist
					bestOff = winStart
				}
			}
			idx = pos + 1
		}
	}
	if bestDist > 3 || bestOff < 0 {
		return nil, 0.0
	}
	conf := 0.5 - float64(bestDist)*0.05
	// Verify remaining anchors are in the vicinity
	if len(anchors) > 1 && bestOff < len(s.Text) {
		winEnd := bestOff + anchorLen + maxDist
		if winEnd > len(s.Text) {
			winEnd = len(s.Text)
		}
		window := s.Text[bestOff:winEnd]
		miss := 0
		for _, other := range anchors[1:] {
			if !strings.Contains(window, other) {
				otherTokens := extractLongTokens(other)
				found := false
				for _, ot := range otherTokens {
					if strings.Contains(window, ot) {
						found = true
						break
					}
				}
				if !found {
					miss++
				}
			}
		}
		if miss > len(anchors)/2 {
			conf -= 0.15
		}
	}
	if conf < 0.1 {
		return nil, 0.0
	}
	return &bestOff, conf
}

// FindAnchorStructural matches anchors by structural delimiter fingerprint.
// Extracts delimiters ({, }, (, ), ;, etc.) from anchors, then searches
// for regions with matching structural patterns using long tokens as guides.
func (s *SigScanner) FindAnchorStructural(anchors []string, maxDist int) (*int, float64) {
	if len(anchors) == 0 {
		return nil, 0.0
	}
	combined := strings.Join(anchors, "")
	targetFP := structuralFingerprint(combined)
	if len(targetFP) < 3 {
		return nil, 0.0 // need enough structural content
	}
	tokens := extractLongTokens(combined)
	if len(tokens) == 0 {
		return nil, 0.0
	}
	for _, tok := range tokens {
		idx := 0
		for attempts := 0; attempts < 30; attempts++ {
			pos := strings.Index(s.Text[idx:], tok)
			if pos < 0 {
				break
			}
			pos += idx
			winStart := pos - len(combined)
			if winStart < 0 {
				winStart = 0
			}
			winEnd := pos + len(combined) + maxDist
			if winEnd > len(s.Text) {
				winEnd = len(s.Text)
			}
			window := s.Text[winStart:winEnd]
			windowFP := structuralFingerprint(window)
			if isSubsequence(targetFP, windowFP) {
				return &pos, 0.35
			}
			idx = pos + 1
		}
	}
	return nil, 0.0
}

// DeriveRegexEnhanced generates a minifier-resilient regex around an anchor offset.
// Unlike DeriveRegex/DeriveRegexWindow, it:
//   - replaces 1-4 char identifiers with wildcard patterns
//   - preserves string literals exactly
//   - preserves JS keywords
//   - collapses whitespace to \s*
//   - keeps structural tokens exact
func (s *SigScanner) DeriveRegexEnhanced(offset, before, after int) string {
	start := offset - before
	if start < 0 {
		start = 0
	}
	end := offset + after
	if end > len(s.Text) {
		end = len(s.Text)
	}
	return buildEnhancedPattern(s.Text[start:end])
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

		// Adjust confidence based on anchor quality
		confidence = AdjustConfidence(confidence, anchors)

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
