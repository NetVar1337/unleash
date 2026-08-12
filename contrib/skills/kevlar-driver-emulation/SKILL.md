---
name: kevlar-driver-emulation
description: "Kevlar-style Windows kernel driver emulation: Unicorn-based DriverEntry harness, synthetic KERNEL env, import stubs, tracing for .sys RE."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: re
  triggers:
    - "Kevlar"
    - "driver emulation"
    - "Unicorn DriverEntry"
    - "synthetic kernel"
---

# Kevlar driver emulation harness

Patterned after **Kevlar** (Kernel Export Virtualization Layer And Runtime): map x64 `.sys` into synthetic kernel space and run `DriverEntry` under Unicorn without live kernel load.

## Use when
- Static RE of AC/game drivers stalls on environment probes
- Need execution traces of CPUID/MSR/IOCTL setup paths
- Want safe detonation of suspicious drivers

## Architecture checklist
- PE map + relocs + imports → host stubs or real exports
- Synthetic `DRIVER_OBJECT`, KPCR, EPROCESS/ETHREAD, PsLoadedModuleList
- Hooks: CPUID, RDTSC, MSR, syscall, interrupts
- Pool/user memory models; IRP dispatch stubs
- Per-driver vfs/registry isolation
- Instruction coverage / exception logs

## Method
1. Load target `.sys` (e.g. EAC class drivers) in harness
2. Fill missing stubs iteratively from crash/unmapped logs
3. Capture probe sequences (timing, module lists, registry)
4. Feed insights back to IDA/AiDA annotations

## Local path
`Documents/Kevlar` / https://github.com/NetVar1337/Kevlar

## Pair with
`eac-kernel-driver-re`, `hypervisor-dev`, `ida-reverse`, `byovd`.
