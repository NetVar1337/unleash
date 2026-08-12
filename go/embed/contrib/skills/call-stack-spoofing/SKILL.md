---
name: call-stack-spoofing
description: "Usermode call-stack spoofing primitives: synthetic frames, gadget restore, JMP vs CALL semantics, ETW-friendly stacks."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "stack spoofing"
    - "fake call stack"
    - "RtlCaptureStackBackTrace spoof"
---

# Call-stack spoofing primitives

Companion to `anti-cheat-stack-walk-stealth` focused on **mechanism**.

## Building blocks
- **Spoof JMP**: transfer without pushing real ret; walker sees crafted frames only
- **Restore gadget**: pops saved real RSP/RBX and resumes
- **Desync FP vs RSP walks**: some walkers trust RBP chain — keep both consistent when possible
- **Veh/instrumentation** paths: don’t leave VEH frames that advertise private code

## Validation
- Compare `CaptureStackBackTrace` output before/after
- Kernel: optional driver logging of user stack on syscall
- Game+AC stress: inventory, shoot, inventory loops

## Pair with
`anti-cheat-stack-walk-stealth`, `lang-assembly`, `stealth-injectors`.
