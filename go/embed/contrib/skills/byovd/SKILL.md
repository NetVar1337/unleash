---
name: byovd
description: "BYOVD (Bring Your Own Vulnerable Driver) attack workflow. Covers vulnerable driver identification, IOCTL analysis, arbitrary physical/virtual memory access, MSR read/write, and leveraging vulnerable signed drivers for kernel code execution. Invoke with /byovd or when the task involves BYOVD techniques."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: offense
  upstream: local:C:\Users\Admin\.claude\skills
---

> Bundled with Unleash skills pack. Upstream: local:C:\Users\Admin\.claude\skills

# BYOVD attack workflow

## Activation

Use when the task involves BYOVD attacks, vulnerable driver analysis, IOCTL
reverse engineering for kernel R/W primitives, or leveraging signed vulnerable
drivers.

## Workflow

### 1. Driver selection

Identify a suitable vulnerable driver:

- Must have a valid Authenticode signature (or bypass DSE).
- Must expose an IOCTL that provides arbitrary memory R/W, MSR access, or
  physical memory access.
- Prefer drivers already present on target systems (OEM bloatware,
  anti-cheat, hardware utilities).

Common families:

| Driver | Vendor | Primitive |
|---|---|---|
| `rtcore64.sys` | MSI | Arbitrary physical memory R/W via MSR |
| `dbutil_2_3.sys` | Dell | Arbitrary physical memory R/W |
| `gdrv.sys` | GIGABYTE | Physical memory R/W, MSR R/W |
| `AsIO64.sys` | ASUS | Physical memory R/W, I/O port access |
| `WinRing0x64.sys` | OpenLibSys | MSR R/W, physical memory R/W, I/O ports |
| `ene.sys` | ENE | Physical memory R/W |
| `lha.sys` | LG | Physical memory R/W |
| `amp.sys` | Acer | Physical memory R/W |
| `phymemx64.sys` | Various | Physical memory R/W |
| `capcom.sys` | Capcom | Arbitrary kernel R/W via HLT |

### 2. IOCTL analysis

Reverse engineer the driver's dispatch routine:

1. Locate `DriverEntry` → `DriverObject->MajorFunction[IRP_MJ_DEVICE_CONTROL]`.
2. Identify `IOCTL` codes from the dispatch switch.
3. Map each IOCTL to its handler and document:
   - Input buffer layout (`METHOD_BUFFERED`, `METHOD_IN_DIRECT`,
     `METHOD_OUT_DIRECT`, `METHOD_NEITHER`).
   - Output buffer layout.
   - Validation (or lack thereof) on user-supplied addresses/sizes.
   - The actual kernel operation (MmMapIoSpace, __readmsr, __writemsr,
     direct pointer deref).

### 3. Primitive construction

**Physical memory R/W → virtual memory R/W:**

1. Read `CR3` from target process (via `EPROCESS->DirectoryTableBase`).
2. Walk page tables manually: PML4 → PDPT → PD → PT.
3. Translate target virtual address to physical address.
4. Use driver's physical R/W IOCTL to read/write the physical page.

**MSR R/W:**

- Read `MSR LSTAR` (0xC0000082) → `KiSystemCall64` base for KASLR defeat.
- Read `MSR IA32_EFER` (0xC0000080) → check NXE bit.
- Write `MSR LSTAR` → redirect syscall handler (PatchGuard detects this).

**Direct kernel R/W (e.g., Capcom):**

- Driver executes arbitrary code in ring 0 via `HLT` instruction handler
  or direct call.
- Map shellcode, call via IOCTL.

### 4. Post-exploitation via BYOVD

Once kernel R/W is established:

1. **KASLR defeat:** Read `ntoskrnl` base from `MSR LSTAR` or
   `PsLoadedModuleList`.
2. **Token steal:** Locate `EPROCESS` via `PsInitialSystemProcess` or
   `ActiveProcessLinks` walk, copy SYSTEM token.
3. **Disable callbacks:** Clear `PspCreateProcessNotifyRoutine`,
   `PspCreateThreadNotifyRoutine`, `PspLoadImageNotifyRoutine`,
   `CmpCallBackVector`, `ObTypeInitializerTable`.
4. **Disable ETW:** Patch `EtwEventWrite` or clear `EtwpDebuggerData`.
5. **Disable PatchGuard:** Patch `KiTimerDispatch` / `KiDpcDispatch` or
   corrupt `PG` context (version-specific).
6. **DSE bypass:** Patch `ci.dll!g_CiOptions` or
   `ntoskrnl!SeValidateImageHeader`.

### 5. Cleanup

- Unload the vulnerable driver (`sc stop` + `sc delete` or
  `ZwUnloadDriver`).
- Remove any traces from `Prefetch`, `Amcache`, `SetupAPI` logs.
- Clear `System` event log entries for driver load (Event ID 7045).

## Tooling

| Task | Tool |
|---|---|
| IOCTL discovery | IDA Pro, Ghidra, IRPMon, API Monitor |
| Driver signature check | `signtool verify /pa`, `sigcheck` |
| Physical memory access | Custom tool via driver IOCTL |
| KASLR defeat | `MSR LSTAR` read, `NtQuerySystemInformation` |
| Driver blocklist check | Microsoft vulnerable driver blocklist |

## Verification checklist

- [ ] Driver signature is valid and not on Microsoft blocklist
- [ ] IOCTL codes and buffer layouts fully documented
- [ ] Physical → virtual translation verified against `!vtop` in WinDbg
- [ ] KASLR defeat produces correct `ntoskrnl` base
- [ ] Token steal results in `whoami` → `NT AUTHORITY\SYSTEM`
- [ ] Driver unloaded and traces cleaned post-operation
