---
name: lang-assembly
description: "Assembly for hooks/shellcode/syscalls: x64/x86 calling conventions, trampolines, PIC, reverse engineering."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  triggers:
    - "x64 assembly"
    - "shellcode"
    - "trampoline"
---

# Assembly (x86/x64) for hooks & shellcode

## Must-know
- Windows x64 calling convention (RCX,RDX,R8,R9, stack shadow)
- RIP-relative addressing; PIC shellcode constraints
- Trampoline construction; register clobber lists
- Syscall stubs; wow64 transitions

## Workflow
1. Write in asm or compile C then carve
2. Validate disassembly in target context
3. Test under DEP/CFG/CET constraints

## Pair with
`assembly-reversal-engineering`, `stealth-injectors`, `zydis-disassembly-engineering` if present.
