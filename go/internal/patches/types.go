// Package patches provides data types and loaders for patch JSON definitions.
package patches

// Patch represents a top-level patch definition loaded from a JSON file.
type Patch struct {
	ID             string         `json:"id"`
	Description    string         `json:"description"`
	Type           string         `json:"type"` // "js_replace", "settings", "mcp_guard", "wrapper", "binary_install"
	ValidateJS     bool           `json:"validate_js"`
	ScanSignatures *bool          `json:"scan_signatures,omitempty"` // nil = default true
	Retired        bool           `json:"retired,omitempty"`
	Disabled       bool           `json:"disabled,omitempty"`
	Patches        []SubPatch     `json:"patches"`
	AnchorStrings  []string       `json:"anchor_strings,omitempty"`
	Settings       map[string]any `json:"settings,omitempty"`
	SettingsPath   string         `json:"settings_path,omitempty"`
	Unset          []string       `json:"unset,omitempty"`
	Category       string         `json:"category,omitempty"`

	// File is the source path this patch was loaded from — used by autoheal.
	// Corresponds to Python's __file annotation.
	File string `json:"-"`
}

// SubPatch represents a single search-replace sub-entry within a Patch.
type SubPatch struct {
	Search        string `json:"search,omitempty"`
	SearchRegex   string `json:"search_regex,omitempty"`
	Replace       string `json:"replace"`
	AppliedMarker string `json:"applied_marker,omitempty"`
	Description   string `json:"description,omitempty"`
	Required      *bool  `json:"required,omitempty"` // nil = context-dependent default (true in scanner, false in patcher)
	Count         *int   `json:"count,omitempty"`    // nil = context-dependent default; 0 = replace all, >0 = max
}

// IsRequired returns the effective required flag for patching (default false)
// or scanning (default true) depending on the context.
func (s *SubPatch) IsRequired(defaultVal bool) bool {
	if s.Required == nil {
		return defaultVal
	}
	return *s.Required
}

// EffectiveCount returns the effective count for replacement.
// For patching: nil → 0 (replace all). For scanner all-optional check: nil → 1.
func (s *SubPatch) EffectiveCount(defaultVal int) int {
	if s.Count == nil {
		return defaultVal
	}
	return *s.Count
}

// ScanRow holds the result of scanning a single patch against the target binary.
type ScanRow struct {
	ID           string   `json:"id"`
	Anchors      []string `json:"anchors,omitempty"`
	AnchorOffset *int     `json:"anchor_offset,omitempty"` // nil if not found
	RegexHit     bool     `json:"regex_hit"`
	MarkerHit    bool     `json:"marker_hit"`
	Status       string   `json:"status"` // "ok", "applied", "drift", "retired", "unclassified"
	Confidence   float64  `json:"confidence"`
	Method       string   `json:"method"` // "marker", "anchor", "regex", "scattered", "fuzzy_ws", "fuzzy_ident", "keyword", "optional", "retired"
	AllOptional  bool     `json:"all_optional"`
}

// ShouldScan returns true if this patch should be included in signature scanning.
// Patches with scan_signatures explicitly set to false are excluded.
func (p *Patch) ShouldScan() bool {
	if p.ScanSignatures == nil {
		return true
	}
	return *p.ScanSignatures
}
