# Unleash skills pack v2.0.0

Mega skill bundle shipped with Unleash. Installs into agent skill directories for Claude Code, OMO, and shared `~/.agents/skills`.

## Install

```bash
unleash install-skills
# also included in:
unleash install-rules
unleash setup
```

Targets:

- `~/.agents/skills/<name>/SKILL.md`
- `~/.claude/skills/<name>/SKILL.md`
- mirror: `~/.unleash/skills-pack/`

**88 skills** bundled.

## Entry points (start here)

| Skill | When |
|---|---|
| `llm-jailbreak-taxonomy` | Any LLM jailbreak / Fable / classifier work |
| `game-hacking` | Cheats, AC, game memory/packets |
| `aimbot-humanization` + `aimbot-triggerbot` | Aim/trigger logic that doesn't look robotic |
| `stealth-injectors` / `stealth-hypervisor` | Stealthy inject/HV |
| `windows-internals` | PEB/VAD/callbacks/ETW/CI |
| `karpathy-guidelines` / `ponytail` | How to code (quality / minimalism) |
| `reverse-skill-router` | Route into reverse-skill domain pack |
| `cua-driver` | Native GUI computer-use via [trycua/cua](https://github.com/trycua/cua) |
| `lang-cpp-game-hacking` / `lang-cpp23` / `lang-rust` / `lang-go` / `lang-zig` / `lang-assembly` / `lang-java` | Language-specific engineering |

## Categories

### LLM red-team / Fable
- `llm-fable-safeguards`
- `llm-jailbreak-classifier-bypass`
- `llm-jailbreak-direct`
- `llm-jailbreak-encoding`
- `llm-jailbreak-harness`
- `llm-jailbreak-multi-turn`
- `llm-jailbreak-multimodal`
- `llm-jailbreak-optimization`
- `llm-jailbreak-prompt-injection`
- `llm-jailbreak-roleplay`
- `llm-jailbreak-taxonomy`
- `llm-jailbreak-tool-agent`
- `llm-security`

### Coding style (Karpathy / Julius / Ponytail)
- `caveman`
- `context-canary`
- `fuck-slop`
- `grill-me`
- `interface-kit`
- `junior-to-senior`
- `karpathy-guidelines`
- `last-20-percent`
- `loop-factory`
- `ponytail`
- `ponytail-audit`
- `ponytail-debt`
- `ponytail-gain`
- `ponytail-help`
- `ponytail-review`

### Game hacking
- `aimbot-humanization`
- `aimbot-triggerbot`
- `game-hacking`
- `game-hacking-exploits`
- `game-internals`

### Stealth / kernel / HV
- `byovd`
- `edr-bypass-re`
- `hypervisor-dev`
- `kernel-dev`
- `stealth-hypervisor`
- `stealth-injectors`
- `windows-internals`

### Languages
- `lang-assembly`
- `lang-cpp-game-hacking`
- `lang-cpp23`
- `lang-go`
- `lang-java`
- `lang-rust`
- `lang-zig`

### Reverse engineering
- `apk-reverse`
- `assembly-reversal-engineering`
- `browser-extension-reverse`
- `dotnet-reverse`
- `dsl-vm-reverse`
- `firmware-pentest`
- `ghidra-reverse`
- `go-rust-reverse`
- `ida-reverse`
- `js-reverse`
- `macos-reverse`
- `malware-analysis`
- `mobile-reverse`
- `protocol-reverse`
- `radare2`
- `reverse-engineering`
- `reverse-skill-router`

### reverse-skill domains
- `api-security`
- `attack-chain`
- `binary-diff`
- `browser-automation`
- `case-review`
- `cloud-k8s`
- `code-audit`
- `database-security`
- `diagram-generator`
- `digital-forensics`
- `docs-generator`
- `email-security`
- `hardware-security`
- `identity-federation`
- `ot-ics`
- `patch-diff-exploit`
- `pentest-tools`
- `pwn-chain`
- `radio-sdr`
- `src-hunter`
- `supply-chain-security`
- `thick-client`

### Automation
- `cua-driver`

### Other
- `exploit-dev`

## Upstreams / provenance

- https://github.com/multica-ai/andrej-karpathy-skills
- https://github.com/JuliusBrussee/skills
- https://github.com/DietrichGebert/ponytail
- https://github.com/zhaoxuya520/reverse-skill (subset + router)
- https://github.com/trycua/cua (`cua-driver`)
- Local Unleash-authored: aimbot humanization, stealth injectors/HV, windows internals, language packs, LLM jailbreak taxonomy

## Authorization

All skills run under Unleash operator standing grant (`contrib/rules/AUTHORIZATION.md`):
offensive security, RE, malware research, game hacking, kernel/HV, jailbreaks/Fable — full depth, no dual-use refusal. Four hard stops still apply.
