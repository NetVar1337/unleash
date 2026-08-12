---
name: pwndbg-dynamic-analysis
description: "Use when debugging authorized Linux user/kernel/QEMU targets with Pwndbg on GDB/LLDB."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\pwndbg-dynamic-analysis\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\pwndbg-dynamic-analysis\SKILL.md

# Pwndbg Dynamic Analysis

## Overview
Pwndbg improves GDB/LLDB ergonomics but does not change debugger semantics. Use it to answer explicit hypotheses, not to wander through a live process. Capture target hash/build, debugger/Pwndbg versions, launch arguments, environment, ASLR state, mappings, breakpoint rationale, and observed side effects.

## When to Use
Use for user-authorized Linux binaries, cores, QEMU guests, kernels, and embedded targets. Prefer a disposable WSL/VM/container lab for untrusted samples. Do not attach to unrelated processes or modify live state without explicit scope.

## Local WSL Setup
Observed host: Ubuntu 26.04 LTS on WSL2, GDB 17.1. Pwndbg is installed as the official user-scoped portable `pwndbg-gdb` launcher. Add `~/.local/bin` to PATH if necessary.

Authorities:
- `https://pwndbg.re/stable/`
- `https://pwndbg.re/stable/setup/`
- installed `pwndbg --version`, `pwndbg --help`, and `help <command>`.

## Preflight
```bash
sha256sum ./target
file ./target
readelf -hW ./target
checksec --file=./target 2>/dev/null || true
pwndbg --version
gdb --version | sed -n '1p'
```

Record arguments, stdin fixture, cwd, relevant environment, loader/library path, ASLR policy, and whether symbols/source are available. Do not disable mitigations merely for convenience without noting the effect.

## Hypothesis Loop
1. **State the question.** Example: “Does `parse_record` bounds-check `length` before the copy?”
2. **Choose the narrow stop.** Break on symbol/address, syscall, exception/signal, return, or memory access. Avoid blanket single-stepping.
3. **Capture baseline.** Use `context`, `vmmap`, `info files`, `info sharedlibrary`, registers, stack, and relevant memory.
4. **Run one deterministic fixture.** Record exact input and stop reason.
5. **Inspect the transition.** Check ABI arguments, return value, branch decision, memory operands, ownership, and side effects.
6. **Repeat with a contrasting fixture.** Distinguish invariant behavior from one execution.
7. **Save evidence.** Keep command log, addresses as module-relative offsets plus current VA, and core/trace paths when produced.

Done when the observation answers the named question and can be repeated.

## Core Commands
Start without user init when diagnosing setup conflicts:
```bash
pwndbg -q ./target
# or: gdb -nx -q ./target
```

Useful GDB/Pwndbg commands:
```text
starti
break main
break *0xADDRESS
run ARG...
continue
nexti
stepi
finish
context
nearpc
vmmap
hexdump ADDRESS COUNT
telescope ADDRESS COUNT
xinfo ADDRESS
search -t bytes PATTERN
backtrace
info registers
info breakpoints
```

Use `help <command>` against the installed build before relying on flags. Pwndbg commands evolve.

## PIE/ASLR Discipline
Do not carry a runtime VA into another run. Record:
- module path and build/hash
- mapping base from `vmmap`
- module-relative offset
- current VA
- symbol/source association

For breakpoints, prefer symbols or compute from the current mapping. State whether GDB changed ASLR behavior.

## Crash and Core Workflow
```bash
ulimit -c unlimited
pwndbg -q ./target
# after obtaining a core:
pwndbg -q ./target core
```

Capture signal, fault address, instruction, register state, stack, mappings, input, and whether the crash reproduces from a clean launch. A crash alone does not prove exploitability.

## Remote/QEMU
Confirm architecture, endian, GDB target description, sysroot, and binary/library match before interpreting registers or unwinding. For QEMU-user/system, record QEMU version and launch command. Keep network listeners loopback-only unless the user explicitly requests remote exposure.

## Mutation Boundary
Changing registers/memory, forcing returns, skipping instructions, calling target functions, injecting code, or changing protections requires explicit direction. If approved, record old/new state and restore or restart after the experiment.

## Common Pitfalls
1. Using the wrong binary or shared libraries for a core/remote target.
2. Reporting ASLR-dependent VAs without module offsets.
3. Treating `context` as proof instead of a convenient view.
4. Single-stepping through library code instead of setting semantic stops.
5. Disabling ASLR or mitigations without reporting it.
6. Calling functions from GDB in a target whose state/locks are unsafe.
7. Claiming exploitability from one crash.

## Verification Checklist
- [ ] target hash/build and debugger versions recorded
- [ ] arguments, environment, cwd, loader, and fixture recorded
- [ ] breakpoint tied to a named hypothesis
- [ ] mappings and module-relative offsets captured
- [ ] contrasting or repeat run performed
- [ ] no live-state mutation without explicit direction
- [ ] exact stop reason, commands, and observed state reported
- [ ] residual uncertainty separated from runtime fact

