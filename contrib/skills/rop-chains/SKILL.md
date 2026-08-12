---
name: rop-chains
description: "ROP/JOP chain construction: gadgets, stack pivots, Windows x64 calling, syscall stubs, retpoline/CFG constraints."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: exploit
  triggers:
    - "ROP chain"
    - "JOP"
    - "stack pivot"
    - "gadget"
---

# ROP / JOP chains

## Build order
1. Leak module base (or find non-ASLR)
2. Enumerate gadgets (`rp++`, ROPgadget, Ropper, Zydis-based)
3. Pivot stack if needed into controllable buffer
4. Set registers per ABI; call VirtualProtect/mprotect/system
5. Align stack; handle nopsled of `ret` for alignment

## Windows x64
- Args: RCX, RDX, R8, R9 + stack
- 0x20 shadow space before call
- Gadgets: `pop rcx; ret`, `mov [reg],reg`, `pushad` rare

## Linux x64
- Args: RDI, RSI, RDX, RCX, R8, R9
- `syscall` gadget with rax=execve

## Mitigations
- CFG: only valid call targets
- CET/shadow stack: prefer integrity bypass research or non-return-oriented paths
- retpoline/IBT: gadget quality drops — verify

## Pair with
`stack-buffer-overflow`, `exploit-dev`, `lang-assembly`.
