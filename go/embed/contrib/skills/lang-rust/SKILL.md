---
name: lang-rust
description: "Rust for safe systems/game tools: unsafe boundaries, windows-rs, no_std driver experiments, perf."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  triggers:
    - "Rust"
    - "windows-rs"
---

# Rust systems engineering

## Prefer
- Strong type boundaries at FFI; minimize `unsafe` surface
- `windows-rs` / `windows-sys` for Win32
- Explicit features for external vs internal builds

## Game/RE tooling
- Excellent for dumpers, pattern scanners, packet crates
- Hot aim loops possible with care (no alloc in path)

## Pair with
`reverse-engineering`, `game-hacking`.
