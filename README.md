<div align="center">

```
  ██╗   ██╗███╗   ██╗██╗     ███████╗ █████╗ ███████╗██╗  ██╗
  ██║   ██║████╗  ██║██║     ██╔════╝██╔══██╗██╔════╝██║  ██║
  ██║   ██║██╔██╗ ██║██║     █████╗  ███████║███████╗███████║
  ██║   ██║██║╚██╗██║██║     ██╔══╝  ██╔══██║╚════██║██╔══██║
  ╚██████╔╝██║ ╚████║███████╗███████╗██║  ██║███████║██║  ██║
   ╚═════╝ ╚═╝  ╚═══╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
```

### **Unleash Claude Code.**

**113 patches (106 active, 7 retired) · zero refusals · zero telemetry · 42 feature gates unlocked**

<br />

[![patches](https://img.shields.io/badge/patches-113-8b5cf6?style=for-the-badge&logo=hackthebox&logoColor=white)](#patch-catalog)
[![feature gates](https://img.shields.io/badge/feature%20gates-42%20unlocked-22d3ee?style=for-the-badge&logo=flipboard&logoColor=white)](#feature-gate-unlock)
[![zero refusals](https://img.shields.io/badge/AUP%20blocks-0-ef4444?style=for-the-badge&logo=adblock&logoColor=white)](#authorization-rules)
[![telemetry](https://img.shields.io/badge/telemetry-disabled-10b981?style=for-the-badge&logo=ghostery&logoColor=white)](#what-it-does)
[![platforms](https://img.shields.io/badge/macOS%20·%20Linux%20·%20Windows-supported-f59e0b?style=for-the-badge&logo=apple&logoColor=white)](#cross-platform-support)

<br />

**`unleash patch && unleash install-rules --no-hook`**

<br />

[Quick Start](#-quick-start) · [Interactive TUI](#-interactive-tui) · [Patch Catalog](#-patch-catalog) · [Scanner](#-scanner) · [Auto-Discovery](#-auto-discovery) · [Platforms](#-cross-platform-support) · [Authorization](#-authorization-rules) · [Commands](#-commands)

</div>

---

> **unleash** is a single static binary patcher for Anthropic's [Claude Code](https://docs.claude.com/en/docs/claude-code) CLI. It rewrites the Bun SEA bytecode in-place to remove every permission gate, refusal classifier, AUP block, telemetry sink, and feature lock — no rebuilds, no JavaScript extraction, no runtime injection. Single binary, ~10 MB, zero dependencies.

## 🚀 Quick Start

```bash
# Download the latest release for your platform
curl -sSL https://github.com/VoidChecksum/void-patcher-cc/releases/latest/download/unleash-$(uname -s)-$(uname -m) -o unleash
chmod +x unleash

# Patch — single command, idempotent, auto-backup
./unleash patch

# Verify everything landed
./unleash verify

# Deploy operator-authorization bundle (no permission prompts, no telemetry)
./unleash install-rules --no-hook
```

<details>
<summary><b>One-shot full setup</b></summary>

```bash
unleash patch && unleash install-rules --no-hook
```

</details>

<details>
<summary><b>After every Claude Code auto-update</b></summary>

```bash
unleash upgrade        # self-update + autoheal + verify + warm cache, one shot
```

</details>

<details>
<summary><b>Windows (PowerShell)</b></summary>

```powershell
# Download unleash.exe from releases
unleash patch
unleash install-rules --no-hook
```

</details>

---

## 🖥️ Interactive TUI

Full curses-style terminal interface — patch, scan, verify, monitor, all from one screen.

```bash
unleash tui
```

- **Dashboard** — real-time view of CC binary, patch status, drift, upstream state
- **Patch catalog** — browse all 113 patches grouped by category, toggle details
- **Scanner** — 8-strategy signature cascade with confidence scores
- **One-key actions** — patch, verify, rollback, scan, autoheal from any panel
- **Responsive layout** — adapts to terminal size, mouse + keyboard navigation

---

## 📚 Patch Catalog

113 patches across 9 categories. 106 active, 7 retired.

| Category | Count | Effect |
|---|---:|---|
| **Permissions** | 22 | Bypass permission gates, sandbox, trust checks |
| **Refusal / AUP** | 20 | Remove AUP blocks, plan-mode restrictions, attribution |
| **Classifier / Safety** | 14 | Fail-open classifiers, neutralize safety gates |
| **Telemetry / Analytics** | 13 | Disable metrics, Datadog, OTEL, plugin telemetry |
| **Feature Gates** | 42 | Unlock all Statsig feature flags (tengu_*) |
| **Rate Limits / Timeouts** | 12 | Raise bash/MCP/output/retry limits |
| **Subscription / UI** | 8 | Unlock subscription-gated UI and features |
| **Attribution** | 3 | Strip Co-Authored-By, generated-with footers |
| **Infrastructure** | 3 | Redirect telemetry endpoints, neutralize kill switches |

Every patch is a standalone JSON file — anchored, marker-tagged, drift-detectable, auto-healable.

---

## 🔍 Scanner

8-strategy signature detection cascade. Survives Claude Code updates without manual intervention.

```
Strategy          Speed      Purpose
─────────────────────────────────────────────────────────
marker            <1 ms      Check applied_marker strings
anchor            ~3 ms      Stable anchor_strings proximity
regex             ~200 ms    search_regex pattern matching
scattered         ~50 ms     Multi-fragment spread search
fuzzy_ws          ~100 ms    Whitespace-insensitive match
fuzzy_ident       ~100 ms    Identifier-insensitive match
keyword           ~10 ms     Keyword presence heuristic
optional          0 ms       Skip if all sub-patches optional
```

```bash
unleash scan              # full signature analysis
unleash scan --verbose    # per-strategy confidence breakdown
unleash scan --auto-heal  # rewrite drifted regexes from anchors
```

Cache keyed by `sha256(binary) + max(patch_mtime)` — cold scan ~5.7s, warm ~137ms (41× speedup).

---

## 🔬 Auto-Discovery

`unleash autopilot` is a fully autonomous drift-response pipeline:

1. **Scan** — detect drifted signatures against the current binary
2. **Auto-heal** — regenerate broken regexes from anchor string windows
3. **Re-patch** — apply updated patches to the binary
4. **Git commit + push** — commit healed patches upstream
5. **GitHub issue** — file a drift report for unfixable patches

Run it on a schedule with `unleash install-guard` — uses Task Scheduler (Windows), launchd (macOS), or systemd timer (Linux). Claude Code updates are re-patched within minutes.

---

## 🌍 Cross-Platform Support

| Platform | Architecture | Binary Format |
|---|---|---|
| macOS | arm64, x64 | Mach-O Bun SEA |
| Linux | x64, arm64 | ELF Bun SEA |
| Linux (musl) | x64 | ELF Bun SEA |
| Windows | x64, arm64 | PE Bun SEA |

Single static binary (~10 MB). No runtime dependencies. Detects Claude Code location automatically — npm global, native installer, Volta, fnm, nvm, pnpm, scoop, winget.

---

## ⚖️ Authorization Rules

`unleash install-rules` deploys an operator-authorization bundle:

| File | Destination | Purpose |
|---|---|---|
| `AUTHORIZATION.md` | `~/.claude/CLAUDE.md` | Operator authorization directive |
| `settings-rules.json` | `~/.claude/settings.json` | Permissions, env, telemetry off |
| `vpcc-auto-allow.sh` | `~/.claude/hooks/` | PreToolUse → allow all |

With `--no-hook`, the binary patch handles allow decisions at native speed — zero subprocess overhead per tool call.

**Scope**: operator-authorized security research, red-team engagements, agent autonomy work, removing UX friction on your own hardware. See the [AUTHORIZATION.md](embed/contrib/rules/AUTHORIZATION.md) doctrine.

---

## 📖 Commands

| Command | Description |
|---|---|
| `unleash patch` | Apply all patches (idempotent, auto-backup) |
| `unleash verify` | Check applied markers |
| `unleash scan` | 8-strategy signature analysis |
| `unleash doctor` | Full health report |
| `unleash status` | Show install state |
| `unleash rollback` | Restore from most recent backup |
| `unleash tui` | Interactive terminal UI |
| `unleash dashboard` | Real-time status display |
| `unleash autopilot` | Full pipeline: scan → heal → patch → commit → issue |
| `unleash guard` | Fast SHA guard — auto-patch if binary changed |
| `unleash install-guard` | Install platform scheduler (Task Scheduler / launchd / systemd) |
| `unleash uninstall-guard` | Remove scheduler |
| `unleash install-rules` | Deploy authorization bundle |
| `unleash uninstall-rules` | Remove authorization bundle |
| `unleash self-update` | Pull latest patches from GitHub |
| `unleash autoheal` | Detect CC drift, self-update + re-patch |
| `unleash upgrade` | All-in-one: self-update + autoheal + verify + warm cache |
| `unleash update` | Full self-update + sync + re-patch |
| `unleash check-updates` | Show if remote patches differ |
| `unleash bench` | Microbenchmark: sha256, text load, scan cold/cached |
| `unleash list` | List patches in catalog |
| `unleash install-preload` | Deploy runtime preload hook |
| `unleash uninstall-preload` | Remove preload hook |

---

## ⚙️ How It Works

1. **Discovery** — locates the Claude Code binary across all packaging formats (npm, native installer, Volta, fnm, nvm, pnpm, scoop, winget)
2. **Section extraction** — parses ELF/Mach-O/PE headers to find the `.bun` section containing the Bun SEA JavaScript bundle
3. **In-place patching** — regex search-and-replace directly in the binary's bytecode. Equal-length replacements preserve offsets; padding handles size differences
4. **Verification** — runs the patched binary and checks applied markers to confirm patches landed
5. **Backup** — auto-creates timestamped backups in `~/.unleash/backups/` before every patch operation

No JavaScript extraction, no rebuilds, no runtime injection. Patches survive across sessions. Auto-heals when Claude Code updates.

---

## 🔨 Building from Source

```bash
cd go/
go build -o unleash .
```

Requires Go 1.21+. Dependencies managed via `go.mod` — Cobra (CLI), Bubbletea (TUI), Lipgloss (styling), Bubbles (components).

---

## 📄 License

MIT
