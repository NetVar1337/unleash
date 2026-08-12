---
name: offset-dumper
description: "Game/engine offset dumper patterns: pattern scan DB, PDB/schema dump, UE/Unity/Source pipelines, version pinning."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
---

# Offset dumper skill

## Outputs
- versioned offset JSON/header
- pattern signatures with uniqueness checks
- struct layouts for entities/view/net

## Pipeline
1. Identify build id
2. Scan patterns / dump schemas
3. Validate in live process
4. Emit C++/Rust bindings
