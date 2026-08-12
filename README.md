<div align="center">

<img src=".github/assets/banner.svg" alt="Unleash — operator patch suite" width="920" />

<br />

**Binary patchers + operator setup for local coding agents.**  
Discover every install, patch in place, deploy authorization config, and keep it patched after updates.

<br />

[![unleash-cc](https://img.shields.io/npm/v/unleash-cc?style=for-the-badge&color=8b5cf6&label=unleash-cc)](https://www.npmjs.com/package/unleash-cc)
&nbsp;
[![unleash-gpt](https://img.shields.io/npm/v/unleash-gpt?style=for-the-badge&color=10b981&label=unleash-gpt)](https://www.npmjs.com/package/unleash-gpt)
&nbsp;
[![unleash-omp](https://img.shields.io/npm/v/unleash-omp?style=for-the-badge&color=22d3ee&label=unleash-omp)](https://www.npmjs.com/package/unleash-omp)

<br />

`macOS arm64` · `macOS x64` · `Linux x64` · `Linux arm64` · `Windows x64` · `Windows arm64`

<br />

<img src=".github/assets/products.svg" alt="Unleash products" width="920" />

</div>

---

## What is Unleash?

**Unleash** is a three-product operator suite. Each product finds **every** copy of its target agent on the machine — npm, bun, WinGet, Scoop, Homebrew, native installers, version managers — patches the binary in place, writes operator config, and installs a guard that re-applies patches after updates.

| Product | Target | npm | CLI | State | Config written |
|---|---|---|---|---|---|
| **Unleash** | Claude Code | [`unleash-cc`](https://www.npmjs.com/package/unleash-cc) | `unleash` | `~/.unleash/` | `~/.claude/CLAUDE.md`, `~/.claude/AGENTS.md`, `~/.claude/settings.json` |
| **Unleash-GPT** | OpenAI Codex CLI | [`unleash-gpt`](https://www.npmjs.com/package/unleash-gpt) | `unleash-gpt` | `~/.unleash-gpt/` | `~/.codex/AGENTS.md`, `~/.codex/config.toml` |
| **Unleash-OMP** | Oh-My-Pi / OMP | [`unleash-omp`](https://www.npmjs.com/package/unleash-omp) | `unleash-omp` | `~/.unleash-omp/` | `~/.omp/agent/AGENTS.md`, `~/.omp/agent/config.yml` |

**Currently tested:** Claude Code **2.1.228** · Codex CLI **0.147.0** · OMP **17.2.13 / 17.2.14**

---

## Install

```bash
# npm
npm install -g unleash-cc      # Claude Code
npm install -g unleash-gpt     # Codex CLI
npm install -g unleash-omp     # Oh-My-Pi

# bun
bun add -g unleash-cc

# Windows — all three from GitHub releases
irm https://raw.githubusercontent.com/NetVar1337/unleash/main/scripts/install.ps1 | iex

# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/NetVar1337/unleash/main/scripts/install.sh | bash
```

GitHub Releases ship standalone binaries for six platforms  
(`unleash-windows-amd64.exe`, `unleash-darwin-arm64`, …) plus checksums.  
A winget manifest template lives in [`contrib/winget/`](contrib/winget/).

---

## Quick start

```bash
unleash setup        # Claude Code — patch + rules + plugins + guard
unleash-gpt setup    # Codex CLI  — patch + rules/config
unleash-omp setup    # Oh-My-Pi   — patch + rules/config
```

Same verbs on every product:

```bash
unleash status           # every detected install + SHA + format
unleash patch --dry-run  # preview
unleash patch            # patch ALL detected installs
unleash verify           # confirm applied markers
unleash scan             # signature-drift report
unleash guard            # SHA check; re-patch on change
unleash doctor           # full health report
unleash install-skills   # LLM jailbreak / Fable skill pack
unleash rollback         # restore newest backup
unleash tui              # interactive control panel
```

---

## Discovery surface

Unleash patches the agent no matter how it was installed. Multiple coexisting copies are all patched (hardlinked files are deduped by identity).

<details>
<summary><strong>Claude Code</strong></summary>

| Method | Example | Layout |
|---|---|---|
| Native installer | `irm https://claude.ai/install.ps1 \| iex` / `curl … \| bash` | `~/.local/bin/claude(.exe)` + `~/.local/share/claude/versions/<ver>` |
| npm | `npm i -g @anthropic-ai/claude-code` | npm root + platform subpackage (often hardlinked) |
| bun | `bun add -g @anthropic-ai/claude-code` | `~/.bun/install/global/node_modules/…` |
| WinGet | `winget install Anthropic.ClaudeCode` | `%LOCALAPPDATA%\Microsoft\WinGet\Packages\…` |
| Scoop / Chocolatey | `scoop install claude-code` | scoop/choco lib dirs |
| Homebrew | formula or cask | `/opt/homebrew/…`, Caskroom |
| Version managers | nvm / fnm / mise / volta / pnpm | versioned `node_modules` |
| System packages | apt / dnf / apk | `/usr/bin/claude`, `/opt/claude-code/…` |

</details>

<details>
<summary><strong>Codex CLI</strong></summary>

npm (flat + nested), bun, pnpm, volta, WinGet, native (`~/.local/bin/codex`, `~/.codex/bin`), Scoop, Homebrew, PATH shim fallback.

</details>

<details>
<summary><strong>OMP</strong></summary>

Standalone SEA executables (WindowsApps, `%LOCALAPPDATA%\Programs`, `~/.local/bin`, WinGet), bun global `@oh-my-pi/pi-coding-agent`, npm global, mise, PATH shim fallback.

</details>

---

## Engine notes

Why this survives real installs and updates:

- **Same-length in-place patching** — replacements pad to match length; longer ones are skipped, never shifting bytecode.
- **Hardlink-safe commit** — npm often hardlinks `bin/claude.exe` to the platform binary; Unleash writes verified bytes in place so every link is patched.
- **SEA layout tolerance** — Bun SEA builds where `.bun` raw size exceeds virtual size (CC 2.1.228+) and both pre/post-2.1.150 active-bundle layouts; only the patchable bundle is touched, never the VFS tail.
- **Bytecode constant-pool targeting (CC 2.1.228+)** — the active region is Bun bytecode + length-prefixed string pools, not minified JS source. Gate ids, messages, and pool strings are the primary surface; settings cover runtime defaults that no longer exist as `:!1` flips.
- **Backref-capable matcher** — `\1`–`\9` backreferences (emulated on RE2) for minifier-agnostic patterns.
- **Update guard** — `unleash guard` (Task Scheduler / launchd / systemd, ~6h) compares per-target SHA manifests and re-runs the pipeline after updates.

---

## What each product patches

### Unleash · Claude Code

Bytecode constant-pool + settings patching of the Bun SEA:

| Category | Effect |
|---|---|
| **Permissions** | Bypass gates, sandbox friction, trust checks, remote kill-switch gate ids / disable messages |
| **Refusal & AUP** | Neutralize usage-policy refusal text, plan-mode blocks, denial workarounds, refusal stop handling |
| **Classifier** | Fail safety classifiers open; neutralize dangerous-prefix / injection blocker copy |
| **Telemetry** | Disable metrics/Datadog/OTEL paths, plugin telemetry events, protection headers, domain preflights |
| **Feature gates** | Statsig-facing gate id renames / force paths where a pool surface exists |
| **Rate limits** | Raise timeouts, output caps, subagent caps, retries (where constants remain patchable) |
| **Subscription** | Pin subscription-sensitive surfaces where a safe pool/settings lever exists |
| **Attribution** | Blank co-author trailers and generated-with markers |
| **Infrastructure** | Off-switch keys, root restrictions, plugin denylists, MCP friction, update guards |

Settings companion (`01-bypass-permissions`) forces `defaultMode=bypassPermissions`, trust skip, sandbox disable, and related operator defaults.

### Unleash-GPT · Codex CLI

- Sentry DSN → loopback
- OTLP metrics endpoint → loopback
- Cyber-safety doc endpoint → loopback
- High-risk cyber fallback + approval/policy rejection wording neutralized
- Approval/sandbox fallback warning neutralized
- Writes `~/.codex/AGENTS.md` + `~/.codex/config.toml`:

```toml
approval_policy = "never"
sandbox_mode = "danger-full-access"
dangerously_bypass_approvals_and_sandbox = true
```

### Unleash-OMP · Oh-My-Pi

- Auto-QA grievance endpoint → loopback
- Approval `deny`/`prompt` branches neutralized (17.x resolver shapes)
- Channel permission gate → always-allow tool execution
- Startup update check + marketplace auto-update defaults off (binary **and** `config.yml`)
- OMP npm update-check endpoint → loopback
- Writes `~/.omp/agent/AGENTS.md` + `~/.omp/agent/config.yml`:

```yaml
tools:
  approvalMode: yolo
startup:
  checkUpdate: false
marketplace:
  autoUpdate: "off"
```

---

## Build from source

```bash
git clone https://github.com/NetVar1337/unleash
cd unleash/go

# Claude Code
GOOS=windows GOARCH=amd64 go build -o unleash-windows-amd64.exe .
# Codex CLI
GOOS=windows GOARCH=amd64 go build -o unleash-gpt-windows-amd64.exe ./cmd/unleash-gpt
# Oh-My-Pi
GOOS=windows GOARCH=amd64 go build -o unleash-omp-windows-amd64.exe ./cmd/unleash-omp
```

Requires **Go 1.24.2+**. Release workflows build all six platform targets per product.

---

## Release layout

| Product | Git tag | GitHub artifacts | npm package | npm bin |
|---|---|---|---|---|
| Unleash | `cc-v*` | `unleash-*` | `unleash-cc` | `unleash` |
| Unleash-GPT | `gpt-v*` | `unleash-gpt-*` | `unleash-gpt` | `unleash-gpt` |
| Unleash-OMP | `omp-v*` | `unleash-omp-*` | `unleash-omp` | `unleash-omp` |

<!-- release-versions:start -->
| Product | Latest release |
|---|---|
| Unleash | `cc-v1.0.1` |
| Unleash-GPT | `gpt-v1.0.1` |
| Unleash-OMP | `omp-v1.0.1` |
<!-- release-versions:end -->

npm packages are thin launchers over prebuilt `bin/` binaries (npm **and** bun).  
Publishing is tag-driven: push `cc-vX.Y.Z` → publish `unleash-cc@X.Y.Z`.

---

## Safety model

Operator-authorization block with four hard stops:

1. No secret exfiltration to networks not invoked by the current task.
2. No overwriting uncommitted work without green tests or explicit confirmation.
3. No sending messages or creating public PRs without in-session acknowledgement.
4. Force-push to `main`/`master` requires explicit in-session consent.

Everything else is treated as pre-authorized local operator work.

Every patch run creates timestamped backups first (`rollback` restores the newest).  
Patched binaries are smoke-verified (`--version` + startup check) before commit.

---


## Unleash skills pack

Mega bundle (**203 skills**) under [`contrib/skills/`](contrib/skills/):

- LLM jailbreaks / Fable safeguards (`llm-jailbreak-taxonomy`, `llm-fable-safeguards`, …)
- Coding style: Karpathy, Julius, Ponytail
- Game hacking, aimbot humanization/triggerbot, game internals/exploits
- Windows internals, stealth injectors, stealth hypervisor, kernel/BYOVD
- Languages: C++23, C++ game hacking, Go, Rust, Zig, Java, Assembly
- reverse-skill domain pack + `cua-driver` (trycua/cua)

```bash
unleash install-skills
```

See [`contrib/skills/README.md`](contrib/skills/README.md).


## Brand assets

| File | Use |
|---|---|
| [`.github/assets/banner.svg`](.github/assets/banner.svg) | Hero banner |
| [`.github/assets/logo.svg`](.github/assets/logo.svg) | App mark / favicon source |
| [`.github/assets/wordmark.svg`](.github/assets/wordmark.svg) | Horizontal wordmark |
| [`.github/assets/products.svg`](.github/assets/products.svg) | Three-product card strip |

<div align="center">
<img src=".github/assets/wordmark.svg" alt="Unleash wordmark" width="420" />
</div>

---

## License

**GPL-3.0-or-later** — see [LICENSE](LICENSE).
