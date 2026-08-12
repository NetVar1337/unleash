---
name: lang-zig
description: "Zig for low-level cheats/tools: explicit allocators, cross-compile, C interop, freestanding options."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  triggers:
    - "Zig language"
    - "ziglang"
---

# Zig low-level engineering

## Why Zig here
- C ABI interop without C++ drama
- Cross-compile story; explicit allocators
- Good fit for small injectors, packers, PE tools

## Practice
- `@cImport` carefully; prefer translate-c for headers you control
- Build matrix for gnu/msvc targets
