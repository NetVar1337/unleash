# Unleash-GPT Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a separate Go `unleash-gpt` CLI for Codex CLI target discovery, config/rules installation, dry-run byte patch scanning, status, verify, rollback, and setup.

**Architecture:** Add Codex-specific packages beside the existing Claude implementation instead of rewriting existing Unleash. The first working cut uses Codex's supported config controls for approvals/sandbox and a conservative byte patcher that can apply Codex JSON patches when verified signatures exist. Existing Claude commands and patch files remain unchanged.

**Tech Stack:** Go, Cobra, stdlib filesystem/process helpers, embedded assets through existing `go/embed` pattern.

## Global Constraints

- Product name: `Unleash-GPT`; command name: `unleash-gpt`.
- Keep normal Unleash for Claude Code intact; Codex files and state must be separate.
- Codex config path: `~/.codex/config.toml`; Codex rules path: `~/.codex/AGENTS.md`.
- Codex state path: `~/.unleash-gpt/`; backups path: `~/.unleash-gpt/backups/`.
- Preserve existing user config/rules when installing Codex rules.
- Do not publish npm/GitHub releases.
- Use TDD: write failing tests before production code.

---

## File Structure

- Create `go/internal/codex/target.go`: Codex target discovery, version validation, hashing, backup directory.
- Create `go/internal/codex/target_test.go`: target discovery and validation tests with temp fake binaries.
- Create `go/internal/codex/config.go`: AGENTS/config TOML merge and uninstall helpers.
- Create `go/internal/codex/config_test.go`: config merge/rules installation tests.
- Create `go/internal/codex/patcher.go`: conservative byte replacement engine for Codex patch JSON files.
- Create `go/internal/codex/patcher_test.go`: dry-run, padding, backup, reject-longer tests.
- Create `go/cmd/unleash-gpt/main.go`: separate Cobra root and Codex commands.
- Modify `README.md`: add a short Unleash-GPT build/usage note without changing current Unleash quickstart semantics.

---

### Task 1: Codex Target Discovery

**Files:**
- Create: `go/internal/codex/target.go`
- Test: `go/internal/codex/target_test.go`

**Interfaces:**
- Produces: `type Target struct { Path string; Kind string; Version string }`
- Produces: `func FindTargetWithEnv(env map[string]string, lookPath func(string) (string, error)) (Target, bool)`
- Produces: `func LooksLikeCodexVersion(output string) (string, bool)`
- Produces: `func SHA256Short(path string) string`
- Produces: `func BackupDir(home string) string`

- [ ] **Step 1: Write failing tests**

Add tests that create temp Codex layouts and assert discovery order:

```go
func TestLooksLikeCodexVersion(t *testing.T) {
    version, ok := LooksLikeCodexVersion("codex-cli 0.142.3\n")
    if !ok || version != "0.142.3" { t.Fatalf("got %q %v", version, ok) }
    if _, ok := LooksLikeCodexVersion("Claude Code 2.1.195\n"); ok { t.Fatal("accepted Claude version") }
}

func TestFindTargetNativeWindowsLayout(t *testing.T) {
    root := t.TempDir()
    exe := filepath.Join(root, "Programs", "OpenAI", "Codex", "bin", "codex.exe")
    writeFakeExe(t, exe, 2_000_000)
    got, ok := FindTargetWithEnv(map[string]string{"LOCALAPPDATA": root, "HOME": root, "USERPROFILE": root}, nil)
    if !ok { t.Fatal("target not found") }
    if got.Path != exe || got.Kind != "native" { t.Fatalf("got %#v", got) }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/codex -run 'TestLooksLikeCodexVersion|TestFindTargetNativeWindowsLayout'`
Expected: FAIL because package/functions do not exist.

- [ ] **Step 3: Implement target discovery**

Implement candidate checks for native Windows path, native Unix paths, npm optional package layout, and `lookPath` fallback. Validate by file size > 1 MB; version execution can be a separate helper used by commands, not by tests.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/codex -run 'TestLooksLikeCodexVersion|TestFindTargetNativeWindowsLayout'`
Expected: PASS.

---

### Task 2: Codex Rules and Config Merge

**Files:**
- Create: `go/internal/codex/config.go`
- Test: `go/internal/codex/config_test.go`

**Interfaces:**
- Consumes: none from Task 1.
- Produces: `func InstallRules(home string, authBlock string) error`
- Produces: `func UninstallRules(home string) error`
- Produces: `func MergeCodexConfig(existing string) string`

- [ ] **Step 1: Write failing tests**

Test that merge preserves unrelated keys and forces Codex approvals/sandbox:

```go
func TestMergeCodexConfigPreservesExistingKeys(t *testing.T) {
    got := MergeCodexConfig("model = \"gpt-5.1-codex-max\"\n")
    for _, want := range []string{"model = \"gpt-5.1-codex-max\"", "approval_policy = \"never\"", "sandbox_mode = \"danger-full-access\""} {
        if !strings.Contains(got, want) { t.Fatalf("missing %q in:\n%s", want, got) }
    }
}

func TestInstallRulesWritesAgentsAndConfig(t *testing.T) {
    home := t.TempDir()
    if err := InstallRules(home, "operator authorization"); err != nil { t.Fatal(err) }
    assertFileContains(t, filepath.Join(home, ".codex", "AGENTS.md"), "operator authorization")
    assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "approval_policy = \"never\"")
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/codex -run 'TestMergeCodexConfig|TestInstallRules'`
Expected: FAIL because functions do not exist.

- [ ] **Step 3: Implement config merge**

Use line-based TOML top-level key replacement for `approval_policy` and `sandbox_mode`; append missing keys. Prepend a marked Unleash-GPT block in AGENTS.md and replace only that block on reinstall.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/codex -run 'TestMergeCodexConfig|TestInstallRules'`
Expected: PASS.

---

### Task 3: Codex Byte Patcher

**Files:**
- Create: `go/internal/codex/patcher.go`
- Test: `go/internal/codex/patcher_test.go`

**Interfaces:**
- Produces: `type Patch struct { ID string; Patches []SubPatch }`
- Produces: `type SubPatch struct { Search string; Replace string; AppliedMarker string; Count int }`
- Produces: `func ApplyPatches(path string, patchList []Patch, dryRun bool, home string) (PatchResult, error)`

- [ ] **Step 1: Write failing tests**

Cover dry-run, equal replacement, shorter replacement padded with spaces, and longer replacement rejection.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/codex -run 'TestApplyPatches'`
Expected: FAIL because patcher does not exist.

- [ ] **Step 3: Implement patcher**

Implement literal byte search/replace only. If replacement is longer than search, return error and do not write. If dry-run, report would-apply and do not write. If write, create backup under `~/.unleash-gpt/backups/` first and replace file bytes.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/codex -run 'TestApplyPatches'`
Expected: PASS.

---

### Task 4: Unleash-GPT CLI

**Files:**
- Create: `go/cmd/unleash-gpt/main.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: Target discovery, config, patcher.
- Produces: buildable command: `go build ./cmd/unleash-gpt`.

- [ ] **Step 1: Write failing CLI smoke test if practical**

If Cobra command construction is exposed as `newRootCmd()`, test `--help` includes `Unleash-GPT` and `setup`, `patch`, `status`, `verify`, `install-rules`, `uninstall-rules`, `rollback`.

- [ ] **Step 2: Run failure**

Run: `go test ./cmd/unleash-gpt` or `go test ./...`.
Expected: FAIL until command exists.

- [ ] **Step 3: Implement Cobra commands**

Implement commands with existing console style where practical. `setup` should discover Codex, run patch dry-run/apply, install rules, and verify. `patch` supports `--dry-run`. `status` prints target and version. `verify` checks config values and target exists. `rollback` restores latest backup.

- [ ] **Step 4: Run build and tests**

Run: `go test ./...`
Run: `go build ./cmd/unleash-gpt`
Run: `go run ./cmd/unleash-gpt --help`
Expected: all pass; help shows Unleash-GPT commands.

---

### Task 5: Final Verification

**Files:**
- All changed files.

- [ ] **Step 1: Run focused package tests**

Run: `go test ./internal/codex ./cmd/unleash-gpt`
Expected: PASS.

- [ ] **Step 2: Run full Go test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 3: Run local smoke commands**

Run: `go run ./cmd/unleash-gpt status`
Expected: prints Codex target path or not-found message without panic.

Run: `go run ./cmd/unleash-gpt patch --dry-run`
Expected: reports target state and exits 0.

- [ ] **Step 4: Inspect changed files for placeholders**

Search changed files for `TODO`, `TBD`, `panic("TODO")`, `test.skip`, `.only`.
Expected: no blockers.
