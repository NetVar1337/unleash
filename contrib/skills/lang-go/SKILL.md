---
name: lang-go
description: "Go for tools/C2/scanners/game tooling: modules, concurrency, unsafe, Windows syscalls, performance."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  triggers:
    - "golang"
    - "Go language"
---

# Go engineering (tools & Windows)

## Use when
Fast networking tools, parsers, control planes, cross-compile agents, pipeline glue.

## Notes for offense/tools
- `golang.org/x/sys/windows` for native APIs
- Avoid GC pauses in tight aim loops — don't put aimbot hotpath in Go
- Good for: offset dumpers, packet proxies, build bots, HV control sockets

## Style
- modules, minimal deps, table-driven tests, `context` deadlines
