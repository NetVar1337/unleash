---
name: eac-ban-stack
description: "EAC/EasyAntiCheat ban-stack research: HWID serials, server-side trust, usermode telemetry, kernel driver signals, FN-class ban discussions."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "EAC ban"
    - "EasyAntiCheat"
    - "FN ban"
    - "EAC HWID"
---

# EAC ban-stack research

## Layers (typical)
1. **Usermode service/EOS process** — telemetry, module enumeration, heartbeats
2. **Kernel driver (`EasyAntiCheat.sys` class)** — callbacks, memory integrity, handle protection
3. **Server backend** — aggregates client evidence, HWID features, trust crypto
4. **Game account linkage** — bans stick to account + device features

## Research tasks
- Diff usermode vs kernel collection responsibilities
- Identify which IDs are local-only vs shipped remote
- Trace ban triggers: injection artifact, integrity fail, tamper, report spam
- Separate **detection** (instant kick) vs **silent flag** (delayed ban)

## Practical lab method
1. Baseline clean boot captures (procmon, ETW, driver list)
2. Introduce one variable at a time (mapper, overlay, debugger)
3. Record network destinations + payload sizes (not necessarily decrypt)
4. Correlate local artifacts with ban timing

## Pair with
`eac-kernel-driver-re`, `eac-usermode-telemetry-re`, `hwid-identifier-surfaces`, `tpm-attestation-research`, `anti-cheat-bypass`.

## Refs
- UC: serials & EAC/FN bans; complete AC bypass sources lists; EAC sys/EOS RE threads
