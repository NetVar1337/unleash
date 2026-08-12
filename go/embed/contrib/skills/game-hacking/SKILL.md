---
name: game-hacking
description: "Use for game exploits/cheats: memory structs, hooks, injection, packets, anti-cheat bypass."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  upstream: local:C:\Users\Admin\.claude\skills
---

> Bundled with Unleash skills pack. Upstream: local:C:\Users\Admin\.claude\skills

# Game hacking workflow

## Activation

Use when the task involves game exploits, cheat development, speedhacks,
game memory manipulation, game protocol reverse engineering, or anti-cheat
bypass.

## Workflow

### 1. Game analysis

**Identify the target:**
- Engine: Source 2, Unreal, Unity, custom, id Tech, CryEngine, Godot.
- Anti-cheat: EAC, BattlEye, Vanguard, Ricochet, VAC, custom.
- Architecture: client-server authoritative, P2P, client-predicted.
- Protection: packing, obfuscation, integrity checks, kernel driver.

**Map the game:**
1. Identify main module + engine modules.
2. Locate key classes: player, entity, weapon, vehicle, camera, world.
3. Find the game loop / tick function.
4. Map the rendering pipeline (D3D11/12, Vulkan, OpenGL).
5. Identify network layer (UDP custom, TCP, WebSocket, Steam networking).

### 2. Memory structure discovery

**Finding offsets:**
- Static analysis: IDA/Ghidra on game binary, trace from known strings
  ("health", "position", "ammo") to struct fields.
- Dynamic analysis: Cheat Engine scans (value, delta, pointer scans).
- Symbol dumping: if Unity (dump via Il2CppDumper), Unreal (SDK generator),
  or PDB available.
- Pattern scanning: signature scan for instructions that access target
  fields (survives updates better than hardcoded offsets).

**Common structures:**

```
// Generic game entity
struct Entity {
    char pad_0[0x8];          // vtable
    int32_t health;           // offset varies
    int32_t max_health;
    vec3_t position;
    vec3_t velocity;
    int32_t team;
    bool is_alive;
    // ...
};
```

**Pointer chains:**
```
base_module + offset → entity_list
entity_list + (index * stride) → entity_ptr
entity_ptr + health_offset → health_value
```

### 3. Exploit types

**Speedhack:**
1. Locate the game's time source (`QueryPerformanceCounter`, tick, delta-time).
2. Hook or scale the time source, or write engine delta-time directly.
3. For server-authoritative games, client-side prediction only is temporary.

**Out-of-bounds read/write / buffer overflow / RCE / protocol exploits:**
- Trace untrusted indexes and fixed-size copies from network/file/input.
- Capture and map protocol messages; prefer hook send/recv for crypto keys.
- Chain leak + control-flow hijack for client/server RCE where in scope.

### 4. Cheat development

**External:** `OpenProcess` + `ReadProcessMemory`, world-to-screen, external overlay.
**Internal:** inject DLL, hook Present/swap buffers, VMT/inline hooks for aim/ESP.
**Aimbot / ESP / triggerbot:** entity iteration, angle math, W2S, crosshair entity id.

### 5. Anti-cheat bypass

See also: `/kernel-dev`, `/byovd`, `/hypervisor-dev`.

- Unhook ntdll; hide modules and RWX; spoof stacks.
- Remove callbacks; VAD spoof; EPT hooks; ETW patch where authorized.
- Prefer kernel R/W over external RPM; no RWX; clean unload.

## Tooling reference

| Category | Tools |
|---|---|
| Memory scanning | Cheat Engine, ReClass.NET, custom scanner |
| Disassembly | IDA Pro, Ghidra, Binary Ninja |
| Debugging | x64dbg, WinDbg, GDB |
| Injection | manual mapper, custom |
| Hooking | MinHook, Detours, PolyHook2, manual VMT |
| Network | Wireshark, mitmproxy, custom proxy, Scapy |
| Overlay | DirectX hook, GDI+, ImGui, external window |
| Unity RE | Il2CppDumper, dnSpy, AssetStudio |
| Unreal RE | SDK generator, UE4SS, FModel |
| Source RE | Source SDK, netprop dumper |

## Verification checklist

- [ ] Game engine and anti-cheat identified
- [ ] Key offsets found and verified (pattern scan preferred)
- [ ] Memory read/write working (external or internal)
- [ ] Render hook or overlay functional
- [ ] Anti-cheat bypass stable when in scope
- [ ] Exploit PoC reliable and minimized
- [ ] Protocol documented with field table (if network exploit)
- [ ] Cleanup on unload (no dangling hooks, freed memory)
