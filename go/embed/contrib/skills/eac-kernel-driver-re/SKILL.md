---
name: eac-kernel-driver-re
description: "EasyAntiCheat.sys-class kernel driver reverse engineering: callbacks, device interfaces, integrity, communication, Rust/C++ RE notes."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  triggers:
    - "EasyAntiCheat.sys"
    - "EAC driver"
    - "kernel AC RE"
---

# EAC kernel driver RE

## Triage
- Version the `.sys` (file version, PDB path leftovers, authenticode)
- Imports: `Flt*`, `Ps*`, `Ob*`, `Cm*`, `Mm*`, `Io*`
- Strings: device names, registry, error codes

## High-value subsystems
1. Process/thread/image notify routines
2. Object callbacks (handle stripping)
3. Minifilter / file integrity
4. Device IOCTL ABI to usermode service
5. Memory scan / module validation workers
6. Timing and environment probes

## Workflow
- Static: IDA/Ghidra + AiDA assistance
- Dynamic: **prefer Kevlar/emulation or HV** before live load
- Document IOCTL codes and structures for usermode pairing

## Pair with
`eac-usermode-telemetry-re`, `kevlar-driver-emulation`, `kernel-callbacks`, `ida-reverse`.

## Refs
- UC: EasyAntiCheat.sys kernel driver RE (Rust notes threads)
