---
name: development-toolchain-curation
description: "Use when auditing or pruning agent skills, MCPs, hooks, and language/RE toolchains."
version: 1.1.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\development-toolchain-curation\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\development-toolchain-curation\SKILL.md

# Development Toolchain Curation

Build a small, audited, class-level development stack rather than installing every popular skill or MCP. The goal is maximum correctness and coverage with minimum overlap, prompt cost, credential exposure, and write authority.

## Trigger

Load this skill when the user asks for the best development skills, coding-agent plugins, MCP servers, hooks, documentation sources, language tooling, reverse-engineering integrations, a review of an existing Hermes development setup, or an audit of a repository's validation/CI toolchain and locally runnable test matrix.

## Workflow

1. **Audit before recommending.** Inspect the live Hermes version, enabled toolsets, installed skills and trust/source, configured MCPs, plugins, hooks, update status, and relevant local pinned catalogs. Never recommend duplicates blindly.
2. **Establish target scope.** Record host OS, languages, editors/IDEs, build systems, licensed RE tools, whether dynamic analysis is authorized, and whether the user asked only for recommendations or for installation.
3. **Use a trust hierarchy.** Prefer bundled Hermes capabilities, official project integrations, primary documentation, actively maintained upstream tools, then narrowly selected community components. Popularity is supporting evidence, not proof of quality.
4. **Separate layers.** Distinguish:
   - Skills: procedures and decision workflows.
   - Plugins: in-process Hermes extensions with broad trust.
   - MCPs: external tool/API capability with process or network boundaries.
   - Hooks: deterministic policy/automation at lifecycle boundaries.
   - Compilers, LSPs, linters, tests, debuggers: actual correctness gates.
   Do not use a skill as a substitute for a compiler or validator.
5. **Rank instead of dumping.** Produce `keep`, `add now`, `add on demand`, and `avoid/redundant` tiers. Pick one primary tool per overlapping category and explain when an alternative wins.
6. **Verify current version status.** Query live registries or primary docs for claims such as stable, preview, prerelease, retired, or LTS. Do not infer stability from a documentation URL.
7. **Prefer project-local quality gates.** Preserve each repository's formatter, linter, test runner, lockfile, compiler, and wrapper. Recommend global hooks only for truly cross-project safety policy.
8. **Secure every extension.** Inspect source, license, scripts, install commands, required credentials, network behavior, tool permissions, and update mechanism before installation. Pin sensitive MCP packages or commits.
9. **Translate recommendations into installation scope.** When the user says “install what you recommend,” install the `add now` tier only. Treat `on demand` entries as conditional: first verify that the host application, license, credentials, and concrete use case exist. Do not turn optional alternatives into a bulk install.
10. **Verify after changes.** Run the relevant Hermes list/test/doctor commands, restart or reload as required, and confirm the actual tools exposed to the model. Prefer a fresh-process end-to-end tool call over status text alone.
11. **Distinguish installed from always active.** A skill is normally indexed and loaded on demand; foreign metadata such as Cursor's `alwaysApply` does not make it universal in Hermes. For a user-requested cross-project behavioral policy, use the supported always-loaded `~/.hermes/SOUL.md` surface after preserving its existing identity and reconciling conflicts with user/project rules. Keep full audited skills installed for provenance and on-demand detail; put only concise standing behavior in `SOUL.md`. Start a fresh session or reset before claiming activation.
12. **Inspect installers before invoking them.** A repository's `install` subcommand may install unrelated compilers, create aliases, or mutate system PATH. Read its implementation/help first; when appropriate, build from pinned source, expose the binary through an existing user-local PATH, and configure project-local flags instead of accepting broader side effects.
13. **Honor security verdicts.** If Hermes blocks a community skill as dangerous, do not bypass the verdict through direct filesystem copying. Report the quarantined subset, install only audited allowed components, and promote only independently reviewed high-signal behavior—not blocked installers, persistence recipes, or shell commands—into global policy.
14. **Roll out by profile, not by assumption.** Hermes profiles are isolated. Audit, install, configure, and verify the default profile and every named profile independently. Prefer a common engineering core plus specialist profile overlays over cloning every skill and MCP everywhere. Use an exact-name manifest for suites and support files, read back each destination profile, and verify long skill names with `skills inspect` rather than relying on the truncated table rendering.
15. **Verify the exposed MCP surface, not just transport.** A successful `hermes mcp test` reports what the server advertises; it does not prove Hermes-side tool filtering. In a fresh process, confirm required tools are registered and mutators are absent. Removing an upstream `--unsafe` flag is not sufficient when typed mutation tools remain available.

## Hermes Installation and Provider Selection

- Enabling a backend plugin only makes its provider available; it may not select that provider. Configure the provider explicitly and set any required managed-gateway flag.
- Provision credentials through echo-free local prompts or secret managers, never by embedding them in command arguments. Verify a real minimum capability rather than trusting token shape or identity metadata alone. If a credential pasted into chat fails validation, remove it from durable config and require rotation instead of leaving an exposed dead secret installed.
- For managed Browser Use, the durable config is `browser.cloud_provider: browser-use` plus `browser.use_gateway: true`. Confirm it from config and exercise the browser in a fresh Hermes process.
- Configuration and plugin changes are startup-scoped. Apply them to long-running gateway processes only after checking how the gateway is managed.
- Before `hermes gateway restart`, inspect gateway service state. On an unmanaged process, restart may offer to install an autostart service or Scheduled Task; answer prompts deliberately rather than accepting defaults accidentally. Verify one healthy gateway PID afterward.
- Treat updater output as one signal, not proof. Run `hermes skills check` and load/inspect the installed skill. If an official skill repeatedly reports an update after a successful safe install, record it as update-metadata drift rather than repeatedly reinstalling it.

## MCP Rules

- Do not add filesystem, shell, Git, memory, web-fetch, or reasoning MCPs when Hermes native tools already cover them well.
- Prefer stdio. For HTTP, bind local servers to `127.0.0.1`, require authentication where applicable, and validate origins.
- Pass only narrowly scoped credentials required by that server.
- Disable server-initiated sampling for untrusted or unnecessary servers.
- Restrict enabled tools to the minimum needed; avoid parallel mutation of shared files, databases, IDBs, or external accounts.
- Use `hermes mcp configure <name>` for list-valued tool selection. Do not pass a JSON array to `hermes config set mcp_servers.<name>.tools.include` and assume it became a list; unknown list-valued keys can persist as one scalar string.
- Treat upstream safety flags and Hermes filtering as separate layers. After removing `--unsafe`, explicitly exclude any remaining typed mutators and verify the registered fresh-process tool names.
- Treat one-click local MCP installation as arbitrary code execution. Show and review the exact command and arguments.

## Plugin and Skill Rules

- Bundled skills and plugins are the preferred baseline.
- Inspect community `SKILL.md`, scripts, license, tool requirements, credentials, and external actions before installation.
- Treat giant skill packs as discovery catalogs. Cherry-pick demonstrated gaps instead of importing hundreds of overlapping instructions globally.
- Plugins run in-process and deserve a higher trust bar than stdio MCPs. Prefer a skill plus CLI, service-gated tool, or MCP when in-process access is unnecessary.

## Hook Rules

- Prefer repository-controlled pre-commit/pre-push/CI checks for formatting, linting, tests, and security scanning.
- Use Hermes `pre_tool_call` hooks only for narrow cross-project safety policy such as blocking destructive commands or unauthorized mutation.
- Avoid global hooks that run every language's linters after every file write.
- Keep first-use consent enabled; validate configured hooks with Hermes hook diagnostics.

## Reverse-Engineering Rules

- Local user-supplied binaries, dumps, firmware, and IDBs may be analyzed when authorized, but begin static and evidence-first.
- Load only the MCP for the active disassembler/debugger. Do not expose IDA, Ghidra, Binary Ninja, Frida, and broad offensive packs globally at once.
- Default to read-only analysis. Renaming, patching, IDB mutation, process-memory writes, kernel-memory writes, or live-target changes require explicit scope.
- Offset reports should include module, build/hash, RVA, signature, uniqueness, confidence, and validation method.
- Prefer actively maintained tool-specific MCPs over broad self-bootstrapping or self-evolving offensive tool routers.

## Documentation Hierarchy

1. Language specification or official project documentation.
2. Compiler/toolchain documentation.
3. Trusted current-doc MCPs.
4. Repository-local docs and source.
5. High-quality technical references such as cppreference.
6. Cheat sheets only for quick recall, never as authority over the compiler/specification.

## Output Shape

Include:

- A short verdict.
- Current setup audit without printing credentials.
- `Keep`, `Add now`, `On demand`, and `Avoid` sections.
- Language/tool matrix with primary docs, LSP, formatter, linter, tests, debugger, security gate, and version status.
- Exact commands only when verified against the current Hermes CLI or upstream installation docs.
- Security and rollback notes.
- A statement of whether anything was actually installed or modified.

## Pitfalls

- Do not equate GitHub stars with quality or maintenance; inspect `pushed_at`, archive state, releases, issues, and licensing.
- Do not call a development branch stable because a URL resolves.
- Do not recommend globally installing multiple overlapping agent methodologies.
- Do not expose credential fragments in audit output.
- Do not harden transient setup failures into permanent negative claims; capture the remediation or retry strategy instead.
- Do not install when the user only asked to find or compare components.
- A symlinked `~/.hermes/skills` can trigger a false containment failure after a community skill scans `SAFE`/`ALLOWED` because the installer compares the resolved destination with the unresolved skills root. Do not treat that as a security verdict. Preserve the scanner result, pin and verify the upstream commit/hash, then copy only the approved skill directory into the resolved skills root and verify `skill_view`; never use this workaround for `BLOCKED` or `DANGEROUS` content.

## References

- See `references/curation-checklist.md` for a reusable audit, scoring, and verification checklist.
- See `references/language-and-re-baseline.md` for a condensed 2026-era language and reverse-engineering baseline; revalidate versions before reuse.
- See `references/hermes-install-verification.md` for converting a ranked recommendation into a minimal installation and verifying plugins, providers, gateway restarts, skills, and MCPs.
- See `references/credential-provisioning.md` for echo-free secret storage, capability-based credential verification, sanitized diagnostics, and rotation/removal handling.
- See `references/community-policy-and-cli-promotion.md` for safely promoting audited community rules into an always-active Hermes policy and installing native source-built CLIs as secondary project gates.
- See `references/validation-pipeline-audits.md` for repository-wide test/CI reachability audits, safe dirty-tree validation in Windows/WSL, false-green detection, and reporting merge blockers with exact evidence.
- See `references/multi-profile-development-rollout.md` for specialized multi-profile skill rollout, scalar-versus-list config handling, least-authority reverse-engineering MCP setup, and fresh-process exposure verification.

