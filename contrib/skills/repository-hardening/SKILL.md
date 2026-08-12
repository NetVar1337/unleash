---
name: repository-hardening
description: "Use to harden a repo end-to-end: inventory, baselines, issues/PRs, security, merge-ready pub."
version: 1.7.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: github
  upstream: C:\Users\Admin\.agents\skills\repository-hardening\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\repository-hardening\SKILL.md

# Repository Hardening

Use this for broad requests to improve one or more active repositories, audit every open issue/PR, incorporate authoritative external data, and leave focused merge-ready changes. This orchestrates narrower GitHub, testing, security, and code-review skills; it does not replace them.

## Definition of Done

A hardening pass is complete only when it has:

1. Refreshed repository and GitHub state.
2. Established reproducible build/test/lint/type/security baselines.
3. Triaged every in-scope issue and PR with evidence.
4. Implemented focused changes on isolated branches/worktrees.
5. Rerun affected checks after the latest base update.
6. Obtained independent review for multi-file or security-sensitive changes.
7. Published focused PRs with honest validation evidence and blockers.
8. Refreshed issue/PR state again before the final report.

Do not equate a local patch, a focused test, or a draft PR with completion of a repository-wide request.

## Workflow

### 1. Inventory the Live State

For every repository, capture:

- default branch, remotes, current HEAD, dirty state, worktrees;
- repository guidance and quality-bar documents;
- package managers, lockfiles, CI workflows, release/deploy paths;
- all open issues and PRs, including draft/base/head/check/review state;
- recent default-branch movement and active maintainer work.

Refresh with `git fetch origin --prune` and GitHub queries. Treat this as a snapshot that can expire during the session.

### 2. Separate Workstreams

Use one isolated branch/worktree per logical concern. Keep these distinct:

- issue fixes;
- baseline quality repairs;
- dependency/security upgrades;
- generated-data or standards ingestion;
- large feature PR review;
- documentation/design stacks.

Never mix unrelated baseline cleanup into a feature fix without explicitly reviewing and justifying each file. If validation exposes unrelated failures, either split them into a separate PR or report them as baseline blockers.

For user-requested multi-agent implementation, use `references/multi-agent-delivery-integration.md`: pin one base SHA, create one worktree/branch per delivery, assign one agent per worktree, verify each local commit independently, and integrate in dependency order. Run bounded waves when concurrency is limited. If the requested single-PR shape conflicts with repository governance, surface that before coding and keep any permitted umbrella PR draft until owner/large-diff gates clear.

### 3. Establish Baselines Precisely

Derive commands from CI and repository guidance rather than guessing. Run checks individually at least once so the exact failing stage is retained.

For read-only audits of test, CI, packaging, observability, and developer-experience posture, follow `references/repository-posture-audit.md`. It emphasizes gate reachability (test collection, path filters, required-job aggregation), release metadata/finalization, logger namespace coverage, documentation drift, concise `file:line` evidence, and clean-tree verification.

For native Visual Studio/MSBuild source-completeness and portability audits, follow `references/native-project-manifest-audit.md` and use `scripts/vcxproj_manifest_audit.py`. Keep filesystem, build-manifest, IDE-filter, and runtime-registration truth separate; classify project-unlisted headers conservatively; retain complete bounded evidence; and prove the final tree still matches the baseline.

Record:

- command;
- revision and base;
- pass/fail/skip count;
- required services or environment;
- whether the failure is introduced by the branch or already on the base.

A command chain that exits nonzero is not evidence that earlier or later unobserved stages passed. Do not claim a full suite passed when only a focused test did.

### 4. Audit Issues and PRs

For each issue:

- reproduce or identify missing reproduction details;
- map it to current code and related PRs;
- decide fix, needs-info, duplicate, backlog, or close;
- state acceptance criteria and ownership.

For each PR:

- inspect incremental diff, full cumulative diff, checks, reviews, comments, migrations, security, and deployment impact;
- trace integrations across runtime boundaries: the process making an API call must receive the correct credential components and least-privilege secret access;
- verify server actions enforce tenant membership capability in addition to platform role/persona;
- verify base freshness and mergeability;
- distinguish generated files from hand-written code;
- recommend an explicit merge order for stacks.

Use `references/active-repositories-and-stacked-prs.md` for moving-state and stacked-PR procedures. Use `references/runtime-boundary-and-validation-pitfalls.md` for deployment-secret tracing, endpoint-specific credential semantics, membership authorization, late-prop React state, hidden mock failures, database-test gating, and provider-idempotency verification. Use `references/stateful-route-external-effects-review.md` for state-machine entry tracing, concurrency interleavings, crash boundaries, provider idempotency assumptions, audit/outbox delivery, and recovery-test design. Use `references/actionable-error-classification.md` for structured status guards, branch precedence, negative overmatch tests, redaction, local HTTP runtime probes, and follow-up review loops. Use `references/dependency-advisory-remediation.md` for audit-path triage, compatible upgrades, peer deduplication, override pitfalls, and before/after verification.
Use `references/agent-runtime-trust-boundary-audit.md` for read-only agent/runtime architecture audits covering model→tool execution, sandbox authority, egress enforcement, plugin pre-load trust, HITL control-plane separation, and verified-fact-versus-inference reporting.

### 5. Incorporate External Standards Safely

Prefer authoritative, versioned primary sources for identity, schema, and relationships. Treat secondary articles as contextual enrichment only unless their claims are independently verified.

For generated standards data:

- pin release/version and checksum;
- preserve provenance and lifecycle fields;
- handle revoked/deprecated records explicitly;
- validate malformed/unknown references;
- require deterministic generation or byte-for-byte `--check` output;
- report exact source version, checksum, record counts, and generation status.

Never copy large third-party prose into product data. Summarize and retain source-scoped references.

### 6. Implement Test-First and Minimally

For a bug, first add a regression test that fails for the intended reason. Preserve a visible RED/GREEN history when repository guidance requests it. Reuse existing abstractions and avoid weakening production contracts merely to satisfy an unrealistic test fixture.

When defaults and their registrations live in separate files, use `references/config-source-parity.md` to classify static, genuinely dynamic, and environment-resolved entries; write a failing set-parity test; verify fallback/provider semantics; and avoid inventing IDs or metadata.

After implementation:

- run focused checks;
- run the repository-required broader gates;
- run `git diff --check`;
- inspect the complete diff and status;
- remove generated local artifacts;
- rerun after any rebase or conflict resolution.

### 7. Independent Review

For two or more edited source files, security-sensitive behavior, billing/auth changes, migrations, or generated standards data, request an independent review before marking ready. Give the reviewer the exact path, base, intent, validation results, and high-risk areas.

Treat every review verdict as bound to an immutable commit range. After resolving findings:

1. rerun all affected focused and broad checks;
2. inspect the complete diff and remove generated local artifacts;
3. commit and push the coherent revision;
4. record the exact remote head SHA; and
5. request a fresh review of that SHA or base...SHA range.

Never reuse approval of an older head as approval of later remediation. If work is interrupted before commit/push, report it as local and unpublished rather than implying the PR was updated.

Use `references/security-remediation-finalization.md` for fail-closed surface audits, generated snapshot verification, baseline comparison, exact-head review, and safe continuation of an existing PR.

### 8. Publish in Focused Units

Before commit/push/PR creation:

1. Refresh `origin` and GitHub state.
2. Confirm the target PR/issue did not merge or materially change during the audit.
3. Rebase or update the branch.
4. Rerun validation affected by the base movement.
5. Commit only intended files with repository-compatible identity and message format.
6. Push and open a draft when review/CI remains; mark ready only after blockers clear.

If the target PR merged while local follow-up work was still in progress, do not rewrite its old branch or replay the whole squash-merged stack. Use `references/mid-flight-pr-merge-followup.md` to isolate the post-merge commit onto current `main`, revalidate it, publish a clearly labeled follow-up PR, classify zero-step CI startup failures, and bind review to the final immutable head.

PR bodies must distinguish:

- what changed;
- why and linked issue;
- impact/anti-goals;
- exact tests run and observed results;
- known baseline failures;
- security or migration notes;
- stack dependency and merge order, if any.

## Pitfalls

- **Stale final report:** A PR can merge while the audit is running. Refresh immediately before publication and again before reporting.
- **Pre-rebase evidence:** Validation from an older base is invalid after rebase until rerun.
- **Silent chained failures:** Run important commands separately before combining them.
- **Draft inflation:** Opening a draft PR is progress, not merge readiness.
- **User-requested mega-PR versus repository policy:** Read the contribution charter before promising one PR. A user preference does not erase one-concern, file-count, runtime-line, ADR, or CODEOWNERS gates. If policy exposes an owner-approved large-diff path and the user insists, use a transparently blocked draft umbrella; otherwise split the work rather than mislabeling it merge-ready.
- **Unverified subagent delivery:** A returned summary is not evidence that a commit exists or tests ran on the committed tree. Verify branch SHA, scope, clean status, diff, and reported commands before cherry-picking.
- **Scope creep:** Split unrelated cleanup found during validation.
- **Missing CI:** No checks on a stacked/non-default base means “not run,” not “passed.” Supply local evidence and document the gap. If a red job has no steps or logs, inspect job JSON and check-run annotations before blaming code; `steps: []` with `runner_id: 0` usually identifies a pre-run infrastructure/account blocker.
- **Generated-file review blindness:** Review generator logic/tests first, then deterministic output. If the intended generator requires an unavailable live service, verify every reconstructed snapshot field against source content (including normalized body bytes, size, hash, and graph description), label the result accurately, and keep the PR draft until intended generation or equivalent reviewer-approved evidence is available.
- **Baseline formatter churn:** Before formatting whole legacy files in a focused security PR, run the formatter against temporary copies from `HEAD`. If the same files fail unchanged, preserve surgical edits, run semantic lint, and document the baseline instead of introducing unrelated whole-file churn.
- **Duplicate PR continuation:** When a branch already backs an open PR, update that PR rather than creating another. Refresh the branch-to-PR mapping immediately before publication and verify the remote head after push.
- **Non-atomic idempotency:** A unique nullable key does not stop two requests from reading null and minting different keys. Claim external side effects with conditional state/key updates, fail concurrent losers before the provider call, reuse persisted keys on retry, and test terminal-state reversals plus race loss.
- **Open-ended write roles:** `role !== viewer` can accidentally authorize future enum values. Use explicit writable-role allowlists and test the real route/action wiring fails before entitlements, external reads, and writes.
- **Force-upgrade security fixes:** Capture the advisory paths first. Automated audit repair may propose incompatible major downgrades; never use force merely to reduce the count, and reject overrides that leave `npm ls` invalid.
- **Credential mounted to the wrong service or wrong gate:** Trace the caller to the actual runtime, evaluate mount conditions against checked-in production values and always-on jobs, and mirror the mount condition in secret-access IAM.
- **Authentication-family confusion:** The same provider can use different Basic-auth usernames across customer and hacker/resource APIs. Verify the exact endpoint family's authoritative docs before replacing handles with token identifiers or designing a migration.
- **Incomplete authorization dimensions:** Platform role or workspace persona alone does not grant write capability; enforce tenant membership role in the server-side mutation.
- **Lint-only React refactors:** Replacing an effect with lazy state can freeze a prop that arrives after mount. Verify the parent lifecycle and test the prop transition.
- **Green tests with caught mock errors:** Read stderr. Missing mock exports can throw inside production `try/catch` while the test still passes; mock and assert the new call.
- **Skipped integration suites with active hooks:** Gate database suites and any top-level cleanup hooks so missing service configuration produces a true skip rather than a hook failure.
- **Cross-platform scripts:** Distinguish a platform-specific package script from an application build failure; use the CI-equivalent underlying command for diagnosis, then fix portability separately if in scope.
- **Shared `node_modules` across worktrees:** Do not junction/symlink one worktree's dependency tree into another when builds enforce a project filesystem root or code generation writes into dependencies (for example Prisma clients). The shared tree can make one worktree's generation invalidate another and Turbopack may reject the out-of-root link. Prefer a lockfile-faithful `npm ci` in each worktree before final typecheck/build; treat the initial link failure as setup evidence, then rerun the canonical build from the isolated tree.

## Final Report Format

Provide:

1. Published artifacts and links.
2. Per-repository validation table.
3. Per-issue/PR status table.
4. External-source provenance and generated-data evidence.
5. Remaining blockers, clearly separated from completed work.
6. Required merge order.

Never call the whole assignment complete when unpublished changes, unresolved review findings, failing required gates, or stale post-rebase validation remain.

