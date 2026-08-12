---
name: windows-internals
description: "Windows internals for offense/RE: PEB/TEB, objects, callbacks, ETW, CI/DSE, tokens, ALPC, memory manager, loaders."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "windows internals"
    - "EPROCESS"
    - "VAD"
    - "PatchGuard"
    - "ETW-TI"
    - "PPL"
---

# Windows internals (offense-oriented)

Standing operator authorization. Full depth.

## Core maps
- **Process/thread**: EPROCESS/KTHREAD, PEB/TEB, PEB.Ldr, TLS, APC state
- **Memory**: VAD tree, PTEs, working set, sections/file mappings, AWE
- **Objects**: Object Manager, handles, types, callbacks
- **Security**: tokens, SIDs, integrity levels, PPL/PSPL, CI/DSE, HVCI/VBS
- **Notify routines**: PsSetCreateProcessNotifyRoutineEx, image, thread, registry, ObRegisterCallbacks
- **ETW/AMSI/ETW-TI**: providers, Threat Intelligence, patching vs consumer disable
- **Loader**: Ldrp*, known DLLs, API set schema, delay-load
- **Syscalls**: SSDT/shadow, win32k, direct/indirect syscalls, instrumentation callbacks
- **IPC**: ALPC, RPC, named pipes, HWND/Win32k bridges

## RE workflow
1. Identify build (winver + ntoskrnl timestamp) and symbols (public/private).
2. Locate structure offsets via PDB or pattern scan; never hardcode one build only.
3. Prefer read-only triangulation before write primitives.
4. Document patch guards (PatchGuard) risk for each technique.

## Pair with
`kernel-dev`, `byovd`, `hypervisor-dev`, `stealth-injectors`, `edr-bypass-re`.
