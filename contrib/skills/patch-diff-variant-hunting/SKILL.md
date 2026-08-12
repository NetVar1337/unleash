---
name: patch-diff-variant-hunting
description: "Use when mining historical CVEs, vendor patches, silent fixes, regressions, or incomplete mitigations for new vulnerability variants that still affect the latest stable release. Converts binary/source diffs into root-cause predicates, searches sibling implementations and exceptional paths, and applies a strict novelty gate so known bugs are not misreported as zero-days."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\patch-diff-variant-hunting\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\patch-diff-variant-hunting\SKILL.md

# Patch-Diff Variant Hunting

## Objective

Use known fixes as maps to unreviewed code. The deliverable is not “old PoC still works” unless the fix genuinely regressed; it is a new latest-stable instance of the same broken security invariant, an incomplete fix, or a fix bypass in a distinct reachable path.

## Candidate Selection

Prefer historical bugs with:

- complex trust-boundary logic rather than one obvious bounds check;
- multiple parsers, protocol versions, platform ports, or service modes;
- duplicated helpers or generated code;
- fixes added only at callers or only after one normalization step;
- emergency patches, defense-in-depth language, or sparse root-cause detail;
- later refactors, rollbacks, vendor forks, or component rewrites;
- high-value products matching the current-version and deployment gate.

Record CVE/advisory, fixed versions, public PoCs, researcher writeups, patch commits, and exact latest stable version before analysis.

## Phase 1: Reconstruct the Original Invariant

Do not begin with the old payload. State:

```text
Untrusted actor controls <field/object/state>.
Trusted component assumes <invariant>.
Missing/late/incorrect check permits <primitive>.
Primitive reaches <sink> under <authority>.
```

Separate:

- trigger syntax;
- vulnerable semantic condition;
- exploit primitive;
- final impact.

Completion criterion: the root cause can generate searches beyond the original function name.

## Phase 2: Diff the Fix

For source patches, inspect full function, callers, tests, and adjacent commits—not only changed lines.

For binary-only patches:

1. obtain same-channel pre/post binaries and symbols where available;
2. hash and record package provenance;
3. match functions using symbols, strings, CFG, constants, and call neighborhoods;
4. classify changed basic blocks and data structures;
5. recover the new predicate, ordering, or ownership rule;
6. identify unchanged callers and parallel implementations;
7. verify the apparent change dynamically with the old trigger.

Produce a fix model:

| Check | Location | Data representation | Effective token/state | Before/after side effect | Failure action |
|---|---|---|---|---|---|

Completion criterion: explain what security property the patch attempts to enforce and where enforcement begins/ends.

## Phase 3: Generate Variant Axes

Search systematically:

- sibling callers omitted from the patch;
- alternate protocol/file-format versions;
- fast path, fallback, retry, recovery, offline, import/export, and compatibility modes;
- default stream versus ADS; local versus remote; live versus snapshot;
- Unicode/ANSI, normalized/raw, absolute/relative, object-name/path-name representations;
- client/server, desktop/Server, x86/x64/ARM, kernel/user, host/guest implementations;
- pre-auth/post-auth and impersonated/reverted token paths;
- parser size/count/offset relationships transformed in a different order;
- object lifecycle changes between the new check and final use;
- integer widths, signedness, truncation, alignment, and unit conversion;
- duplicated vendor forks and statically linked copies;
- cleanup/error paths that skip the new invariant.

Build a matrix; do not free-associate variants.

## Phase 4: Search by Predicate

Translate the fix into queries:

- CodeQL/Semgrep/Coccinelle for source patterns;
- compiler-aware grep for helper call sequences;
- IDA/Ghidra scripts for call-before-check or missing-callee patterns;
- binary feature matching for old/new basic-block shapes;
- runtime tracing for consumers that reach the sink without the new validation event.

Example abstractions:

```text
open(path) -> validate(path) -> reopen(path)       # identity not preserved
size = count * width without checked multiplication
authorize(token A) -> revert -> act(token B)
normalize(name) in one branch, raw name in sibling branch
validate leaf reparse tag but reopen through mutable parent
```

Completion criterion: every hit is tied to the root-cause predicate, not merely a shared API.

## Phase 5: Latest-Stable and Novelty Validation

For each candidate:

1. reproduce on exact latest stable bytes;
2. prove the old public issue is fixed in its original path;
3. prove the new path remains vulnerable;
4. compare root cause, trigger, component, and impact with public descriptions;
5. search whether the variant path has already been disclosed;
6. record why this is a distinct instance, incomplete fix, or regression.

If the original old PoC simply works unchanged, investigate regression provenance; do not assume novelty.

Completion criterion: `zero-day-target-eligibility` passes with a written distinction from the known issue.

## Phase 6: Minimize and Bisect

- minimize the trigger around the variant-specific condition;
- build a negative control exercising the fixed original path;
- bisect packages/commits to find introduction or regression;
- determine whether the variant predates, survives, or reappears after the original fix;
- quantify affected stable releases without losing latest-version proof;
- derive a regression test that covers all sibling paths.

## High-Value Fix Smells

- validation added after partial parsing;
- denylist of one path/name/tag rather than object identity validation;
- check performed before queueing asynchronous work;
- check under caller impersonation, side effect after revert;
- count capped but secondary count/stream/metadata unbounded;
- only one protocol opcode or content-type patched;
- security check copied rather than centralized;
- exception converted to generic error while state remains mutated;
- defense-in-depth patch with no corresponding tests;
- fix in UI/client while server endpoint remains reachable;
- recovery/offline implementation not updated with online component.

## Common Pitfalls

1. Running public PoCs without reconstructing root cause.
2. Calling a byte-level patch difference a security difference.
3. Ignoring servicing branches and independently updated components.
4. Searching only renamed functions rather than semantic predicates.
5. Treating an incomplete fix already discussed publicly as a new zero-day.
6. Failing to prove the original path is fixed and the sibling path is distinct.
7. Overfitting to compiler noise in binary diffs.
8. Skipping error, fallback, and recovery paths.

## Verification Checklist

- [ ] Original invariant and primitive reconstructed
- [ ] Pre/post artifacts provenance and hashes recorded
- [ ] Fix predicate and enforcement boundary recovered
- [ ] Variant matrix covers sibling and exceptional paths
- [ ] Semantic searches implemented
- [ ] Original public path shown fixed
- [ ] Candidate shown vulnerable on latest stable
- [ ] Novelty distinction documented
- [ ] Trigger minimized and negative control included
- [ ] Introduction/regression range established where possible
- [ ] Generalized regression test proposed

