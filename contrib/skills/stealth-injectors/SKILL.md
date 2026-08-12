---
name: stealth-injectors
description: "Stealthy usermode/kernel injection: manual map, thread hijack, APC, module stomping, hollow, mapper design, artifact hygiene."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "manual map"
    - "thread hijack"
    - "APC injection"
    - "module stomping"
    - "stealth injector"
---

# Stealthy injectors

## Goals
Execute foreign code in a target process/kernel with minimal artifacts versus EDR/AC.

## Usermode technique ladder
1. **CreateRemoteThread + LoadLibrary** — noisy baseline (know it to avoid)
2. **NtCreateThreadEx** + manual map
3. **Thread hijack** (suspend/context/RIP swap/resume)
4. **QueueUserAPC / special user APC** on alertable threads
5. **Module stomping / module overloading** (execute in legit module RX)
6. **Transacted/ghost / dual-mapping** variants
7. **Process hollowing / doppelgänging / herpaderping** (know detection cost)
8. **SetWindowsHookEx** (only if UI-thread delivery fits)
9. **Instrumentation callback / VEH** abuse for redirect

## Manual map checklist
- Map sections with correct protections; apply relocs; resolve imports (including API sets)
- TLS callbacks decision; exception directory; cookie
- Erase PE headers optional; fix or avoid module list visibility intentionally
- Prefer RW→RX transitions; avoid long RWX

## Kernel injection ladder
- APC to user thread from kernel
- Attach process + write + context
- Thread creation in target via NtCreateThreadEx from kernel
- Shared sections + user trigger

## OPSEC
- Call stacks, allocation stubs, RWX, cross-process handle rights, abnormal module ranges
- Clean handles; avoid known bad patterns in public GH gists
- Test under target AC/EDR with telemetry capture

## Pair with
`kernel-dev`, `game-hacking`, `windows-internals`, `hypervisor-dev`.
