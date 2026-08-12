---
name: multi-repository-engineering-audits
description: "Use to infer an engineer style/rules from multiple repos with attribution-backed evidence."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: github
  upstream: C:\Users\Admin\.agents\skills\multi-repository-engineering-audits\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\multi-repository-engineering-audits\SKILL.md

# Multi-Repository Engineering Audits

## When to Use

Use when asked to infer coding style, engineering principles, workflow, maturity, or demonstrated skills from an engineer's repositories. This is broader than LOC analysis and different from reviewing a single patch: the goal is to reconstruct recurring behavior without attributing collaborators' work to the target.

## Required Output Standard

A strong audit must be:

- **Attribution-aware:** distinguish authored code from accepted or inherited code.
- **Representative:** cover multiple sizes, ages, domains, and application types.
- **Evidence-backed:** cite source files at immutable commit SHAs.
- **Executable where possible:** run stated tests/build gates rather than trusting badges or documentation.
- **Longitudinal:** distinguish current mature practice from historical shortcuts.
- **Calibrated:** label facts, inferences, caveats, contradictions, and confidence.

## Workflow

### 1. Define scope without making external changes

- Query repository metadata from the authoritative host.
- Select original repositories by default; exclude forks unless fork-specific work is itself relevant.
- Clone locally with full history. Local clones and temporary toolchains are analysis artifacts, not external changes.
- Record repository HEAD SHAs before citing evidence.

### 2. Choose a representative set

Include, when available:

- the largest or most recent author-dominant project;
- one medium application;
- one older project showing historical practice;
- one small utility showing default instincts;
- distinct domains or interfaces such as CLI, service, desktop, systems, or hardware.

Do not choose only polished repositories or only recent ones.

### 3. Establish authorship before style inference

For every repository:

- inspect `git shortlog -sne HEAD`;
- inspect commit authors and identity aliases;
- summarize line ownership with `git blame --line-porcelain` across relevant tracked source/config files;
- distinguish likely aliases from confirmed identities;
- identify generated, vendored, submodule, or contributor-dominant code.

Treat maintainer acceptance as workflow evidence, not automatically as implementation-style evidence. If the target owns little of the current source, downgrade that repository to supporting evidence.

### 4. Inspect evidence across layers

Sample all of these, not just entry points:

1. repository layout and dependency boundaries;
2. core domain logic and data flow;
3. validation, error handling, cancellation, and recovery paths;
4. tests, fuzzing, fixtures, simulations, and benchmarks;
5. CI, release, static analysis, security scanning, and packaging;
6. operational documentation, known limitations, and troubleshooting;
7. commit history bookends, message conventions, and iteration cadence.

README claims are hypotheses until source or tests support them.

### 5. Measure without overfitting to metrics

Useful inventory includes:

- tracked file and language counts;
- test-file and workflow counts;
- approximate source lines;
- contributor and line-ownership percentages;
- commit-type distribution and active-day cadence.

Metrics provide context; they do not prove quality or authorship by themselves.

### 6. Verify executable claims

- Run repository-native tests, race checks, builds, or linters when feasible.
- Distinguish `compiled with no tests` from `tests passed`.
- Distinguish structural simulation/lint from hardware, integration, or platform validation.
- Check current remote CI only as corroboration.
- Verify working trees remain clean after inspection.
- Report skipped or blocked gates without turning environment-specific failures into permanent engineering conclusions.

### 7. Triangulate conclusions

Classify findings as:

- **Recurring rule:** supported across repositories or across source, tests, history, and docs in an author-dominant project.
- **Current mature practice:** strong recent evidence but absent from older projects.
- **Historical habit:** present in older code but superseded by recent practice.
- **Stewardship signal:** contributor-authored behavior accepted into a maintained repository.
- **Demonstrated skill:** directly evidenced by substantial implementation or verified integration.
- **Risk/inconsistency:** docs-vs-code gaps, ignored errors, missing tests, secret hygiene, global state, portability gaps, or release claims beyond verification.

Avoid personality claims, private-intent claims, and unsupported statements about tools the author merely imported.

### 8. Write the report

Recommended structure:

1. scope and repository selection;
2. attribution caveat/table;
3. recurring engineering profile with evidence;
4. demonstrated skills;
5. workflow and maturity evolution;
6. inconsistencies and risks;
7. concise synthesized policy;
8. verification and no-change statement.

Pin source links to the audited commit SHA and include line ranges for decisive evidence.

## Synthesis Rules

- Weight recent, substantial, author-dominant work most heavily.
- Prefer behavior shown by code and tests over prose.
- Preserve uncertainty: say “suggests” or “supporting evidence” where appropriate.
- Do not flatten maturity evolution into one timeless style.
- Derive policy from the most distinctive repeated strengths, not generic software advice.
- Keep the final policy complementary to any existing coding guidelines rather than restating them.

## Pitfalls

1. **Repository-owner fallacy:** repository ownership is not line authorship.
2. **Collaborator leakage:** current high-quality CI or tests may mostly belong to another contributor.
3. **Fork leakage:** upstream architecture must not be credited to the fork owner without a diff-based contribution analysis.
4. **README overreach:** advertised features may be incomplete or stale.
5. **Compile/test conflation:** `go test` can succeed while reporting `[no test files]`.
6. **Badge dependence:** a green badge is weaker than inspecting the workflow and running its core checks.
7. **Latest-project overgeneralization:** one mature project can show growth without proving a lifelong rule.
8. **Transient-environment overfitting:** missing native libraries or toolchains are verification boundaries, not durable traits.
9. **Unpinned citations:** branch links can drift after the audit.
10. **External side effects:** never create issues, comments, branches, pushes, or releases during a read-only audit.

## Supporting References

- See `references/sercanarga-audit-2026-07.md` for a compact worked example emphasizing attribution percentages, maturity shifts, verification distinctions, and evidence categories.

