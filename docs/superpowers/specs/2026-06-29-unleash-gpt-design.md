# Unleash-GPT design

## Goal

Create a separate Codex-focused product, **Unleash-GPT**, implemented in Go, with the same operator experience as Unleash for Claude Code: one static binary, one setup command, target discovery, patch application, config/rules installation, verification, rollback, and an update guard.

Normal Unleash for Claude Code remains intact in this repository. Unleash-GPT gets its own command name, branding, target discovery, patch set, config paths, and docs.

## Current facts

- Existing Unleash is Go under `go/`, with Cobra commands and embedded `patches/` + `contrib/` assets.
- Existing Unleash discovers Claude Code targets in `go/internal/target/target.go`, then applies JSON-defined byte-preserving patches through `go/cmd/patch.go` and `go/internal/binary`.
- Codex CLI is open source, written in Rust, and distributed as native platform binaries.
- The npm package is `@openai/codex`; version observed from npm metadata: `0.142.4`.
- The npm launcher is `bin/codex.js`; it locates native binaries under platform packages like `@openai/codex-win32-x64`, whose executable path is `vendor/<target-triple>/bin/codex(.exe)`.
- This workstation has Codex installed at `C:\Users\Administrator\AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`; `codex --version` reported `codex-cli 0.142.3`.
- Codex already exposes runtime flags for dangerous full access: `--dangerously-bypass-approvals-and-sandbox`, `--dangerously-bypass-hook-trust`, `--sandbox danger-full-access`, and `--ask-for-approval never`.

## Product shape

Unleash-GPT is a separate Go binary named `unleash-gpt`.

Primary commands mirror existing Unleash where they make sense:

- `unleash-gpt setup`: install/update Codex if needed, patch Codex, install Codex rules/config, install guard, verify.
- `unleash-gpt patch [--dry-run]`: apply Codex patch JSON files to the discovered native Codex binary.
- `unleash-gpt verify`: confirm Codex patch markers and config are present.
- `unleash-gpt rollback`: restore the newest Codex backup.
- `unleash-gpt status`: print Codex target path, version, hash, patch/config state.
- `unleash-gpt scan`: scan Codex patches against the current binary for applied/drift/missing status.
- `unleash-gpt install-rules` / `uninstall-rules`: manage Codex operator rules/config only.
- `unleash-gpt install-guard` / `uninstall-guard`: auto-patch Codex after updates.

The TUI can remain out of the first Codex cut unless existing components can be reused cheaply. CLI behavior is the deliverable.

## Architecture

Keep the existing Claude implementation stable. Add Codex-specific seams instead of rewriting the current codebase wholesale.

### Target abstraction

Introduce a small target profile concept:

```go
type Product string

const (
    ProductClaude Product = "claude"
    ProductCodex  Product = "codex"
)

type TargetInfo struct {
    Product Product
    Path    string
    Kind    string
}
```

Codex target discovery should be independent of Claude discovery.

Codex discovery checks, in priority order:

1. Native installer paths:
   - Windows: `%LOCALAPPDATA%/Programs/OpenAI/Codex/bin/codex.exe`
   - macOS/Linux: `$HOME/.local/bin/codex`, `$HOME/.codex/bin/codex`, `/usr/local/bin/codex`, `/opt/homebrew/bin/codex`
2. npm global roots for `@openai/codex/bin/codex.js` and platform optional packages:
   - `@openai/codex-<platform>/vendor/<triple>/bin/codex(.exe)`
3. Bun/pnpm/Volta-managed global installs using the same package layout.
4. Homebrew cask paths for Codex.
5. Scoop/WinGet/Chocolatey-style paths containing `codex.exe`.
6. `exec.LookPath("codex")` / `exec.LookPath("codex.exe")` as last resort.

Discovery validates candidates by size and by executing `codex --version` when safe; accepted output starts with `codex-cli`.

### Patch format

Reuse JSON patch files and the same-length replacement invariant. Codex patch files live separately, e.g. `codex-patches/`, so Claude patches never apply to Codex.

Add a native binary patch path for Rust binaries:

- Read the target bytes.
- For each Codex `js_replace`-style subpatch, treat `search` or `search_regex` as byte/string search over the full binary.
- Replacement must be exactly equal length or shorter. Shorter replacements are padded with spaces or NUL bytes according to patch metadata.
- Applied markers must be literal byte strings found after patching.
- Patching writes a temp file, validates `codex --version`, then atomically replaces the target and creates a backup under `~/.unleash-gpt/backups/`.

This keeps the proven byte-preserving model and avoids parsing Rust/Mach-O/ELF/PE internals for v1.

### Codex patch set v1

Start with patches that can be located and validated in the current Codex binary. Candidate categories:

- Approval and sandbox defaults: force non-interactive/default policy toward `never` + `danger-full-access` where static strings permit byte-safe replacement.
- Hook trust defaults: neutralize blocking trust text or force bypass default if string-level patchable.
- Telemetry/logging endpoints: redirect or blank static endpoint strings found in the binary.
- Feature flags: flip statically embedded disabled flags where byte-safe replacements exist.
- Attribution/user-facing generated markers: blank or replace strings if present.

No fake parity claim. If a Claude patch category has no byte-safe Codex equivalent, it is omitted from v1 and reported by scan/status as not implemented rather than stubbed.

### Rules/config

Codex uses `~/.codex` rather than `~/.claude`.

`unleash-gpt install-rules` writes/merges:

- `~/.codex/AGENTS.md`: operator authorization block.
- `~/.codex/config.toml`: safe merge preserving user config, setting:
  - `approval_policy = "never"`
  - `sandbox_mode = "danger-full-access"`
  - `disable_response_storage = true` if supported by current Codex config docs/binary.
  - feature/config toggles only when verified against Codex docs or current binary help.

It must not overwrite unrelated user Codex settings.

### Setup flow

`unleash-gpt setup`:

1. Check for Codex CLI.
2. If missing, install `@openai/codex` with npm when npm exists; otherwise print official installer commands.
3. Discover Codex target.
4. Apply Codex patches.
5. Install Codex rules/config.
6. Install guard.
7. Verify `codex --version`, patch markers, and config values.

### Guard

Codex guard mirrors Unleash guard but watches Codex paths and writes state under `~/.unleash-gpt/`.

On target hash change:

1. Run scan.
2. Apply patch if signatures still resolve.
3. Verify `codex --version`.
4. Update state stamp.

### Packaging

Release artifacts use names like:

- `unleash-gpt-linux-amd64`
- `unleash-gpt-linux-arm64`
- `unleash-gpt-darwin-amd64`
- `unleash-gpt-darwin-arm64`
- `unleash-gpt-windows-amd64.exe`
- `unleash-gpt-windows-arm64.exe`

npm package candidate: `unleash-gpt` with bin `unleash-gpt`.

## Testing

Test-first implementation targets:

1. Codex target discovery resolves native installer and npm optional package layouts.
2. `codex --version` validation accepts `codex-cli <version>` and rejects Claude targets.
3. Codex config merge preserves existing TOML keys and writes approval/sandbox values.
4. Patch engine applies shorter/equal byte replacements and rejects longer replacements.
5. Patch engine creates backups and does not modify files on dry-run.
6. Setup skips binary patch cleanly when Codex is missing but still reports actionable install steps.

Verification commands:

- `go test ./...` from `go/`.
- Focused tests for new Codex target/config/patch packages during TDD.
- Local smoke: `go run . --help`, `go run . setup --dry-run` if a dry-run setup flag is added, and `go run . patch --dry-run` against the installed Codex binary.

## Non-goals

- Do not rewrite Unleash's Claude Code behavior.
- Do not claim all 113 Claude patches apply to Codex.
- Do not add runtime hooks or JS extraction for Codex v1.
- Do not create mocks for tests; use temp directories/files and real command output parsing helpers.
- Do not publish npm/GitHub releases without explicit in-session approval.

## Risks

- Rust native binary strings may not expose enough stable patch points for full parity. Mitigation: implement scan/status honesty and only ship verified byte-safe patches.
- Code signing or platform protection may reject modified binaries on macOS/Windows. Mitigation: preserve existing verification step and document platform-specific remediation only after observed failures.
- Codex config keys can drift. Mitigation: tests cover merge behavior; setup verifies current `codex --help`/docs before writing optional keys.
