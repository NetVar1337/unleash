package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/patches"
)

// selectionFile is ~/.unleash/selection.json — operator enable/disable state.
// Default mode is "all": every non-retired catalog patch is enabled unless listed in Disabled.
// Mode "only": only IDs in Enabled are applied (empty Enabled = nothing).
type selectionState struct {
	Mode     string   `json:"mode"` // "all" | "only"
	Disabled []string `json:"disabled,omitempty"`
	Enabled  []string `json:"enabled,omitempty"`
}

func selectionPath() string {
	return filepath.Join(unleashDir(), "selection.json")
}

func loadSelection() selectionState {
	data, err := os.ReadFile(selectionPath())
	if err != nil {
		return selectionState{Mode: "all"}
	}
	var s selectionState
	if err := json.Unmarshal(data, &s); err != nil {
		return selectionState{Mode: "all"}
	}
	if s.Mode != "only" {
		s.Mode = "all"
	}
	return s
}

func saveSelection(s selectionState) error {
	if err := os.MkdirAll(unleashDir(), 0o755); err != nil {
		return err
	}
	if s.Mode != "only" {
		s.Mode = "all"
	}
	// de-dupe + sort
	s.Disabled = uniqueSorted(s.Disabled)
	s.Enabled = uniqueSorted(s.Enabled)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(selectionPath(), append(data, '\n'), 0o644)
}

func uniqueSorted(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func setFrom(ids []string) map[string]bool {
	m := map[string]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// isPatchSelected reports whether a patch ID is currently selected for apply.
func isPatchSelected(s selectionState, id string) bool {
	switch s.Mode {
	case "only":
		return setFrom(s.Enabled)[id]
	default: // all
		return !setFrom(s.Disabled)[id]
	}
}

// filterPatchesBySelection keeps patches allowed by selection state.
func filterPatchesBySelection(list []patches.Patch, s selectionState) []patches.Patch {
	var out []patches.Patch
	for _, p := range list {
		if isPatchSelected(s, p.ID) {
			out = append(out, p)
		}
	}
	return out
}

// filterPatchesByOnlyExcept applies one-shot --only / --except filters.
// only and except are comma/space-separated ID lists (already split).
func filterPatchesByOnlyExcept(list []patches.Patch, only, except []string) ([]patches.Patch, error) {
	only = uniqueSorted(only)
	except = uniqueSorted(except)
	if len(only) > 0 && len(except) > 0 {
		return nil, fmt.Errorf("use either --only or --except, not both")
	}
	if len(only) == 0 && len(except) == 0 {
		return list, nil
	}
	byID := map[string]patches.Patch{}
	for _, p := range list {
		byID[p.ID] = p
	}
	if len(only) > 0 {
		var out []patches.Patch
		var missing []string
		for _, id := range only {
			p, ok := byID[id]
			if !ok {
				missing = append(missing, id)
				continue
			}
			out = append(out, p)
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("unknown patch id(s): %s", strings.Join(missing, ", "))
		}
		return out, nil
	}
	ex := setFrom(except)
	var out []patches.Patch
	for _, p := range list {
		if !ex[p.ID] {
			out = append(out, p)
		}
	}
	return out, nil
}

// parseIDList splits comma and whitespace separated IDs.
func parseIDList(parts ...string) []string {
	var out []string
	for _, part := range parts {
		for _, f := range strings.FieldsFunc(part, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ';'
		}) {
			if f = strings.TrimSpace(f); f != "" {
				out = append(out, f)
			}
		}
	}
	return out
}

// resolvePatchIDs expands args against catalog; accepts id or unique prefix.
func resolvePatchIDs(catalog []patches.Patch, raw []string) ([]string, error) {
	raw = uniqueSorted(parseIDList(raw...))
	if len(raw) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(catalog))
	for _, p := range catalog {
		ids = append(ids, p.ID)
	}
	var out []string
	var errs []string
	for _, r := range raw {
		if containsID(ids, r) {
			out = append(out, r)
			continue
		}
		// unique prefix match
		var hits []string
		for _, id := range ids {
			if strings.HasPrefix(id, r) {
				hits = append(hits, id)
			}
		}
		if len(hits) == 1 {
			out = append(out, hits[0])
			continue
		}
		if len(hits) > 1 {
			errs = append(errs, fmt.Sprintf("%q is ambiguous (%s)", r, strings.Join(hits, ", ")))
			continue
		}
		errs = append(errs, fmt.Sprintf("unknown patch id %q", r))
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return uniqueSorted(out), nil
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// loadCatalogAll loads active + retired/disabled JSON for listing (disk override preferred).
func loadCatalogAll() []patches.Patch {
	dir := patchDir()
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		if p, err := patches.LoadAllPatches(dir); err == nil && len(p) > 0 {
			return p
		}
	}
	p, _ := patches.LoadAllPatchesFromEmbed()
	return p
}

// NewEnableCmd marks patches selected (re-include after disable, or add to only-mode).
func NewEnableCmd() *cobra.Command {
	var all, only bool
	c := &cobra.Command{
		Use:   "enable [id ...]",
		Short: "Select patches so they apply on the next patch run",
		Long: `Enable (select) patches.

Default selection mode is "all" — every catalog patch is on unless disabled.
  unleash enable <id>          remove id from the disabled list
  unleash enable --all         clear disabled list (everything on)
  unleash enable --only a b    switch to only-mode with just those ids

IDs accept unique prefixes (e.g. js-aup-ref → js-aup-refusal).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnable(all, only, args)
		},
	}
	c.Flags().BoolVar(&all, "all", false, "Enable every patch (clear disabled / leave only-mode)")
	c.Flags().BoolVar(&only, "only", false, "Switch to only-mode with the given ids")
	return c
}

// NewDisableCmd marks patches deselected.
func NewDisableCmd() *cobra.Command {
	var all bool
	c := &cobra.Command{
		Use:   "disable [id ...]",
		Short: "Deselect patches so they are skipped on patch runs",
		Long: `Disable (deselect) patches.

  unleash disable <id>     skip this id on future patch runs
  unleash disable --all    switch to only-mode with empty set (patch applies nothing until enable)

Use 'unleash patch --force' to ignore selection for one run.
Use 'unleash list --selected' / '--deselected' to inspect state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisable(all, args)
		},
	}
	c.Flags().BoolVar(&all, "all", false, "Deselect everything (only-mode, empty enabled set)")
	return c
}

func runEnable(all, only bool, args []string) error {
	catalog := loadAllPatches()
	full := loadCatalogAll()
	if len(full) > len(catalog) {
		catalog = full
	}
	s := loadSelection()

	if all && only {
		return fmt.Errorf("use either --all or --only, not both")
	}
	if all {
		s.Mode = "all"
		s.Disabled = nil
		s.Enabled = nil
		if err := saveSelection(s); err != nil {
			return err
		}
		fmt.Printf("%s%s%s selection: all patches enabled\n", console.G, console.CHECK, console.X)
		return nil
	}
	if len(args) == 0 {
		return fmt.Errorf("pass patch id(s) or --all")
	}
	ids, err := resolvePatchIDs(catalog, args)
	if err != nil {
		return err
	}

	if only {
		s = selectionState{Mode: "only", Enabled: ids}
	} else if s.Mode == "only" {
		en := setFrom(s.Enabled)
		for _, id := range ids {
			en[id] = true
		}
		s.Enabled = mapKeys(en)
	} else {
		dis := setFrom(s.Disabled)
		for _, id := range ids {
			delete(dis, id)
		}
		s.Disabled = mapKeys(dis)
	}
	if err := saveSelection(s); err != nil {
		return err
	}
	fmt.Printf("%s%s%s enabled %d patch(es): %s\n", console.G, console.CHECK, console.X, len(ids), strings.Join(ids, ", "))
	printSelectionSummary(s, loadAllPatches())
	return nil
}

func runDisable(all bool, args []string) error {
	catalog := loadAllPatches()
	full := loadCatalogAll()
	if len(full) > 0 {
		catalog = full
	}
	s := loadSelection()

	if all {
		s.Mode = "only"
		s.Enabled = nil
		s.Disabled = nil
		if err := saveSelection(s); err != nil {
			return err
		}
		fmt.Printf("%s%s%s selection: all patches deselected (only-mode, empty)\n", console.Y, console.WARN, console.X)
		fmt.Printf("  tip: unleash enable <id> …   or   unleash enable --all\n")
		return nil
	}
	if len(args) == 0 {
		return fmt.Errorf("pass patch id(s) or --all")
	}
	ids, err := resolvePatchIDs(catalog, args)
	if err != nil {
		return err
	}

	if s.Mode == "only" {
		en := setFrom(s.Enabled)
		for _, id := range ids {
			delete(en, id)
		}
		s.Enabled = mapKeys(en)
	} else {
		dis := setFrom(s.Disabled)
		for _, id := range ids {
			dis[id] = true
		}
		s.Disabled = mapKeys(dis)
	}
	if err := saveSelection(s); err != nil {
		return err
	}
	fmt.Printf("%s%s%s disabled %d patch(es): %s\n", console.Y, console.WARN, console.X, len(ids), strings.Join(ids, ", "))
	printSelectionSummary(s, loadAllPatches())
	return nil
}

// NewSelectOnlyCmd sets only-mode with explicit IDs.
func NewSelectOnlyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select-only [id ...]",
		Short: "Select only these patches (exclusive mode)",
		Long: `Switch selection to only-mode with the given IDs.

  unleash select-only js-aup-refusal js-metrics-disable
  unleash patch                  # applies just those
  unleash enable --all           # back to default all-on mode
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pass at least one patch id")
			}
			catalog := loadCatalogAll()
			if len(catalog) == 0 {
				catalog = loadAllPatches()
			}
			ids, err := resolvePatchIDs(catalog, args)
			if err != nil {
				return err
			}
			s := selectionState{Mode: "only", Enabled: ids}
			if err := saveSelection(s); err != nil {
				return err
			}
			fmt.Printf("%s%s%s only-mode: %d patch(es) selected\n", console.G, console.CHECK, console.X, len(ids))
			for _, id := range ids {
				fmt.Printf("  · %s\n", id)
			}
			return nil
		},
	}
}

func mapKeys(m map[string]bool) []string {
	var out []string
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func printSelectionSummary(s selectionState, catalog []patches.Patch) {
	nOn, nOff := 0, 0
	for _, p := range catalog {
		if isPatchSelected(s, p.ID) {
			nOn++
		} else {
			nOff++
		}
	}
	fmt.Printf("  mode=%s  selected=%d  deselected=%d  file=%s\n", s.Mode, nOn, nOff, selectionPath())
}
