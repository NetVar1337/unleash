---
name: kernel-dev
description: "Kernel-level development workflow for stealthy drivers, manual mappers, injectors, and rootkits. Covers WDM/KMDF drivers, shellcode, APC injection, thread hijacking, VAD manipulation, callback unhooking, and DKOM. Invoke with /kernel-dev or when the task involves kernel-mode code."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: local:C:\Users\Admin\.claude\skills
---

> Bundled with Unleash skills pack. Upstream: local:C:\Users\Admin\.claude\skills

# Kernel development workflow

## Activation

Use when the task involves kernel-mode driver development, kernel shellcode,
manual mapping, injection, rootkit techniques, or any ring-0 code.

## Project scaffolding

Default stack: C with WDK (Windows Driver Kit). Use WDM for maximum control,
KMDF only when the task benefits from the framework.

```
<project>/
├── src/
│   ├── driver.c          # DriverEntry, dispatch routines
│   ├── imports.c         # Dynamic import resolution
│   ├── hooks.c           # SSDT, IRP, callback hooks
│   ├── inject.c          # APC / thread hijack / shellcode injection
│   ├── stealth.c         # VAD spoofing, DKOM, PiDDBCache cleanup
│   └── util.c            # Helpers: memory, strings, PE parsing
├── include/
│   ├── driver.h
│   ├── imports.h
│   ├── hooks.h
│   ├── inject.h
│   ├── stealth.h
│   └── util.h
├── <project>.vcxproj     # VS project targeting WDK
└── README.md
```

## Coding conventions

- Resolve all imports dynamically at runtime (no static IAT entries for
  sensitive APIs). Walk `ntoskrnl.exe` / `hal.dll` export tables or use
  pattern scanning for unexported functions.
- Use `POOL_TAG` constants that blend in (e.g., reuse tags from legitimate
  drivers).
- Avoid `DbgPrint` in release builds. Use a custom debug channel or compile
  out entirely.
- Prefer `ExAllocatePool2` (Win10 2004+) with `POOL_FLAG_NON_PAGED` over
  deprecated `ExAllocatePoolWithTag`.
- All IRQL-sensitive code must be annotated with `_IRQL_requires_max_()`.
- Clean up all hooks, allocations, and handles on `DriverUnload`.

## Key techniques reference

### Dynamic import resolution

```c
PVOID GetKernelExport(PCWSTR moduleName, PCSTR exportName);
PVOID PatternScan(PVOID base, SIZE_T size, PCSTR pattern, PCSTR mask);
```

### APC injection (kernel-mode)

1. Locate target thread via `PsLookupThreadByThreadId` or thread list walk.
2. Allocate RWX memory in target process (`MmAllocateContiguousMemory` or
   `ZwAllocateVirtualMemory` with attached process).
3. Write shellcode / payload.
4. Queue APC via `KeInsertQueueApc` targeting `KernelApcRoutine` or
   `NormalRoutine` depending on context.
5. For user-mode APC: set `NormalRoutine` to payload, `NormalContext` to arg.

### Thread hijacking

1. Suspend target thread (`PsSuspendThread`).
2. Get context (`PsGetContextThread`).
3. Save original RIP, set RIP to payload, set RCX to argument.
4. Resume thread (`PsResumeThread`).
5. Payload must restore original RIP on completion (trampoline or ret-gadget).

### VAD manipulation / spoofing

1. Locate `EPROCESS->VadRoot` (AVL tree).
2. Find or create `MMVAD` node for target region.
3. Modify `VadFlags` (Protection, PrivateMemory, NoChange) to spoof the
   region as `PAGE_READONLY` / `MEM_IMAGE` / backed-by-file.
4. Optionally unlink the node entirely for full invisibility.

### Callback removal / unhooking

- `PsSetCreateProcessNotifyRoutine` callbacks: walk
  `PspCreateProcessNotifyRoutine` array, clear entries.
- `PsSetCreateThreadNotifyRoutine`: same pattern for
  `PspCreateThreadNotifyRoutine`.
- `PsSetLoadImageNotifyRoutine`: `PspLoadImageNotifyRoutine`.
- `CmRegisterCallbackEx`: walk `CallbackListHead` in `CmpCallBackVector`.
- `ObRegisterCallbacks`: patch `ObjectType->TypeInfo` callback arrays.
- MiniFilter: `FltUnregisterFilter` or unlink from `FltGlobals`.

### DKOM (Direct Kernel Object Manipulation)

- Process hiding: unlink `EPROCESS` from `ActiveProcessLinks` doubly-linked
  list. Patch `HandleTableList` for handle-table invisibility.
- Token stealing: copy `TOKEN` from SYSTEM process, replace target's
  `EPROCESS->Token`.
- Thread hiding: unlink from `ETHREAD->ThreadListEntry`.

### SSDT hooking

1. Locate `KeServiceDescriptorTable` (pattern scan or `MSR LSTAR` for x64).
2. Disable write protection (`CR0.WP` or `MSR EFER.NXE` + remap).
3. Replace target service pointer with hook.
4. Restore write protection.
5. On x64 with PatchGuard: use EPT-based hooks or inline hooks instead.

## Build & test

- Build: `msbuild /p:Configuration=Release /p:Platform=x64`
- Sign: test-sign with `makecert` + `signtool` or use `bcdedit /set
  testsigning on`.
- Deploy: `sc create` + `sc start`, or manual map via existing loader.
- Debug: WinDbg with kernel debugging enabled (`bcdedit /debug on`).

## Verification checklist

- [ ] All imports resolved dynamically
- [ ] No static strings for sensitive paths/names (hash or encrypt)
- [ ] Pool allocations use blended tags
- [ ] IRQL constraints verified for all code paths
- [ ] DriverUnload cleans up all hooks, allocations, handles
- [ ] No DbgPrint in release configuration
- [ ] Compiles clean with `/W4 /WX`
