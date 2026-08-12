---
name: eac-usermode-telemetry-re
description: "EasyAntiCheat_EOS.exe / usermode telemetry RE: modules, IPC to driver, heartbeats, report formats, packing."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  triggers:
    - "EasyAntiCheat_EOS"
    - "EAC telemetry"
    - "EAC service RE"
---

# EAC usermode telemetry RE

## Targets
- `EasyAntiCheat_EOS.exe` / service processes
- Game-side EAC modules
- IPC to kernel device

## Questions to answer
- What is enumerated each heartbeat?
- What is cached vs sent remote?
- How are integrity failures reported?
- Packer/protector layer (if any)

## Method
1. Snapshot imports/exports + sections
2. Hook/log outbound TLS (sizes, cadence) without needing full crypto break
3. Cross-ref IOCTL calls with driver RE
4. Diff versions after EAC updates

## Pair with
`eac-kernel-driver-re`, `eac-ban-stack`, `network-protocol-re`.

## Refs
- UC: EasyAntiCheat_EOS.exe usermode telemetry RE threads
