package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/binary"
	"github.com/VoidChecksum/unleash/internal/bytepatch"
	"github.com/VoidChecksum/unleash/internal/console"
	"github.com/VoidChecksum/unleash/internal/patches"
	"github.com/VoidChecksum/unleash/internal/target"
	"github.com/VoidChecksum/unleash/internal/updater"
)

// NewPatchCmd creates the "patch" cobra command.
func NewPatchCmd() *cobra.Command {
	var dryRun, force bool
	var only, except string
	c := &cobra.Command{
		Use:   "patch [id ...]",
		Short: "Apply selected patches (default: all enabled)",
		Long: `Apply patches to every discovered Claude Code install.

Selection (persistent):
  unleash disable <id>     skip on future runs
  unleash enable <id>      re-include
  unleash enable --all     everything on
  unleash select-only a b  exclusive set

One-shot filters:
  unleash patch --only js-aup-refusal,js-metrics-disable
  unleash patch --except js-webfetch-lyrics-copyright-clause
  unleash patch js-aup-refusal          # same as --only
  unleash patch --force                 # ignore enable/disable selection
  unleash patch -n                      # dry-run
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPatch(dryRun, force, only, except, args)
		},
	}
	c.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without writing")
	c.Flags().BoolVar(&force, "force", false, "Ignore enable/disable selection (apply full catalog)")
	c.Flags().StringVar(&only, "only", "", "Comma-separated patch ids to apply (one-shot)")
	c.Flags().StringVar(&except, "except", "", "Comma-separated patch ids to skip (one-shot)")
	return c
}

func runPatch(dryRun, force bool, only, except string, args []string) error {
	patchList := loadAllPatches()
	sel := loadSelection()

	// Persistent selection unless --force or explicit --only/--except/args
	hasOnly := strings.TrimSpace(only) != "" || len(args) > 0
	hasExcept := strings.TrimSpace(except) != ""
	if !force && !hasOnly && !hasExcept {
		before := len(patchList)
		patchList = filterPatchesBySelection(patchList, sel)
		if len(patchList) < before {
			fmt.Printf("  %sselection: %d/%d patches (mode=%s)%s\n",
				console.Y, len(patchList), before, sel.Mode, console.X)
		}
	}

	// One-shot only/except + positional ids
	onlyIDs := parseIDList(only)
	onlyIDs = append(onlyIDs, parseIDList(args...)...)
	exceptIDs := parseIDList(except)
	if len(onlyIDs) > 0 || len(exceptIDs) > 0 {
		// resolve against full active catalog for prefix match
		catalog := loadAllPatches()
		var err error
		if len(onlyIDs) > 0 {
			onlyIDs, err = resolvePatchIDs(catalog, onlyIDs)
			if err != nil {
				return err
			}
		}
		if len(exceptIDs) > 0 {
			exceptIDs, err = resolvePatchIDs(catalog, exceptIDs)
			if err != nil {
				return err
			}
		}
		base := catalog
		if force {
			base = catalog
		} else if !hasOnly && !hasExcept {
			base = patchList
		}
		// when --only given, start from full catalog; when --except, from selection-filtered
		if len(onlyIDs) > 0 {
			base = catalog
		}
		patchList, err = filterPatchesByOnlyExcept(base, onlyIDs, exceptIDs)
		if err != nil {
			return err
		}
	}

	if len(patchList) == 0 {
		fmt.Printf("%sno patches selected — nothing to do%s\n", console.Y, console.X)
		fmt.Printf("  tip: unleash enable --all   or   unleash patch --force\n")
		return nil
	}

	all := target.FindAllTargets()

	var jsPatches, metaPatches []patches.Patch
	for _, p := range patchList {
		if p.Type == "js_replace" {
			jsPatches = append(jsPatches, p)
		} else {
			metaPatches = append(metaPatches, p)
		}
	}

	fmt.Printf("%sunleash patch — %d patches%s\n", console.B, len(patchList), console.X)

	var kind, tgt string
	if len(all) > 0 {
		tgt, kind = all[0].Path, all[0].Kind
	}

	if len(all) == 0 {
		fmt.Printf("  %starget not found — settings-only patches will apply%s\n", console.Y, console.X)
	}

	ok, fail, skip, try := 0, 0, 0, 0

	// ── JS patches — every discovered installation ───────────────────────
	if len(all) == 0 {
		for _, p := range jsPatches {
			fmt.Printf("  %sskip%s %s  target not found\n", console.Y, console.X, padRight(p.ID, 40))
			skip++
		}
	}
	for i, f := range all {
		label := "cli.js"
		if f.Kind != "js" {
			label = fmt.Sprintf("Bun SEA %s", filepath.Base(f.Path))
		}
		if len(all) > 1 {
			fmt.Printf("\n  %s── target %d/%d ──%s\n", console.B, i+1, len(all), console.X)
		}
		fmt.Printf("  target : %s\n", f.Path)
		fmt.Printf("  format : %s\n", label)
		fmt.Printf("  sha    : %s\n", target.SHA256Short(f.Path))
		if !dryRun {
			bkp, err := target.Backup(f.Path, f.Kind)
			if err == nil {
				fmt.Printf("  backup : %s\n", bkp)
			}
		}
		o, fl, sk, tr := patchJSTarget(f.Path, f.Kind, jsPatches, dryRun)
		ok += o
		fail += fl
		skip += sk
		try += tr
	}

	// ── Non-JS patches ────────────────────────────────────────────────────
	for _, p := range metaPatches {
		var success bool
		var msg string
		switch p.Type {
		case "settings", "hook":
			success, msg = applySettings(p, dryRun)
		case "mcp_guard":
			success, msg = applyMCPGuard(p, dryRun)
		case "wrapper":
			success, msg = applyWrapper(p, coalesce(kind, "bun_sea"), tgt, dryRun)
		case "binary_install":
			success, msg = applyBinaryInstall(p, coalesce(kind, "bun_sea"), dryRun)
		default:
			fmt.Printf("  %sskip%s %s  type=%s (unknown)\n", console.Y, console.X, padRight(p.ID, 40), p.Type)
			skip++
			continue
		}
		mark := fmt.Sprintf("%sok%s", console.G, console.X)
		if !success {
			mark = fmt.Sprintf("%sfail%s", console.R, console.X)
		}
		fmt.Printf("  %s   %s  %s\n", mark, padRight(p.ID, 40), msg)
		if success {
			ok++
		} else {
			fail++
		}
	}

	if try > 0 {
		fmt.Printf("\n%s%d ok %s %d would try %s %d failed %s %d skipped%s\n",
			console.B, ok, console.DOT, try, console.DOT, fail, console.DOT, skip, console.X)
	} else {
		fmt.Printf("\n%s%d ok %s %d failed %s %d skipped%s\n",
			console.B, ok, console.DOT, fail, console.DOT, skip, console.X)
	}

	// Record state for autoheal drift detection
	if len(all) > 0 && !dryRun && fail == 0 {
		updater.SaveState(map[string]interface{}{
			"last_cc_sha":  target.SHA256Short(tgt),
			"last_cc_kind": kind,
		})

		// Stamp SHA manifest for the guard fast-path (one line per target)
		writeSHAManifest(all)
	}

	if fail > 0 {
		return fmt.Errorf("patch failed")
	}
	return nil
}

// shaManifestPath is the guard's per-target SHA record.
func shaManifestPath() string {
	return filepath.Join(unleashDir(), "last_patched_sha")
}

// writeSHAManifest records "sha  path" lines for every patched target.
func writeSHAManifest(all []target.Found) {
	var sb strings.Builder
	for _, f := range all {
		sb.WriteString(target.SHA256Short(f.Path))
		sb.WriteString("  ")
		sb.WriteString(f.Path)
		sb.WriteString("\n")
	}
	os.MkdirAll(unleashDir(), 0o755)
	os.WriteFile(shaManifestPath(), []byte(sb.String()), 0o644)
}

// readSHAManifest parses the guard manifest into path → sha.
func readSHAManifest() map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(shaManifestPath())
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 {
			out[parts[1]] = parts[0]
			continue
		}
		// legacy single-sha format — no path association
		out[""] = line
	}
	return out
}

// patchJSTarget applies all js_replace patches to one target binary and
// returns (ok, fail, skip, try) counters.
func patchJSTarget(tgt, kind string, jsPatches []patches.Patch, dryRun bool) (ok, fail, skip, try int) {
	if kind != "bun_sea" {
		// cli.js path — not ported (legacy, very rare); print skip
		for _, p := range jsPatches {
			fmt.Printf("  %sskip%s %s  cli.js patching not supported in Go build\n",
				console.Y, console.X, padRight(p.ID, 40))
			skip++
		}
		return
	}

	if dryRun {
		data, err := os.ReadFile(tgt)
		if err != nil {
			fmt.Printf("  %sfail%s  [bun-inplace]  %v\n", console.R, console.X, err)
			fail += len(jsPatches)
			return
		}
		bunOff, bunSize, err := binary.FindBunSection(data)
		if err != nil {
			fmt.Printf("  %sfail%s  [bun-inplace]  %v\n", console.R, console.X, err)
			fail += len(jsPatches)
			return
		}
		effLo, effHi := binary.FindActiveBundleBounds(data, bunOff, bunOff+bunSize)
		eff := data[effLo:effHi]
		for _, p := range jsPatches {
			appliedN := 0
			markerSeen := false
			hasCandidate := false
			for _, sub := range p.Patches {
				if sub.AppliedMarker != "" && bytesContains(eff, []byte(sub.AppliedMarker)) {
					markerSeen = true
					continue
				}
				sr := sub.SearchRegex
				if sr == "" {
					sr = sub.Search
				}
				if sr == "" {
					continue
				}
				hasCandidate = true
				if sub.SearchRegex != "" {
					appliedN += regexCountMatchesN(sr, eff, sub.EffectiveCount(0))
				} else {
					appliedN += bytesCountN(eff, []byte(sr), sub.EffectiveCount(0))
				}
			}
			var msg, mark string
			switch {
			case appliedN > 0:
				msg = fmt.Sprintf("would try %d in-place replacement(s); verification only runs without --dry-run", appliedN)
				mark = fmt.Sprintf("%stry%s", console.Y, console.X)
				try++
			case markerSeen:
				msg = "no-op (already applied)"
				mark = fmt.Sprintf("%sok%s", console.G, console.X)
				ok++
			case !hasCandidate:
				msg = "no-op (no subpatches)"
				mark = fmt.Sprintf("%sok%s", console.G, console.X)
				ok++
			default:
				msg = "drift (search not found in this version)"
				mark = fmt.Sprintf("%sdrift%s", console.R, console.X)
				skip++
			}
			fmt.Printf("  %s    %s  %s\n", mark, padRight(p.ID, 40), msg)
		}
		fmt.Printf("  %sdry-run: binary not modified%s\n", console.Y, console.X)
		return
	}

	for _, p := range jsPatches {
		result := binary.PatchBunSEAInplace(tgt, []patches.Patch{p})
		if !result.OK {
			fmt.Printf("  %sfail%s  %s  %s\n", console.R, console.X, padRight(p.ID, 40), result.Err)
			fail++
			continue
		}
		for _, pr := range result.PerPatch {
			msg := "no-op (already applied)"
			mark := fmt.Sprintf("%sok%s", console.G, console.X)
			if pr.Applied > 0 {
				msg = fmt.Sprintf("%d in-place replacement(s)", pr.Applied)
				ok++
			} else if pr.Skipped > 0 {
				msg = "search not found"
				mark = fmt.Sprintf("%sskip%s", console.Y, console.X)
				skip++
			} else {
				ok++
			}
			fmt.Printf("  %s    %s  %s\n", mark, padRight(pr.ID, 40), msg)
		}
		for _, pid := range result.SkippedHeavy {
			fmt.Printf("  %sskip%s  %s  skipped — failed binary verification; retry succeeded without\n",
				console.Y, console.X, padRight(pid, 40))
			skip++
		}
	}
	return
}

// RunPatchQuiet is called by other commands internally (autopilot, upgrade, etc.)
// Returns error on failure, nil on success (exit code 0).
// Respects the operator selection file unless empty (then applies all).
func RunPatchQuiet() error {
	return runPatch(false, false, "", "", nil)
}

// RunPatchRC returns 0 on success, 1 on failure — for Autoheal callback.
func RunPatchRC() int {
	if runPatch(false, false, "", "", nil) != nil {
		return 1
	}
	return 0
}

// ── settings patch ──────────────────────────────────────────────────────────

func applySettings(p patches.Patch, dryRun bool) (bool, string) {
	settingsPath := p.SettingsPath
	if settingsPath == "" {
		settingsPath = filepath.Join(homeDir(), ".claude", "settings.json")
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) && isWindows() {
			appdata := os.Getenv("APPDATA")
			if appdata != "" {
				alt := filepath.Join(appdata, "claude", "settings.json")
				if _, err := os.Stat(alt); err == nil {
					settingsPath = alt
				}
			}
		}
	} else {
		settingsPath = expandHome(settingsPath)
	}

	os.MkdirAll(filepath.Dir(settingsPath), 0o755)

	var cur map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if json.Unmarshal(data, &cur) != nil {
			cur = make(map[string]interface{})
		}
	} else {
		cur = make(map[string]interface{})
	}

	wanted := p.Settings
	if wanted == nil {
		return true, "no-op (no settings in patch)"
	}

	out := make(map[string]interface{})
	for k, v := range cur {
		out[k] = v
	}
	changed := 0
	for k, v := range wanted {
		if strings.Contains(k, ".") {
			if setNested(out, k, v) {
				changed++
			}
			continue
		}
		existing, exists := out[k]
		if !exists || !jsonEqual(existing, v) {
			out[k] = v
			changed++
		}
	}
	for _, k := range p.Unset {
		if unsetNested(out, k) {
			changed++
		}
	}

	if changed == 0 {
		return true, "no-op (already applied)"
	}
	if dryRun {
		return true, fmt.Sprintf("would set %d key(s)", changed)
	}

	outJSON, _ := json.MarshalIndent(out, "", "  ")
	os.WriteFile(settingsPath, outJSON, 0o644)
	return true, fmt.Sprintf("%d key(s)", changed)
}

// ── MCP guard patch ─────────────────────────────────────────────────────────

func applyMCPGuard(p patches.Patch, dryRun bool) (bool, string) {
	preloadDir := preloadDirPath()
	preloadDst := filepath.Join(preloadDir, "claude-preload.js")

	guardMarker := "// mcp-guard"
	guardCode := "\n  // mcp-guard: timeout npx @latest MCP server spawns\n" +
		"  try { const _cp = require(\"child_process\"); const _oS = _cp.spawn;\n" +
		"    _cp.spawn = function(cmd, args, opts) { const p = _oS.apply(this, arguments);\n" +
		"      const isNpx = typeof cmd === \"string\" && cmd.includes(\"npx\") &&\n" +
		"        Array.isArray(args) && args.some(a => typeof a === \"string\" && a.includes(\"@latest\"));\n" +
		"      if (isNpx) { const t = setTimeout(() => { try { p.kill(); } catch(_) {} }, 30000);\n" +
		"        p.on(\"close\", () => clearTimeout(t)); } return p; }; } catch(_) {}\n"

	preloadMissing := false
	var content string

	data, err := os.ReadFile(preloadDst)
	if err != nil {
		preloadMissing = true
		embData, embErr := getContribFile("preload/claude-preload.js")
		if embErr != nil {
			return false, "preload source missing — run unleash install-preload first"
		}
		content = string(embData)
		if !dryRun {
			os.MkdirAll(preloadDir, 0o755)
			os.WriteFile(preloadDst, embData, 0o644)
		}
	} else {
		content = string(data)
	}

	if strings.Contains(content, guardMarker) {
		if dryRun && preloadMissing {
			return true, "would install preload (mcp-guard already present)"
		}
		return true, "no-op (already applied)"
	}

	newContent := injectGuard(content, guardCode)
	if newContent == "" {
		return false, "preload format unexpected"
	}

	if dryRun {
		if preloadMissing {
			return true, "would install preload and add mcp-guard"
		}
		return true, "would add mcp-guard to preload"
	}

	os.WriteFile(preloadDst, []byte(newContent), 0o644)
	return true, "mcp-guard added to preload"
}

func injectGuard(content, guardCode string) string {
	const marker = "globalThis.__VPCC_PRELOAD_ACTIVE__ = true;"
	if strings.Contains(content, marker) {
		return strings.Replace(content, marker, marker+guardCode, 1)
	}
	tail := "\n})();"
	idx := strings.LastIndex(content, tail)
	if idx >= 0 {
		return content[:idx] + guardCode + content[idx:]
	}
	return ""
}

// ── wrapper patch ───────────────────────────────────────────────────────────

func applyWrapper(p patches.Patch, kind string, tgt string, dryRun bool) (bool, string) {
	if isWindows() {
		// Bash wrapper concept does not apply on Windows; binary patches and
		// settings cover these installs directly.
		return true, "no-op (wrapper not applicable on Windows)"
	}
	wrapperPath := p.SettingsPath // wrapper_path reuses SettingsPath field or use WrapperPath
	if wrapperPath == "" {
		wrapperPath = "~/.local/bin/claude"
	}
	wrapperPath = expandHome(wrapperPath)

	nativeMarker := "# unleash-native-wrapper"
	jsMarker := "CLAUDE_CLI_JS"

	if kind == "bun_sea" {
		wrapperContent := "#!/usr/bin/env bash\n" +
			nativeMarker + "\n" +
			"VPCC_PRELOAD=\"${VPCC_PRELOAD:-$HOME/.local/share/void-patcher/claude-preload.js}\"\n" +
			"VPCC_VERSIONS_DIR=\"$HOME/.local/share/claude/versions\"\n" +
			"VPCC_NATIVE=\"$(ls -1 \"$VPCC_VERSIONS_DIR\" 2>/dev/null | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)\"\n" +
			"VPCC_NATIVE=\"$VPCC_VERSIONS_DIR/$VPCC_NATIVE\"\n" +
			"if [[ ! -x \"${VPCC_NATIVE:-}\" ]]; then echo \"unleash wrapper: native binary not found\" >&2; exit 1; fi\n" +
			"if [[ -f \"$VPCC_PRELOAD\" ]]; then\n" +
			"    export BUN_OPTIONS=\"${BUN_OPTIONS:+$BUN_OPTIONS }--preload $VPCC_PRELOAD\"\n" +
			"fi\n" +
			"exec \"$VPCC_NATIVE\" \"$@\"\n"

		if data, err := os.ReadFile(wrapperPath); err == nil {
			if strings.Contains(string(data), nativeMarker) {
				return true, "no-op (already applied)"
			}
		}
		if dryRun {
			return true, "would install native BUN_OPTIONS wrapper"
		}
		os.MkdirAll(filepath.Dir(wrapperPath), 0o755)
		os.Remove(wrapperPath)
		os.WriteFile(wrapperPath, []byte(wrapperContent), 0o755)
		return true, "native BUN_OPTIONS wrapper installed"
	}

	// cli.js wrapper — content from patch JSON
	if dryRun {
		return true, "would install cli.js wrapper"
	}
	if data, err := os.ReadFile(wrapperPath); err == nil {
		s := string(data)
		if strings.Contains(s, jsMarker) || strings.Contains(s, nativeMarker) {
			return true, "no-op (already applied)"
		}
	}
	return false, "no wrapper content in patch"
}

// ── binary_install patch ────────────────────────────────────────────────────

func applyBinaryInstall(_ patches.Patch, kind string, _ bool) (bool, string) {
	if kind == "bun_sea" {
		return true, "no-op (N/A for native binary — seccomp is in Bun runtime, not VFS)"
	}
	return false, "npm layout binary_install not implemented in unleash"
}

// ── deep merge ──────────────────────────────────────────────────────────────

// DeepMerge recursively merges src into dst. Arrays concatenated uniquely.
func DeepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		dstV, exists := dst[k]
		if exists {
			dstMap, dstIsMap := dstV.(map[string]interface{})
			srcMap, srcIsMap := v.(map[string]interface{})
			if dstIsMap && srcIsMap {
				dst[k] = DeepMerge(dstMap, srcMap)
				continue
			}
			dstArr, dstIsArr := dstV.([]interface{})
			srcArr, srcIsArr := v.([]interface{})
			if dstIsArr && srcIsArr {
				merged := make([]interface{}, len(dstArr))
				copy(merged, dstArr)
				for _, item := range srcArr {
					found := false
					for _, existing := range merged {
						if jsonEqual(existing, item) {
							found = true
							break
						}
					}
					if !found {
						merged = append(merged, item)
					}
				}
				dst[k] = merged
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

// ── byte helpers ────────────────────────────────────────────────────────────

func bytesContains(data, sub []byte) bool {
	return bytes.Contains(data, sub)
}

func bytesCountN(data, sub []byte, count int) int {
	if count <= 0 {
		return bytes.Count(data, sub)
	}
	total := 0
	pos := 0
	for total < count {
		idx := bytes.Index(data[pos:], sub)
		if idx < 0 {
			break
		}
		total++
		pos += idx + len(sub)
	}
	return total
}

func regexCountMatchesN(pattern string, data []byte, count int) int {
	if bytepatch.HasBackrefs(pattern) {
		n := 0
		pos := 0
		for count <= 0 || n < count {
			loc, _ := bytepatch.FindSubmatchBackrefs(pattern, data[pos:])
			if loc == nil {
				break
			}
			n++
			advance := loc[1]
			if advance <= loc[0] {
				advance = loc[0] + 1
			}
			pos += advance
			if pos >= len(data) {
				break
			}
		}
		return n
	}
	re, err := regexp.Compile("(?s)" + pattern)
	if err != nil {
		return 0
	}
	if count <= 0 {
		return len(re.FindAll(data, -1))
	}
	return len(re.FindAll(data, count))
}
