---
name: x64dbg-anti-debugger
description: "Anti-debugger techniques and bypass research in x64dbg workflows: PEB flags, NtQuery, object hides, timing, TLS, self-debug."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  triggers:
    - "anti-debug"
    - "x64dbg"
    - "BeingDebugged"
    - "ScyllaHide"
---

# Anti-debugger (x64dbg workflow)

## Common checks
- PEB `BeingDebugged`, NtGlobalFlag, heap flags
- `NtQueryInformationProcess` (DebugPort, DebugObjectHandle, DebugFlags)
- `NtQuerySystemInformation` / handle table scans
- Parent process / window class heuristics
- RDTSC/QPC timing, trap flag, hardware BP detection
- TLS callbacks before entry

## RE procedure
1. Break early (system BP / TLS)
2. Patch or hide debug object as appropriate for lab
3. Prefer ScyllaHide-class plugins carefully — note detection of common hides
4. Log which check fired (don't blind-NOP everything)

## Pair with
`debugger`, `ida-reverse`, `anti-cheat-bypass`, `windows-internals`.

## Refs
- UC: x64dbg anti-debugger threads
