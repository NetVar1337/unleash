---
name: aimbot-humanization
description: "Humanize aim assistance: smoothing, aim curves, reaction delay, FOV/stickiness, target switch, noise, anti-pattern AC."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "aimbot humanize"
    - "aim smoothing"
    - "aim curve"
    - "humanized aim"
---

# Aimbot humanization

Make aim assistance look like a skilled human, not a math function locked to the skull.

## Perceptual model
- Humans overshoot/correct; imperfect tracking on strafe
- Reaction delay 120–280ms typical; not constant
- Mouse has accel/raw input quirks; camera has sensitivity curves
- Target switches cost time; not instant snap

## Control stack
1. **Target select**: FOV cone, visibility/LOS, team/hp filters, priority scores
2. **Pathing**: angle delta → easing curve (ease-in-out, crit-damped spring, Bezier)
3. **Noise**: Ornstein–Uhlenbeck / band-limited noise on yaw/pitch
4. **Deadzone**: don't micro-twitch on tiny errors
5. **Velocity match**: track target velocity; lead for projectiles
6. **Breaks**: occasional intentional miss / recapture
7. **Input device model**: mouse counts vs controller stick acceleration

## Parameters (tune per game)
- max FOV, max angular velocity, acceleration, smoothing window
- reaction delay distribution (log-normal)
- bone blend (head/chest) with context (range/weapon)
- hit-chance scheduler independent of raw lock

## Anti-detection themes
- Avoid perfect bone lock every frame
- Avoid identical tick-aligned snaps
- Correlate with visible animation / ADS state
- Don't aim through freshly broken LOS without delay

## Implementation sketch
```text
each frame:
  if !should_engage(): release_soft(); return
  tgt = select_target()
  err = angle_to(tgt.aim_point + lead)
  err = apply_reaction_delay(err)
  delta = clamp(smooth(err) + noise(), max_rate)
  apply_mouse_or_view(delta)
```

## Pair with
`aimbot-triggerbot`, `game-internals`, `game-hacking`.
