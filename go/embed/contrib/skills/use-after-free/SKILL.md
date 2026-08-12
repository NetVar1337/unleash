---
name: use-after-free
description: "Use-after-free exploitation: dangling refs, reclaim/spray, type confusion, C++ vptr hijack, kernel pool UAF notes."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: exploit
  triggers:
    - "use after free"
    - "UAF"
    - "dangling pointer"
    - "double free"
---

# Use-after-free (UAF)

## Root patterns
- Explicit free/delete then use
- Callback after teardown
- Iterator into cleared container
- Refcount underflow → premature destroy (`integer-overflow` underflow)

## Exploit path
1. Stabilize free trigger
2. Identify dangling object type/size
3. Spray same-size controlled allocations
4. Corrupt virtual function table / function ptr / length+buffer
5. Call path that trusts dangling object

## C++ tips
- vptr first qword/qword on MSVC/Itanium layouts — confirm
- Fake vtable in controlled spray; ensure RX or pivot to stack ROP if DEP

## Kernel pool UAF (high level)
- Pool backend/lookaside differences
- Quota/process context
- Prefer data-only (token) when SMEP/SMAP/CIG

## Pair with
`heap-exploitation`, `kernel-dev`, `c-review`.
