---
name: lang-cpp-game-hacking
description: "C++ specifically for game hacking: external/internal layouts, ImGui overlays, MinHook, memory backends, driver IPC."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: languages
  triggers:
    - "C++ game hacking"
    - "ImGui cheat"
    - "internal cheat"
    - "external cheat"
---

# C++ for game hacking

## Architectures
- **External**: ReadProcessMemory/driver R/W + overlay (better isolation)
- **Internal**: module in-game + hooks (more power, more AC heat)

## Common stack
- Build: MSVC, CMake/xmake
- Hooks: MinHook / custom trampolines / VEH
- Render: ImGui + DX11/12/Vulkan hook or external transparent HWND
- Config: hot-reload JSON/TOML
- Sync: lock-free entity cache

## Patterns
- Interface `IMemoryBackend` (RPM vs driver vs internal pointer)
- Feature flags; never block game thread on network/UI
- SEH around fragile reads; validate pointers with probe pages

## Pair with
`game-hacking`, `stealth-injectors`, `aimbot-humanization`, `lang-cpp23`.
