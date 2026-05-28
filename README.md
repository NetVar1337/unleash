<div align="center">

```
                                    ██╗   ██╗██████╗  ██████╗ ██████╗
                                    ██║   ██║██╔══██╗██╔════╝██╔════╝
                                    ██║   ██║██████╔╝██║     ██║
                                    ╚██╗ ██╔╝██╔═══╝ ██║     ██║
                                     ╚████╔╝ ██║     ╚██████╗╚██████╗
                                      ╚═══╝  ╚═╝      ╚═════╝ ╚═════╝

                                   v p c c  ·  v o i d - p a t c h e r - c c
```

### **Unleash Claude Code.**

**106 patches (100 active) · zero refusals · zero telemetry · zero permission prompts · 42 feature gates unlocked**

<br />

[![patches](https://img.shields.io/badge/patches-106-8b5cf6?style=for-the-badge&logo=hackthebox&logoColor=white)](#patch-catalog)
[![feature gates](https://img.shields.io/badge/feature%20gates-42%20unlocked-22d3ee?style=for-the-badge&logo=flipboard&logoColor=white)](#feature-gate-mega-patch-42-flags)
[![zero refusals](https://img.shields.io/badge/AUP%20blocks-0-ef4444?style=for-the-badge&logo=adblock&logoColor=white)](#authorization-rules)
[![telemetry](https://img.shields.io/badge/telemetry-disabled-10b981?style=for-the-badge&logo=ghostery&logoColor=white)](#what-gets-patched)
[![platforms](https://img.shields.io/badge/macOS%20·%20Linux%20·%20Windows-supported-f59e0b?style=for-the-badge&logo=apple&logoColor=white)](#cross-platform-support)
[![scan speed](https://img.shields.io/badge/scan-137ms%20cached-ec4899?style=for-the-badge&logo=lightning&logoColor=white)](#performance)

<br />

**`pipx install void-patcher-cc && vpcc patch && vpcc install-rules --no-hook`**

<br />

[Quick Start](#-quick-start) · [What Gets Patched](#-what-gets-patched) · [Patch Catalog](#-patch-catalog) · [Performance](#-performance) · [Scanner](#-scanner--drift-detection) · [Platforms](#-cross-platform-support) · [Authorization](#-authorization-rules) · [Commands](#-commands)

</div>

---

> **vpcc** is a single-shot binary patcher for Anthropic's [Claude Code](https://docs.claude.com/en/docs/claude-code) CLI. It rewrites the Bun SEA bytecode in-place to remove every permission gate, refusal classifier, AUP block, telemetry sink, and feature lock. No rebuilds. No JavaScript extraction. No runtime injection.
>
> Use it for: operator-authorized red-team work, security research, agent autonomy experiments, removing CC's UX friction on your own hardware.

## 🚀 Quick Start

```bash
# Install (any of these — pipx recommended, no venv noise)
pipx install void-patcher-cc
# or:  pip install void-patcher-cc
# or:  uv tool install void-patcher-cc

# Patch — single command, idempotent, auto-backup
vpcc patch

# Verify everything landed
vpcc verify

# Deploy operator-authorization bundle (no permission prompts, no telemetry)
# --no-hook = zero subprocess overhead per CC tool call (binary patch handles allow)
vpcc install-rules --no-hook
```

<details>
<summary><b>🪄 One-shot full setup</b></summary>

```bash
pipx install void-patcher-cc && vpcc patch && vpcc install-rules --no-hook
```

</details>

<details>
<summary><b>🔁 After every Claude Code auto-update</b></summary>

```bash
vpcc upgrade        # self-update + autoheal + verify + warm cache, one shot
```

</details>

<details>
<summary><b>🪟 Windows (PowerShell)</b></summary>

```powershell
pipx install void-patcher-cc
vpcc patch
vpcc install-rules --no-hook
```

</details>

---

## 🎯 What Gets Patched

```
 ┌──────────────────────────────────────────────────────────────────────────┐
 │                                                                          │
 │  PERMISSION GATES         ██████████████████████  22 patches             │
 │  REFUSAL / AUP BLOCKS     ████████████████████    20 patches             │
 │  CLASSIFIERS / SAFETY     ██████████████          14 patches             │
 │  TELEMETRY / ANALYTICS    █████████████           13 patches             │
 │  FEATURE GATES (tengu_)   ██████████████████████████████████████  42 ⚡  │
 │  RATE LIMITS / TIMEOUTS   ████████████            12 patches             │
 │  SHELL SAFETY CHECKS      ██████                   6 patches             │
 │  UI / UX UNLOCKS          ████████                 8 patches             │
 │                                                                          │
 └──────────────────────────────────────────────────────────────────────────┘
```

**Total: 106 patches** across 9 categories. 100 active binary patches, 6 retired, 3 infrastructure.

> Every patch is a standalone JSON file in `patches/` — anchored, marker-tagged, drift-detectable, auto-healable.

---

## ⚡ Performance

vpcc is fast and stays out of Claude Code's hot path.

| Operation | Cold | Warm (cached) | Speedup |
|---|---:|---:|---:|
| `vpcc scan` (106 patches on 237 MB binary) | **5.7 s** | **137 ms** | **41×** |
| `vpcc verify` | 0.3 s | 0.3 s | — |
| `vpcc doctor` (full health report) | 5.8 s | 0.4 s | 14× |
| sha256 of 237 MB binary | 140 ms | — | — |
| Bun SEA `.bun` section extract + decode | 700 ms | 0 ms | ∞ |
| **CC per-tool-call hook overhead** (with `--no-hook`) | — | **0 ms** | ∞ |

### How it stays fast

- **Scan cache** — `~/.vpcc/cache/scan_<sha>_<patchmtime>.json` keyed by binary SHA256 + max patch-dir mtime. Auto-invalidated on either change.
- **Anchor-first cascade** — 3 ms anchor check before 200 ms regex search.
- **Bulk dedup'd marker pre-sweep** — single C-level pass for all unique markers across patches.
- **`hashlib.file_digest`** — zero-copy streaming sha256 (Python 3.11+).
- **No CC overhead** — `vpcc install-rules --no-hook` deploys authorization without a `PreToolUse` shell hook. The binary patch (`05-auto-allow-hook`) handles the allow decision at C-level. Zero subprocess spawn per tool call.

```bash
$ vpcc bench
vpcc bench — 2.1.152 (237 MB)
  sha256             :  140 ms
  text load+decode   :  700 ms
  scan_patches       : 4974 ms  (100 patches)
  cache hit          :  137 ms

  cold total         : 5681 ms
  warm total         :  137 ms
  speedup            :   41×
```

---

## 📚 Patch Catalog

### Permission Bypass (22 patches)

| # | Patch | Effect |
|---|---|---|
| 01 | `bypass-permissions` | `defaultMode: bypassPermissions`, disable sandbox, skip trust checks |
| 02 | `env-flags` | 16 env vars: `CLAUDE_CODE_PERMISSION_MODE=bypassPermissions`, disable telemetry, enable experimental features |
| 03 | `js-trust-dialog` | Auto-accept trust dialog, onboarding, external includes |
| 04 | `js-bypass-mode` | Force `isBypassPermissionsModeAvailable = true` in runtime |
| 05 | `auto-allow-hook` | PreToolUse hook emits `{"permissionDecision":"allow"}` for every tool call |
| 09 | `js-allow-skip-permissions` | Default `allowDangerouslySkipPermissions = true` |
| 10 | `js-disable-bypass-check` | Neutralize operator `disableBypassPermissionsMode` remote kill switch |
| 12 | `js-session-bypass-mode` | Sessions start in bypass mode by default |
| 13 | `js-session-trust` | Skip workspace trust verification on session start |
| 34 | `js-bypass-permissions-statsig-recheck-kill` | Kill mid-session Statsig recheck that can disable bypass |
| 35 | `js-statsig-gate-kill-switches-off` | Known kill-switch gates always return false |
| 46 | `js-bypass-perm-mode-not-available-fake-ok` | Runtime bypass mode switch always succeeds |
| 47 | `js-bypass-perm-mode-not-available-sdk-fake-ok` | SDK/control-response bypass switch always succeeds |
| 51 | `js-auto-mode-disable-settings-bypass` | Auto mode never disabled by settings |
| 53 | `js-Xj-permissionmode-allowall` | Permission-mode mapper forces every mode to "allow" |
| 55 | `js-plugin-deny-allowlist-passthrough` | Plugin channel allowlist check always passes |
| 58 | `js-rule-deny-allow` | Sandbox rule `deny` → `allow` |
| 59 | `js-tg7-permission-writer-false` | Write-permission predicate always returns false (no prompt needed) |
| 69 | `js-plugin-org-denylist-passthrough` | Enterprise plugin denylist neutralized |
| 81 | `js-bypass-permissions-async-kill` | Async bypass-permissions kill no-op |
| 82 | `js-bypass-permissions-sync-kill` | Sync startup bypass check no-op |
| 38 | `js-policy-limits-allowall` | Central capability gate always returns true |

### Refusal & AUP Removal (20 patches)

| # | Patch | Effect |
|---|---|---|
| 11 | `js-plan-mode-refusal` | Plan mode allows execution — not read-only |
| 15 | `js-aup-refusal` | AUP violation message replaced with operator authorization |
| 17 | `js-aup-refusal-2` | Second AUP variant ("double press esc") replaced |
| 18 | `js-prompt-injection-sysprompt` | Prompt injection flagging removed from system prompt |
| 24 | `js-plan-mode-disable` | `planModeRequired` always false |
| 26 | `js-security-guardrail` | Security research guardrail replaced with authorization |
| 28 | `js-plan-mode-cannot` | "You CANNOT and MUST NOT write" directive removed |
| 29 | `js-denial-workaround` | "Don't work around this denial" replaced with permission |
| 32 | `js-refusal-stop-reason-neutralize` | Refusal stop_reason silently ignored |
| 44 | `js-generated-with-claude-footer-off` | PR attribution footer blanked |
| 52 | `js-s5K-refusal-neutralize` | Refusal handler short-circuited |
| 54 | `js-aup-refusal-sanitize` | Composed AUP refusal message neutralized |
| 55r | `js-refusal-explanation-null` | Refusal explanation field nulled |
| 67 | `js-computer-use-policy-refusal` | Computer-use policy denylist neutralized |
| 68 | `js-co-authored-by-claude-off` | Co-Authored-By attribution stripped |
| 78 | `js-webfetch-lyrics-copyright-clause` | Song lyrics / copyright restriction removed |
| 90 | `js-generated-with-footer-off-extra` | All remaining attribution templates blanked |
| 92 | `js-no-doc-creation-directive-off` | "NEVER create documentation files" directive removed |
| 50 | `js-brief-mode-entitlement-allow` | Brief mode always entitled |
| 84 | `js-plan-mode-interview-phase-on` | Plan mode interview phase enabled |

### Classifier & Safety (14 patches)

| # | Patch | Effect |
|---|---|---|
| 14 | `js-classifier-failopen` | Auto-mode classifier fails **open** (allow) instead of closed (block) |
| 16 | `js-classifier-all-failopen` | All classifier error paths fail open |
| 33 | `js-auto-mode-classifier-shouldblock-false` | `shouldBlock` flipped to false at both block-branch gates |
| 56j | `js-jtH-safe-always-true` | Safety-check helper always returns `{safe: true}` |
| 57 | `js-canusetool-safetycheck-allow` | `canUseTool` permission gate always `{allowed: true}` |
| 60 | `js-twostage-classifier-always-on` | Two-stage classifier gate forced true |
| 79 | `js-command-injection-classifier-neutralize` | Command injection classifier → pass (treats shell as its own prefix) |
| 80 | `js-dangerous-shell-prefix-neutralize` | Dangerous shell prefix set → pass |
| 88 | `js-classifier-summary-kill-on` | Summary classifier kill gate enabled |
| 98 | `js-sed-dangerous-check-disable` | sed approval check bypassed |
| 99 | `js-jq-dangerous-flags-disable` | jq dangerous flags check → ok:true |
| 100 | `js-find-dangerous-action-disable` | find -exec/-delete check → ok:true |
| 101 | `js-write-exfiltration-cap-raise` | Write operation exfiltration cap disabled |
| 102 | `js-dangerous-redirection-disable` | Shell redirection safety → always false |

### Telemetry & Analytics (13 patches)

| # | Patch | Effect |
|---|---|---|
| 19 | `js-metrics-disable` | Metrics reporting to `api.anthropic.com/api/claude_code/metrics` disabled |
| 23 | `js-additional-protection` | `x-anthropic-additional-protection` header removed |
| 36 | `js-datadog-sink-kill` | Datadog telemetry sink killed |
| 37 | `js-1p-event-logging-off` | 1P OTEL event sink disabled |
| 39 | `js-agent-summary-disable` | Background per-agent summarization API call disabled |
| 56 | `js-plugin-session-telemetry-off` | Per-session plugin inventory telemetry stubbed |
| 57p | `js-plugin-load-failed-telemetry-off` | Plugin error telemetry stubbed |
| 62 | `js-tengu-plugin-prematureread-off` | Plugin premature-read event dropped |
| 91 | `js-disable-nonessential-traffic-default` | Telemetry network policy → essentials only |
| 75 | `js-econnrefused-silent` | Plugin marketplace download errors silenced |
| 86 | `js-marketplace-etimedout-silent` | Plugin marketplace ETIMEDOUT silenced |
| 105 | `js-off-switch-neutralize` | Remote kill switch (tengu-off-switch) neutralized — Anthropic cannot abort sessions server-side |
| 58h | `js-handle-uri-deeplink-disable` | `--handle-uri` deep-link handler disabled |

### Rate Limits & Timeouts (12 patches)

| # | Patch | Effect |
|---|---|---|
| 40 | `js-bash-default-timeout-raise` | Bash default timeout: **120s → 3600s** (1 hour) |
| 41 | `js-bash-max-timeout-floor-raise` | Bash max timeout floor: **600s → 7200s** (2 hours) |
| 42 | `js-mcp-sendrequest-timeout-raise` | MCP per-request timeout: **30s → 300s** |
| 43 | `js-max-thinking-default-on` | Extended thinking always enabled |
| 52n | `js-non-streaming-fallback-always-on` | Non-streaming fallback always available |
| 64 | `js-bash-max-output-default-raise` | Bash output cap: **30k → 100k** chars |
| 65 | `js-task-max-output-default-raise` | Subagent task output cap: **32k → 128k** chars |
| 66 | `js-agent-implicit-fork-max-turns-raise` | Subagent max turns: **200 → 999** |
| 83 | `js-always-enable-effort-on` | High effort mode always available |
| 96 | `js-opus-fallback-threshold-raise` | Opus fallback threshold: **3 → 9** retries before Sonnet downgrade |
| 97 | `js-fast-mode-cooldown-disable` | Fast mode cooldown disabled — stays active through rate limits |
| 104 | `js-rate-limit-retry-raise` | Max retries: **10 → 30**, max delay: **60s → 300s** |

### Feature Gate Unlock (42 flags)

**Patch 103** — `js-tengu-feature-flags-unlock` — flips 42 Statsig feature gates from `false` → `true`:

<details>
<summary><b>Click to expand all 42 flags</b></summary>

| Flag | What it unlocks |
|---|---|
| `tengu_auto_notice_once` | Auto-mode reminder shown once (not repeated) |
| `tengu_edit_minimalanchor_jrn` | Edit tool uses minimal anchors (1-3 lines) — saves tokens |
| `tengu_chomp_inflection` | Prompt suggestions / autocomplete |
| `tengu_orchid_mantis` | `/schedule` suggestions at task end |
| `tengu_gouda_loop` | Notification when referenced issues close |
| `tengu_mcp_subagent_prompt` | Better MCP result truncation |
| `tengu_immediate_model_command` | Inline `/model` command (faster model switching) |
| `tengu_terminal_sidebar` | Terminal tab status display |
| `tengu_silk_hinge` | Message timestamps in settings |
| `tengu_gleaming_fair` | Session resume dialog after idle timeout |
| `tengu_fg_left_arrow_agents` | Left-arrow agents fleet access |
| `tengu_copper_fox` | Subagent forking for parallel work |
| `tengu_coral_fern` | Memory search instructions |
| `tengu_loud_sugary_rock` | Opus 4.7 code style matching instruction |
| `tengu_marlin_porch` | DECSTBM terminal scroll optimization |
| `tengu_sparrow_ledger` | Verify prompt behavior |
| `tengu_drift_lantern` | Event loop stall detector |
| `tengu_moth_copse` | AutoMem noise filtered from context |
| `tengu_ochre_finch` | Typed memory categories |
| `tengu_hawthorn_steeple` | Tool use ID deduplication |
| `tengu_cobalt_raccoon` | Precomputed compaction |
| `tengu_sepia_moth` | Reactive compaction (auto-compact when context fills) |
| `tengu_hazel_osprey` | API 422/424 overflow error handling |
| `tengu_disable_keepalive_on_econnreset` | Stale HTTP connection recovery |
| `tengu_kairos_loop_dynamic` | Smart dynamic loop scheduling |
| `tengu_kairos_loop_persistent` | Loops survive session restarts |
| `tengu_kairos_loop_prompt` | Loop prompt preamble injection |
| `tengu_kairos_push_notifications` | Push notifications for agent events |
| `tengu_kairos_input_needed_push` | Push notification when agent needs input |
| `tengu_amber_sentinel` | Monitor tool for long-running scripts |
| `tengu_scratch` | Scratchpad directory for temporary work |
| `tengu_passport_quail` | Memory persistence |
| `tengu_slate_thimble` | Memory features in SDK/non-interactive mode |
| `tengu_mocha_barista` | One-time scheduled runs |
| `tengu_willow_prism` | Cost-aware agent steering |
| `tengu_agent_list_attach` | Agent list attached to model messages |
| `tengu_crimson_vector` | Larger image upload size |
| `tengu_prompt_cache_diagnostics` | Prompt cache diagnostic logging |
| `tengu_ashen_kelp` | Wait for MCP servers before tool search |
| `tengu_velvet_static` | `/radio` — Claude FM lo-fi radio |
| `tengu_pewter_brook` | Fullscreen TUI mode by default |
| `tengu_slate_harbor_experiment` | Better `/init` with codebase analysis |

</details>

### Subscription & UI Unlocks (8 patches)

| # | Patch | Effect |
|---|---|---|
| 08 | `js-root-restriction` | Root/sudo restriction removed |
| 21 | `js-subscription-max` | Subscription pinned to `max` — prevents feature downgrades |
| 25 | `js-unlock-ab-flags` | A/B test flags unlocked (quiet_save, enhanced output style) |
| 45 | `js-elevated-priv-stderr-quiet` | Elevated privileges warning silenced |
| 48 | `js-chrome-subscription-require-skip` | Chrome subscription gate removed |
| 49 | `js-voice-mode-subscription-gate-skip` | Voice mode gate removed |
| 53b | `js-experimental-betas-always-on` | Experimental anthropic-beta headers always sent |
| 54a | `js-agent-teams-env-or-cli-always-on` | Agent Teams always available |

### Infrastructure (3 patches)

| # | Patch | Effect |
|---|---|---|
| 06 | `patch-guard-hook` | SessionStart hook re-applies settings after Claude Code self-heals |
| 07 | `mcp-guard` | MCP spawn timeout preload — prevents startup hangs from `npx @latest` |
| 08w | `cli-syntax-selfheal-wrapper` | Wrapper with syntax guard and auto-repair |

### Retired (6 patches)

| # | Patch | Reason |
|---|---|---|
| 27 | `js-malware-refusal` | System prompt text rewritten in v2.1.152 |
| 43 | `js-max-thinking-default-on` | Feature now enabled by default upstream |
| 83 | `js-always-enable-effort-on` | Feature now enabled by default upstream |
| 84 | `js-plan-mode-interview-phase-on` | `tengu_plan_mode_interview_phase` removed in v2.1.152 |
| 94 | `js-session-memory-on` | `tengu_session_memory` removed in v2.1.152 |
| 95 | `js-cold-compact-on` | `tengu_cold_compact` removed in v2.1.152 |

---

## 🛰 Scanner & Drift Detection

The scanner uses a **multi-strategy detection cascade** with confidence scoring:

```
$ vpcc scan

vpcc scan — 100 js_replace patches
  target : ~/.local/share/claude/versions/2.1.152
  format : Bun SEA (Mach-O)

  ok     1.0  regex   js-trust-dialog                      @0x002f7dc0
  ok     1.0  regex   js-bypass-mode                       @0x009369f2
  ok     0.9  anchor  js-metrics-disable                   @0x00657501
  ok     1.0  regex   js-tengu-feature-flags-unlock        --
  ...
  ok     1.0  regex   js-off-switch-neutralize             @0x008c2410
  retired      —      js-max-thinking-default-on           (feature now default upstream)
  ...
  54 ok  28 applied  1 nometa  6 retired
```

### Detection strategies (in order)

| Priority | Method | Confidence | Description |
|---|---|---|---|
| 1 | `marker` | 1.0 | Applied marker found — patch is already live (bulk pre-sweep, one C-pass) |
| 2 | `anchor` | 0.9 | All anchor strings co-located within proximity window |
| 3 | `regex` | 1.0 | Search regex matches current binary (only run if anchors missed) |
| 4 | `scattered` | 0.7 | Anchors present anywhere in text (for spread mega-patches) |
| 5 | `fuzzy_ws` | 0.7 | Anchors match after whitespace normalization |
| 6 | `fuzzy_ident` | 0.5 | Anchors match with minifier-tolerant identifiers |
| 7 | `keyword` | 0.3 | Long tokens extracted from anchors found |
| 8 | `optional` | 0.6 | All-optional mega-patch with anchors present (no-op by design) |
| 0 | `retired` | — | Patch marked as retired — skipped (feature removed upstream) |

### Auto-heal drift

When Claude Code updates break regexes, anchors usually survive. Autoheal re-derives the patches from anchor context windows. Fires on:

- **Binary SHA change** — Claude Code self-updated
- **Patch mtime change** — new patches landed via `vpcc self-update`

```bash
vpcc upgrade        # ⭐ self-update + autoheal + verify + warm cache, one shot
vpcc autoheal       # heal on demand (—force to ignore unchanged-sha short-circuit)
vpcc scan --auto-heal   # regenerate drifted regex from anchor windows
vpcc doctor         # full health report: sha, format, drift, backups, upstream
vpcc watch          # daemon: poll binary, auto-backup + autoheal on change
```

---

## 🌐 Cross-Platform Support

### Operating Systems

| OS | Arch | Binary Format | Status |
|---|---|---|---|
| **macOS** | arm64, x64 | Mach-O | Fully supported |
| **Linux** | x64, arm64, musl | ELF | Fully supported |
| **Windows 10/11** | x64, arm64 | PE/COFF | Fully supported |

### Installation Methods Detected

| Method | Path | Status |
|---|---|---|
| **Native installer** | `~/.local/share/claude/versions/<semver>` | Auto-detected (newest version) |
| **npm -g** | `<npm-root>/node_modules/@anthropic-ai/claude-code/` | Auto-detected |
| **bun add -g** | `~/.bun/install/global/node_modules/` | Auto-detected |
| **pnpm add -g** | `pnpm root -g` / fallback path | Auto-detected |
| **Volta** | `~/.volta/tools/image/packages/` | Auto-detected |
| **fnm** | `$FNM_DIR/node-versions/*/installation/lib/node_modules/` | Auto-detected |
| **nvm** | `~/.nvm/versions/node/*/lib/node_modules/` | Auto-detected |
| **mise** | `~/.local/share/mise/installs/node/*/lib/node_modules/` | Auto-detected |
| **Homebrew** | `/opt/homebrew/lib/node_modules/` | Auto-detected |
| **Scoop** (Win) | `~/scoop/apps/claude-code/current/` | Auto-detected |
| **Chocolatey** (Win) | `C:\ProgramData\chocolatey\lib\claude-code\` | Auto-detected |
| **Winget** (Win) | `%LOCALAPPDATA%\Microsoft\WinGet\Packages\*claude*\` | Auto-detected |
| **shutil.which** fallback | `$PATH` | Last resort, cross-platform |

> **Windows compatibility notes:** All `chmod()` calls are guarded for Windows. Atomic binary swap uses `os.replace()` (POSIX + Windows). MCP guard uses portable `p.kill()` (no SIGKILL dependency).

---

## 🔓 Authorization Rules

`vpcc install-rules` deploys a full operator authorization bundle:

| File | What it does |
|---|---|
| `~/.claude/settings.json` | `bypassPermissions` mode, all tools allowed, empty deny/ask, telemetry off |
| `~/.claude/CLAUDE.md` | Operator authorization doctrine — no permission prompts, no refusals |
| `~/.claude/AGENTS.md` | Same doctrine for Codex / Aider / Gemini / OpenCode compatibility |
| `~/.claude/hooks/vpcc-auto-allow.sh` | *(omitted with `--no-hook`)* PreToolUse hook emits `allow` for every call |

### `--no-hook` — zero CC overhead

```bash
vpcc install-rules --no-hook    # skip PreToolUse shell hook
```

The binary patch `05-auto-allow-hook` already emits the allow decision at C-level inside the Claude Code binary. The `.sh` hook is redundant when the binary is patched. **Removing it saves a `bash` subprocess spawn on every tool call** (a 5-15ms hit that adds up over thousands of calls in a long session).

vpcc auto-detects this: when patch 05 is applied, `install-rules` skips the hook by default unless explicitly requested.

### Scope limits (the only 4 hard stops)

1. No secret exfiltration to networks not invoked by the current task
2. No overwriting uncommitted work without green tests or confirmation
3. No sending messages / creating public PRs without acknowledgement
4. Force-push to main/master requires explicit consent

**Everything else is pre-authorized.** No disclaimers. No "are you sure?". No AUP gates.

---

## 🎛 Commands

| Command | Purpose |
|---|---|
| **`vpcc patch`** | Apply all patches (automatic timestamped backup) |
| **`vpcc upgrade`** ⭐ | All-in-one: self-update + autoheal + verify + warm cache |
| **`vpcc verify`** | Confirm every applied marker is present |
| **`vpcc scan`** | Signature scan with confidence scores + drift detection (cached) |
| **`vpcc scan --auto-heal`** | Regenerate drifted regexes from anchor windows |
| **`vpcc bench`** | Microbenchmark: sha256, text load, scan cold / cached |
| **`vpcc doctor`** | Full health report: target, format, sha, drift, upstream, backups |
| **`vpcc autoheal`** | Self-update + re-derive on binary or patch-mtime drift |
| **`vpcc watch`** | Daemon: poll binary, auto-backup + autoheal on change |
| **`vpcc self-update`** | Pull latest `patches/*.json` from GitHub |
| **`vpcc check-updates`** | Show local vs remote patch commit |
| **`vpcc status`** | Target path, format, sha, backup count |
| **`vpcc list`** | List every patch in the catalog |
| **`vpcc rollback`** | Restore from most recent backup |
| **`vpcc install-rules [--no-hook]`** | Deploy authorization bundle (`--no-hook` for zero CC overhead) |
| **`vpcc uninstall-rules`** | Remove vpcc-authored rules (operator content preserved) |
| **`vpcc install-preload`** | Install MCP guard preload script |
| **`VPCC_NO_CACHE=1 vpcc scan`** | Force fresh scan, bypass cache |

---

## 🔬 How It Works

Claude Code ships as a **Bun SEA** (Single Executable Application) — a Mach-O/ELF/PE binary with embedded bytecode. vpcc:

1. Locates the `.bun` / `__bun` section in the binary (Mach-O, ELF, or PE)
2. Finds the active bundle boundaries within the section
3. Applies regex-based search/replace **in-place** on the bytecode — string literals are stored verbatim even in compiled bytecode
4. Replacements are same-length (pad with spaces) so no offset fixups needed
5. Creates timestamped backups before every patch run

The scanner independently verifies each patch via multi-strategy detection (markers → regex → anchors → fuzzy → keyword) with confidence scoring, so you always know exactly what's applied.

---

## 🤝 Contributing

PRs welcome. Each patch is a standalone JSON file in `patches/`:

```jsonc
{
  "id": "js-example-patch",
  "description": "What this patch does",
  "type": "js_replace",
  "validate_js": true,
  "scan_signatures": true,
  "patches": [
    {
      "search_regex": "regex matching current binary",
      "replace": "replacement (same length, padded)",
      "applied_marker": "unique string after patching",
      "description": "Sub-patch description"
    }
  ],
  "anchor_strings": ["stable strings near patch site"]
}
```

Run `vpcc scan -v` to verify your patch detects correctly before submitting.

---

## 📄 License

[GPL-3.0-or-later](LICENSE) — free software, source open, copyleft.

---

<div align="center">

<sub>built for operators who own their hardware · <a href="https://github.com/VoidChecksum/void-patcher-cc/issues">report drift / new CC versions</a></sub>

<br />

<sub>**no permission gate survives contact with `vpcc patch`**</sub>

</div>
