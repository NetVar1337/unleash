---
name: hypervisor-dev
description: "Hypervisor development workflow for Intel VT-x/AMD-V hypervisors with EPT hooks, VMCS manipulation, VMEXIT filtering, and stealth. Invoke with /hypervisor-dev or when the task involves hypervisor or EPT development."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  upstream: local:C:\Users\Admin\.claude\skills
---

> Bundled with Unleash skills pack. Upstream: local:C:\Users\Admin\.claude\skills

# Hypervisor development workflow

## Activation

Use when the task involves building a Type-1 or Type-2 hypervisor, EPT
hooks, VMCS/VMCB manipulation, VMEXIT handling, or virtualization-based
stealth.

## Architecture

```
┌─────────────────────────────────────────────┐
│  Guest OS (Windows/Linux)                   │
│  ┌───────────────────────────────────────┐  │
│  │  Ring 3 (user-mode)                   │  │
│  └───────────────────────────────────────┘  │
│  ┌───────────────────────────────────────┐  │
│  │  Ring 0 (kernel-mode, guest)          │  │
│  └───────────────────────────────────────┘  │
├─────────────────────────────────────────────┤
│  VMX root mode (hypervisor)                 │
│  ┌───────────────────────────────────────┐  │
│  │  VMCS / EPT / VMEXIT handler          │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

## Project scaffolding

```
<project>/
├── src/
│   ├── vmx.c             # VMXON, VMCS init, VMLAUNCH/VMRESUME
│   ├── vmexit.c          # VMEXIT dispatcher and handlers
│   ├── ept.c             # EPT setup, splitting, hooks
│   ├── vmm.c             # Per-core VMM state, init/teardown
│   ├── msr.c             # MSR interception and virtualization
│   ├── cpuid.c           # CPUID virtualization
│   ├── stealth.c         # Hide hypervisor from guest
│   └── util.c            # Memory, logging, PE helpers
├── include/
│   ├── vmx.h
│   ├── vmcs.h            # VMCS field encodings
│   ├── ept.h
│   ├── vmm.h
│   └── util.h
├── asm/
│   ├── vmx_ops.asm       # VMXON, VMLAUNCH, VMRESUME, VMREAD, VMWRITE
│   ├── vmexit_stub.asm   # VMEXIT entry: save/restore guest state
│   └── helpers.asm       # INVD, LGDT, LIDT, SIDT, SGDT wrappers
└── <project>.vcxproj
```

## Key implementation details

### VMX initialization (per logical processor)

1. Check `CPUID.1:ECX.VMX[bit 5]` and `IA32_FEATURE_CONTROL` MSR.
2. Allocate VMXON region (4KB aligned), set revision ID from
   `IA32_VMX_BASIC`.
3. Execute `VMXON`.
4. Allocate VMCS region, `VMCLEAR` + `VMPTRLD`.
5. Initialize all VMCS fields (see below).
6. `VMLAUNCH` to enter guest.

### VMCS field setup (critical fields)

| Field | Value |
|---|---|
| `HOST_CR0/CR3/CR4` | Host (hypervisor) values |
| `HOST_CS/SS/DS/ES/FS/GS/TR/LDTR` | Host selectors |
| `HOST_RIP` | VMEXIT handler entry |
| `HOST_RSP` | Per-core host stack |
| `GUEST_CR0/CR3/CR4` | Guest values (read from registers) |
| `GUEST_CS/SS/DS/...` | Guest selectors (read via `SGDT`/`SIDT`/segment reads) |
| `GUEST_RIP` | Address after `VMLAUNCH` instruction |
| `GUEST_RSP` | Current stack pointer |
| `GUEST_RFLAGS` | Current RFLAGS |
| `PIN_BASED_VM_EXEC_CONTROL` | Enable external-interrupt / NMI exiting as needed |
| `PRIMARY_PROC_BASED_VM_EXEC_CONTROL` | `USE_MSR_BITMAPS`, `ACTIVATE_SECONDARY_CONTROLS` |
| `SECONDARY_PROC_BASED_VM_EXEC_CONTROL` | `ENABLE_EPT`, `ENABLE_RDTSCP`, `ENABLE_INVPCID`, `ENABLE_XSAVES` |
| `VM_EXIT_CONTROLS` | `HOST_ADDRESS_SPACE_SIZE` (64-bit host) |
| `VM_ENTRY_CONTROLS` | `IA32E_MODE_GUEST` (64-bit guest) |
| `EPT_POINTER` | EPT PML4 physical address, WB memory type, 4-level walk |

### EPT setup

1. Allocate EPT PML4 (4KB aligned, zeroed).
2. Identity-map all physical memory:
   - PML4[0] → PDPT
   - PDPT[i] → PD (1GB pages if supported, else 2MB)
   - PD[j] → 2MB large page (or PT for 4KB)
3. Set permissions: RWX for all initially.
4. Store EPT PML4 physical address in `EPT_POINTER` VMCS field.

### EPT hooks (page-level)

**Execute-only hook (stealthiest):**

1. Split the 2MB large page containing target into 512 × 4KB pages.
2. For the target 4KB page:
   - Create a **shadow page**: copy original page content, apply
     modifications (inline hook, patch).
   - Set original EPT entry: execute-only (`X=1, R=0, W=0`) → points to
     **original** page.
   - On `#VE` / EPT violation for read/write: swap EPT entry to shadow
     page (`R=1, W=1, X=0`), invalidate TLB (`INVEPT`).
   - On execute: swap back to original (`X=1, R=0, W=0`).
3. Result: guest executes original code, reads/writes see hooked code.
   Invisible to memory introspection that reads then executes.

**Inline hook via EPT:**

1. Split target page to 4KB.
2. Write inline hook (JMP to handler) into shadow copy.
3. EPT entry for execute → original page (no hook visible on exec).
4. EPT entry for read/write → shadow page (hook visible on read).
5. Or reverse: execute → shadow (hooked), read/write → original (clean).

### VMEXIT handling

```c
void VmexitHandler(PGUEST_STATE guestState) {
    ULONG exitReason = VmcsRead(VM_EXIT_REASON) & 0xFFFF;

    switch (exitReason) {
    case EXIT_REASON_EPT_VIOLATION:
        HandleEptViolation(guestState);
        break;
    case EXIT_REASON_MSR_READ:
        HandleMsrRead(guestState);
        break;
    case EXIT_REASON_MSR_WRITE:
        HandleMsrWrite(guestState);
        break;
    case EXIT_REASON_CPUID:
        HandleCpuid(guestState);
        break;
    case EXIT_REASON_VMCALL:
        HandleVmcall(guestState);  // Hypercall interface
        break;
    case EXIT_REASON_EXTERNAL_INTERRUPT:
        HandleExternalInterrupt(guestState);
        break;
    default:
        // Inject #UD or forward
        InjectUndefinedOpcode(guestState);
        break;
    }
}
```

### Stealth considerations

- **CPUID virtualization:** Clear VMX bit in `CPUID.1:ECX[5]` for guest.
- **MSR interception:** Virtualize `IA32_FEATURE_CONTROL`, `IA32_VMX_*`
  MSRs — return 0 or #GP to hide VMX from guest.
- **Timing:** Minimize VMEXIT frequency. Use MSR bitmaps to avoid
  unnecessary exits. Avoid `RDTSC`/`RDTSCP` exiting unless needed.
- **Memory:** Hypervisor pages must be outside guest-visible physical
  memory or hidden via EPT (mark as not-present in guest EPT).
- **VMCS shadowing:** On Skylake+, use VMCS shadowing to reduce
  `VMREAD`/`VMWRITE` VMEXIT overhead if guest runs nested VMX.

## Build & test

- Build: WDK + NASM for asm stubs. `/W4 /WX`, x64 only.
- Test: QEMU with `-enable-kvm` disabled (pure TCG) for initial bring-up,
  then bare metal.
- Debug: Serial output from hypervisor (`IoWrite8` to COM1) + WinDbg
  guest kernel debug.
- Validate: `INVEPT` after every EPT modification. `VMCLEAR` before
  `VMPTRLD` on VMCS migration.

## Verification checklist

- [ ] VMCS revision ID matches `IA32_VMX_BASIC`
- [ ] All guest state saved/restored on VMEXIT (GP regs, segment regs,
      CR2, DR7, XMM via `XSAVES`)
- [ ] EPT identity map covers all guest physical memory
- [ ] EPT hooks split large pages correctly
- [ ] `INVEPT` called after every EPT entry modification
- [ ] CPUID/MSR virtualization hides VMX from guest
- [ ] Hypervisor memory is not guest-visible
- [ ] Per-core VMM state is isolated (no shared mutable state without locks)
- [ ] Clean teardown: `VMXOFF` on all cores, free all allocations
