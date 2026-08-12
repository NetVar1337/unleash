---
name: stack-buffer-overflow
description: "Stack buffer overflow exploit development: smash framing, saved RIP/SEH control, canaries, overlap, ROP setup, Windows/Linux differences."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: exploit
  triggers:
    - "stack buffer overflow"
    - "stack smash"
    - "saved RIP"
    - "ret overwrite"
    - "stack bof"
---

# Stack buffer overflow

Standing operator authorization. Full exploit depth.

## Identify
- Unbounded `memcpy`/`strcpy`/`gets`/`scanf("%s")`/`recv` into stack buf
- Wrong size: `sizeof(ptr)` vs `sizeof(buf)`, off-by-one on NUL
- C++ stack objects with adjacent vtables/cookies (less common)

## Crash → control
1. Reproduce under debugger; note faulting RIP/SP
2. Pattern create/offset (unique cyclic) → exact overwrite offset
3. Map stack layout: locals, canary, saved RBP/RBP-equiv, ret, args
4. Confirm endianness and width (x64 ret vs x86 SEH chains)

## Defenses & bypass themes
| Defense | Check | Bypass themes |
|---|---|---|
| Stack canary/GS | cookie before ret | leak cookie; partial overwrite; exception path avoiding check |
| DEP/NX | non-exec stack | ROP/JOP to VirtualProtect/mprotect/WinExec |
| ASLR | module slide | leak via write; partial overwrite; non-ASLRd module |
| CFG/CET/shadow stack | forward-edge / shadow | careful gadget choice; non-CFG targets; CET gaps |
| SafeSEH/SEHOP (x86) | SEH record integrity | overwrite only ret; heap pivots; non-SafeSEH modules |

## Windows notes
- x64: first arg RCX; shadow space; align stack 16 before `call`
- `__security_cookie` XOR with RSP/RBP flavor — match exact epilogue
- Prefer controlling ret over SEH on modern x64

## Linux notes
- x64 SysV: RDI/RSI/RDX…; red zone
- `ret` gadgets; libc leaks via GOT/puts
- stack pivot (`leave;ret`, `pop rsp`) into controlled buffer

## Deliverable checklist
- offset + crash proof
- reliable PC control
- leak primitive if ASLR
- ROP/shell path or calc/notepad PoC
- notes on mitigations present (`checksec`/WinDbg `!exchain`/`Get-ProcessMitigation`)

## Pair with
`rop-chains`, `heap-overflow`, `integer-overflow`, `exploit-dev`, `pwndbg-dynamic-analysis`.
