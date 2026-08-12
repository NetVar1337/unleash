---
name: binary-obfuscation-deconstruction
description: "Use when authorized binaries use opaque predicates, CFF, MBA, or self-modifying obfuscation."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\binary-obfuscation-deconstruction\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\binary-obfuscation-deconstruction\SKILL.md

# Binary Obfuscation Deconstruction

## Overview
Modern protectors compose transforms. A flattened function may also contain opaque predicates, encoded state, indirect dispatch, MBA expressions, bogus blocks, exceptions, and self-modifying code. Attack one semantic mechanism at a time and retain an equivalence check after every rewrite. The goal is a smaller verified model, not source-code aesthetics.

Tigress is a useful controlled laboratory because it documents transformations and deliberately generates diverse variants. Do not treat one Tigress/OLLVM pattern as a universal detector for commercial protectors.

## Authorization Boundary
Use only on owned or authorized binaries, CTFs, research corpora, or isolated malware. Online game artifacts are offline research targets only. Do not develop anti-cheat evasion, live-client instrumentation, integrity bypasses, cheat logic, credential extraction, or unauthorized proprietary-code redistribution.

Start read-only. File/process rewriting, forced branches, target execution, or instrumentation requires explicit scope and rollback.

## Success Contract
Select the observable boundary:
- restored CFG and reachability
- simplified expressions/data flow
- recovered indirect targets
- readable analysis IR/pseudocode
- behaviorally equivalent native rewrite

Define outputs and side effects that must be preserved: return values, memory writes, calls, exceptions, flags where externally visible, termination, and ordering. Done when the boundary and comparison oracle are written.

## Transform Inventory
Before rewriting, label candidate mechanisms with evidence:
- dispatcher dominance and state variable: control-flow flattening
- solver-invariant branch: opaque predicate
- high operation density with small semantic slice: MBA/data encoding
- many indirect transfers: encoded targets, jump tables, return dispatch, or virtualization
- impossible/dead blocks: bogus control flow
- SEH/VEH/signals: exception-driven transfer
- executable writes/version changes: self-modifying code
- call/return imbalance: return-address or stack dispatch
- alias-heavy table state: encoded pointers or state machines

Do not label a mechanism from visual complexity alone.

## Common Analysis Substrate
1. Fix instruction boundaries, architecture mode, function bounds, and exception/unwind regions.
2. Lift exact semantics with bit widths, partial registers, flags, and memory effects.
3. Build CFG with unresolved edges represented explicitly, not dropped.
4. Track definitions/uses, dominators/postdominators, loops, and side-effecting operations.
5. Add dynamic edge/code-version evidence without replacing unseen edges with “dead.”
6. Maintain an original-to-normalized address map.

Completion criterion: every transformation step can identify which IR nodes/edges it changed and how to compare old/new semantics.

## Opaque Predicate Workflow
1. Slice backward from branch condition through flags, registers, and memory.
2. Separate input/environment dependencies from constants and protector-only state.
3. Simplify at exact width; preserve overflow, shifts, and undefined/unspecified behavior.
4. Query both condition and negation with an SMT solver under explicit preconditions.
5. If one side is unsatisfiable, retain the proof/query/model metadata and prune only that edge.
6. If both are satisfiable, use generated models as contrasting runtime fixtures.
7. If unknown/timeout, keep both edges and classify unresolved.

A branch observed one way in 100 traces is not opaque. Environment-dependent checks are not opaque merely because the lab is constant.

## Control-Flow Flattening Workflow
1. Identify candidate dispatcher header/latch, state definition, and block-to-dispatch backedges.
2. Compute how each case transforms dispatcher state.
3. Recover successor sets statically when expressions are solvable; supplement with contrasting traces.
4. Distinguish real state transitions from bogus cases and exception exits.
5. Rewire each real block to proven successors in a separate normalized CFG.
6. Preserve loops and join conditions; do not duplicate blocks blindly.
7. Run reachability/DCE only after successor recovery.
8. Compare path conditions and side effects between flattened and normalized CFGs.

Dispatcher state can be encoded, split, aliased, or computed from flags. Match semantics, not a `switch` pattern.

## Indirect Branch Recovery
For each indirect call/jump/return:
- slice the target expression
- determine table/base/index provenance
- apply relocations and module mappings
- solve bounded index/range constraints
- enumerate only justified executable targets
- confirm with xrefs, table layout, runtime edges, or exception metadata
- retain an unknown-target edge for incomplete sets

Never mark a recovered target set complete from one trace. Validate address-space and code version at execution time.

## MBA and Data-Encoding Workflow
1. Define exact input/output bit-vector widths and signedness.
2. Backward-slice from the observable result.
3. Remove computations proven independent of the result/path.
4. Constant-fold and normalize associative/commutative forms.
5. Use algebraic rewriting, SMT equivalence, or synthesis with an explicit grammar/cost.
6. Ask the solver for a counterexample to old == new under preconditions.
7. Differential-test edge values: zero, all-ones, sign boundaries, carries, shift boundaries, and random fixtures.

A synthesized short expression is an equivalent candidate, not proof of original source intent. Preserve memory, flags, and exceptional behavior if they are observable.

## Bogus Blocks and Dead Code
Remove a block only when at least one holds:
- predecessor condition is proven unsatisfiable
- no executable predecessor exists in a complete recovered CFG
- all effects are proven dead relative to the success contract

Dynamic non-observation is supporting evidence, not proof. Watch for asynchronous callbacks, exceptions, indirect entries, exports, and computed jumps.

## Exception-Driven Control Flow
Build an explicit model of:
- handler registration and ordering
- protected ranges and unwind metadata
- exception kind and faulting instruction
- handler edits to context/IP/SP
- continuation target and side effects

Treat exceptions as CFG edges. Do not “fix” a faulting instruction until proving it is not deliberate dispatch.

## Self-Modifying and JIT Code
1. Record write address/range, writer, old/new bytes, protection, and execution timestamp.
2. Assign monotonically increasing code versions per page/range.
3. Decode traces against bytes valid at that timestamp.
4. Build CFG nodes keyed by `(module, RVA, code_version)`.
5. Separate unpack-once, polymorphic regeneration, and input-dependent generation.

A single static dump cannot represent multiple executed versions. Preserve versioned artifacts and hashes.

## Anti-Disassembly and Overlapping Code
Decode from proven control-flow entries using exact bytes and mode. Validate fall-through and targets independently. Represent overlapping instructions as separate paths rather than forcing one linear sweep. Cross-check critical boundaries with Zydis and one independent decoder.

## Iterative Pass Order
A practical fixed-point loop:
1. exact lift and constant propagation
2. stack/load-store propagation with alias checks
3. opaque-predicate resolution
4. indirect-target/dispatcher recovery
5. CFG simplification
6. backward slicing and DCE
7. MBA/data-expression simplification
8. repeat until metrics stabilize

Record per-pass metrics: blocks, edges, unresolved targets, live IR operations, solver outcomes, and test mismatches. Stop if a pass increases uncertainty or breaks fixtures.

## Validation
- structural CFG consistency and target validity
- SMT counterexample search for local rewrites
- differential execution over controlled fixtures
- path/edge comparison for recovered branches
- memory/call/exception trace comparison for side-effecting regions
- translation validation if lowering through LLVM and the model applies

Test both positive and negative/error behavior. Timeouts and unsupported semantics remain explicit unknowns.

## Reporting
Include target hash/build, transform inventory, evidence for each label, observable boundary, original/normalized CFG metrics, solver queries/results, dynamic coverage, code versions, address map, rewritten artifacts/hashes, differential tests, and unresolved edges/dependencies.

## Common Pitfalls
1. Treating “never observed” as unreachable.
2. Solving predicates without environmental preconditions.
3. Losing bit widths, flags, partial registers, or exception effects.
4. Removing the dispatcher before recovering every successor.
5. Declaring a finite indirect-target set from one run.
6. Simplifying MBA with unbounded integer algebra instead of bit vectors.
7. Decoding self-modifying code against stale bytes.
8. Rewriting the original binary before local equivalence passes.

## Verification Checklist
- [ ] authorization and observable boundary recorded
- [ ] exact instruction/IR semantics established
- [ ] each transform classified from evidence
- [ ] unknown edges and code versions retained
- [ ] predicates checked in both directions
- [ ] dispatcher successors and indirect targets justified
- [ ] MBA rewrites counterexample-checked and differentially tested
- [ ] pass metrics improve without fixture regressions
- [ ] no live anti-cheat/service bypass or unrequested mutation occurred

