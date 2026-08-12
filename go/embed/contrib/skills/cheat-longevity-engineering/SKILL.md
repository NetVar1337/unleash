---
name: cheat-longevity-engineering
description: "Engineering cheats for longevity under AC: minimize ban surfaces, feature risk tiers, OPSEC build/deploy, silent flags, update cadence."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "cheat longevity"
    - "silent ban"
    - "ban evasion engineering"
    - "undetected design"
---

# Cheat longevity engineering ("write cheat unless bans")

Goal: design features and delivery so research builds last longer under modern AC — not magic immunity.

## Risk tier features
| Tier | Examples | Notes |
|---|---|---|
| Lower | external read radar, web radar | still HWID/telemetry risk |
| Medium | internal ESP with careful stacks | stack walks, overlays |
| High | aim write, input synthesis | behavioral + integrity |
| Extreme | kernel RW without HV hygiene | callback hell |

## Longevity principles
1. **Read > write** when possible
2. Prefer **signed execution contexts** / legitimate modules for call stacks (`anti-cheat-stack-walk-stealth`)
3. Separate **loader** trust from **payload** trust
4. Assume **silent bans** — canary accounts, staged rollout
5. Strip debug strings; deterministic builds; no public GH paste signatures
6. Rotate primitives after AC patches; keep offset DB versioned
7. HWID/TPM: understand what you cannot cheaply forge (`tpm-attestation-research`)

## Delivery pipelines
- Manual map vs LoadLibrary risk profile
- HV read vs driver IOCTL vs usermode RPM
- UEFI/boot loaders change measured boot story

## Pair with
`game-hacking`, `eac-ban-stack`, `stealth-injectors`, `valthrun-style-stack`, `aimbot-humanization`.

## Refs
- UC: write cheat unless bans discussions
