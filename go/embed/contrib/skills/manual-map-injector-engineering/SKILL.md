---
name: manual-map-injector-engineering
description: "Stealth/manual-map injector engineering from real injector lineages: LoadLibrary vs NtCreateThreadEx vs APC vs hijack vs kernel map; W^X; SEC_IMAGE; WOW64."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "manual map injector"
    - "Sastasha"
    - "Xenos"
    - "kdmapper"
    - "stealth injector"
---

# Manual-map / stealth injector engineering

Synthesizes patterns from injector trees (Sastasha-class, Xenos/Blackbone-class, Xenox options, kdmapper-style driver delivery).

## Technique matrix
| Method | Pros | Cons |
|---|---|---|
| LoadLibrary | simple | module list artifact |
| NtCreateThreadEx | flexible | start-address heuristics |
| Thread hijack | no new thread object | race/suspend artifacts |
| APC | stealthy if alertable | delivery constraints |
| Manual map | no module list | private RX / stack walks |
| Kernel map / APC | powerful | driver trust + PG |

## Hardening checklist (payload delivery)
- Architecture detect x64/WOW64
- Relocs, imports, delayed imports, TLS, exceptions/unwind
- Section protect final W^X (no long RWX)
- Optional header wipe / name unlink
- Least-privilege handles; transient opens
- File-backed `SEC_IMAGE` dual views when useful
- Call stack spoof on sensitive APIs (`anti-cheat-stack-walk-stealth`)

## Local corpora
- `Desktop/Injectors/Sastasha Injector v1.7*`
- `Desktop/Injectors/Xenos-master`, `Xenox v2.3.2`
- `Desktop/Injectors/kdmapper v3.0.1`
- Hypervisor-SVM / VEN / Milkyway trees as available

## Pair with
`stealth-injectors`, `pe-tools`, `driver-comm`, `anti-cheat-stack-walk-stealth`.
