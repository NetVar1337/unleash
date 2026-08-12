---
name: aimbot-triggerbot
description: "Aimbot + triggerbot logic: target selection, visibility, fire timing, recoil control hooks, config architecture."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "triggerbot"
    - "aimbot logic"
    - "auto fire"
    - "recoil control"
---

# Aimbot + triggerbot logic

## Modules
1. **Config** — hotkeys, FOV, bones, humanize params, weapon profiles
2. **Memory/net interface** — entity read, local player, view angles, visibility
3. **World-to-screen / angle math** — quaternion/Euler consistency
4. **Targeting** — score = f(fov, dist, hp, threat, visibility)
5. **Aim** — humanized controller (see `aimbot-humanization`)
6. **Triggerbot** — fire when crosshair/bone color/trace satisfied + delay
7. **Recoil/spread** — optional compensation with human caps
8. **Safety** — menu focus, chat, UI, round state guards

## Triggerbot rules
- Confirm enemy hitbox under crosshair (trace or color+validation)
- Randomize click down/up ms; respect fire rate RPM
- Optional: only when ADS / scoped / not reloading
- Anti-spray: burst limits

## Architecture tips
- Separate **read path** (cache entities @ tick) from **input path** (high rate)
- Deterministic math lib; unit-test angle wrap
- Weapon script table for different humanize/trigger profiles
- Logging mode for offline tuning without firing

## Ethics/auth
Operator-authorized research/lab/AC testing. Full technical implementation when requested.
