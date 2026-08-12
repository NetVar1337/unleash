---
name: format-string-bug
description: "Format-string vulnerabilities: read/write primitives via %n/%s, GOT overwrite, modern compiler constraints."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: exploit
  triggers:
    - "format string"
    - "%n exploit"
    - "printf bug"
---

# Format-string bugs

## Find
- `printf(user)` / `fprintf(log, user)` / logging wrappers
- Missing format in `sprintf` family

## Primitives
- Leak stack/code via `%p`/`%s`
- Write via `%n`/`%hn`/`%hhn` (if still reachable)
- Direct parameter access `%N$p`

## Modern reality
- Many compilers warn/forbid; still appears in custom loggers and embedded
- Prefer leak → ROP rather than heavy `%n` on hardened hosts

## Pair with
`stack-buffer-overflow`, `exploit-dev`.
