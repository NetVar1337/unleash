---
name: assembly-reversal-engineering
description: "Use when writing assembly or analyzing authorized native binaries at ASM level."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: local:C:\Users\Admin\.agents\skills
---

> Bundled with Unleash skills pack. Upstream: local:C:\Users\Admin\.agents\skills

# Assembly and Reversal Engineering Skill

Use this skill for assembly development and authorized native-binary analysis. Begin static and read-only, preserve provenance, and separate observed facts from hypotheses.

## When to Use

- Writing or reviewing x86, x86-64, ARM, or AArch64 assembly.
- Investigating PE, ELF, Mach-O, COFF, object files, libraries, or crash dumps.
- Recovering calling conventions, control flow, data flow, structures, or compiler behavior.
- Comparing source-level intent with emitted machine code.

Do not use it to mutate a live target, patch a binary, bypass access controls, or change an IDB without explicit authorization and scope.

## Prerequisites

Use `terminal` to record:

- Target path and SHA-256.
- File format, architecture, endianness, image base, and build identifiers.
- Tool versions and exact analysis artifact.
- Whether symbols, source, PDB/DWARF, relocation data, or compiler metadata are available.

Preferred local stack on Windows includes LLVM object tools, Clang/LLDB, NASM/NDISASM, IDA/IDALib, Ghidra, x64dbg, Zydis, and Graphify where applicable. Availability must be checked live; never infer it from this document.

## How to Run

1. Use `terminal` to hash and identify the artifact.
2. Use `read_file` for project instructions, map files, linker scripts, and neighboring source.
3. Perform static triage before debugging or emulation.
4. Keep an evidence table with module, RVA/VA, bytes, interpretation, confidence, and validation method.
5. Escalate to dynamic analysis only when static evidence cannot settle the question.
6. Validate recovered semantics with an independent method.

## Quick Reference

### Assembly build loop

```bash
nasm -f win64 input.asm -o input.obj
clang input.obj -o output.exe
llvm-objdump -d --no-show-raw-insn output.exe
```

Select the object format and linker for the actual target. Confirm the ABI before writing prologues, stack frames, unwind metadata, calls, or SIMD code.

### Static triage

```bash
llvm-readobj --file-headers --sections --symbols <artifact>
llvm-objdump -d --source --demangle <artifact>
llvm-nm --demangle <artifact>
```

For raw shellcode or data, specify architecture, mode, and load address explicitly. A raw byte stream has no reliable metadata.

### Core ABI checks

| Boundary | Verify |
|---|---|
| Calls | argument registers/stack, shadow space, red zone, return registers |
| Stack | alignment at every call, local layout, unwindability |
| Registers | caller/callee-saved rules, vector-register state |
| Data | width, signedness, alignment, endianness, packing |
| Control flow | direct/indirect targets, tail calls, exception edges |
| Relocations | image-relative vs absolute addresses, GOT/IAT/PLT use |

## Procedure

### 1. Fix the coordinate system

Record module name, preferred image base, runtime base if known, RVA, VA, and file offset. Never mix these in notes. Completion requires every referenced address to state its coordinate type.

### 2. Establish function boundaries

Use symbols and unwind metadata first, then cross-references, prologue patterns, switch tables, and control-flow reachability. Compiler optimizations invalidate simplistic prologue matching.

### 3. Recover semantics bottom-up

Annotate inputs, outputs, clobbers, stack slots, globals, and call targets. Name entities only when evidence supports the role; keep provisional names visibly provisional.

### 4. Track data flow

Follow definitions and uses through registers, memory aliases, phi-like merges, and calls. For pointers, distinguish the address, pointee, and object lifetime. For virtual dispatch, identify object layout and vtable provenance before naming a class.

### 5. Form and test hypotheses

Each hypothesis must include supporting evidence, contradictory evidence, confidence, and a falsification test. Prefer cross-reference checks, a second disassembler, a small source-to-assembly reproduction, or constrained runtime observation.

### 6. Escalate carefully

Before dynamic debugging, specify process, build/hash, breakpoint location, and expected observation. IDB changes, binary patches, process-memory writes, or live-target mutation require explicit user authorization.

## Packer and obfuscation branch

Load the related packer, obfuscation, trace-guided, or virtualization skill only when evidence indicates that class of protection. Do not label unfamiliar optimized code as obfuscated. Record entropy, section anomalies, import behavior, control-flow regularity, and runtime transitions before choosing an unpacking strategy.

## Pitfalls

- Confusing RVA, VA, and file offset.
- Treating decompiler output as source truth.
- Guessing types from one use site.
- Ignoring calling-convention and unwind rules.
- Renaming or retyping an IDB during read-only reconnaissance.
- Assuming compiler-generated control flow is malicious obfuscation.
- Claiming a signature is reliable without uniqueness and boundary validation.
- Testing a different build than the one that was hashed.

## Verification

- [ ] Target build and SHA-256 are recorded.
- [ ] Architecture, ABI, format, image base, and tool versions are recorded.
- [ ] Every important address names its coordinate system.
- [ ] Function boundaries and call targets have evidence.
- [ ] Recovered types and names carry confidence.
- [ ] At least one independent validation method confirmed key semantics.
- [ ] Static/read-only scope was preserved unless broader mutation was explicitly authorized.
- [ ] Findings distinguish facts, hypotheses, and unresolved questions.
