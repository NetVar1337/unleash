---
name: virtualization-deobfuscation
description: "Use to recover semantics from VMProtect/Themida/Tigress/custom VM bytecode."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\virtualization-deobfuscation\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\virtualization-deobfuscation\SKILL.md

# Virtualization Deobfuscation

## Overview
Virtualization replaces native semantics with an interpreter whose ISA, dispatch, state, encoding, and handlers may be randomized per build. The robust objective is not to assign folklore names to handlers. It is to recover a verified relation between protected inputs, state transitions, branches, memory effects, and outputs.

Back Engineering Labs’ current guidance explicitly treats its VMProtect 2 handler-identification project as legacy and brittle. Its May 9, 2026 Themida work advocates incremental lifting, stack/load-store propagation, constant propagation, DCE, and control-flow recovery with minimal VM-specific logic. Apply that principle across protectors.

## Authorization Boundary
Use for owned or explicitly authorized binaries, research challenges, and isolated malware analysis. Game binaries may be studied only offline and lawfully. Do not disable anti-cheat, instrument live multiplayer clients, bypass integrity/attestation, extract secrets, or create cheating capabilities.

Tracing target execution, emulation, process-memory access, rewriting, or replacing protected code requires explicit scope. Static analysis remains the default.

## Vocabulary Without Assumptions
Track observed roles, not fixed registers:
- **VM entry/exit:** native-to-virtual and virtual-to-native boundaries
- **virtual context:** state representing virtual registers/flags/stack
- **VIP:** virtual instruction position or equivalent cursor
- **VSP:** virtual stack position or equivalent state
- **dispatcher:** mechanism selecting the next semantic fragment
- **handler/semantic fragment:** code contributing a virtual operation
- **bytecode/operand stream:** encoded VM program and operands
- **transform:** encoding/decoding or rolling-key operation

A role can move across registers/memory, be split among blocks, be implicit, or be interlinked with other handlers. Assign it only from data-flow evidence.

## Choose the Recovery Product
1. **Behavioral formula:** input/output relation for a bounded pure region.
2. **Recovered CFG:** virtual blocks and branch conditions.
3. **Normalized IR:** lifted side effects suitable for analysis.
4. **Readable pseudocode:** semantically useful but not necessarily recompilable.
5. **Recompiled/native replacement:** highest validation burden and explicit mutation scope.

Done when the chosen product and equivalence boundary are written before building automation.

## Stage 1 — Establish VM Boundaries
1. Start from proven native anchors, calls, protected macros if source exists, or stable transitions from unpacking.
2. Identify state initialization, context save/restore, bytecode source, dispatcher loop, exits, and interactions with native APIs.
3. Record module/RVA, architecture, image version, calling convention, live inputs/outputs, memory regions, and exception behavior.
4. Compare at least two entries or builds before labeling a pattern protector-wide.

Done when a bounded VM invocation can be described as a state transformer with explicit inputs and outputs.

## Stage 2 — Recover State Roles by Invariants
Use value and data-flow invariants:
- a cursor advances/decodes from a bytecode-like region
- a stack/context pointer indexes repeated VM state accesses
- a rolling key feeds operand transforms
- dispatch targets depend on decoded opcode/state
- exits restore native ABI-visible state

Corroborate with dynamic traces or symbolic dependencies. Do not hardcode “VIP is RSI” or a handler catalog unless pinned to one hash/build and guarded by validation.

## Static Incremental-Lifting Pipeline
Use when code and VM stream are sufficiently available:
1. Decode native instructions from proven boundaries.
2. Lift exact side effects into an IR preserving bit widths, flags, memory, and partial registers.
3. Model the VM stack/context as memory first; do not prematurely invent virtual registers.
4. Propagate stores to loads where aliasing is proven.
5. Propagate constants from immutable sections and decoded operand streams.
6. Simplify address arithmetic and normalize register fragments/flags.
7. Eliminate dead computations relative to selected observable outputs.
8. Recover indirect targets and split blocks only when target sets are justified.
9. Iterate propagation, simplification, DCE, and CFG recovery to a fixed point.
10. Introduce VM-specific semantics only for operations the generic passes cannot expose.

Completion criterion: each iteration reduces a named metric—live IR operations, unresolved indirect targets, unknown memory dependencies, or dispatcher cycles—without failing equivalence fixtures.

Do not lower to LLVM merely because it is available. Back Engineering Labs reports that one-to-one lifting/lowering can produce cleaner results than forcing all semantics through LLVM. Choose IR based on precise semantics and validation tooling.

## Dynamic Trace/Symbolic Pipeline
Use when self-modification, encrypted operands, or opaque dispatch defeat static recovery:
1. Define exact VM entry/exit and symbolize only intended inputs.
2. Capture ordered instructions, code versions, register state needed for replay, memory reads/writes, branches, exceptions, and module mappings.
3. Replay with precise instruction semantics.
4. Taint from protected inputs and selected state; concretize VM machinery only when independence is proven.
5. Build formulas forward, then backward-slice from observable outputs and path conditions.
6. Simplify/synthesize expressions while preserving width, signedness, and undefined behavior.
7. Invert user-dependent path predicates to obtain contrasting inputs.
8. Merge traces at shared prefixes into guarded expressions/CFG.
9. Mark uncovered paths and symbolic-address limitations explicitly.

Jonathan Salwan’s VMProtect experiments demonstrate this path-oriented method for bounded pure functions, but also document limitations: path explosion, symbolic memory, loops/calls, and incomplete multi-path recovery. Do not generalize a successful pure-function result to arbitrary stateful code.

## Virtual Branch Recovery
Classify a transition using dependencies, not native `jcc` appearance:
- unconditional virtual transfer
- input-dependent conditional transfer
- dispatcher recurrence
- native call/return bridge
- exception-mediated transfer
- VM exit

For a candidate virtual condition, prove that changing a solver-derived or contrast input changes the virtual successor while keeping unrelated setup fixed. Record both traces and the formula slice.

## MBA and Data-Encoding Simplification
1. Preserve exact bit-vector widths and flag semantics.
2. Slice from selected output/state.
3. Constant-fold VM-only state.
4. Canonicalize commutative operations and extensions/truncations.
5. Use SMT equality checks or synthesis for bounded expressions.
6. Apply compiler optimization only after semantics are represented without accidental undefined behavior.
7. Validate candidate simplifications on solver counterexamples and concrete differential fixtures.

A shorter formula is not automatically the original source and may not preserve poison/undefined semantics in a compiler IR.

## Nested and Interlinked VMs
For nested virtualization or handlers that execute partial semantics:
- keep a call/entry stack of VM contexts
- tag operations by context and code version
- recover observable state at each boundary
- permit semantic fragments to span multiple dispatcher visits
- use slicing to group fragments by contribution rather than address adjacency

Stop seeking one-handler/one-opcode correspondence when evidence contradicts it.

## Equivalence Ladder
1. **Decoder:** native bytes decode identically in an independent decoder.
2. **Trace replay:** replay reproduces recorded registers/memory at checkpoints.
3. **Local semantic:** lifted fragment matches original over concrete and solver-generated states.
4. **Path:** recovered formula matches protected path outputs/side effects.
5. **CFG:** contrasting inputs reach corresponding recovered successors.
6. **Function:** differential tests cover boundaries, errors, loops, and memory effects.
7. **Translation:** if LLVM is used, apply translation validation such as Alive2 where its model applies, plus concrete testing.

Report timeouts, unsupported instructions, unconstrained memory, and uncovered paths. “Solver returned unknown” is not equivalence.

## Tool Routing
- IDA database and decompiler: `ida-pro-mcp`
- trace acquisition/normalization: `trace-guided-deobfuscation`
- opaque predicates/CFF/MBA: `binary-obfuscation-deconstruction`
- exact x86 semantics checks: `zydis-disassembly-engineering`
- terminal static work: `radare2-terminal-re`

Historical projects such as `backengineering/vmp2`, NoVmp/VTIL, and version-specific handler tables are architecture case studies, not universal current solutions.

## Required Report
Include target hash/build; protector/version confidence; VM entry/exit RVAs; observable state boundary; state-role evidence; static/dynamic method; IR semantics; simplification metrics; recovered branches/paths; unsupported cases; equivalence fixtures and solver results; artifact paths/hashes; and residual protection.

## Common Pitfalls
1. Hardcoding handler names/register roles from another build.
2. Treating every dispatcher target as one virtual instruction.
3. Losing partial-register, flag, or memory-alias semantics during lifting.
4. Concretizing input-dependent VM state.
5. Claiming whole-function recovery from one trace.
6. Ignoring code versions in self-modifying handlers.
7. Trusting prettier LLVM/pseudocode without translation validation.
8. Recompiling/reinjecting before the recovered model is independently equivalent.

## Verification Checklist
- [ ] authorization and offline boundary recorded
- [ ] VM invocation inputs/outputs and recovery product defined
- [ ] state roles proven by invariants/data flow
- [ ] lifting preserves widths, flags, memory, and code versions
- [ ] VM-specific rules minimized and build-scoped
- [ ] user-dependent branches tested with contrasting inputs
- [ ] path/memory/loop limitations explicit
- [ ] recovered semantics differentially or formally checked
- [ ] no live anti-cheat/service bypass or unrequested rewriting occurred

