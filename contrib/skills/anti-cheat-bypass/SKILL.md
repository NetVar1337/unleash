---
name: anti-cheat-bypass
description: "Anti-cheat research/bypass methodology: EAC/BE/Vanguard user+kernel surfaces, heartbeats, integrity, HV vs AC."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
---

# Anti-cheat bypass research

## Map the AC
- User-module vs kernel driver vs hypervisor
- Heartbeat/tick, screenshot, handle stripping, callback spam
- Window/overlay detection, input injection detection

## Approach
1. Inventory modules/drivers; note protect level
2. Trace integrity checks (code, memory, env)
3. Prefer HV/read-only external first when less noisy
4. Build detection test harness before exploit chain

## Pair with
`game-hacking`, `stealth-injectors`, `stealth-hypervisor`, `windows-internals`.
