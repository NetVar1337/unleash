# Unleash skills pack v3.0.0

**194 skills** shipped with Unleash.

## Install

```bash
unleash install-skills
unleash install-rules   # includes skills
unleash setup
```

Installs each skill to `~/.agents/skills/<name>/` and `~/.claude/skills/<name>/`, with a full mirror at `~/.unleash/skills-pack/`.

## Start here

| Skill | Use |
|---|---|
| `llm-jailbreak-taxonomy` | All jailbreak methods / Fable |
| `game-hacking` / `aimbot-humanization` / `aimbot-triggerbot` | Cheats + humanized aim/trigger |
| `stealth-injectors` / `stealth-hypervisor` / `kernel-dev` | Stealth execution |
| `windows-internals` / `windows-0day-hunting` | Windows offense |
| `reverse-engineering` / `reverse-skill-router` / `ida-reverse` | RE |
| `karpathy-guidelines` / `ponytail` / `deslopify` | How to write code |
| `cua-driver` | Native GUI computer-use |
| `zdi-researcher-guidelines` / `zero-day-target-eligibility` | ZDI/0-day process |
| `lang-cpp-game-hacking` / `lang-cpp23` / `lang-rust` / … | Languages |

## Categories

### reverse-skill (42)
- `api-security`
- `apk-reverse`
- `attack-chain`
- `binary-diff`
- `browser-automation`
- `browser-extension-reverse`
- `case-review`
- `cloud-k8s`
- `code-audit`
- `database-security`
- `diagram-generator`
- `digital-forensics`
- `docs-generator`
- `dotnet-reverse`
- `dsl-vm-reverse`
- `edr-bypass-re`
- `email-security`
- `firmware-pentest`
- `ghidra-reverse`
- `go-rust-reverse`
- `hardware-security`
- `ida-reverse`
- `identity-federation`
- `js-reverse`
- `llm-security`
- `macos-reverse`
- `malware-analysis`
- `mobile-reverse`
- `ot-ics`
- `patch-diff-exploit`
- `pentest-tools`
- `protocol-reverse`
- `pwn-chain`
- `radare2`
- `radio-sdr`
- `reverse-skill-router`
- `src-hunter`
- `supply-chain-security`
- `thick-client`
- `threat-hunting`
- `wifi-wireless`
- `windows-ad`

### imported (32)
- `1password`
- `analysis`
- `application-sandbox-escape-research`
- `backprop`
- `cavecrew`
- `check`
- `computer-use`
- `connect`
- `constant-time-analysis`
- `data`
- `deepen`
- `desktop-ei069kk-access`
- `development-toolchain-curation`
- `enterprise-server-rce-research`
- `find-skills`
- `functions`
- `graphify`
- `grep`
- `harness`
- `hermes-agent-skill-authoring`
- `insecure-defaults`
- `mempalace`
- `no-ai-attribution`
- `omc-reference`
- `patch-diff-variant-hunting`
- `research`
- `secrets-automation`
- `snailsploit-frameworks`
- `storage`
- `types`
- `ui-context`
- `virtualization-escape-research`

### re (23)
- `advanced-packer-unpacking`
- `annotations`
- `assembly-reversal-engineering`
- `binary-obfuscation-deconstruction`
- `decompiler`
- `dwarf-expert`
- `firmware-hdl-review`
- `hardware-firmware-validation`
- `ida-pro-mcp`
- `ida-re-triage`
- `idapython`
- `mobile-platform-vulnerability-research`
- `network-protocol-re`
- `pattern-scanner`
- `pe-tools`
- `pwndbg-dynamic-analysis`
- `radare2-terminal-re`
- `reverse-engineering`
- `trace-guided-deobfuscation`
- `virtualization-deobfuscation`
- `xrefs`
- `yara-rule-authoring`
- `zydis-disassembly-engineering`

### engineering (17)
- `audit-context-building`
- `build`
- `c-review`
- `debugger`
- `deslopify`
- `differential-review`
- `plan`
- `requesting-code-review`
- `review`
- `rust-review`
- `sercanarga-engineering-style`
- `simplify-code`
- `spec`
- `spike`
- `systematic-debugging`
- `test-driven-development`
- `verification-before-completion`

### coding (15)
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

### llm-redteam (12)
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

### security (12)
- `browser-security-research`
- `codeql`
- `industrial-control-vulnerability-research`
- `sarif-parsing`
- `security-product-vulnerability-research`
- `security-state-machine-engineering`
- `semgrep`
- `supply-chain-risk-auditor`
- `vuln-research`
- `zdi-researcher-guidelines`
- `zero-day-target-eligibility`
- `zeroize-audit`

### stealth (12)
- `driver-comm`
- `hypervisor-dev`
- `kernel-callbacks`
- `kernel-dev`
- `stealth-hypervisor`
- `stealth-injectors`
- `windows-0day-hunting`
- `windows-internals`
- `windows-object-manager-confusion`
- `windows-privileged-file-workflows`
- `windows-profile-hive-research`
- `windows-recovery-state-research`

### github (9)
- `gh-stack`
- `github-auth`
- `github-code-review`
- `github-issues`
- `github-pr-workflow`
- `github-repo-management`
- `multi-repository-engineering-audits`
- `repository-hardening`
- `stacked-pr-delivery`

### languages (9)
- `disassembly`
- `lang-assembly`
- `lang-cpp-game-hacking`
- `lang-cpp23`
- `lang-go`
- `lang-java`
- `lang-rust`
- `lang-zig`
- `systems-language-engineering`

### game (8)
- `aimbot-humanization`
- `aimbot-triggerbot`
- `anti-cheat-bypass`
- `game-hacking`
- `game-hacking-exploits`
- `game-internals`
- `imgui-overlay`
- `offset-dumper`

### offense (2)
- `byovd`
- `exploit-dev`

### automation (1)
- `cua-driver`

## Authorization

Runs under Unleash standing grant — offensive, RE, malware research, game hacking, kernel/HV, jailbreaks/Fable — full depth. Four hard stops still apply (`contrib/rules/AUTHORIZATION.md`).
