---
name: valthrun-style-stack
description: "Valthrun-style external/HV game stacks: kernel driver + usermode interface + UEFI loader + overlay/radar; CS2-class architecture notes."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "Valthrun"
    - "CS2 overlay"
    - "kernel read driver"
    - "UEFI driver loader"
---

# Valthrun-style read stack

Architecture pattern seen in external/HV game tooling releases:

## Components
| Piece | Role |
|---|---|
| Usermode overlay/radar | consume world state |
| Driver interface DLL | IOCTL/user API |
| Kernel driver | read physical/virtual memory |
| UEFI/ISO loader | early driver bring-up path |
| Zenith-class installer | package/driver deployment |

## Design lessons
- Split **trust** (loader/driver) from **features** (overlay)
- Version stamp every artifact (`_0157087` style hashes)
- Prefer read-only features for longevity (`cheat-longevity-engineering`)
- UEFI loaders interact with measured boot / TPM stories

## Local path
`Desktop/Valthrun/Release/*`

## Pair with
`hypervisor-memory-introspection`, `driver-comm`, `game-hacking`, `tpm-attestation-research`.
