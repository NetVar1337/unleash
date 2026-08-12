---
name: kernel-callbacks
description: "Windows kernel callback tradecraft: enum/remove/spoof create-process/image/thread/object callbacks, minifilter notes."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
---

# Kernel callbacks

## Common
- Ps* notify routines
- ObRegisterCallbacks
- Cm callbacks
- Minifilters

## Research path
Enumerate → attribute owner module → assess PatchGuard risk → decide remove vs filter vs HV hide
