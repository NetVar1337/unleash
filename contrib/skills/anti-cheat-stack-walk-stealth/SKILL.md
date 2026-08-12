---
name: anti-cheat-stack-walk-stealth
description: "Anti-cheat stack-walking stealth: return-address spoofing, stack pivoting for call stacks, gadget frames, RtlUserThreadStart fakes, ETW/AC walk bypass."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "stack walk"
    - "return address spoof"
    - "call stack spoof"
    - "anti-cheat stack"
    - "spoof stack"
---

# Anti-cheat stack-walking stealth

Many ACs (EAC/BE/Vanguard-class and customs) validate **call stacks** on sensitive syscalls, game functions, and input paths. Goal: make illicit call sites look like legitimate game/engine/module frames.

## What ACs walk
- Usermode: `RtlCaptureStackBackTrace` / frame pointers / unwind metadata
- Kernel: stack walk from syscall traps; validate user return addresses
- ETW-TI / kernel callbacks correlating stacks to modules
- “Dirty” origins: manually mapped ranges, RWX, non-backed anonymous exec

## Detection signals you create
| Artifact | Why bad |
|---|---|
| Return addr in private RX | manual map / shellcode |
| Missing unwind / bad RSP alignment | spoof incomplete |
| Module gap / unbacked pages | hollowed/stomped poorly |
| Direct syscall stub without ntdll | known pattern |
| Identical spoofed stack every call | ML/heuristic |

## Stealth technique ladder

### 1. Stay in legitimate modules
- Module stomp / execute inside game or system DLL text
- Prefer existing executable sections with correct backing

### 2. Return-address spoofing
- Build synthetic stack with valid gadgets ending in expected callers
- Classic: push fake frames + JMP to API with spoofed ret into gadget that restores
- Ensure each spoofed RIP is **inside a real module** with plausible prologue/epilogue

### 3. Stack pivot + unwind-friendly frames
- On x64 Windows, prefer chains consistent with RUNTIME_FUNCTION data when walkers use unwind
- Align RSP 16-byte; respect homing space if faking CALL

### 4. Gadget frame quality
- Use `jmp [rbx]` / `add rsp,imm; ret` style restore gadgets from kernel32/ntdll/game
- Avoid obvious gadget caves of NOPs only

### 5. Indirect syscalls with spoofed usermode stack
- Even with syscall instruction, usermode stack above often still walked
- Spoof before transition; restore after

### 6. Thread start spoof
- Fake originating at `RtlUserThreadStart` / game main thread loops
- Avoid CreateRemoteThread start addresses in private memory

### 7. HV-assisted (advanced)
- Hide pages from AC reads while keeping execute (split-view EPT) — pair `stealth-hypervisor`
- Still spoof stacks; HV alone may not fix usermode walkers

## Implementation checklist
1. Log real stacks from legit game paths (baseline corpus)
2. Build spoof templates matching those module sequences
3. Per-call jitter: vary intermediate frames within legit set
4. Validate under AC with stack-trace logging / kernel debugger
5. Test both FP-based and unwind-based walkers

## Pseudo-flow
```text
prepare_fake_stack(template_from_legit_capture)
push restore_gadget
push spoofed_ret_into_game_code
jmp Nt* or game_function
# restore_gadget cleans RSP to real stack
```

## Pair with
`anti-cheat-bypass`, `stealth-injectors`, `game-hacking`, `lang-assembly`, `windows-internals`, `stealth-hypervisor`.

## Don’t ship
- Single static spoof stack forever
- Return addresses into non-exec or unmapped pages
- Ignoring kernel-side walks on syscall path
