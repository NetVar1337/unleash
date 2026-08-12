---
name: heap-overflow
description: "Heap buffer overflow: adjacent chunk smash, metadata overwrite, freelist poison setup, Windows/Linux allocator notes."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: exploit
  triggers:
    - "heap overflow"
    - "heap buffer overflow"
    - "chunk smash"
    - "LFH overflow"
---

# Heap buffer overflow

## Identify
- Copy/loop past heap buffer into next chunk/object
- Wrong capacity after realloc shrink
- Integer wrap → small alloc + big copy (pair `integer-overflow`)

## What you hit
- **Inline object fields** (vtables, function ptrs, length, pointers) — best case
- **Allocator metadata** (glibc chunks, LFH/Backend, segment heaps)
- **Guard pages** (if present) — need precise sizes

## Glibc (ptmalloc) themes
- Overwrite next chunk size/prev_size → fake free / consolidate tricks
- Modern tcache/safe-linking constraints — poison with obfuscated ptrs
- Leak via unsorted bin / fd bk before write-what-where

## Windows heap themes
- NT heap LFH vs backend vs segment heap (Win10+)
- Encode freelist pointers; variable encoding keys
- Prefer **typed object corruption** over raw freelist when LFH randomized
- Look for useful adjacent classes (C++ objects with vptr)

## Method
1. Bucket size: control alloc sizes to land victim adjacent
2. Spray/groom order; avoid free noise
3. Overflow content: padding + precise field/metadata rewrite
4. Trigger virtual call / free / unlink of corrupted object

## Pair with
`heap-exploitation`, `use-after-free`, `integer-overflow`, `exploit-dev`.
