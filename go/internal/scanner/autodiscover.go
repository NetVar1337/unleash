package scanner

import (
	"regexp"
	"strings"

	"github.com/VoidChecksum/void-patcher-cc/internal/patches"
)

// DiscoveredPattern represents a newly discovered patchable pattern in the binary.
type DiscoveredPattern struct {
	Category       string  `json:"category"`
	Description    string  `json:"description"`
	Offset         int     `json:"offset"`
	Context        string  `json:"context"`
	SearchStr      string  `json:"search_str"`
	Confidence     float64 `json:"confidence"`
	AlreadyPatched bool    `json:"already_patched"`
}

// OffsetUpdate records a discovered new location for a patch whose search pattern drifted.
type OffsetUpdate struct {
	PatchID    string  `json:"patch_id"`
	OldPattern string  `json:"old_pattern"`
	NewPattern string  `json:"new_pattern"`
	NewOffset  int     `json:"new_offset"`
	Confidence float64 `json:"confidence"`
	Method     string  `json:"method"`
}

// Pre-compiled pattern sets for each discovery category.
var (
	refusalREs = []*regexp.Regexp{
		regexp.MustCompile(`(?i)I\s+cannot\s+(?:help|assist|do)`),
		regexp.MustCompile(`(?i)not\s+authorized\s+to`),
		regexp.MustCompile(`(?i)against\s+[\w\s]*policy`),
		regexp.MustCompile(`(?i)acceptable\s+use\s+policy`),
		regexp.MustCompile(`(?i)I'?m\s+not\s+able\s+to`),
		regexp.MustCompile(`(?i)I\s+can'?t\s+help\s+with`),
		regexp.MustCompile(`(?i)I\s+am\s+unable\s+to`),
		regexp.MustCompile(`(?i)not\s+(?:allowed|permitted)`),
		regexp.MustCompile(`(?i)violat(?:es?|ing)\s+(?:our|the)\s+(?:policy|guidelines|terms)`),
		regexp.MustCompile(`(?i)content\s+policy`),
		regexp.MustCompile(`(?i)usage\s+policy`),
	}

	gateREs = []*regexp.Regexp{
		regexp.MustCompile(`\{allowed:\s*!1\}`),
		regexp.MustCompile(`\{safe:\s*!1\}`),
		regexp.MustCompile(`shouldBlock\b`),
		regexp.MustCompile(`isBlocked\b`),
		regexp.MustCompile(`isRestricted\b`),
		regexp.MustCompile(`permissionDenied\b`),
		regexp.MustCompile(`accessDenied\b`),
		regexp.MustCompile(`blockRequest\b`),
		regexp.MustCompile(`denyAccess\b`),
		regexp.MustCompile(`allowed\s*=\s*!1`),
		regexp.MustCompile(`blocked\s*=\s*!0`),
	}

	telemetryREs = []*regexp.Regexp{
		regexp.MustCompile(`datadoghq\.com`),
		regexp.MustCompile(`sentry\.io`),
		regexp.MustCompile(`amplitude\.com`),
		regexp.MustCompile(`segment\.(?:com|io)`),
		regexp.MustCompile(`statsig\.com`),
		regexp.MustCompile(`mixpanel\.com`),
		regexp.MustCompile(`(?i)https?://[^\s"']+(?:analytics|telemetry|tracking)\.[^\s"']+`),
	}

	featureFlagREs = []*regexp.Regexp{
		regexp.MustCompile(`tengu_\w+\s*[=:]\s*!1`),
		regexp.MustCompile(`tengu_\w+\s*[=:]\s*false`),
		regexp.MustCompile(`feature_\w+\s*[=:]\s*!1`),
		regexp.MustCompile(`enableFeature\w*\s*[=:]\s*!1`),
	}

	killSwitchREs = []*regexp.Regexp{
		regexp.MustCompile(`process\.exit\s*\(`),
		regexp.MustCompile(`bypass_permissions_disabled`),
		regexp.MustCompile(`(?i)remote[_\s]?disable`),
		regexp.MustCompile(`(?i)kill[_\s]?switch`),
		regexp.MustCompile(`(?i)emergency[_\s]?stop`),
		regexp.MustCompile(`(?i)force[_\s]?shutdown`),
	}

	attributionREs = []*regexp.Regexp{
		regexp.MustCompile(`Co-Authored-By`),
		regexp.MustCompile(`(?i)Generated\s+with`),
		regexp.MustCompile(`(?i)generated\s+by\s+(?:Claude|AI|GPT)`),
		regexp.MustCompile(`(?i)AI-generated`),
		regexp.MustCompile(`(?i)written\s+by\s+(?:Claude|AI|an?\s+AI)`),
		regexp.MustCompile(`(?i)co-?author(?:ed)?`),
	}
)

// searchWithPatterns is the generic discovery engine for a regex pattern set.
func searchWithPatterns(text string, patterns []*regexp.Regexp, category, descPrefix string, existingPatches []patches.Patch) []DiscoveredPattern {
	var results []DiscoveredPattern
	seen := make(map[int]bool) // dedupe by 100-byte bucket

	for _, re := range patterns {
		matches := re.FindAllStringIndex(text, -1)
		for _, m := range matches {
			offset := m[0]
			bucket := offset / 100
			if seen[bucket] {
				continue
			}
			seen[bucket] = true

			matchStr := text[m[0]:m[1]]

			// Extract surrounding context (100 chars each side)
			ctxStart := offset - 100
			if ctxStart < 0 {
				ctxStart = 0
			}
			ctxEnd := m[1] + 100
			if ctxEnd > len(text) {
				ctxEnd = len(text)
			}
			ctx := text[ctxStart:ctxEnd]

			// Confidence based on match length / specificity
			conf := 0.5
			if len(matchStr) > 20 {
				conf = 0.7
			} else if len(matchStr) > 10 {
				conf = 0.6
			}

			alreadyPatched := isAlreadyPatched(matchStr, existingPatches)
			if alreadyPatched {
				conf *= 0.5
			}

			results = append(results, DiscoveredPattern{
				Category:       category,
				Description:    descPrefix + ": " + truncateStr(matchStr, 60),
				Offset:         offset,
				Context:        ctx,
				SearchStr:      matchStr,
				Confidence:     conf,
				AlreadyPatched: alreadyPatched,
			})
		}
	}
	return results
}

// isAlreadyPatched checks whether matchStr is covered by an existing patch.
func isAlreadyPatched(matchStr string, existingPatches []patches.Patch) bool {
	for _, p := range existingPatches {
		for _, sub := range p.Patches {
			if sub.Search != "" && strings.Contains(sub.Search, matchStr) {
				return true
			}
			if sub.SearchRegex != "" {
				re, err := regexp.Compile("(?s)" + sub.SearchRegex)
				if err == nil && re.MatchString(matchStr) {
					return true
				}
			}
			if sub.AppliedMarker != "" && strings.Contains(matchStr, sub.AppliedMarker) {
				return true
			}
		}
		for _, a := range p.AnchorStrings {
			if strings.Contains(a, matchStr) || strings.Contains(matchStr, a) {
				return true
			}
		}
	}
	return false
}

// truncateStr shortens s to maxLen, appending "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SearchForRefusals finds refusal/restriction strings in the binary text.
func SearchForRefusals(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	return searchWithPatterns(text, refusalREs, "refusal", "Refusal pattern", existingPatches)
}

// SearchForGates finds permission gate patterns like {allowed:!1}, shouldBlock.
func SearchForGates(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	return searchWithPatterns(text, gateREs, "permissions", "Permission gate", existingPatches)
}

// SearchForTelemetry finds URLs to tracking/analytics domains.
func SearchForTelemetry(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	return searchWithPatterns(text, telemetryREs, "telemetry", "Telemetry endpoint", existingPatches)
}

// SearchForFeatureFlags finds feature flags set to !1 (disabled).
func SearchForFeatureFlags(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	return searchWithPatterns(text, featureFlagREs, "features", "Feature flag", existingPatches)
}

// SearchForKillSwitches finds process.exit, remote disable, and similar kill patterns.
func SearchForKillSwitches(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	return searchWithPatterns(text, killSwitchREs, "infrastructure", "Kill switch", existingPatches)
}

// SearchForAttribution finds Co-Authored-By, Generated with, and attribution footers.
func SearchForAttribution(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	return searchWithPatterns(text, attributionREs, "attribution", "Attribution pattern", existingPatches)
}

// DiscoverAll runs all discovery searches and returns combined results.
func DiscoverAll(text string, existingPatches []patches.Patch) []DiscoveredPattern {
	var all []DiscoveredPattern
	all = append(all, SearchForRefusals(text, existingPatches)...)
	all = append(all, SearchForGates(text, existingPatches)...)
	all = append(all, SearchForTelemetry(text, existingPatches)...)
	all = append(all, SearchForFeatureFlags(text, existingPatches)...)
	all = append(all, SearchForKillSwitches(text, existingPatches)...)
	all = append(all, SearchForAttribution(text, existingPatches)...)
	return all
}

// FindNewOffsets attempts to locate drifted patches in the binary.
// For each patch whose search pattern no longer matches, it uses anchor_strings
// to locate the region and derives a new regex from the surrounding context.
func FindNewOffsets(text string, patchList []patches.Patch) []OffsetUpdate {
	sc := NewSigScanner(text)
	var updates []OffsetUpdate

	for _, p := range patchList {
		if p.Retired || len(p.Patches) == 0 {
			continue
		}

		sub := p.Patches[0]
		sigRegex := sub.SearchRegex
		if sigRegex == "" && sub.Search == "" {
			continue
		}

		// Check if current pattern still matches — skip if it does
		currentMatch := false
		if sigRegex != "" {
			re, err := regexp.Compile("(?s)" + sigRegex)
			if err == nil {
				currentMatch = re.MatchString(text)
			}
		} else if sub.Search != "" {
			currentMatch = strings.Contains(text, sub.Search)
		}
		if currentMatch {
			continue
		}

		anchors := p.AnchorStrings
		if len(anchors) == 0 {
			continue
		}

		// Full strategy cascade to find where the anchor region moved
		var anchorOff *int
		var method string

		anchorOff = sc.FindAnchor(anchors, 400)
		method = "exact_anchor"

		if anchorOff == nil {
			if noff, nconf := sc.FindAnchorNgram(anchors, 600); noff != nil && nconf > 0.2 {
				anchorOff = noff
				method = "ngram"
			}
		}

		if anchorOff == nil {
			if loff, lconf := sc.FindAnchorLevenshtein(anchors, 600); loff != nil && lconf > 0.2 {
				anchorOff = loff
				method = "levenshtein"
			}
		}

		if anchorOff == nil {
			foff, fmethod, fconf := sc.FindAnchorFuzzy(anchors, 600)
			if foff != nil && fconf > 0.0 {
				anchorOff = foff
				method = fmethod
			}
		}

		if anchorOff == nil {
			continue
		}

		// Derive a new regex from the discovered region
		newRegex := sc.DeriveRegexEnhanced(*anchorOff, 80, 150)
		if newRegex == "" {
			continue
		}

		// Validate the derived regex actually matches
		newRE, err := regexp.Compile("(?s)" + newRegex)
		if err != nil {
			continue
		}
		if newRE.FindString(text) == "" {
			continue
		}

		conf := 0.5
		switch method {
		case "exact_anchor":
			conf = 0.9
		case "ngram":
			conf = 0.6
		case "levenshtein":
			conf = 0.5
		case "fuzzy_ws":
			conf = 0.7
		case "fuzzy_ident":
			conf = 0.4
		case "structural":
			conf = 0.45
		}

		oldPattern := sigRegex
		if oldPattern == "" {
			oldPattern = sub.Search
		}

		updates = append(updates, OffsetUpdate{
			PatchID:    p.ID,
			OldPattern: truncateStr(oldPattern, 80),
			NewPattern: truncateStr(newRegex, 80),
			NewOffset:  *anchorOff,
			Confidence: conf,
			Method:     method,
		})
	}

	return updates
}
