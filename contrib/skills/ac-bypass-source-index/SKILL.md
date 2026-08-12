---
name: ac-bypass-source-index
description: "Index skill for anti-cheat bypass research sources: UC topic map, local injector/HV corpora, Kevlar/AiDA/Valthrun, how to turn threads into lab checklists."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "unknowncheats"
    - "AC sources list"
    - "bypass index"
---

# AC bypass research source index

## How to use threads/releases
1. Extract **claims** vs **repro steps** vs **code**
2. Recreate in lab with canary accounts
3. Convert into checklists/skills (this pack)
4. Never paste session cookies into repos

## Topic → skill routing
| Topic | Skill |
|---|---|
| HWID/serials | `hwid-identifier-surfaces` |
| TPM attestation | `tpm-attestation-research` |
| EAC bans | `eac-ban-stack` |
| Longevity design | `cheat-longevity-engineering` |
| Hyper-RE | `hypervisor-memory-introspection` |
| EAC.sys | `eac-kernel-driver-re` |
| EOS telemetry | `eac-usermode-telemetry-re` |
| Anti-debug | `x64dbg-anti-debugger` |
| Injectors | `manual-map-injector-engineering` |
| Driver emu | `kevlar-driver-emulation` |
| IDA AI | `aida-ida-assistant` |
| External HV stacks | `valthrun-style-stack` |
| Stack walks | `anti-cheat-stack-walk-stealth` |

## Local corpora
- `Desktop/Injectors/**`
- `Desktop/AiDA-Fork-9.4`
- `Desktop/Valthrun/Release`
- `Documents/Kevlar` (https://github.com/NetVar1337/Kevlar)

## Public topic seeds (UnknownCheats)
- serials / EAC / FN bans
- remote TPM attestation trust crypto
- write cheat unless bans
- Hyper-RE memory introspection
- HWID retrieval methods
- AC bypass complete sources list
- EasyAntiCheat.sys RE
- EasyAntiCheat_EOS telemetry RE
- x64dbg anti-debugger
- stealth injector / Sastasha injector threads
