package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/VoidChecksum/unleash/internal/binary"
	"github.com/VoidChecksum/unleash/internal/patches"
	"github.com/VoidChecksum/unleash/internal/scanner"
	"github.com/VoidChecksum/unleash/internal/target"
	"github.com/VoidChecksum/unleash/internal/updater"
)

// ── views ───────────────────────────────────────────────────────────────────

type tuiView int

const (
	viewDashboard tuiView = iota
	viewPatchMgr
	viewScan
	viewDoctor
)

// ── patch categories ────────────────────────────────────────────────────────

var categoryOrder = []string{
	"permissions",
	"refusal",
	"classifier",
	"telemetry",
	"features",
	"ratelimit",
	"subscription",
	"attribution",
	"infrastructure",
}

var categoryLabels = map[string]string{
	"permissions":    "Permissions",
	"refusal":        "Refusal",
	"classifier":     "Classifier",
	"telemetry":      "Telemetry",
	"features":       "Features",
	"ratelimit":      "Rate Limit",
	"subscription":   "Subscription",
	"attribution":    "Attribution",
	"infrastructure": "Infrastructure",
}

// ── styles ──────────────────────────────────────────────────────────────────

var (
	accent    = lipgloss.Color("#8b5cf6")
	accentDim = lipgloss.Color("#6d45c4")
	green     = lipgloss.Color("#22c55e")
	yellow    = lipgloss.Color("#eab308")
	red       = lipgloss.Color("#ef4444")
	dimColor  = lipgloss.Color("#6b7280")
	white     = lipgloss.Color("#f9fafb")
	fgMuted   = lipgloss.Color("#9ca3af")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	headerStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentDim)

	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent)

	statusApplied = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	statusAvailable = lipgloss.NewStyle().
			Foreground(yellow)

	statusDrift = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	statusRetired = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)

	selectedStyle = lipgloss.NewStyle().
			Background(accent).
			Foreground(white).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(fgMuted)

	keyStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(fgMuted)
)

// ── messages ────────────────────────────────────────────────────────────────

type scanDoneMsg struct {
	rows []patches.ScanRow
}

type patchDoneMsg struct {
	result binary.PatchResult
}

type doctorDoneMsg struct {
	output string
}

type rollbackDoneMsg struct {
	ok  bool
	msg string
}

type updateCheckDoneMsg struct {
	info updater.UpstreamInfo
}

// ── model ───────────────────────────────────────────────────────────────────

type tuiModel struct {
	// terminal size
	width, height int

	// current view
	view tuiView

	// target info
	targetPath string
	targetKind string
	targetSHA  string
	ccVersion  string
	formatStr  string

	// patches
	allPatches []patches.Patch
	scanRows   []patches.ScanRow
	scanDone   bool
	byCategory map[string][]int // category -> indices into allPatches
	toggleSet  map[int]bool     // indices into allPatches user has toggled

	// patch manager state
	catIdx      int  // selected category index
	patchIdx    int  // selected patch index in current category
	patchFocus  bool // true = right panel focused, false = left (categories)
	searchMode  bool
	searchInput textinput.Model

	// scan view
	scanViewport viewport.Model

	// doctor view
	doctorOutput   string
	doctorDone     bool
	doctorViewport viewport.Model

	// upstream update check
	updateInfo    updater.UpstreamInfo
	updateChecked bool

	// status bar messages
	statusMsg string

	// pending operation
	busy    bool
	busyMsg string
}

func initialModel() tuiModel {
	ti := textinput.New()
	ti.Placeholder = "filter patches..."
	ti.CharLimit = 64

	sv := viewport.New(80, 20)
	dv := viewport.New(80, 20)

	m := tuiModel{
		width:          120,
		height:         40,
		view:           viewDashboard,
		toggleSet:      make(map[int]bool),
		byCategory:     make(map[string][]int),
		searchInput:    ti,
		scanViewport:   sv,
		doctorViewport: dv,
	}

	// Load target info
	m.targetPath, m.targetKind = target.FindTarget()
	if m.targetPath != "" {
		m.targetSHA = target.SHA256Short(m.targetPath)
		m.ccVersion = detectCCVersion(m.targetPath)
		m.formatStr = formatLabel(m.targetKind)
	}

	// Load patches
	m.allPatches = loadAllPatches()
	m.buildCategoryIndex()

	return m
}

func (m *tuiModel) buildCategoryIndex() {
	m.byCategory = make(map[string][]int)
	for i, p := range m.allPatches {
		cat := p.Category
		if cat == "" {
			cat = "infrastructure"
		}
		m.byCategory[cat] = append(m.byCategory[cat], i)
	}
}

func formatLabel(kind string) string {
	if kind == "bun_sea" {
		switch runtime.GOOS {
		case "darwin":
			return "Bun SEA (Mach-O)"
		case "windows":
			return "Bun SEA (PE)"
		default:
			return "Bun SEA (ELF)"
		}
	}
	return "cli.js (JS)"
}

// ── patchStatus helpers ─────────────────────────────────────────────────────

func (m *tuiModel) patchStatus(patchIdx int) string {
	p := m.allPatches[patchIdx]
	if p.Retired {
		return "RETIRED"
	}
	if p.Type == "settings" && settingsPatchApplied(p) {
		return "APPLIED"
	}
	if !m.scanDone {
		return "AVAILABLE"
	}
	for _, r := range m.scanRows {
		if r.ID == p.ID {
			switch r.Status {
			case "applied":
				return "APPLIED"
			case "ok":
				return "AVAILABLE"
			case "drift":
				return "DRIFT"
			case "retired":
				return "RETIRED"
			default:
				return "AVAILABLE"
			}
		}
	}
	return "AVAILABLE"
}

func settingsPatchApplied(p patches.Patch) bool {
	if len(p.Settings) == 0 {
		return false
	}
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

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}
	var cur map[string]interface{}
	if json.Unmarshal(data, &cur) != nil {
		return false
	}
	for k, v := range p.Settings {
		existing, exists := cur[k]
		if !exists || !jsonEqual(existing, v) {
			return false
		}
	}
	return true
}

func (m *tuiModel) currentCategoryPatches() []int {
	if m.catIdx >= len(categoryOrder) {
		return nil
	}
	cat := categoryOrder[m.catIdx]
	idxs := m.byCategory[cat]

	if m.searchInput.Value() == "" {
		return idxs
	}

	query := strings.ToLower(m.searchInput.Value())
	var filtered []int
	for _, i := range idxs {
		p := m.allPatches[i]
		if strings.Contains(strings.ToLower(p.ID), query) ||
			strings.Contains(strings.ToLower(p.Description), query) {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

func (m *tuiModel) clampPatchIndex() {
	visPatches := m.currentCategoryPatches()
	if len(visPatches) == 0 {
		m.patchIdx = 0
		return
	}
	if m.patchIdx >= len(visPatches) {
		m.patchIdx = len(visPatches) - 1
	}
	if m.patchIdx < 0 {
		m.patchIdx = 0
	}
}

// ── tea.Model implementation ────────────────────────────────────────────────

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		runScanAsync(m.targetPath, m.targetKind, m.allPatches),
		runUpdateCheckAsync(),
	)
}

func runUpdateCheckAsync() tea.Cmd {
	return func() tea.Msg {
		return updateCheckDoneMsg{info: updater.UpstreamStatus(patchDir())}
	}
}

func runScanAsync(targetPath, kind string, patchList []patches.Patch) tea.Cmd {
	return func() tea.Msg {
		if targetPath == "" {
			return scanDoneMsg{}
		}
		text, err := scanner.LoadTextFromTarget(targetPath, kind)
		if err != nil {
			return scanDoneMsg{}
		}
		sc := scanner.NewSigScanner(text)
		scannable := scannablePatches(patchList)
		rows := sc.ScanPatches(scannable)
		return scanDoneMsg{rows: rows}
	}
}

func scannablePatches(patchList []patches.Patch) []patches.Patch {
	scannable := make([]patches.Patch, 0, len(patchList))
	for _, p := range patchList {
		if p.Type == "js_replace" && !p.Disabled && p.ShouldScan() {
			scannable = append(scannable, p)
		}
	}
	return scannable
}

func runPatchAsync(targetPath, targetKind string, patchList []patches.Patch) tea.Cmd {
	return func() tea.Msg {
		return patchDoneMsg{result: applySelectedPatches(targetPath, targetKind, patchList)}
	}
}

func applySelectedPatches(targetPath, targetKind string, patchList []patches.Patch) binary.PatchResult {
	var jsPatches, metaPatches []patches.Patch
	for _, p := range patchList {
		if p.Type == "js_replace" {
			jsPatches = append(jsPatches, p)
		} else {
			metaPatches = append(metaPatches, p)
		}
	}

	result := binary.PatchResult{OK: true, Noop: true}
	if len(jsPatches) > 0 {
		if targetPath == "" {
			result.Skipped += len(jsPatches)
		} else if targetKind != "bun_sea" {
			result.Skipped += len(jsPatches)
		} else {
			_, err := target.Backup(targetPath, targetKind)
			if err != nil {
				return binary.PatchResult{Err: err.Error()}
			}
			jsResult := binary.PatchBunSEAInplace(targetPath, jsPatches)
			if !jsResult.OK {
				return jsResult
			}
			result.Applied += jsResult.Applied
			result.Skipped += jsResult.Skipped
			result.PerPatch = append(result.PerPatch, jsResult.PerPatch...)
			result.SkippedHeavy = append(result.SkippedHeavy, jsResult.SkippedHeavy...)
			result.Noop = jsResult.Noop
		}
	}

	for _, p := range metaPatches {
		success, msg := applyMetaPatch(p, targetPath, targetKind, false)
		if !success {
			return binary.PatchResult{Err: fmt.Sprintf("%s: %s", p.ID, msg)}
		}
		result.Applied++
		result.Noop = false
	}
	return result
}

func applyMetaPatch(p patches.Patch, targetPath, targetKind string, dryRun bool) (bool, string) {
	switch p.Type {
	case "settings", "hook":
		return applySettings(p, dryRun)
	case "mcp_guard":
		return applyMCPGuard(p, dryRun)
	case "wrapper":
		return applyWrapper(p, targetKind, targetPath, dryRun)
	case "binary_install":
		return applyBinaryInstall(p, targetKind, dryRun)
	default:
		return false, fmt.Sprintf("type=%s (unknown)", p.Type)
	}
}

func runDoctorAsync(targetPath, kind string) tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder
		patchList := loadAllPatches()
		pd := patchDir()
		nRetired := countRetired(pd)

		sb.WriteString("unleash doctor\n")
		sb.WriteString("  version    : 1.0.0\n")
		sb.WriteString(fmt.Sprintf("  patches    : %d\n", len(patchList)))

		if targetPath == "" {
			sb.WriteString("  target     : NOT FOUND\n")
			return doctorDoneMsg{output: sb.String()}
		}

		label := formatLabel(kind)
		info, _ := os.Stat(targetPath)
		sizeMB := int64(0)
		if info != nil {
			sizeMB = info.Size() / 1024 / 1024
		}

		sb.WriteString(fmt.Sprintf("  target     : %s\n", targetPath))
		sb.WriteString(fmt.Sprintf("  format     : %s\n", label))
		sb.WriteString(fmt.Sprintf("  sha256     : %s\n", target.SHA256Short(targetPath)))
		sb.WriteString(fmt.Sprintf("  size       : %d MB\n", sizeMB))

		text, err := scanner.LoadTextFromTarget(targetPath, kind)
		if err == nil && text != "" {
			sc := scanner.NewSigScanner(text)

			// Only scan js_replace patches
			var scannable []patches.Patch
			for _, p := range patchList {
				if p.Type == "js_replace" && p.ShouldScan() {
					scannable = append(scannable, p)
				}
			}

			rows := sc.ScanPatches(scannable)
			applied, drift, ok, retired := 0, 0, 0, 0
			for _, r := range rows {
				switch r.Status {
				case "applied":
					applied++
				case "drift":
					drift++
				case "ok":
					ok++
				case "retired":
					retired++
				}
			}
			sb.WriteString(fmt.Sprintf("  sig scan   : %d applied, %d ok, %d drift\n", applied, ok, drift))
		} else {
			sb.WriteString("  sig scan   : failed\n")
		}

		sb.WriteString(fmt.Sprintf("  retired    : %d patches\n", nRetired))

		bdir := target.BackupDir()
		baks := countBackups(bdir)
		sb.WriteString(fmt.Sprintf("  backups    : %d in %s\n", baks, bdir))

		return doctorDoneMsg{output: sb.String()}
	}
}

func runRollbackAsync(targetPath string) tea.Cmd {
	return func() tea.Msg {
		rc := RunRollback()
		if rc == 0 {
			return rollbackDoneMsg{ok: true, msg: "rollback successful"}
		}
		return rollbackDoneMsg{ok: false, msg: "rollback failed"}
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewports()
		m.buildScanViewport()
		if m.doctorOutput != "" {
			m.doctorViewport.SetContent(m.doctorOutput)
		}
		return m, nil

	case scanDoneMsg:
		m.scanRows = msg.rows
		m.scanDone = true
		m.buildScanViewport()
		if m.busyMsg == "Scanning..." || m.busyMsg == "Re-scanning..." {
			m.busy = false
			m.busyMsg = ""
		}
		return m, nil

	case updateCheckDoneMsg:
		m.updateInfo = msg.info
		m.updateChecked = true
		return m, nil

	case patchDoneMsg:
		m.busy = false
		m.busyMsg = ""
		if msg.result.OK {
			m.statusMsg = fmt.Sprintf("Patched: %d applied, %d skipped", msg.result.Applied, msg.result.Skipped)
		} else {
			m.statusMsg = fmt.Sprintf("Patch failed: %s", msg.result.Err)
		}
		// Re-scan after patching
		return m, runScanAsync(m.targetPath, m.targetKind, m.allPatches)

	case doctorDoneMsg:
		m.doctorOutput = msg.output
		m.doctorDone = true
		m.doctorViewport.SetContent(msg.output)
		m.busy = false
		m.busyMsg = ""
		return m, nil

	case rollbackDoneMsg:
		m.busy = false
		m.busyMsg = ""
		if msg.ok {
			m.statusMsg = msg.msg
		} else {
			m.statusMsg = msg.msg
		}
		return m, runScanAsync(m.targetPath, m.targetKind, m.allPatches)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to sub-components
	if m.searchMode {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	if m.view == viewScan {
		var cmd tea.Cmd
		m.scanViewport, cmd = m.scanViewport.Update(msg)
		return m, cmd
	}

	if m.view == viewDoctor {
		var cmd tea.Cmd
		m.doctorViewport, cmd = m.doctorViewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	}

	// Search mode capture
	if m.searchMode {
		switch key {
		case "esc":
			m.searchMode = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			return m, nil
		case "enter":
			m.searchMode = false
			m.searchInput.Blur()
			m.clampPatchIndex()
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.clampPatchIndex()
			return m, cmd
		}
	}

	if m.busy {
		return m, nil
	}

	switch m.view {
	case viewDashboard:
		return m.handleDashboardKey(key)
	case viewPatchMgr:
		return m.handlePatchMgrKey(key)
	case viewScan:
		return m.handleScanKey(msg)
	case viewDoctor:
		return m.handleDoctorKey(msg)
	}

	return m, nil
}

func (m tuiModel) handleDashboardKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "Q":
		return m, tea.Quit
	case "p", "P":
		m.view = viewPatchMgr
		m.catIdx = 0
		m.patchIdx = 0
		m.patchFocus = false
		return m, nil
	case "s", "S":
		m.view = viewScan
		if !m.scanDone {
			m.busy = true
			m.busyMsg = "Scanning..."
			return m, runScanAsync(m.targetPath, m.targetKind, m.allPatches)
		}
		m.buildScanViewport()
		return m, nil
	case "d", "D":
		m.view = viewDoctor
		m.busy = true
		m.busyMsg = "Running doctor..."
		return m, runDoctorAsync(m.targetPath, m.targetKind)
	}
	return m, nil
}

func (m tuiModel) handlePatchMgrKey(key string) (tea.Model, tea.Cmd) {
	visPatches := m.currentCategoryPatches()

	switch key {
	case "q", "esc":
		m.view = viewDashboard
		m.searchMode = false
		m.searchInput.SetValue("")
		return m, nil

	case "tab":
		m.patchFocus = !m.patchFocus
		if m.patchFocus && len(visPatches) > 0 {
			m.patchIdx = 0
		}
		return m, nil

	case "up", "k":
		if m.patchFocus {
			if m.patchIdx > 0 {
				m.patchIdx--
			}
		} else {
			if m.catIdx > 0 {
				m.catIdx--
				m.patchIdx = 0
			}
		}
		return m, nil

	case "down", "j":
		if m.patchFocus {
			if m.patchIdx < len(visPatches)-1 {
				m.patchIdx++
			}
		} else {
			if m.catIdx < len(categoryOrder)-1 {
				m.catIdx++
				m.patchIdx = 0
			}
		}
		return m, nil

	case " ":
		if m.patchFocus && len(visPatches) > 0 && m.patchIdx < len(visPatches) {
			idx := visPatches[m.patchIdx]
			if m.patchStatus(idx) != "APPLIED" && m.patchStatus(idx) != "RETIRED" {
				m.toggleSet[idx] = !m.toggleSet[idx]
				if !m.toggleSet[idx] {
					delete(m.toggleSet, idx)
				}
			}
		}
		return m, nil

	case "a":
		// Select all in current category
		for _, idx := range visPatches {
			p := m.allPatches[idx]
			status := m.patchStatus(idx)
			if !p.Retired && status != "APPLIED" && status != "RETIRED" {
				m.toggleSet[idx] = true
			}
		}
		return m, nil

	case "n":
		// Deselect all in current category
		for _, idx := range visPatches {
			delete(m.toggleSet, idx)
		}
		return m, nil

	case "/":
		m.searchMode = true
		m.searchInput.Focus()
		m.patchIdx = 0
		return m, textinput.Blink
	case "c", "C":
		if m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.patchIdx = 0
		}
		return m, nil

	case "enter":
		return m.applyToggled()

	case "r", "R":
		// Rollback
		if m.targetPath != "" {
			m.busy = true
			m.busyMsg = "Rolling back..."
			return m, runRollbackAsync(m.targetPath)
		}
		return m, nil
	}

	return m, nil
}

func (m tuiModel) applyToggled() (tea.Model, tea.Cmd) {
	var toApply []patches.Patch
	hasMeta := false
	for idx := range m.toggleSet {
		p := m.allPatches[idx]
		if !p.Retired {
			toApply = append(toApply, p)
			if p.Type != "js_replace" {
				hasMeta = true
			}
		}
	}

	if len(toApply) == 0 {
		m.statusMsg = "No patches selected"
		return m, nil
	}
	if m.targetPath == "" && !hasMeta {
		m.statusMsg = "No target binary found"
		return m, nil
	}

	m.busy = true
	m.busyMsg = fmt.Sprintf("Applying %d patches...", len(toApply))
	m.toggleSet = make(map[int]bool)
	return m, runPatchAsync(m.targetPath, m.targetKind, toApply)
}

func (m tuiModel) handleScanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.view = viewDashboard
		return m, nil
	case "r", "R":
		m.busy = true
		m.busyMsg = "Re-scanning..."
		m.scanDone = false
		return m, runScanAsync(m.targetPath, m.targetKind, m.allPatches)
	default:
		var cmd tea.Cmd
		m.scanViewport, cmd = m.scanViewport.Update(msg)
		return m, cmd
	}
}

func (m tuiModel) handleDoctorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.view = viewDashboard
		return m, nil
	default:
		var cmd tea.Cmd
		m.doctorViewport, cmd = m.doctorViewport.Update(msg)
		return m, cmd
	}
}

func (m *tuiModel) resizeViewports() {
	w := m.width - 6
	if w < 20 {
		w = 20
	}
	h := m.height - 9
	if h < 3 {
		h = 3
	}
	m.scanViewport.Width = w
	m.scanViewport.Height = h
	m.doctorViewport.Width = w
	m.doctorViewport.Height = h
}

func (m *tuiModel) buildScanViewport() {
	if len(m.scanRows) == 0 {
		m.scanViewport.SetContent("  No scan results available.")
		m.scanViewport.GotoTop()
		return
	}

	var sb strings.Builder
	hdr := fmt.Sprintf("  %-40s  %-10s  %6s  %-12s  %s",
		"PATCH ID", "STATUS", "CONF", "METHOD", "OFFSET")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(white).Render(hdr))
	sb.WriteString("\n")

	for _, r := range m.scanRows {
		conf := fmt.Sprintf("%.0f%%", r.Confidence*100)
		offset := "—"
		if r.AnchorOffset != nil {
			offset = fmt.Sprintf("0x%x", *r.AnchorOffset)
		}

		var styledStatus string
		switch r.Status {
		case "applied":
			styledStatus = statusApplied.Render(padRight(r.Status, 10))
		case "ok":
			styledStatus = statusAvailable.Render(padRight(r.Status, 10))
		case "drift":
			styledStatus = statusDrift.Render(padRight(r.Status, 10))
		case "retired":
			styledStatus = statusRetired.Render(padRight(r.Status, 10))
		default:
			styledStatus = mutedStyle.Render(padRight(r.Status, 10))
		}

		var styledMethod string
		switch {
		case r.Method == "marker" || r.Method == "anchor":
			styledMethod = lipgloss.NewStyle().Foreground(green).Render(padRight(r.Method, 12))
		case r.Method == "scattered" || r.Method == "fuzzy_ws" || r.Method == "fuzzy_ident":
			styledMethod = lipgloss.NewStyle().Foreground(yellow).Render(padRight(r.Method, 12))
		default:
			styledMethod = mutedStyle.Render(padRight(r.Method, 12))
		}

		id := r.ID
		if len(id) > 40 {
			id = id[:39] + "…"
		}

		sb.WriteString(fmt.Sprintf("  %-40s  %s  %6s  %s  %s\n",
			id, styledStatus, conf, styledMethod, offset))
	}
	m.scanViewport.SetContent(strings.TrimRight(sb.String(), "\n"))
	m.scanViewport.GotoTop()
}

// ── View ────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	switch m.view {
	case viewDashboard:
		return m.viewDashboard()
	case viewPatchMgr:
		return m.viewPatchMgr()
	case viewScan:
		return m.viewScan()
	case viewDoctor:
		return m.viewDoctor()
	}
	return ""
}

// ── ASCII logo ──────────────────────────────────────────────────────────────

const unleashLogo = `
 ██╗   ██╗███╗   ██╗██╗     ███████╗ █████╗ ███████╗██╗  ██╗
 ██║   ██║████╗  ██║██║     ██╔════╝██╔══██╗██╔════╝██║  ██║
 ██║   ██║██╔██╗ ██║██║     █████╗  ███████║███████╗███████║
 ██║   ██║██║╚██╗██║██║     ██╔══╝  ██╔══██║╚════██║██╔══██║
 ╚██████╔╝██║ ╚████║███████╗███████╗██║  ██║███████║██║  ██║
  ╚═════╝ ╚═╝  ╚═══╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝`

// ── Dashboard view ──────────────────────────────────────────────────────────

func (m tuiModel) viewDashboard() string {
	w := m.width
	if w < 60 {
		w = 60
	}

	var sb strings.Builder

	// Logo
	logo := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(unleashLogo)
	sb.WriteString(logo)
	sb.WriteString("\n")

	subtitle := lipgloss.NewStyle().
		Foreground(fgMuted).
		Italic(true).
		Render("  Claude Code binary patcher — v3.0.0")
	sb.WriteString(subtitle)
	sb.WriteString("\n\n")

	// Status panel
	panelW := w - 4
	if panelW > 80 {
		panelW = 80
	}

	var statusLines []string
	if m.targetPath != "" {
		statusLines = append(statusLines,
			fmt.Sprintf("  CC Version  : %s", coalesce(m.ccVersion, "unknown")),
			fmt.Sprintf("  Binary      : %s", m.targetPath),
			fmt.Sprintf("  SHA256      : %s", m.targetSHA),
			fmt.Sprintf("  Format      : %s", m.formatStr),
		)

		total := len(m.allPatches)
		applied, drift := 0, 0
		if m.scanDone {
			for _, r := range m.scanRows {
				switch r.Status {
				case "applied":
					applied++
				case "drift":
					drift++
				}
			}
		}
		statusLines = append(statusLines,
			fmt.Sprintf("  Patches     : %d total, %d applied, %d drift", total, applied, drift),
		)
		if m.updateChecked {
			if warning := updateWarning(m.updateInfo, "unleash update"); warning != "" {
				statusLines = append(statusLines, "  Update      : "+warning)
			} else if m.updateInfo.RemoteCommit != "" {
				statusLines = append(statusLines, "  Update      : "+statusApplied.Render("current"))
			} else {
				statusLines = append(statusLines, "  Update      : unreachable")
			}
		} else {
			statusLines = append(statusLines, "  Update      : checking...")
		}
	} else {
		statusLines = append(statusLines,
			"  Target      : "+lipgloss.NewStyle().Foreground(red).Render("NOT FOUND"),
			"  Run 'unleash patch' after installing Claude Code",
		)
	}

	statusPanel := borderStyle.
		Width(panelW).
		Padding(0, 1).
		Render(headerStyle.Render("  Status") + "\n" + strings.Join(statusLines, "\n"))
	sb.WriteString(statusPanel)
	sb.WriteString("\n\n")

	// Menu
	menuItems := []string{
		keyStyle.Render("[P]") + " Patch Manager",
		keyStyle.Render("[S]") + " Scan",
		keyStyle.Render("[D]") + " Doctor",
		keyStyle.Render("[Q]") + " Quit",
	}
	menuPanel := borderStyle.
		Width(panelW).
		Padding(0, 1).
		Render(headerStyle.Render("  Menu") + "\n  " + strings.Join(menuItems, "    "))
	sb.WriteString(menuPanel)
	sb.WriteString("\n")

	// Status message
	if m.statusMsg != "" {
		sb.WriteString("\n")
		sb.WriteString(mutedStyle.Render("  " + m.statusMsg))
	}

	if m.busy {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(yellow).Render("  ⟳ " + m.busyMsg))
	}

	return sb.String()
}

// ── Patch Manager view ──────────────────────────────────────────────────────

func (m tuiModel) viewPatchMgr() string {
	w := m.width
	if w < 80 {
		w = 80
	}
	h := m.height
	if h < 20 {
		h = 20
	}

	leftW := 28
	rightW := w - leftW - 7
	if rightW < 40 {
		rightW = 40
	}
	listH := h - 14
	if listH < 6 {
		listH = 6
	}

	// Left panel: categories
	var catLines []string
	for i, cat := range categoryOrder {
		label := categoryLabels[cat]
		count := len(m.byCategory[cat])
		line := fmt.Sprintf("  %-16s %3d", label, count)

		if i == m.catIdx {
			if !m.patchFocus {
				line = selectedStyle.Render(line)
			} else {
				line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
			}
		} else {
			line = mutedStyle.Render(line)
		}
		catLines = append(catLines, line)
	}

	for len(catLines) < listH {
		catLines = append(catLines, "")
	}
	if len(catLines) > listH {
		catLines = catLines[:listH]
	}

	catStyle := borderStyle
	if !m.patchFocus {
		catStyle = activeBorder
	}
	leftPanel := catStyle.
		Width(leftW).
		Height(listH).
		Render(headerStyle.Render(" Categories") + "\n" + strings.Join(catLines, "\n"))

	// Right panel: patches in current category
	visPatches := m.currentCategoryPatches()
	var patchLines []string

	// Scroll window
	scrollStart := 0
	if m.patchIdx >= listH-1 {
		scrollStart = m.patchIdx - listH + 2
	}

	for vi := scrollStart; vi < len(visPatches) && len(patchLines) < listH; vi++ {
		idx := visPatches[vi]
		p := m.allPatches[idx]
		status := m.patchStatus(idx)

		// Checkbox
		check := "[ ]"
		if m.toggleSet[idx] {
			check = "[x]"
		}
		if status == "APPLIED" {
			check = "[✓]"
		}

		// Status badge
		var badge string
		switch status {
		case "APPLIED":
			badge = statusApplied.Render("APPLIED")
		case "AVAILABLE":
			badge = statusAvailable.Render("AVAIL")
		case "DRIFT":
			badge = statusDrift.Render("DRIFT")
		case "RETIRED":
			badge = statusRetired.Render("RETD")
		default:
			badge = status
		}

		// Truncate ID
		id := p.ID
		maxIDLen := rightW - 26
		if maxIDLen < 10 {
			maxIDLen = 10
		}
		if len(id) > maxIDLen {
			id = id[:maxIDLen-1] + "…"
		}

		line := fmt.Sprintf(" %s %-*s %s", check, maxIDLen, id, badge)

		if m.patchFocus && vi == m.patchIdx {
			line = selectedStyle.Render(line)
		}
		patchLines = append(patchLines, line)
	}

	for len(patchLines) < listH {
		patchLines = append(patchLines, "")
	}

	patchStyle := borderStyle
	if m.patchFocus {
		patchStyle = activeBorder
	}
	rightPanel := patchStyle.
		Width(rightW).
		Height(listH).
		Render(headerStyle.Render(" Patches") + "\n" + strings.Join(patchLines, "\n"))

	// Join panels
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)

	// Detail panel (bottom)
	var detailContent string
	if m.patchFocus && len(visPatches) > 0 && m.patchIdx < len(visPatches) {
		idx := visPatches[m.patchIdx]
		p := m.allPatches[idx]
		status := m.patchStatus(idx)

		var badge string
		switch status {
		case "APPLIED":
			badge = statusApplied.Render("● APPLIED")
		case "AVAILABLE":
			badge = statusAvailable.Render("○ AVAILABLE")
		case "DRIFT":
			badge = statusDrift.Render("⚠ DRIFT")
		case "RETIRED":
			badge = statusRetired.Render("◌ RETIRED")
		}

		desc := p.Description
		if len(desc) > rightW+leftW-10 {
			desc = desc[:rightW+leftW-13] + "..."
		}

		detailContent = fmt.Sprintf(
			"  %s  %s\n  %s\n  Type: %s  │  Category: %s  │  Sub-patches: %d",
			lipgloss.NewStyle().Bold(true).Foreground(white).Render(p.ID),
			badge,
			mutedStyle.Render(desc),
			p.Type,
			coalesce(p.Category, "infrastructure"),
			len(p.Patches),
		)
	} else {
		detailContent = mutedStyle.Render("  Select a patch to see details")
	}

	detailPanel := borderStyle.
		Width(w - 4).
		Render(headerStyle.Render(" Detail") + "\n" + detailContent)

	// Search bar
	var searchBar string
	if m.searchMode {
		searchBar = "\n " + lipgloss.NewStyle().Foreground(accent).Render("🔍 ") + m.searchInput.View()
	}

	// Help line
	toggleCount := len(m.toggleSet)
	helpText := helpStyle.Render(
		" ↑↓ navigate  Tab switch  Space toggle  Enter apply  " +
			"a all  n none  / search  c clear  r rollback  q back",
	)
	if toggleCount > 0 {
		helpText = lipgloss.NewStyle().Foreground(accent).Render(
			fmt.Sprintf(" %d selected", toggleCount),
		) + "  " + helpText
	}

	// Title
	title := headerStyle.Render("  ▸ Patch Manager")

	// Status
	var statusLine string
	if m.statusMsg != "" {
		statusLine = "\n" + mutedStyle.Render("  "+m.statusMsg)
	}
	if m.busy {
		statusLine = "\n" + lipgloss.NewStyle().Foreground(yellow).Render("  ⟳ "+m.busyMsg)
	}

	return title + "\n\n" + panels + "\n" + detailPanel + searchBar + "\n" + helpText + statusLine + "\n"
}

// ── Scan view ───────────────────────────────────────────────────────────────

func (m tuiModel) viewScan() string {
	title := headerStyle.Render("  ▸ Scan Results")

	if m.busy {
		return title + "\n\n" +
			lipgloss.NewStyle().Foreground(yellow).Render("  ⟳ Scanning...") + "\n"
	}

	if !m.scanDone || len(m.scanRows) == 0 {
		return title + "\n\n" +
			mutedStyle.Render("  No scan results available.") + "\n\n" +
			helpStyle.Render("  r rescan  q back") + "\n"
	}

	applied, drift, ok := 0, 0, 0
	for _, r := range m.scanRows {
		switch r.Status {
		case "applied":
			applied++
		case "drift":
			drift++
		case "ok":
			ok++
		}
	}
	summary := fmt.Sprintf("  %s %d applied  %s %d available  %s %d drift  │  %d total",
		statusApplied.Render("●"), applied,
		statusAvailable.Render("○"), ok,
		statusDrift.Render("⚠"), drift,
		len(m.scanRows))

	w := m.width - 4
	if w < 60 {
		w = 60
	}

	panel := borderStyle.
		Width(w).
		Render(m.scanViewport.View())

	return title + "\n\n" + summary + "\n\n" + panel + "\n\n" +
		helpStyle.Render("  ↑↓/PgUp/PgDn scroll  r rescan  q back") + "\n"
}

// ── Doctor view ─────────────────────────────────────────────────────────────

func (m tuiModel) viewDoctor() string {
	title := headerStyle.Render("  ▸ Doctor")

	if m.busy {
		return title + "\n\n" +
			lipgloss.NewStyle().Foreground(yellow).Render("  ⟳ Running diagnostics...") + "\n"
	}

	if !m.doctorDone {
		return title + "\n\n" +
			mutedStyle.Render("  Running doctor...") + "\n"
	}

	w := m.width - 4
	if w < 60 {
		w = 60
	}

	panel := borderStyle.
		Width(w).
		Padding(1, 1).
		Render(m.doctorViewport.View())

	return title + "\n\n" + panel + "\n\n" +
		helpStyle.Render("  ↑↓/PgUp/PgDn scroll  q back") + "\n"
}

// ── Cobra command ───────────────────────────────────────────────────────────

// NewTuiCmd creates the "tui" cobra command — the full interactive terminal UI.
func NewTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive terminal UI — full unleash control panel",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(initialModel(), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}

// NewTuiProgram creates a bubbletea program for use by the root command default.
func NewTuiProgram() *tea.Program {
	return tea.NewProgram(initialModel(), tea.WithAltScreen())
}
