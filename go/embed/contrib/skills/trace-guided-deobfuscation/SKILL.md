---
name: trace-guided-deobfuscation
description: "Use when static analysis fails on self-modifying/virtualized code; acquire and slice traces."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\trace-guided-deobfuscation\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\trace-guided-deobfuscation\SKILL.md

# Trace-Guided Deobfuscation

## Overview
A trace is a measured path, not the program. Use traces to expose runtime code, compare inputs, recover state dependencies, and drive symbolic or IR analysis. Preserve enough execution context to replay semantics and distinguish application behavior from unpacker/VM machinery.

DynamoRIO `drmemtrace`, QBDI, Intel Pin, debugger scripting, emulators, and hardware branch tracing provide different events and perturbation levels. Select from required evidence; no backend is universally invisible or complete.

## Authorization Boundary
Use only on owned/authorized targets in an isolated lab. Game executables are offline research artifacts only. Do not trace live multiplayer clients, bypass anti-cheat or attestation, conceal instrumentation, extract secrets, automate gameplay, or connect modified clients to services.

Execution, instrumentation, attachment, snapshots, and memory reads require explicit user scope. Writes, forced control flow, injected code, or protection changes require separate explicit direction.

## Define the Trace Question
Write one question before acquisition:
- Where does unpacked code first execute?
- Which operations make output depend on selected inputs?
- What virtual branch condition selects two successors?
- Which code versions execute?
- Which edges differ between fixtures?
- Which memory cells carry VM/application state?

Then define the smallest event schema that answers it. Collecting every event can make analysis worse.

## Backend Selection
| Need | Prefer | Limitation to record |
|---|---|---|
| full instructions + memory | DBI such as DynamoRIO/QBDI/Pin | perturbation, unsupported instructions/syscalls |
| precise interactive state | debugger/Pwndbg/IDA | high overhead, anti-debug visibility |
| deterministic isolated semantics | emulator | incomplete OS/device/environment model |
| low-overhead branch history | Intel PT/hardware trace | decode gaps, limited data values |
| page writes/protections | debugger/DBI/OS events | races and observation timing |

QBDI documents callback-driven instruction/memory instrumentation across several architectures. DynamoRIO `drmemtrace` records retired user-mode instructions and memory accesses for offline analysis. Verify the exact installed version and event guarantees before relying on either.

## Reproducibility Manifest
Record:
- sample and dependency hashes
- OS/kernel/VM snapshot and architecture
- tool/backend versions and configuration
- command line, cwd, environment, locale, time source, CPU features
- input fixture and external files/network simulation
- ASLR policy and module load map
- process/thread selection and start/stop predicates
- trace schema/version, compression, and artifact hash

Done when another clean snapshot can rerun the same fixture and reach the same semantic boundary, even if absolute addresses differ.

## Canonical Event Schema
Use precision-preserving integer fields, preferably hex strings for 64-bit addresses:
```text
seq, process_id, thread_id, timestamp/order
module_id, module_hash, mapping_base, rva
code_version, instruction_bytes, length, architecture_mode
register reads/writes or checkpoints
memory address, size, value/hash, read/write
branch kind, taken, target
exception/signal and continuation
mapping/protection/write-to-code events
```

Do not rely on formatted disassembly as semantics. Store bytes and decode against the correct architecture/code version.

## Acquisition Workflow
1. Start from a clean snapshot and immutable sample copy.
2. Capture module mappings before target execution and on load/unload.
3. Arm tracing at a semantic boundary; avoid startup noise unless unpacking is the question.
4. Record thread creation and per-thread order.
5. Capture executable-page writes and protection transitions.
6. Stop at a deterministic exit boundary and flush atomically.
7. Hash trace and side artifacts; retain dropped-event/overflow counters.
8. Repeat the same fixture once to measure nondeterminism.

Done when event loss is known and stable behavior can be separated from scheduling/environment noise.

## Address and Module Normalization
Convert every executable event to `(module_hash, RVA, code_version)` where possible. Runtime/private/JIT regions get a stable region ID derived from allocation provenance and captured bytes. Keep current VA only as run-local evidence.

Track load/unload and reused addresses. Never merge events solely because their VAs match across time or runs.

## Self-Modifying Code
For each executable range:
1. Initialize version from mapped/captured bytes.
2. On a write, record writer, range, before/after bytes, and ordering.
3. Increment version before subsequent execution.
4. Decode an instruction using bytes valid for that sequence point.
5. Key CFG/coverage nodes by code version.

If instrumentation cannot order writes and execution reliably, classify the region as unresolved rather than decoding stale bytes.

## Trace Reduction
Apply lossless normalization first:
- remove backend/runtime modules by explicit module identity
- canonicalize addresses and thread IDs
- coalesce repeated identical dispatcher cycles only while retaining count and boundaries
- retain all side-effecting events and first/last state
- split into semantic epochs: loader, materialization, VM entry, dispatcher, native call, VM exit

Then use backward slicing from selected outputs, path conditions, memory effects, or API calls. A reduced trace must retain a map back to original event sequence numbers.

## Differential Tracing
Run baseline and contrasting fixtures with controlled environment:
1. Align at stable semantic anchors, not raw sequence number.
2. Compare edge sets, code versions, memory dependencies, calls, exceptions, and outputs.
3. Identify earliest divergence influenced by changed input.
4. Separate scheduler/address noise from semantic divergence.
5. Generate additional fixtures to test candidate conditions.

Use multiple runs per fixture before interpreting unstable differences.

## Taint and Symbolic Replay
1. Symbolize only intended input bytes/registers/memory.
2. Replay exact instruction bytes and concrete state checkpoints.
3. Taint registers, flags, and memory with explicit alias/overlap rules.
4. Concretize subexpressions only when proven independent of symbolic inputs and selected outputs.
5. Backward-slice from output/path predicate.
6. Query solver for a contrasting branch model.
7. Rerun the real target with the model and verify the predicted transition.

A satisfiable model is not runtime validation. An unsatisfiable result is scoped to modeled semantics and constraints. Report solver unknown/timeouts.

## Path Exploration
Maintain a path ledger:
- normalized path/edge hash
- input fixture/model
- reached VM/native boundaries
- path predicates and solver status
- output/side-effect digest
- uncovered candidate branches

Prioritize branches that depend on selected inputs and contribute to observables. Bound depth, loop iterations, time, solver queries, and trace size. Preserve incomplete coverage honestly; path explosion is a real limit.

## Trace-to-IR Pipeline
1. Decode bytes with current code version.
2. Lift exact instruction effects.
3. Replay and compare checkpoints.
4. Remove backend/VM-independent operations through proven slicing.
5. Merge traces at shared prefixes using validated branch predicates.
6. Simplify with bit-vector-aware passes.
7. Emit normalized IR/CFG with provenance to trace events and original RVAs.

Route VM semantics to `virtualization-deobfuscation` and transform-specific rewrites to `binary-obfuscation-deconstruction`.

## Validation
- trace repeatability and event-loss counters
- independent decoder check for critical bytes
- replay state equals recorded checkpoints
- contrasting input follows solver-predicted branch
- reduced trace preserves selected output/side-effect digest
- recovered formula/IR differentially matches multiple fixtures
- code-version and module normalization survive ASLR/reloads

## Artifact Layout
```text
case/
  manifest.json
  sample.sha256
  mappings.json
  trace.raw.zst
  trace.normalized.zst
  code_versions/
  fixtures/
  path_ledger.json
  findings.md
```

Use a documented schema version and atomic writes. Never include credentials or unrelated process memory.

## Common Pitfalls
1. Treating one path as full CFG coverage.
2. Comparing ASLR-dependent VAs across runs.
3. Ignoring thread interleaving and dropped events.
4. Decoding self-modifying instructions from final bytes.
5. Tainting everything until formulas become unusable.
6. Concretizing VM state that actually depends on user input.
7. Trusting a solver model without replaying it on the target.
8. Tracing production services or live anti-cheat-protected games.

## Verification Checklist
- [ ] authorization, isolation, and trace question recorded
- [ ] backend chosen from required evidence and limitations
- [ ] reproducibility manifest and hashes complete
- [ ] mappings, RVAs, threads, and code versions normalized
- [ ] event loss/nondeterminism measured
- [ ] reduction retains original sequence provenance
- [ ] symbolic models replayed on the target
- [ ] coverage and path-explosion limits explicit
- [ ] recovered IR/formulas differentially validated
- [ ] no live-service evasion or unrequested mutation occurred

