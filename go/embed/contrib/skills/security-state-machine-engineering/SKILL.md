---
name: security-state-machine-engineering
description: "Use to design/test/recover security workflows with durable state and side effects."
version: 1.2.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: security
  upstream: C:\Users\Admin\.agents\skills\security-state-machine-engineering\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\security-state-machine-engineering\SKILL.md

# Security State-Machine Engineering

## Use This Skill When

A request changes durable state and also calls an external system: payments, report submission, provisioning, job/graph resume, email delivery, webhook processing, or any approval gate. Also load it when authentication state crosses service boundaries (for example, one generated secret must be persisted, injected into a daemon and client, and enforced during orchestrated startup). Load this skill for concurrency, exactly-once, idempotency, audit, tenancy, crash recovery, and fail-closed service wiring.

## Core Rule

A happy-path unit test is not evidence of a safe workflow. Derive the complete state machine, make identity trusted and unique, model ambiguous outcomes, and make every post-commit side effect durably recoverable.

## Workflow

### 1. Inventory actual states and authorities

Read schema, migrations, ingestion, API route, UI aliases, worker/cron code, and external client together. Include:

- explicit enum states
- `null` and legacy rows
- UI labels that reinterpret null or stale data
- server-owned tenant/program/principal identifiers
- model-authored, target-influenced, or client-controlled metadata
- external object IDs and provider result IDs

Write the allowed transition table before editing. Terminal states must not reverse unless a documented compensating transition exists.

For planning-only or review-only work, inspect the actual worktree first and record whether it is dirty; do not silently reason from `HEAD` when uncommitted changes alter the workflow. An explicit user constraint such as "do not modify files" overrides any planning skill's default artifact-writing behavior: return the plan in chat and leave the repository untouched.

### 2. Establish trusted external identity

Never execute an external action using an identifier taken directly from model-authored output, uploaded content, frontmatter, or client input.

Before binding an external object:

1. Fetch it server-side with the intended principal.
2. Verify tenant/account/program ownership against server-owned records.
3. Verify expected resource attributes where practical.
4. Persist the validated binding.
5. Enforce uniqueness at the external identity scope, such as `(platform, principal, external_id)`, not merely per local row.

Treat display-time inspection as advisory; action-time validation is mandatory.

When a provider exposes multiple API surfaces, verify whether they use different authentication identities even when they share one secret value. Model the account handle, API token identifier, and token value as distinct fields; never default one identity to another. Verify each API surface independently, require a complete tuple at runtime and deployment boundaries, and invalidate ambiguous legacy credentials rather than guessing. See `references/split-api-credential-identities.md`.

### 3. Model claim and recovery explicitly

Use an explicit in-progress state rather than overloading a nullable idempotency key. Persist:

- actor
- principal/account identity
- stable idempotency key
- attempt/lease timestamp
- external identity

Resolve definite preflight failures before claiming. Distinguish:

- **definite failure before side effect** — safely return to reviewable state
- **ambiguous timeout/network failure** — remain in-progress and reconcile
- **durable external success** — finalize only with a durable result identifier

Never mark success merely because an HTTP response was 2xx if the authoritative external result ID is absent.

### 4. Commit local terminal effects atomically

The transaction that makes a decision terminal should also create its audit record and durable follow-up marker. Do not use:

```text
commit terminal state -> write audit -> call secondary system
```

A crash between arrows permanently loses work. Prefer:

```text
transaction(terminal state + audit + pending outbox)
worker/cron -> claim outbox -> reconcile external/thread state -> deliver -> mark delivered
```

Terminal retries must attempt reconciliation rather than blindly short-circuiting.

### 5. Make the consumer idempotent

An outbox alone is not enough. Handle the crash after the external call succeeds but before `DELIVERED` is persisted:

- use a claim lease, retry counter, and fencing token/generation; condition release and acknowledgement on the current token
- preserve the durable source mode when claiming: a single `claimed` state must carry `claimedFromState` (or use mode-specific claim states). Never recover a stale reconciliation claim from `confirming`/`ambiguous` to `pending`; doing so silently re-enables remote create and defeats at-most-once safety
- remember that a local fencing token cannot revoke remote I/O from a paused stale worker: split `claimed` (safe to reclaim only when it came from pre-create `pending`) from `creating` (ambiguous after expiry), and never automatically issue another create after the side-effect boundary unless remote idempotency is proven
- freeze the exact destination identity from immutable resource provenance (for example, the finding's producing thread), not from a mutable current-thread pointer merely sampled at event creation; never re-resolve that pointer on retries
- when an exact interrupt is discovered lazily, pass its ID into the fenced transition and persist it in the same CAS that enters `creating`; validate its ID, kind, and trusted binding before create, and include both delivery and interrupt IDs in remote metadata so a lost create response remains reconcilable
- on reconciliation, require known/listed runs to match the complete stored correlation tuple; do not let a local interrupt ID mask absent or conflicting remote metadata
- inspect external/thread state before retrying, paginate reconciliation to authoritative exhaustion, detect stalled pagination, and do not treat current absence as permission to recreate an `ambiguous` attempt
- distinguish accepted/pending work from confirmed delivery; acknowledge only after the remote run reaches an acceptable terminal state and the pinned interrupt/postcondition is gone
- avoid duplicate timeline/audit events with unique keys or conditional updates
- if an active run already exists, wait rather than create another
- if the interrupt is gone but board state is stale, repair only the board
- define how a stopped runtime is woken while retaining the pinned thread, or visibly park/dead-letter the event; endless `runtime not ready` retries are not eventual delivery
- release definite pre-create failures back to pending with bounded error detail; bound attempts and retain a visible dead-letter/manual-reconciliation state

### 6. Verify provider idempotency instead of assuming it

An `Idempotency-Key` header proves nothing unless the provider documents and honors it. Confirm:

- supported endpoint
- retention window
- key scope
- behavior across credentials/principals
- response after an ambiguous retry

If the contract is unavailable, rely on local uniqueness, preflight/reconciliation reads, explicit in-progress state, and conservative recovery. State the evidence gap in the PR.

## Test-First Matrix

Write failing tests before production code. At minimum cover:

### State compatibility
- legacy/null row shown as actionable by the UI
- each allowed transition
- each forbidden terminal reversal
- migration/backfill behavior

### Shared-state concurrency
- approve / approve
- approve / reject
- reject / reject (exactly one transition, audit, and follow-up)
- duplicate external ID on two local rows
- duplicate finalizer
- compare-and-set loser produces no audit and no external/follow-up action
- audit insertion failure rolls back the terminal transition

Use a disposable real database when possible. If unavailable, use a shared mutable fake that applies compare-and-set predicates. Mock call-shape assertions alone do not prove concurrency safety; manually returning `{count: 0}` proves only the branch behavior, not real database locking, predicate re-evaluation, or rollback semantics.

### Policy provisioning and successful-state caching
- backend exception stops the caller in enforcing mode
- explicit negative acknowledgement (`applied=false`, missing result ID, rejected policy) also stops the caller
- failed attempts do not enter the success cache and retry on the next turn
- cache identity includes the resource/workspace and a canonical hash of the effective policy
- semantically identical reordered/deduplicated inputs compile to the same hash and reuse the applied policy
- a changed effective policy re-provisions, including a change back to an older policy
- the cache represents the currently applied policy, not every policy ever seen; prefer `resource -> current_hash` over a historical set of `(resource, hash)` pairs
- serialize cache-check → external apply → positive acknowledgement → cache update so concurrent callers cannot pass while provisioning is incomplete
- audit/warn/non-enforcing modes and explicit operator opt-outs preserve their documented behavior

See `references/fail-closed-policy-provisioning.md` for a compact implementation pattern and RED/GREEN test sequence.

### Failure and crash windows
- missing credentials before claim
- explicit 4xx/rate limit
- ambiguous transport failure
- crash after claim
- external success without result ID
- crash after external success
- crash after terminal transaction
- audit failure
- follow-up delivery failure
- crash after delivery but before acknowledgement
- stale claim recovery
- stale claim recovery preserves the claimed-from mode: expired pre-create `pending` work may return to `pending`, but expired claims originating from `confirming` or `ambiguous` remain reconciliation-only and can never regain create permission
- stale worker resumes after lease takeover and is fenced from local acknowledgement; if it may already have crossed the remote-create boundary, replacement workers reconcile only and never issue a blind second create
- stale pre-create `claimed` lease returns to pending, while stale in-create lease becomes explicit `ambiguous`
- exact interrupt ID is durably written by the fenced `creating` transition before the remote request, including the crash between that CAS and request dispatch
- mutable current-thread pointer changes before event creation/backfill, but destination is selected from the resource's immutable producing-thread provenance
- lost create response later recovers both run ID and pinned interrupt/destination fingerprint from remote metadata
- ambiguous create remains reconciliation-only when no matching run is currently visible
- accepted run remains `confirming` until the pinned interrupt/postcondition is authoritatively gone
- ambiguous create reconciled when the matching run is beyond the first result page
- thread pointer changes after event creation but delivery remains pinned to the original interrupt
- runtime stopped before delivery: idempotent wake or explicit parked/dead-letter outcome
- pre-migration terminal row with no outbox event

### Authorization, input shape, and trust
- unauthenticated user
- read-only role and unknown future role (allowlist, not denylist)
- cross-tenant engagement/finding mismatch
- cross-tenant existence-oracle behavior: scope the resource lookup through the write-role membership predicate so absent, foreign-tenant, viewer, and unknown-role resources are indistinguishable
- malformed JSON, arrays/scalars, missing decision, unknown decision
- extra fields that try to smuggle an alternate action or external identifier
- every explicit, null, legacy, and terminal state for a disabled/approval path
- hostile external ID in model-authored metadata
- program/principal mismatch

For a temporarily disabled side-effect path, verify the unconditional refusal is
reached only after authentication/tenant authorization but before credentials,
claims, writes, audits, external calls, or resume operations. Search the complete
immutable tree—not only the diff—for direct endpoints, wrappers, routes, workers,
sidecars, CLIs, adapter methods, and dynamically assembled agent/tool registries.
Classify remaining primitives as remotely reachable, agent reachable, scheduled,
trusted-library/CLI-only, dormant export, generated display data, historical, or
test-only. An exported helper without a production caller is not a currently
reachable path, but a temporary fail-closed change should still make callable
compatibility boundaries refuse before client I/O when practical. This prevents
a future caller from silently reactivating the unsafe action.

Also sweep every reachable UI and agent-facing instruction surface: queue lists,
detail pages, empty states, monitors, badges, parent/orchestrator prompts,
subagent descriptions, tool docstrings, loaded skills/playbooks, package
docstrings, schema comments, recovery-module comments, and generated production
catalogs/manifests. Removing an approval button is insufficient when another
screen, generated snapshot, parent prompt, protocol, adapter, or state transition
still describes approval as submission. Resume/finalizer contracts should accept
only supported decisions at both type and runtime boundaries; disabled decisions
must not enter legacy in-progress/tracking states.

When production serves generated skill/catalog data, update the source first,
then regenerate or deterministically patch only the matching record. Verify body,
byte count, content hash, and graph metadata against normalized source, and add a
production-path regression test. Probe malformed and extra fields with the
installed validator version, including whitespace-only required notes; do not
assume unknown fields are stripped or rejected. Distinguish necessary read-only
authorization lookups from mutating or external side effects in the review report.
See `references/disabled-side-effect-review.md` for the full audit and concurrency
checklist and `references/fail-closed-surface-closure.md` for transitive guidance,
compatibility-boundary, generated-artifact, and remediation checks.

## Review and Publication Gate

Before calling the change merge-ready:

1. Pin the requested base and head to immutable commit SHAs before review; use those SHAs for every diff. Re-check symbolic HEAD and working-tree status after long tests so concurrent branch advances or unrelated edits do not silently widen the attested range.
2. Run focused tests, migration validation, type/lint checks, full suite, build, and diff checks. When native PostgreSQL is unavailable, a PostgreSQL-compatible embedded engine can exercise migration SQL, backfill, check constraints, and foreign keys—but label this accurately and do not treat it as evidence for native locking, rollback, or concurrent-worker behavior.
3. Exercise the real surrounding runtime path with a controlled external stand-in when live credentials are unavailable; clearly disclose the stand-in.
4. Request an independent security/concurrency review after green validation.
5. If review finds blockers, keep the PR draft, document them, add new RED tests, fix, and re-review.
6. Never equate green existing tests with merge readiness after a reviewer identifies uncovered state or crash paths.

### Validation environment hygiene

Run each quality gate in the environment its suite expects. Long-lived shells can retain build-only variables and silently change test selection—for example, a placeholder database URL may turn normally skipped integration tests into live database tests. Likewise, an inherited `PYTHONPATH` can make a project-managed virtual environment import incompatible global packages.

- Prefer a fresh or explicitly sanitized environment for each canonical gate.
- Before unit suites, unset build-only database/integration selectors unless the run intentionally provisions the real service.
- Before project-managed Python tests, remove inherited module-path overrides unless the repository requires them.
- Treat an environment-contaminated nonzero run as failed, explain why it selected the wrong mode, and rerun the canonical command cleanly; never retroactively label the first run green.
- Record both the failed command and the clean rerun when publishing evidence.

See `references/validation-environment-isolation.md` for reusable shell patterns and reporting language.

If the branch advances during review, keep findings tied to the original immutable
range and state exactly which commit the tests exercised. A test-only follow-up
does not retroactively mean the original range had that coverage. See
`references/immutable-review-scope.md`.

## Common Pitfalls

- Treating `null` as impossible while the UI treats it as staged
- Trusting an LLM/frontmatter external ID because the user still clicks Approve
- Uniqueness per finding instead of per external intent/principal
- Claiming before credentials/preflight, creating unrejectable limbo
- Finalizing with a null external result ID
- Writing audit after terminal commit
- Calling resume/webhook/email directly with no durable marker
- Assuming an outbox consumer is idempotent without analyzing its acknowledgement crash window
- Keeping an unsafe partial PR because its focused tests pass
- Correcting a source skill/prompt while leaving the generated production catalog or parent orchestrator guidance stale
- Calling a dormant compatibility helper "safe" solely because no current caller reaches it, instead of making the boundary fail before I/O
- Running a broad formatter on legacy files, then restoring whole files and accidentally discarding already-validated semantic fixes
- Citing tests from an intermediate worktree after reset/restore changed the code under test
- Checking authorization only after loading a tenant resource, creating a 404/403 existence oracle
- Disabling a decision in the web wrapper while the interrupt, worker, or direct-resume boundary still accepts it
- Resetting legacy external or ambiguous states to a fresh/re-hunt state during generic startup recovery
- Overwriting `pending`, `confirming`, and `ambiguous` with one provenance-free `claimed` state, then recovering every expired claim to `pending`; this converts reconciliation-only work back into create-capable work
- Calling a destination "pinned" because a mutable current-thread pointer was copied once, even though immutable producing-thread provenance exists on the resource
- Discovering an interrupt before create but persisting its ID only after the response; the fenced transition into `creating` must durably carry the exact interrupt ID
- Removing a dormant side-effect implementation while preserving its dead dependency island and obsolete tests

## Reference

See `references/external-side-effect-workflow.md` for a compact transition template, review checklist, and reproduction patterns derived from a real approval/submission workflow.

See `references/fail-closed-surface-closure.md` when disabling an unsafe external action across routes, agent guidance, generated manifests, compatibility APIs, and state transitions.

See `references/trusted-draft-binding-and-resume-outbox.md` for endpoint-specific credential identity, canonical interrupt/draft binding, dispatch ambiguity, metadata-reconciled graph resume, multi-item board derivation, and the at-most-once-versus-liveness tradeoff when neither provider exposes enforceable idempotency.

See `references/fail-closed-review-gaps.md` for tenant-safe resource lookup, deepest-boundary decision validation, legacy startup-state preservation, dead dependency cleanup, and metadata-reconciled durable resume delivery.

See `references/split-api-credential-identities.md` for providers whose REST and GraphQL surfaces share a secret but require distinct account-handle and token-identifier usernames, including conservative legacy migration and deployment wiring.

See `references/conservative-outbox-remote-create.md` for the claim/creating/ambiguous/confirming state protocol required when a remote create lacks enforceable idempotency, including exact interrupt pinning, metadata recovery, exhaustive reconciliation, completion confirmation, and migration-test boundaries.

See `references/service-authentication-boundary-tdd.md` for fail-closed daemon startup, stable per-stack secret provisioning, daemon/client propagation, readiness wiring, request-level verification, and honest fallback evidence when container tooling is unavailable.

