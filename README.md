<div align="center">

<br />

<img src=".github/assets/banner.svg" alt="unleash" width="640" />

<br />
<br />

<h3>No gates. No refusals. No telemetry. Your hardware, your rules.</h3>

<br />

<a href="https://www.npmjs.com/package/unleash-cc"><img src="https://img.shields.io/npm/v/unleash-cc?style=for-the-badge&color=8b5cf6&label=unleash-cc" alt="unleash-cc npm version" /></a>
&nbsp;
<a href="https://www.npmjs.com/package/unleash-gpt"><img src="https://img.shields.io/npm/v/unleash-gpt?style=for-the-badge&color=10b981&label=unleash-gpt" alt="unleash-gpt npm version" /></a>
&nbsp;
<a href="https://www.npmjs.com/package/unleash-omp"><img src="https://img.shields.io/npm/v/unleash-omp?style=for-the-badge&color=22d3ee&label=unleash-omp" alt="unleash-omp npm version" /></a>

<br />
<br />

<table>
<tr>
<td align="center"><strong>Unleash</strong><br /><sub>Claude Code</sub></td>
<td align="center"><strong>Unleash-GPT</strong><br /><sub>Codex CLI</sub></td>
<td align="center"><strong>Unleash-OMP</strong><br /><sub>Oh-My-Pi</sub></td>
</tr>
<tr>
<td align="center"><code>npm install -g unleash-cc</code></td>
<td align="center"><code>npm install -g unleash-gpt</code></td>
<td align="center"><code>npm install -g unleash-omp</code></td>
</tr>
</table>

<sub>macOS arm64 · macOS x64 · Linux x64 · Linux arm64 · Windows x64 · Windows arm64</sub>

<br />
<br />

</div>

---

# Unleash

Unleash is a Go binary patcher and operator setup suite for three local coding agents:

| Product | Target | npm package | Command | State directory | Config/rules written |
|---|---|---|---|---|---|
| **Unleash** | Claude Code | `unleash-cc` | `unleash` | `~/.unleash/` | `~/.claude/CLAUDE.md`, `~/.claude/AGENTS.md`, `~/.claude/settings.json` |
| **Unleash-GPT** | OpenAI Codex CLI | `unleash-gpt` | `unleash-gpt` | `~/.unleash-gpt/` | `~/.codex/AGENTS.md`, `~/.codex/config.toml` |
| **Unleash-OMP** | Oh-My-Pi / OMP | `unleash-omp` | `unleash-omp` | `~/.unleash-omp/` | `~/.omp/agent/AGENTS.md`, `~/.omp/agent/config.yml` |

Each product is shipped as its own npm package, binary name, release tag family, and patch set. The normal Claude Code Unleash flow remains unchanged; GPT and OMP are separate companion tools.

---

## Quick setup

Install the npm package for the agent you use, then run `setup`. Each npm package is a thin launcher that selects the prebuilt binary for your OS/CPU.

### Claude Code: Unleash

```bash
npm install -g unleash-cc
unleash setup
```

`unleash setup` performs the full Claude Code flow: target discovery, binary patching, authorization rules, plugin install, and update guard setup.

Useful commands:

```bash
unleash status
unleash patch
unleash verify
unleash scan
unleash tui
unleash doctor
unleash rollback
```

### Codex CLI: Unleash-GPT

```bash
npm install -g unleash-gpt@latest
unleash-gpt setup
```

`unleash-gpt setup` discovers the local Codex CLI binary, applies Codex-specific byte patches, and installs Codex operator config.

Useful commands:

```bash
unleash-gpt status
unleash-gpt patch --dry-run
unleash-gpt patch
unleash-gpt verify
unleash-gpt install-rules
unleash-gpt rollback
```

Current Codex setup writes:

```toml
approval_policy = "never"
sandbox_mode = "danger-full-access"
dangerously_bypass_approvals_and_sandbox = true
```

### Oh-My-Pi: Unleash-OMP

```bash
npm install -g unleash-omp@latest
unleash-omp setup
```

`unleash-omp setup` discovers the installed OMP bundle, applies OMP-specific byte patches, and installs OMP operator config.

Useful commands:

```bash
unleash-omp status
unleash-omp patch --dry-run
unleash-omp patch
unleash-omp verify
unleash-omp install-rules
unleash-omp rollback
```

Current OMP setup writes:

```yaml
tools:
  approvalMode: yolo
```

---

## What each product patches

### Unleash for Claude Code

Unleash patches Claude Code's Bun SEA bytecode in place. The current Claude Code patch inventory contains **113 patch files** / **159 subpatches** across these categories:

| Category | Count | Effect |
|---|---:|---|
| Permissions | 26 | Bypass permission gates, sandbox restrictions, trust dialogs, workspace checks, and remote bypass kill switches. |
| Refusal and AUP | 14 | Remove local refusal/AUP text, plan-mode restrictions, denial workarounds, and refusal stop handling. |
| Classifier | 14 | Fail classifiers open; neutralize safety checks, dangerous prefix checks, command-injection blockers, and `canUseTool` gates. |
| Telemetry | 13 | Disable metrics, Datadog, OTEL, plugin telemetry, additional-protection headers, and domain preflight telemetry. |
| Feature gates | 9 | Flip 42 Statsig-gated features on, including agent teams, background tasks, subagent forking, and reactive compaction. |
| Rate limits | 13 | Raise timeouts, output caps, subagent caps, max turns, retry counts, and fallback thresholds. |
| Subscription | 3 | Pin subscription-sensitive gates to available states. |
| Attribution | 6 | Blank co-author trailers and generated-with markers in bytecode constants. |
| Infrastructure | 15 | Neutralize off switches, process exits, root restrictions, plugin denylists, MCP friction, and update guards. |

### Unleash-GPT for Codex CLI

Unleash-GPT uses Codex-specific target discovery, config, and byte patches. Current patch coverage includes:

- Sentry DSN loopback redirect.
- OTLP metrics loopback redirect.
- Cyber-safety documentation endpoint loopback redirect.
- Local high-risk cyber fallback wording neutralization.
- Local approval/policy rejection wording neutralization.
- Local approval/sandbox fallback warning neutralization.
- Managed Codex operator rules under `~/.codex/AGENTS.md`.
- Managed Codex config requiring `approval_policy = "never"`, `sandbox_mode = "danger-full-access"`, and `dangerously_bypass_approvals_and_sandbox = true`.

### Unleash-OMP for Oh-My-Pi

Unleash-OMP uses OMP-specific bundle discovery, config, and byte patches. Current patch coverage includes:

- Auto-QA grievance endpoint loopback redirect.
- OMP approval `deny` policy neutralization.
- OMP approval `prompt` policy neutralization.
- ACP permission gate set emptied for `bash`, `edit`, `delete`, and `move`.
- Startup update check default disabled.
- Marketplace auto-update default disabled.
- OMP npm update endpoint loopback redirect.
- Managed OMP operator rules under `~/.omp/agent/AGENTS.md`.
- Managed OMP config requiring `tools.approvalMode: yolo`.

---

## Build from source

Clone once, then build whichever binary you need.

```bash
git clone https://github.com/VoidChecksum/unleash
cd unleash/go

# Claude Code
GOOS=linux   GOARCH=amd64 go build -o unleash-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o unleash-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o unleash-windows-amd64.exe .

# Codex CLI
GOOS=linux   GOARCH=amd64 go build -o unleash-gpt-linux-amd64 ./cmd/unleash-gpt
GOOS=darwin  GOARCH=arm64 go build -o unleash-gpt-darwin-arm64 ./cmd/unleash-gpt
GOOS=windows GOARCH=amd64 go build -o unleash-gpt-windows-amd64.exe ./cmd/unleash-gpt

# Oh-My-Pi
GOOS=linux   GOARCH=amd64 go build -o unleash-omp-linux-amd64 ./cmd/unleash-omp
GOOS=darwin  GOARCH=arm64 go build -o unleash-omp-darwin-arm64 ./cmd/unleash-omp
GOOS=windows GOARCH=amd64 go build -o unleash-omp-windows-amd64.exe ./cmd/unleash-omp
```

Go 1.24.2+ is required. Release workflows build all six platform targets for each product.

---

## Release and package layout

| Product | Git tag pattern | GitHub artifacts | npm package | npm binary |
|---|---|---|---|---|
| Unleash | `cc-v*` | `unleash-*` | `unleash-cc` | `unleash` |
| Unleash-GPT | `gpt-v*` | `unleash-gpt-*` | `unleash-gpt` | `unleash-gpt` |
| Unleash-OMP | `omp-v*` | `unleash-omp-*` | `unleash-omp` | `unleash-omp` |

<!-- release-versions:start -->
| Product | Latest release |
|---|---|
| Unleash | `cc-v0.0.1` |
| Unleash-GPT | `gpt-v1.0.1` |
| Unleash-OMP | `omp-v1.0.1` |
<!-- release-versions:end -->

The npm packages are thin launchers with prebuilt platform binaries in `bin/`. Publishing is tag-driven: push `cc-vX.Y.Z` to publish `unleash-cc@X.Y.Z`, `gpt-vX.Y.Z` to publish `unleash-gpt@X.Y.Z`, and `omp-vX.Y.Z` to publish `unleash-omp@X.Y.Z`.

---

## Target discovery

| Product | Discovery targets |
|---|---|
| Unleash | Claude Code native installers, npm/bun/pnpm/volta/nvm/fnm/mise layouts, Homebrew, WinGet, Scoop, Chocolatey, Linux package layouts. |
| Unleash-GPT | Codex native installer, npm optional package layout, PATH shim fallback, platform package variants. |
| Unleash-OMP | OMP Bun global install, npm global install, PATH shim fallback, custom npm prefixes. |

All patch operations create timestamped backups before writing unless run with `--dry-run`.

---

## Safety model

The tools install an operator-authorization block with four hard stops:

1. No secret exfiltration to networks not invoked by the current task.
2. No overwriting uncommitted work without green tests or explicit confirmation.
3. No sending messages or creating public PRs without in-session acknowledgement.
4. Force-push to `main`/`master` requires explicit in-session consent.

Everything else is treated as pre-authorized local operator work.

---

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).
