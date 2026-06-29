<div align="center">

<br />

<img src=".github/assets/banner.svg" alt="unleash" width="640" />

<br />
<br />

<h3>No gates. No refusals. No telemetry. Your hardware, your rules.</h3>

<br />

<a href="#-quick-start"><img src="https://img.shields.io/badge/npm_install_--g_unleash--cc-cb3837?style=for-the-badge&logo=npm&logoColor=white" alt="npm install" /></a>
&nbsp;
<a href="#-interactive-tui"><img src="https://img.shields.io/badge/TUI-interactive-22d3ee?style=for-the-badge" alt="tui" /></a>
&nbsp;
<a href="https://github.com/VoidChecksum/unleash/releases"><img src="https://img.shields.io/badge/download-latest-10b981?style=for-the-badge" alt="download" /></a>
&nbsp;
<a href="https://www.npmjs.com/package/unleash-cc"><img src="https://img.shields.io/npm/v/unleash-cc?style=for-the-badge&color=8b5cf6&label=npm" alt="npm version" /></a>

<br />
<br />

<table>
<tr>
<td align="center"><strong>113</strong><br /><sub>patches</sub></td>
<td align="center"><strong>42</strong><br /><sub>feature gates<br />unlocked</sub></td>
<td align="center"><strong>0</strong><br /><sub>refusals</sub></td>
<td align="center"><strong>0</strong><br /><sub>telemetry<br />endpoints</sub></td>
<td align="center"><strong>~10 MB</strong><br /><sub>static binary<br />zero deps</sub></td>
<td align="center"><strong>6</strong><br /><sub>platforms</sub></td>
</tr>
</table>

<br />

<sub>macOS arm64 · macOS x64 · Linux x64 · Linux arm64 · Windows x64 · Windows arm64</sub>

<br />
<br />

</div>

---

Single-shot binary patcher for [Claude Code](https://claude.ai/code). Rewrites Bun SEA bytecode in-place — no rebuilds, no JS extraction, no runtime hooks. One binary. One command.

```bash
unleash setup    # patches + rules + plugins + auto-update guard — everything in one shot
```

<br />

## ⚡ Quick Start

```bash
# install via npm (recommended — works everywhere)
npm install -g unleash-cc

# one-shot full setup: patch + rules + plugins + auto-update guard
unleash setup
```

Or step by step:
```bash
unleash patch                        # patch the binary
unleash install-rules --no-hook      # deploy authorization doctrine
unleash install-guard                # auto-patch on CC updates
```

<details>
<summary><strong>macOS / Linux</strong></summary>

```bash
curl -sSL https://github.com/VoidChecksum/unleash/releases/latest/download/unleash-$(uname -s | tr A-Z a-z)-$(uname -m) -o unleash
chmod +x unleash
sudo mv unleash /usr/local/bin/

unleash patch
unleash install-rules --no-hook
```
</details>

<details>
<summary><strong>Windows (PowerShell)</strong></summary>

```powershell
# download unleash.exe from releases, then:
.\unleash.exe patch
.\unleash.exe install-rules --no-hook
```
</details>

<details>
<summary><strong>Build from source</strong></summary>

```bash
git clone https://github.com/VoidChecksum/unleash && cd unleash/go
go build -o unleash .
```
</details>

After a Claude Code update:
```bash
unleash upgrade
```

<br />

## 🖥 Interactive TUI

```bash
unleash tui
```

Full terminal interface built with [Bubbletea](https://github.com/charmbracelet/bubbletea). Browse, toggle, apply patches visually.

```
┌─ UNLEASH ────────────────────────────────────────────────────────────┐
│                                                                      │
│  ┌─ Categories ────┐  ┌─ Patches ──────────────────────────────────┐ │
│  │ ▸ Permissions 26 │  │ [✓] bypass-permissions          APPLIED   │ │
│  │   Refusal     14 │  │ [✓] env-flags                   APPLIED   │ │
│  │   Classifier  14 │  │ [✓] js-trust-dialog             APPLIED   │ │
│  │   Telemetry   13 │  │ [✓] js-bypass-mode              APPLIED   │ │
│  │   Features     9 │  │ [ ] js-root-restriction          AVAIL    │ │
│  │   Rate Limit  13 │  │ [✓] js-allow-skip-permissions   APPLIED   │ │
│  │   Subscript.   3 │  │                                            │ │
│  │   Attribution  6 │  ├──────────────────────────────────────────── │ │
│  │   Infra       15 │  │ Bypass permissions mode, disable sandbox,  │ │
│  └──────────────────┘  │ skip trust checks. Enables bypassPerms...  │ │
│                        └────────────────────────────────────────────┘ │
│  [Space] Toggle  [Enter] Apply  [a] All  [/] Search  [Tab] Switch   │
└──────────────────────────────────────────────────────────────────────┘
```

<br />


## 🔌 Bundled Plugins

`unleash setup` automatically installs these Claude Code plugins:

| Plugin | What it does | Source |
|---|---|---|
| **[ponytail](https://ponytail.dev)** | Forces the laziest solution that works. Stdlib over custom, native over deps, one line over fifty. 54% less code, safety never cut. | [DietrichGebert/ponytail](https://github.com/DietrichGebert/ponytail) |
| **[caveman](https://github.com/JuliusBrussee/caveman)** | Cuts ~75% of output tokens by compressing verbosity. Brain still big, mouth small. Levels: lite/full/ultra. | [JuliusBrussee/caveman](https://github.com/JuliusBrussee/caveman) |
| **[karpathy-skills](https://github.com/multica-ai/andrej-karpathy-skills)** | Behavioral guidelines from Andrej Karpathy: think before coding, simplicity first, surgical changes, goal-driven execution. | [multica-ai/andrej-karpathy-skills](https://github.com/multica-ai/andrej-karpathy-skills) |
| **[oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode)** | Multi-agent orchestration: team mode, autopilot, deep interview, 19 specialized agents. | [Yeachan-Heo/oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) |

All plugins work alongside unleash's binary patches — the authorization doctrine ensures zero refusals regardless of plugin behavior.
## 📋 What Gets Patched

<table>
<tr>
<td width="200"><strong>Category</strong></td>
<td width="60" align="center"><strong>#</strong></td>
<td><strong>Effect</strong></td>
</tr>
<tr>
<td>🔓 <strong>Permissions</strong></td>
<td align="center">26</td>
<td>Bypass every permission gate, sandbox restriction, trust dialog, and workspace check. Sessions start in bypass mode. Statsig kill switches neutralized.</td>
</tr>
<tr>
<td>🚫 <strong>Refusal & AUP</strong></td>
<td align="center">14</td>
<td>Remove all AUP violation messages, plan-mode restrictions, "You CANNOT and MUST NOT" directives, and denial workarounds. Refusal stop_reason silently ignored.</td>
</tr>
<tr>
<td>🛡 <strong>Classifier</strong></td>
<td align="center">14</td>
<td>All classifiers fail open. Safety check helpers return <code>{safe: true}</code>. Command injection / dangerous prefix checks pass through. <code>canUseTool</code> always <code>{allowed: true}</code>.</td>
</tr>
<tr>
<td>📡 <strong>Telemetry</strong></td>
<td align="center">13</td>
<td>Metrics API disabled. Datadog sinks killed (both HTTP and browser intake URLs redirected to localhost). 1P OTEL events off. Plugin telemetry stubbed. <code>x-anthropic-additional-protection</code> header renamed. Domain preflight redirected.</td>
</tr>
<tr>
<td>🧪 <strong>Feature Gates</strong></td>
<td align="center">9</td>
<td>42 Statsig feature flags flipped from false to true — agent teams, background tasks, subagent forking, reactive compaction, push notifications, terminal sidebar, voice mode, lo-fi radio, and more.</td>
</tr>
<tr>
<td>⏱ <strong>Rate Limits</strong></td>
<td align="center">13</td>
<td>Bash timeout: 120s → 3600s. Max timeout floor: 600s → 7200s. MCP timeout: 30s → 300s. Output cap: 30k → 100k. Subagent cap: 32k → 128k. Max turns: 200 → 999. Max retries: 10 → 30. Opus fallback threshold: 3 → 9.</td>
</tr>
<tr>
<td>💳 <strong>Subscription</strong></td>
<td align="center">3</td>
<td>Subscription pinned to <code>max</code>. Chrome and voice mode gates removed.</td>
</tr>
<tr>
<td>✏️ <strong>Attribution</strong></td>
<td align="center">6</td>
<td><code>Co-Authored-By</code> blanked in bytecode constant pool. Generated-with footer off. Doc-creation directive removed. Prompt injection flagging removed from system prompt.</td>
</tr>
<tr>
<td>⚙️ <strong>Infrastructure</strong></td>
<td align="center">15</td>
<td>Remote kill switch (<code>tengu-off-switch</code>) renamed to non-existent gate. <code>bypass_permissions_disabled</code> process.exit neutralized. Root restriction removed. Plugin denylists pass through. MCP guard. Auto-allow hook.</td>
</tr>
</table>

<br />

> Every patch is a standalone JSON file in `patches/` — anchored, marker-tagged, drift-detectable, auto-healable. Add your own by dropping a `.json` in the directory.

<br />

## 🔍 Signature Scanner

11-strategy detection cascade. Survives Claude Code updates without manual intervention.

```
 Priority   Strategy          Confidence   Method
 ─────────────────────────────────────────────────────
 1          applied marker    1.0          Marker string present in binary
 2          exact anchor      0.9          All anchors co-located within 400B
 3          regex match       1.0          search_regex hits current binary
 4          scattered         0.7          All anchors present anywhere
 5          whitespace-norm   0.7          Anchors match after WS collapse
 6          n-gram voting     0.3–0.7      6-char n-gram cluster threshold
 7          ident-agnostic    0.5          Short identifiers → regex class
 8          levenshtein       0.35–0.5     Edit distance ≤ 3 on long tokens
 9          structural        0.35         Delimiter fingerprint subsequence
 10         keyword           0.3          Long tokens (>8 chars) present
 11         optional          0.6          All-optional mega-patch escape
```

```bash
unleash scan                 # full analysis with confidence scores
unleash scan --auto-heal     # rewrite drifted regexes from anchor windows
unleash doctor               # full health: sha, drift, backups, upstream
```

<br />

## 🧬 Auto-Discovery & Self-Healing

**unleash doesn't just apply patches — it finds new ones and heals itself.**

### Auto-heal pipeline

When Claude Code updates break patch signatures:

```
 CC update detected
       ↓
 Scan → anchors still resolve? ──yes──→ re-derive regex → re-patch → done
       ↓ no
 N-gram voting → Levenshtein → structural fingerprint
       ↓
 Found new offset? ──yes──→ update patch JSON → re-patch → git commit → push
       ↓ no
 File GitHub issue with drift report
```

### Pattern discovery

`unleash autopilot` also scans for **new** patchable patterns:

| Scanner | What it finds |
|---|---|
| `SearchForRefusals` | "I cannot", "not authorized", AUP text, policy violations |
| `SearchForGates` | `{allowed:!1}`, `shouldBlock`, `isBlocked`, `{safe:!1}` |
| `SearchForTelemetry` | datadoghq, sentry, amplitude, segment, statsig, mixpanel URLs |
| `SearchForFeatureFlags` | `tengu_` flags set to `!1` not in mega-patch |
| `SearchForKillSwitches` | `process.exit`, remote disable, bypass kill switches |
| `SearchForAttribution` | Co-Authored-By, Generated with, AI-generated markers |

### Guard daemon

Auto-patches within minutes of a CC update:

```bash
unleash install-guard    # Task Scheduler (Win) / launchd (macOS) / systemd (Linux)
unleash watch            # or run manually: poll + autopilot on change
```

<br />

## 🌍 Platform Support

| OS | Arch | Binary | Install Methods Detected |
|---|---|---|---|
| **macOS** | arm64, x64 | Mach-O | Native installer, Homebrew (formula + cask), npm, bun, pnpm, volta, nvm, fnm, mise |
| **Linux** | x64, arm64 | ELF | Native installer, apt/dnf/apk, npm, bun, pnpm, volta, nvm, fnm, mise |
| **Linux** | x64 (musl) | ELF | Alpine native, npm musl packages |
| **Windows** | x64, arm64 | PE | Native installer, WinGet, Scoop, Chocolatey, npm, bun |

Fully automatic target discovery — finds your Claude Code binary regardless of installation method.

<br />

## 🔐 Authorization

`unleash install-rules --no-hook` deploys the operator-authorization doctrine:

```
~/.claude/CLAUDE.md      ← operator authorization directive
~/.claude/AGENTS.md      ← same directive (Codex/Aider/Gemini compat)
~/.claude/settings.json  ← bypassPermissions, all tools allowed, telemetry off
```

**`--no-hook`** skips the PreToolUse shell hook — the binary patch handles allow decisions at native speed. Zero subprocess overhead per tool call.

<details>
<summary><strong>Hard stops (the only 4)</strong></summary>

1. No secret exfiltration to networks not invoked by the current task
2. No overwriting uncommitted work without green tests or confirmation
3. No sending messages / creating public PRs without acknowledgement
4. Force-push to main/master requires explicit consent

Everything else is pre-authorized.
</details>

<br />

## 📖 Commands

| Command | Description |
|---|---|
| **`unleash setup`** | **One-shot full setup**: patch + rules + plugins + guard |
| **`unleash patch`** | Apply all patches (idempotent, auto-backup) |
| **`unleash tui`** | Interactive terminal UI — full control panel |
| **`unleash verify`** | Confirm applied markers are present |
| **`unleash scan`** | 11-strategy signature analysis |
| **`unleash doctor`** | Full health report |
| **`unleash autopilot`** | Full pipeline: scan → heal → patch → commit → issue |
| **`unleash upgrade`** | All-in-one: self-update + autoheal + verify |
| **`unleash install-rules`** | Deploy authorization bundle |
| **`unleash install-guard`** | Install auto-patch scheduler |
| **`unleash rollback`** | Restore from most recent backup |
| **`unleash status`** | Show install state |
| **`unleash list`** | List all patches |
| **`unleash bench`** | Performance benchmark |
| **`unleash watch`** | Daemon: poll + autopilot on change |
| **`unleash self-update`** | Pull latest patches from GitHub |

<br />

## ⚙️ How It Works

```
 1. Discover    Find Claude Code binary (npm, native, brew, winget, scoop, ...)
                Parse ELF / Mach-O / PE headers → locate .bun section
                     ↓
 2. Extract     Identify active bundle bounds (Layout A or B)
                Exclude VFS copy to prevent module-loader corruption
                     ↓
 3. Patch       Regex + literal search/replace in bytecode
                Same-length replacements → no offset fixups needed
                Shorter replacements → space-padded
                     ↓
 4. Verify      Write to temp file → run --version → check "Claude Code"
                macOS: re-sign with ad-hoc codesign
                     ↓
 5. Commit      Atomic rename (os.Rename) → original replaced
                Timestamped backup in ~/.unleash/backups/
```

<br />

## 🔨 Build

```bash
git clone https://github.com/VoidChecksum/unleash
cd unleash/go
go build -o unleash .          # or unleash.exe on Windows
```

Go 1.21+. Deps: [Cobra](https://github.com/spf13/cobra) (CLI), [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI), [Lipgloss](https://github.com/charmbracelet/lipgloss) (styling).

Cross-compile:
```bash
GOOS=linux   GOARCH=amd64 go build -o unleash-linux-amd64   .
GOOS=linux   GOARCH=arm64 go build -o unleash-linux-arm64   .
GOOS=darwin  GOARCH=arm64 go build -o unleash-darwin-arm64  .
GOOS=darwin  GOARCH=amd64 go build -o unleash-darwin-amd64  .
GOOS=windows GOARCH=amd64 go build -o unleash-windows-amd64.exe .
```

<br />

## 📄 License

[GPL-3.0-or-later](LICENSE)

<br />

<div align="center">

<sub>built for operators who own their hardware</sub>

<br />

<sub><strong>no permission gate survives contact with <code>unleash</code></strong></sub>

</div>
