---
name: integer-overflow
description: "Integer overflow/underflow bugs: width conversion, mul/add wrap, size calc to alloc/copy, signedness, exploits to heap/stack corruption."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: exploit
  triggers:
    - "integer overflow"
    - "integer underflow"
    - "wraparound"
    - "size_t mul"
    - "signedness bug"
---

# Integer overflow & underflow

## Bug patterns
1. **Add/mul wrap** before `malloc(n * size)` → small alloc, large copy
2. **Truncation** `size_t` → `uint32`/`uint16` on boundary checks
3. **Signed/unsigned mix** — negative length passes `< max` then huge `size_t`
4. **Off-by-one** length including/excluding NUL
5. **Custom saturating math done wrong** (checked add that doesn't)

## Underflow specifics
- `len - hdr` when `len < hdr` → enormous unsigned
- Loop `for (i = n-1; i >= 0; i--)` with unsigned `i`
- Refcount `--` at zero → free-while-live / UAF setup

## From integer to memory corruption
```
bad_size = wrap(count * elem)
p = alloc(bad_size)          # small
copy(src, p, count * elem)   # large → heap overflow
```
Also: index OOB via wrapped index; stack alloc via VLAs/`alloca` with wrapped size.

## Hunting
- CodeQL/semgrep: mul then alloc; unchecked casts
- Diff size checks vs copy lengths
- Fuzz with maxed integers (`0xffffffff`, `1<<31`, `0`)

## Exploit notes
- Prefer stable heap layout after wrap-induced overflow
- Record exact widths (32 vs 64) per build
- On C++ `size_t`/`int` APIs (Win32 `int cb`) watch 2GB boundary

## Pair with
`heap-overflow`, `heap-exploitation`, `stack-buffer-overflow`, `c-review`.
