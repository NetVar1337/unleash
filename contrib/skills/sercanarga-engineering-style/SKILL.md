---
name: sercanarga-engineering-style
description: "Use when applying Sercan Arga evidence-backed engineering style."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: engineering
  upstream: C:\Users\Admin\.agents\skills\sercanarga-engineering-style\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\sercanarga-engineering-style\SKILL.md

# Sercan Arğa Engineering Style

## When to Use

Use for implementation, debugging, refactoring, review, CI, release, systems, networking, firmware, Go, C, or HDL work. Apply the transferable rules below; do not mechanically copy language-specific patterns into unrelated stacks. Repository-local rules and the user's explicit request take precedence.

## Core Rules

1. **Trace the real system boundary first.** Understand the actual data path and hardware/runtime constraints before choosing a fix.
2. **Keep the production path explicit.** Prefer straightforward functions, standard-library primitives, early returns, and clear package boundaries over hidden control flow.
3. **Use structure where the domain earns it.** Favor `cmd/` entry points, `internal/` implementation packages, narrow infrastructure interfaces, constructor wiring, and named ordered passes for stateful or order-sensitive transformations.
4. **Validate before side effects.** Establish the authoritative input, then validate required fields, resource bounds, alignment, widths, overlap, identity, and cross-field consistency before mutation, generation, synthesis, or hardware access.
5. **Preserve evidence and provenance.** Distinguish measured, unknown, and synthetic/fallback data. Never invent missing hardware or binary state; retain the authoritative capture and record source/tool/time provenance for generated artifacts.
6. **Separate correctness from optional capability.** Fail when a state invalidates correctness, integrity, or recovery. Warn and degrade only for capabilities explicitly defined as optional, and provide a deliberate bypass rather than silently weakening guarantees.
7. **Make diagnostics operational.** Wrap errors with operation and resource context, include actionable recovery guidance, and never describe structural lint as proof of runtime or driver compatibility.
8. **Bound concurrency and own lifecycle.** Limit parallelism, support cancellation and graceful shutdown, put timeouts around I/O, protect shared state explicitly, and avoid locks across network or disk operations.
9. **Test contracts, not only happy paths.** Cover invariants, malformed inputs, edge cases, races, and generated artifacts. Hardware, concurrency, and artifact-path changes should leave a deterministic hardware-free regression check where feasible.
10. **Use layered gates.** Format, vet/static-analysis, lint, race tests, language-specific analyzers, sanitizers/Valgrind, and HDL lint as applicable. Do not substitute one gate for another.
11. **Protect artifact integrity.** Inventory outputs, hash distributable artifacts, reject path traversal/symlinks/duplicates where manifests cross trust boundaries, and use atomic durable replacement for critical metadata.
12. **Treat delivery as implementation.** Provide documented build/run/diagnostic paths, secret-free environment examples, reproducible packaging, checksums for distributed binaries, and warnings around privileged or destructive behavior.
13. **Treat security as normal quality.** Sanitize input, parameterize queries, rate-limit exposed APIs, scan code and dependencies, preserve bounds and access policies, and never copy older plaintext-secret or unchecked-error patterns.
14. **Keep changes logically separated.** Distinguish feature, regression-test, CI, and corrective work when the repository workflow supports it.
15. **Prefer small focused packages and practical tools.** Build a direct utility first; introduce richer internal structure only when domain complexity and tests justify it.
16. **Match the current repository, not historical quirks.** Recent substantial work is the stronger signal. Do not imitate old ad-hoc commit messages or legacy shortcuts.

## Demonstrated Skills

- Go CLI, backend, and desktop engineering with idiomatic package layout (`cmd/`, `internal/`), interfaces, dependency wiring, and cross-compilation.
- Network tooling, bounded worker pools, cancellation, DNS/IP/ASN scanning, caching, proxies, and rate limiting.
- Service engineering with PostgreSQL, Redis, GORM, Docker Compose, Swagger, transactions, pagination, and graceful lifecycle management.
- PCIe/VFIO/NVMe firmware modeling, BAR/MSI-X behavior, register/capability validation, Vivado automation, and Go/C/SystemVerilog integration.
- CI/CD, release automation, static analysis, race detection, fuzzing, sanitizers, Valgrind, Cocotb, and HDL lint.
- Input sanitization, parameterized persistence, API integration with Gemini/OpenAI, dependency maintenance, and cross-platform delivery.

## Evidence and Provenance

Audited 2026-07-13 from public repositories. Main evidence:

- `PCILeechGen` at `f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28`: explicit model invariants and contextual errors in [`internal/firmware/devicemodel/validate.go`](https://github.com/sercanarga/PCILeechGen/blob/f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28/internal/firmware/devicemodel/validate.go); evidence-preserving model contracts in [`builder_contract_test.go`](https://github.com/sercanarga/PCILeechGen/blob/f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28/internal/firmware/devicemodel/builder_contract_test.go); secure, durable artifact manifests in [`manifest.go`](https://github.com/sercanarga/PCILeechGen/blob/f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28/internal/firmware/output/manifest.go); explicit limits on what must not be fabricated or overstated in [`KNOWN_ISSUES`](https://github.com/sercanarga/PCILeechGen/blob/f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28/KNOWN_ISSUES); layered Go/C/HDL checks in [CI](https://github.com/sercanarga/PCILeechGen/blob/f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28/.github/workflows/ci.yml); and deterministic build/check targets in the [Makefile](https://github.com/sercanarga/PCILeechGen/blob/f2d6a5fa80eb6cdf8f7cf7f45f3696828e7b4d28/Makefile).
- `insider-challenge` at `e9bd7f7d7950afe8d39d4d59a5a9dd9c9687fce9`: layered handler/service/repository/domain/config organization, narrow repository interfaces, dependency wiring, bounded lifecycle management, request timeouts, graceful shutdown, Docker Compose, Swagger, PostgreSQL, and Redis; see its [architecture documentation](https://github.com/sercanarga/insider-challenge/blob/e9bd7f7d7950afe8d39d4d59a5a9dd9c9687fce9/README.md#L92-L122) and [`message_sender.go`](https://github.com/sercanarga/insider-challenge/blob/e9bd7f7d7950afe8d39d4d59a5a9dd9c9687fce9/internal/service/message_sender.go).
- `ipmap` at `75d366e0047cf83baf9f86390c6132909e1ba2cd`: standard-library validation in [`modules/validators.go`](https://github.com/sercanarga/ipmap/blob/75d366e0047cf83baf9f86390c6132909e1ba2cd/modules/validators.go) and race/lint/cross-platform build gates in [CI](https://github.com/sercanarga/ipmap/blob/75d366e0047cf83baf9f86390c6132909e1ba2cd/.github/workflows/ci.yml). Recent implementation is substantially contributor-authored, so this is supporting evidence rather than the primary personal-style signal.
- `fuckregex` at `de08b38e02fa4f578a2f549be139f9de6a799822`: input binding/sanitization, early returns, cached DB reuse, and explicit API errors in [`handler/generate.go`](https://github.com/sercanarga/fuckregex/blob/de08b38e02fa4f578a2f549be139f9de6a799822/handler/generate.go).

## Caveats

- This is an inference from public work, not a statement of the author's private rules.
- `PCILeechGen` is recent and substantially more rigorous than older small utilities, so it carries the most weight.
- Testing maturity is uneven: `PCILeechGen` has extensive automated checks, while several smaller repositories expose little or no repository-level suite. Some `ipmap` tests are scaffolding rather than behavioral verification.
- `ipmap` has significant contributor-authored implementation. Conclusions prioritize original, author-dominant work such as `PCILeechGen`, `insider-challenge`, and `franslate`.
- Adopt the newer security posture; do not imitate older unchecked-error handling, plaintext local secret storage, tracked environment files, or incomplete HTTP response cleanup.
- Treat documented platform support and feature lists as claims to verify: older projects can depend on obsolete native toolchains, and documentation can run ahead of delivered behavior.

## Verification

Before completion, show the repository's own formatter/linter/test/build gates that ran, and distinguish skipped platform/hardware checks from passed checks.
