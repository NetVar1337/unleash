---
name: driver-comm
description: "Usermode↔kernel driver communication: IOCTL design, shared sections, events, security descriptors, stealth IOCTL."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
---

# Driver communication

## Channels
- DeviceIoControl IOCTL
- Shared memory sections + events
- Ports/ALPC (rare)

## Design
- Validate IRQL, buffer methods (METHOD_BUFFERED preferred)
- ACL device object; avoid \Device\EasyToFind names
- Version handshake; deny unknown clients
