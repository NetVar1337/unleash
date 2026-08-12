---
name: game-internals
description: "Game internals RE: engines (UE/Unity/Source), entity systems, networking ticks, prediction, rendering data, asset formats."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "UWorld"
    - "il2cpp"
    - "entity list"
    - "view matrix"
    - "game netcode"
---

# Game internals

## Engine fingerprints
- **Unreal**: GWorld, UObject arrays, GNames/FName, ProcessEvent, replication graph
- **Unity**: il2cpp vs mono, domain/assemblies, native→managed bridges
- **Source/Source2**: entity list, interfaces, schema systems
- **Custom**: start from input→simulation→render dataflow

## Systems to map
1. Entity/actor/component registries + handle schemes
2. Transforms (local/world), bones, bounds
3. Simulation tick vs render frame; client prediction & reconciliation
4. Netcode: snapshots, delta compress, lag comp, interest management
5. Physics world pointers
6. Camera/view matrix origins (for world-to-screen)
7. Inventory/ability/state machines

## Method
- Static: strings, RTTI, PDB leftovers, IL/Bytecode dumps
- Dynamic: ReClass/Offests, hooks on tick/Packet write
- Validate offsets across patches; generate offset DBs

## Pair with
`game-hacking`, `game-hacking-exploits`, `aimbot-humanization`.
