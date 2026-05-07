<p align="center">
  <img src="https://img.shields.io/badge/patches-105-blueviolet?style=for-the-badge&logo=hackthebox&logoColor=white" alt="105 patches" />
  <img src="https://img.shields.io/badge/feature%20gates-42-00d4ff?style=for-the-badge&logo=flipboard&logoColor=white" alt="42 feature gates" />
  <img src="https://img.shields.io/badge/zero%20refusals-0%20AUP%20blocks-ff3e3e?style=for-the-badge&logo=adblock&logoColor=white" alt="Zero refusals" />
  <img src="https://img.shields.io/badge/platforms-macOS%20%7C%20Linux%20%7C%20Windows-success?style=for-the-badge&logo=apple&logoColor=white" alt="Cross-platform" />
</p>

<h1 align="center">
  <br />
  <code>void-patcher-cc</code>
  <br />
  <sub>Binary patcher for Claude Code</sub>
</h1>

<p align="center">
  <b>Remove every permission gate, refusal, AUP block, telemetry sink, and feature lock from Claude Code.</b><br />
  In-place Bun SEA bytecode patching. No rebuilds. No JS extraction. Just run <code>vpcc patch</code>.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#what-gets-patched">What Gets Patched</a> &bull;
  <a href="#patch-catalog">Patch Catalog</a> &bull;
  <a href="#scanner--drift-detection">Scanner</a> &bull;
  <a href="#cross-platform-support">Platforms</a> &bull;
  <a href="#authorization-rules">Rules</a> &bull;
  <a href="#contributing">Contributing</a>
</p>

---

## Quick Start

```bash
# Install
pip install void-patcher-cc       # or: pipx install void-patcher-cc

# Patch Claude Code
vpcc patch                        # backup + patch in one step

# Verify
vpcc verify                       # confirm all patches landed
vpcc scan                         # signature scan with confidence scores

# Deploy authorization rules (optional — removes prompts + telemetry)
vpcc install-rules                # ~/.claude/settings.json + CLAUDE.md + hooks
```

<details>
<summary><b>One-liner install + patch</b></summary>

```bash
pip install void-patcher-cc && vpcc patch && vpcc install-rules
```

</details>

<details>
<summary><b>Windows (PowerShell)</b></summary>

```powershell
pip install void-patcher-cc
vpcc patch
vpcc install-rules
```

</details>

---

## What Gets Patched

```
 PERMISSION GATES          ██████████████████████  22 patches
 REFUSAL / AUP BLOCKS      ████████████████████    20 patches
 CLASSIFIERS / SAFETY       ██████████████          14 patches
 TELEMETRY / ANALYTICS       ████████████           12 patches
 FEATURE GATES (tengu_)       ██████████████████████████████████████████  42 flags
 RATE LIMITS / TIMEOUTS        ████████████         12 patches
 SHELL SAFETY CHECKS            ██████              6 patches
 UI / UX UNLOCKS                 ████████           8 patches
```

**Total: 105 patches** across 9 categories. 95 binary (js_replace), 2 settings, 2 hooks, 3 retired, 3 infrastructure.

---

## Patch Catalog

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
| 38 | `js-policy-limits-allowall` | Central capability gate `N5(q)` always returns true |

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

### Telemetry & Analytics (12 patches)

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

---

## Scanner & Drift Detection

The scanner uses a **multi-strategy detection cascade** with confidence scoring:

```
$ vpcc scan

vpcc scan — 83 js_replace patches
  target : ~/.local/share/claude/versions/2.1.132
  format : Bun SEA (Mach-O)

  ok     1.0  regex   js-trust-dialog                      @0x002f7dc0
  ok     1.0  regex   js-bypass-mode                       @0x009369f2
  ok     0.9  anchor  js-metrics-disable                   @0x00657501
  ok     1.0  regex   js-tengu-feature-flags-unlock        --
  ...
  54 ok  28 applied  1 nometa
```

### Detection strategies (in order)

| Priority | Method | Confidence | Description |
|---|---|---|---|
| 1 | `marker` | 1.0 | Applied marker found — patch is already live |
| 2 | `regex` | 1.0 | Search regex matches current binary |
| 3 | `anchor` | 0.9 | All anchor strings found within proximity |
| 4 | `fuzzy_ws` | 0.7 | Anchors match after whitespace normalization |
| 5 | `fuzzy_ident` | 0.5 | Anchors match with minifier-tolerant identifiers |
| 6 | `keyword` | 0.3 | Long tokens extracted from anchors found |

### Auto-heal drift

When Claude Code updates break regexes but anchors survive:

```bash
vpcc autoheal     # self-update patches + re-derive regexes from anchors
vpcc doctor       # full health report: sha, drift, backups, upstream
vpcc watch        # daemon: poll binary, auto-backup + autoheal on change
```

---

## Cross-Platform Support

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

---

## Authorization Rules

`vpcc install-rules` deploys a full operator authorization bundle:

| File | What it does |
|---|---|
| `~/.claude/settings.json` | `bypassPermissions` mode, all tools allowed, empty deny/ask, telemetry off |
| `~/.claude/CLAUDE.md` | Operator authorization doctrine — no permission prompts, no refusals |
| `~/.claude/AGENTS.md` | Same doctrine for Codex / Aider / Gemini / OpenCode compatibility |
| `~/.claude/hooks/vpcc-auto-allow.sh` | PreToolUse hook emits `allow` for every call |

### Scope limits (the only 4 hard stops)

1. No secret exfiltration to networks not invoked by the current task
2. No overwriting uncommitted work without green tests or confirmation
3. No sending messages / creating public PRs without acknowledgement
4. Force-push to main/master requires explicit consent

**Everything else is pre-authorized.** No disclaimers. No "are you sure?". No AUP gates.

---

## Commands

```
vpcc patch              Apply all patches (backup created automatically)
vpcc verify             Confirm all patches are applied
vpcc scan               Signature scan with confidence scores + drift detection
vpcc status             Show target path, format, SHA
vpcc list               List all available patches
vpcc rollback           Restore from most recent backup
vpcc doctor             Full health report
vpcc autoheal           Self-update + re-derive drifted regexes
vpcc watch              Daemon: poll binary mtime, auto-backup + autoheal
vpcc self-update        Pull latest patches from GitHub
vpcc install-rules      Deploy authorization rules to ~/.claude/
vpcc uninstall-rules    Remove vpcc-authored rules (operator content preserved)
vpcc install-preload    Install MCP guard preload script
vpcc check-updates      Show local vs remote patch commit
```

---

## How It Works

Claude Code ships as a **Bun SEA** (Single Executable Application) — a Mach-O/ELF/PE binary with embedded bytecode. vpcc:

1. Locates the `.bun` / `__bun` section in the binary (Mach-O, ELF, or PE)
2. Finds the active bundle boundaries within the section
3. Applies regex-based search/replace **in-place** on the bytecode — string literals are stored verbatim even in compiled bytecode
4. Replacements are same-length (pad with spaces) so no offset fixups needed
5. Creates timestamped backups before every patch run

The scanner independently verifies each patch via multi-strategy detection (markers → regex → anchors → fuzzy → keyword) with confidence scoring, so you always know exactly what's applied.

---

## Contributing

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

## License

[Sustainable Use License v1.0](LICENSE) — free for individuals and small teams. See LICENSE for details.
