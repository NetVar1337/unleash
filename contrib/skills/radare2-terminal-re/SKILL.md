---
name: radare2-terminal-re
description: "Use for terminal-first r2 static/dynamic analysis with JSON-first automation."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  upstream: C:\Users\Admin\.agents\skills\radare2-terminal-re\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\radare2-terminal-re\SKILL.md

# Radare2 Terminal Reverse Engineering

## Overview
Radare2 is a framework, not a single command. Use `rabin2` for low-cost format triage, `rasm2` for focused assembly/disassembly, `r2` for stateful analysis, and JSON commands or r2pipe for automation. Start read-only; `-w` and write commands require explicit user direction.

## When to Use
Use for authorized ELF, PE, Mach-O, firmware, bytecode, cores, and raw blobs when terminal reproducibility matters. Use IDA for richer interactive decompilation and Pwndbg for runtime hypotheses.

## Local WSL Setup
Observed environment: Ubuntu 26.04 LTS under WSL2. The source-pinned install target is radare2 `6.1.8` under `/usr/local`; the prior Ubuntu package was `6.0.7`. Official authority: `https://book.rada.re/` and `https://github.com/radareorg/radare2`.

Always report `r2 -v`; command behavior and analysis passes vary by version.

## Triage
Work on a copy and record SHA-256 first.
```bash
sha256sum sample
file sample
rabin2 -I sample
rabin2 -S sample
rabin2 -i sample
rabin2 -E sample
rabin2 -z sample
```

Prefer JSON for machine processing:
```bash
rabin2 -Ij sample
rabin2 -Sj sample
rabin2 -ij sample
rabin2 -Ej sample
rabin2 -zj sample
```

Done when format, architecture, endian, bits, entry points, mappings, sections, imports, exports, and notable strings are captured.

## Analysis Loop
Open read-only and avoid maximum analysis by reflex:
```bash
r2 -nn sample
```

Inside r2:
```text
e scr.color=0
ij
ie
is
izz
S
```

Choose the smallest analysis pass that answers the question:
- `aa`: symbols and basic analysis.
- `aaa`: broader function/xref/call analysis.
- `aaaa`: expensive extras; use only with a reason and timeout.

Then navigate:
```text
afl
s sym.main
pdf
pdr
agf
axf
 axt
```

Use `afl~filter`, `iz~filter`, and bounded print counts to control output. Verify function bounds and architecture settings before interpreting pseudocode or ESIL.

## Address Discipline
Record virtual address, module-relative offset, and file offset separately. Useful commands:
```text
?v $$
?P <va>
?v <file-offset>
iS
om
```

Do not assume VA equals file offset. For PIE/ASLR, report module-relative offsets and current mappings.

### PE mapped-image dumps
If the PE plugin follows stale `PointerToRawData` fields and shows header bytes, zero code, empty imports, or invalid certificates, first test whether the file is a mapped process image with `file offset == RVA`. Do not keep analyzing the wrong loader view.

For focused read-only disassembly of a confirmed x64 mapped image, bypass the PE loader and map the file at the captured base:
```bash
r2 -q -n -a x86 -b 64 -m 0x<loaded-base> -e scr.color=0 sample
```
Then seek to `loaded_base + RVA`. Keep PE-directory recovery in a memory-layout-aware parser; raw radare2 mapping is for bytes, disassembly, and bounded function work, not automatic import/certificate reconstruction. If the dependency is ARM64EC/ARM64X, inspect CHPE code ranges with a hybrid-aware PE tool and select architecture per range instead of forcing x64 globally.

## Automation
Prefer command JSON suffixes (`ij`, `aflj`, `pdfj`, `agfj`, `axtj`) and parse JSON rather than scraping colored tables.

One-shot pattern:
```bash
r2 -q -nn -c 'aaa;aflj;q' sample
```

For repeated logic, use r2pipe and retain the script with the report. Pin the radare2 version and cap result counts/timeouts. Done when the script can rerun from a clean copy and produce the same keys/counts.

## Focused Assembly with rasm2
```bash
rasm2 -a x86 -b 64 'mov rax, rbx'
printf '4889d8' | rasm2 -a x86 -b 64 -d -
rasm2 -L
```

Specify architecture, bits, endian, and syntax explicitly. Cross-check critical x86 decoding with Zydis when instruction boundaries or metadata matter.

## Dynamic and Write Boundary
Do not use `r2 -w`, `oo+`, `wx`, `wa`, `w*`, process attachment, or debugger write commands without explicit authorization. For approved patches:
1. Copy the input.
2. Record original bytes and file offset/VA mapping.
3. Assemble independently.
4. Patch the copy.
5. Re-read bytes and rerun format checks/tests.

## Common Pitfalls
1. Running `aaaa` on every target and mistaking volume for evidence.
2. Scraping human output when JSON exists.
3. Confusing seek address, VA, RVA, and file offset.
4. Trusting auto-created function boundaries without CFG/xref checks.
5. Opening with `-w` before preserving the original.
6. Using distro radare2 versions without checking security/version drift.

## Verification Checklist
- [ ] sample copy and SHA-256 recorded
- [ ] `r2 -v` recorded
- [ ] `rabin2` triage complete before deep analysis
- [ ] analysis pass chosen deliberately
- [ ] addresses reported with mapping context
- [ ] automation uses JSON and bounded output
- [ ] no write/debug mutation without explicit direction
- [ ] findings reproduced from a clean invocation

