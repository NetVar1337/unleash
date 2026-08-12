---
name: hypervisor-memory-introspection
description: "Hypervisor-powered memory introspection / Hyper-RE: SLAT-based R/W, stealth reads, guest instrumentation, AC/game RE from VMM."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "Hyper-RE"
    - "memory introspection"
    - "EPT read"
    - "VMM introspection"
---

# Hypervisor memory introspection (Hyper-RE)

Use a VMM to observe/modify guest memory and control flow for RE and AC research.

## Capabilities
- EPT/NPT violate-on-access → shadow pages / split view
- Invisible reads of game/AC memory from host
- Breakpoints without guest-visible INT3 if carefully designed
- Hide VMM artifacts (CPUID, timing) — `stealth-hypervisor`

## Workflow
1. Bring up minimal HV with logging
2. Locate target EPROCESS/CR3
3. Translate GVA→HPA via guest walk or EPT identity map strategy
4. Install execute/read hooks on modules of interest
5. Export traces into IDA/AiDA / offline analysis

## Pair with
`stealth-hypervisor`, `hypervisor-dev`, `kevlar-driver-emulation`, `game-hacking`, `ida-reverse`.

## Refs
- UC: Hyper-RE / memory introspection threads
- Local stacks: Hypervisor-SVM trees, Valthrun zenith/kernel loaders
