// Package updater handles self-update, patch syncing, autoheal, and upstream status.
package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── constants ───────────────────────────────────────────────────────────────

const (
	Repo    = "NetVar1337/unleash"
	Branch  = "main"
	APIBase = "https://api.github.com/repos/" + Repo
	UA      = "unleash-updater/1.0"
)

var (
	// StateDir is ~/.unleash
	StateDir string
	// StateFile is ~/.unleash/state.json
	StateFile string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	StateDir = filepath.Join(home, ".unleash")
	StateFile = filepath.Join(StateDir, "state.json")
}

// ── HTTP helpers ────────────────────────────────────────────────────────────

func doReq(url, accept string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Accept", accept)

	// Support GITHUB_TOKEN / GH_TOKEN for authenticated requests.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if token := os.Getenv("GH_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// ── remote queries ──────────────────────────────────────────────────────────

// RemoteHeadSHA returns the latest commit SHA touching path on Branch.
// Returns "" on any network or parse error.
func RemoteHeadSHA(path string) string {
	url := fmt.Sprintf("%s/commits?sha=%s&path=%s&per_page=1", APIBase, Branch, path)
	data, err := doReq(url, "application/json", 20*time.Second)
	if err != nil {
		return ""
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(data, &commits); err != nil || len(commits) == 0 {
		return ""
	}
	return commits[0].SHA
}

// DownloadTarball fetches a gzipped tarball of the given ref.
// Prefers codeload.github.com (works on all OS / avoids 415 on some Windows
// urllib stacks); falls back to api.github.com/tarball.
func DownloadTarball(sha string) ([]byte, error) {
	codeloadURL := fmt.Sprintf("https://codeload.github.com/%s/tar.gz/%s", Repo, sha)
	data, err := doReq(codeloadURL, "application/x-gzip", 60*time.Second)
	if err == nil {
		return data, nil
	}
	// Fallback to API tarball endpoint. GitHub rejects application/octet-stream
	// on this JSON API route with HTTP 415; the response redirects to codeload.
	return doReq(fmt.Sprintf("%s/tarball/%s", APIBase, sha), "application/vnd.github+json", 60*time.Second)
}

// DownloadPatchFiles fetches patches/*.json directly through the contents API.
// This avoids downloading the whole repository tarball just to sync patch JSON.
func DownloadPatchFiles(ref string) (map[string][]byte, error) {
	url := fmt.Sprintf("%s/contents/patches?ref=%s", APIBase, ref)
	data, err := doReq(url, "application/vnd.github+json", 30*time.Second)
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	staged := map[string][]byte{}
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".json") {
			continue
		}
		if entry.DownloadURL == "" {
			return nil, fmt.Errorf("%s missing download_url", entry.Name)
		}
		content, err := doReq(entry.DownloadURL, "application/vnd.github.raw", 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name, err)
		}
		staged[entry.Name] = content
	}
	if len(staged) == 0 {
		return nil, fmt.Errorf("contents API returned no patches")
	}
	return staged, nil
}

func downloadPatchFilesFromTarball(ref string) (map[string][]byte, error) {
	tarBytes, err := DownloadTarball(ref)
	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}

	gzr, err := gzip.NewReader(bytes.NewReader(tarBytes))
	if err != nil {
		return nil, fmt.Errorf("tar open failed: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	staged := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read failed: %v", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		parts := splitTarPath(hdr.Name)
		if len(parts) < 3 || parts[1] != "patches" || !strings.HasSuffix(parts[2], ".json") {
			continue
		}
		content, err := io.ReadAll(tr)
		if err != nil {
			continue
		}
		staged[parts[2]] = content
	}
	if len(staged) == 0 {
		return nil, fmt.Errorf("tarball contained no patches/")
	}
	return staged, nil
}

// ── state persistence ───────────────────────────────────────────────────────

// LoadState reads ~/.unleash/state.json into a map. Returns empty map on any error.
func LoadState() map[string]interface{} {
	data, err := os.ReadFile(StateFile)
	if err != nil {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

// SaveState merges kv into the existing state and writes it back.
func SaveState(kv map[string]interface{}) {
	_ = os.MkdirAll(StateDir, 0o755)
	s := LoadState()
	for k, v := range kv {
		s[k] = v
	}
	s["updated_at"] = time.Now().UTC().Format("2006-01-02T15:04:05+00:00")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(StateFile, data, 0o644)
}

// ── patch dir sync ──────────────────────────────────────────────────────────

// SyncPatches downloads patches/*.json into patchDir.
// Returns (changed_file_count, commit_sha). changed == -1 signals an error
// with the second value containing the error message.
func SyncPatches(patchDir string, sha string) (int, string) {
	targetSHA := sha
	if targetSHA == "" {
		if rs := RemoteHeadSHA("patches"); rs != "" {
			targetSHA = rs
		} else {
			targetSHA = Branch
		}
	}

	staged, err := DownloadPatchFiles(targetSHA)
	if err != nil {
		tarStaged, tarErr := downloadPatchFilesFromTarball(targetSHA)
		if tarErr != nil {
			return -1, fmt.Sprintf("download failed: contents API: %v; tarball: %v", err, tarErr)
		}
		staged = tarStaged
	}

	// Validate every staged file before touching patchDir.
	for name, content := range staged {
		var obj map[string]interface{}
		if err := json.Unmarshal(content, &obj); err != nil {
			return -1, fmt.Sprintf("remote %s: invalid JSON — %v", name, err)
		}
		if _, ok := obj["id"]; !ok {
			return -1, fmt.Sprintf("remote %s: missing 'id' field", name)
		}
	}

	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		return -1, fmt.Sprintf("mkdir failed: %v", err)
	}

	// Collect existing .json files.
	existing := map[string]bool{}
	entries, _ := os.ReadDir(patchDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			existing[e.Name()] = true
		}
	}

	// Stage into sibling .unleash-new files, then atomic-rename as a group.
	type stagedPair struct{ tmp, dst string }
	var stagedPaths []stagedPair

	rollbackStaged := func() {
		for _, sp := range stagedPaths {
			_ = os.Remove(sp.tmp)
		}
	}

	for name, content := range staged {
		dst := filepath.Join(patchDir, name)
		// Skip unchanged files.
		if cur, err := os.ReadFile(dst); err == nil && string(cur) == string(content) {
			continue
		}
		tmp := filepath.Join(patchDir, "."+name+".unleash-new")
		if err := os.WriteFile(tmp, content, 0o644); err != nil {
			rollbackStaged()
			return -1, fmt.Sprintf("stage failed: %v", err)
		}
		stagedPaths = append(stagedPaths, stagedPair{tmp, dst})
	}

	// Atomic rename pass.
	changed := 0
	for _, sp := range stagedPaths {
		if err := os.Rename(sp.tmp, sp.dst); err != nil {
			// On Windows, os.Rename can fail if dst is open. Try remove+rename.
			_ = os.Remove(sp.dst)
			if err2 := os.Rename(sp.tmp, sp.dst); err2 != nil {
				// Not fatal — continue with what we can.
				continue
			}
		}
		changed++
	}

	// Remove stale files (exist locally but not in remote).
	for name := range existing {
		if _, ok := staged[name]; !ok {
			_ = os.Remove(filepath.Join(patchDir, name))
			changed++
		}
	}

	SaveState(map[string]interface{}{
		"patches_commit": targetSHA,
		"patches_count":  len(staged),
	})

	return changed, targetSHA
}

// splitTarPath splits a tar entry name into path components, handling both
// forward-slash and backslash separators.
func splitTarPath(name string) []string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimSuffix(name, "/")
	if name == "" {
		return nil
	}
	return strings.Split(name, "/")
}

// ── autoheal ────────────────────────────────────────────────────────────────

// AutohealCallbacks bundles the function callbacks autoheal needs.
type AutohealCallbacks struct {
	// FindTarget returns (path, kind). path=="" means not installed.
	FindTarget func() (string, string)
	// SHA256Short returns the short SHA256 of a file.
	SHA256Short func(path string) string
	// LoadPatches loads patches from patchDir (unused here but kept for parity).
	LoadPatches func()
	// CmdVerify runs verify, returns 0 on success.
	CmdVerify func() int
	// CmdPatch runs patch (non-dry-run), returns 0 on success.
	CmdPatch func() int
	// CmdRollback runs rollback. May be nil.
	CmdRollback func() error
}

// Autoheal detects Claude Code drift and self-updates + re-patches if needed.
//
// Returns: 0 = ok (unchanged), 1 = healed, 2 = drift but verified, 3 = failure.
func Autoheal(cb AutohealCallbacks, patchDir string, force, quiet bool) int {
	log := func(msg string) {
		if !quiet {
			fmt.Println(msg)
		}
	}

	target, kind := cb.FindTarget()
	if target == "" {
		log("unleash autoheal: Claude Code not installed — nothing to do")
		return 0
	}

	curSHA := cb.SHA256Short(target)
	state := LoadState()

	var lastSHA string
	if v, ok := state["last_cc_sha"].(string); ok {
		lastSHA = v
	}
	var lastPMT float64
	if v, ok := state["last_patch_mtime"].(float64); ok {
		lastPMT = v
	}

	// Largest mtime across patch dir.
	curPMT := maxPatchMtime(patchDir)

	binDrifted := curSHA != lastSHA
	patchDrifted := int64(curPMT) != int64(lastPMT)

	if !binDrifted && !patchDrifted && !force {
		log(fmt.Sprintf("unleash autoheal: CC unchanged (%s), patches unchanged — skip", curSHA))
		return 0
	}

	if binDrifted {
		log(fmt.Sprintf("unleash autoheal: CC drift detected (%s → %s)", lastSHA, curSHA))
	} else if patchDrifted {
		log(fmt.Sprintf("unleash autoheal: patch updates detected (mtime %d → %d)", int64(lastPMT), int64(curPMT)))
	}

	// Step 1 — verify current patches against new binary.
	rc := cb.CmdVerify()
	if rc == 0 {
		log("unleash autoheal: patches still valid, updating state")
		SaveState(map[string]interface{}{
			"last_cc_sha":      curSHA,
			"last_cc_kind":     kind,
			"last_patch_mtime": int64(curPMT),
		})
		return 2
	}

	// Step 2 — patches broken, pull latest from GitHub.
	log("unleash autoheal: patches broken, syncing latest from GitHub")
	changed, shaOrErr := SyncPatches(patchDir, "")
	if changed < 0 {
		log(fmt.Sprintf("unleash autoheal: sync failed — %s", shaOrErr))
		return 3
	}
	if len(shaOrErr) >= 7 {
		log(fmt.Sprintf("unleash autoheal: synced %d file(s) @ %s", changed, shaOrErr[:7]))
	} else {
		log(fmt.Sprintf("unleash autoheal: synced %d file(s) @ %s", changed, shaOrErr))
	}

	// Step 3 — re-apply patches.
	rc = cb.CmdPatch()
	if rc != 0 {
		log("unleash autoheal: re-patch failed — rolling back to pre-patch backup")
		if cb.CmdRollback != nil {
			if err := cb.CmdRollback(); err != nil {
				log(fmt.Sprintf("unleash autoheal: rollback failed — %v", err))
			}
		}
		return 3
	}

	// Step 4 — confirm verify passes after fresh patch.
	rcV := cb.CmdVerify()
	if rcV != 0 {
		log("unleash autoheal: post-patch verify failed — rolling back")
		if cb.CmdRollback != nil {
			if err := cb.CmdRollback(); err != nil {
				log(fmt.Sprintf("unleash autoheal: rollback failed — %v", err))
			}
		}
		return 3
	}

	target2, _ := cb.FindTarget()
	finalPMT := maxPatchMtime(patchDir)
	finalSHA := curSHA
	if target2 != "" {
		finalSHA = cb.SHA256Short(target2)
	}

	SaveState(map[string]interface{}{
		"last_cc_sha":      finalSHA,
		"last_cc_kind":     kind,
		"last_patch_mtime": int64(finalPMT),
	})
	log("unleash autoheal: healed")
	return 1
}

// maxPatchMtime returns the largest mtime (as unix seconds) across *.json in dir.
func maxPatchMtime(dir string) float64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var max int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		mt := info.ModTime().Unix()
		if mt > max {
			max = mt
		}
	}
	return float64(max)
}

// ── upstream status ─────────────────────────────────────────────────────────

// UpstreamInfo holds the result of comparing local patch sync state with remote HEAD.
type UpstreamInfo struct {
	LocalCommit     string `json:"local_commit"`
	RemoteCommit    string `json:"remote_commit"`
	Drift           bool   `json:"drift"`
	UpdateAvailable bool   `json:"update_available"`
	StateMissing    bool   `json:"state_missing"`
	LocalFiles      int    `json:"local_files"`
}

// UpstreamStatus compares local patches commit vs remote HEAD.
func UpstreamStatus(patchDir string) UpstreamInfo {
	return UpstreamStatusWithRemote(patchDir, RemoteHeadSHA("patches"))
}

// UpstreamStatusWithRemote compares local patch sync state with a supplied remote SHA.
// It is split out so callers and tests can avoid a second network request after
// already resolving the remote commit.
func UpstreamStatusWithRemote(patchDir, remote string) UpstreamInfo {
	state := LoadState()
	var local string
	if v, ok := state["patches_commit"].(string); ok {
		local = v
	}

	count := 0
	entries, _ := os.ReadDir(patchDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}

	stateMissing := remote != "" && local == ""
	drift := remote != "" && local != "" && local != remote
	return UpstreamInfo{
		LocalCommit:     local,
		RemoteCommit:    remote,
		Drift:           drift,
		UpdateAvailable: stateMissing || drift,
		StateMissing:    stateMissing,
		LocalFiles:      count,
	}
}
