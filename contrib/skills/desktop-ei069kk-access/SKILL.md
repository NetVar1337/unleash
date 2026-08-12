---
name: desktop-ei069kk-access
description: "Use when connecting to, administering, or transferring files to and from DESKTOP-EI069KK via the `work` SSH/RDP aliases, or when diagnosing that remote-access path from either C3PO or DESKTOP-EI069KK."
version: 1.0.0
license: Private
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\desktop-ei069kk-access\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\desktop-ei069kk-access\SKILL.md

# DESKTOP-EI069KK Access

## Overview

Use the established `work` connection without rediscovering credentials or creating competing aliases. The client is `C3PO`; the target is `DESKTOP-EI069KK`. SSH and RDP use the target's `Admin` account.

Do not print, copy into logs, or rewrite passwords, private keys, or raw credential files. The private access bundle is already installed and the SSH host key is pinned.

## Determine the Machine Role

Run:

```powershell
hostname
whoami
```

- On `C3PO`, follow **Client Workflow**.
- On `DESKTOP-EI069KK`, follow **Server Workflow**.
- On any other host, stop and report that the `work` setup is not installed there.

Role detection is complete when the hostname matches one of the two expected machines.

## Client Workflow — C3PO

### SSH

Open an interactive session:

```powershell
ssh work
```

Run one command without opening a persistent shell:

```powershell
ssh work whoami
ssh work "dir C:\Users\Admin"
```

Expected identity:

```text
desktop-ei069kk\admin
```

The default remote shell is `cmd.exe`. Use `dir` and `cls`, or enter `powershell` when PowerShell commands and aliases such as `ls` and `clear` are wanted.

### Remote Desktop

```powershell
rdp work
```

The profile targets `DESKTOP-EI069KK\Admin` and prompts for the existing Admin password. Do not substitute the legacy `RemoteAccess` credential.

### Upload Files or Directories

Upload into `C:\Users\Admin`:

```powershell
work-put .\artifact.zip
work-put .\project
```

Choose a remote path relative to `C:\Users\Admin`:

```powershell
work-put .\artifact.zip Downloads/artifact.zip
work-put .\project Documents/project
```

### Download Files or Directories

Download into the current local directory:

```powershell
work-get Downloads/artifact.zip .
work-get Documents/project .\project
```

Both helpers recurse automatically for directories.

### Standard OpenSSH Transfer Tools

The `work` SSH alias also works directly with:

```powershell
scp .\artifact.zip work:
scp -r .\project work:Documents/project
scp work:Downloads/artifact.zip .
sftp work
```

Prefer relative remote paths with forward slashes. A remote Windows drive colon can be confused with SCP's `host:path` separator.

A transfer is complete only after the destination exists and, for important artifacts, source and destination hashes match:

```powershell
Get-FileHash .\artifact.zip -Algorithm SHA256
ssh work "certutil -hashfile C:\Users\Admin\artifact.zip SHA256"
```

## Server Workflow — DESKTOP-EI069KK

This machine is the target. Work directly under `C:\Users\Admin` unless the user specifies another destination.

Check the access services:

```powershell
Get-Service sshd, TermService
```

Both services must report `Running` before claiming SSH/RDP availability.

Relevant server paths:

```text
C:\ProgramData\ssh\sshd_config
C:\ProgramData\ssh\administrators_authorized_keys
C:\Users\Admin\desktop-ei069kk-access
```

Because `Admin` is an administrator, Windows OpenSSH normally authorizes it through `administrators_authorized_keys`, not only `C:\Users\Admin\.ssh\authorized_keys`.

Inbound relative SCP/SFTP paths resolve from `C:\Users\Admin`. When asked to send a file back to C3PO, do not assume reverse SSH is configured. Tell the user or the C3PO-side agent to initiate:

```powershell
work-get <remote-path> <local-path>
```

Use the private access repository only for maintaining the access bundle. Never display `credentials/connection.json` or private-key contents.

## Diagnostics

From C3PO, verify the resolved SSH profile without connecting:

```powershell
ssh -G work | Select-String '^(user|hostname|port|identityfile|stricthostkeychecking) '
```

Expected essentials are user `Admin`, SSH port `22`, strict host-key checking, and the installed dedicated identity.

Check endpoint reachability:

```powershell
Test-NetConnection DESKTOP-EI069KK -Port 22
Test-NetConnection DESKTOP-EI069KK -Port 3389
```

Live-check SSH identity:

```powershell
ssh -o BatchMode=yes -o ConnectTimeout=8 work whoami
```

If hostname resolution fails but the known LAN address is reachable, diagnose DNS before changing the alias. Do not disable strict host-key checking to bypass a mismatch; compare the presented key with the pinned access bundle first.

## Common Pitfalls

1. **`ls` or `clear` fails after SSH.** The remote default is `cmd.exe`; use `dir`/`cls` or enter `powershell`.
2. **RDP selects `RemoteAccess`.** Remove any stale `TERMSRV/desktop-ei069kk` credential and reopen `rdp work`; the profile must name `DESKTOP-EI069KK\Admin`.
3. **SCP parses an absolute Windows path as another host.** Use a path relative to `C:\Users\Admin`, preferably with `/` separators.
4. **A transfer command returned zero but the wrong destination was used.** Inspect the destination and hash important files before reporting completion.
5. **A host-key mismatch appears.** Stop. Never use `StrictHostKeyChecking=no`; validate or rotate the pinned host key deliberately.
6. **The skill was just installed but Pi does not list it.** Start a new Pi session; skill discovery is cached at session startup.

## Verification Checklist

- [ ] Machine role identified as C3PO or DESKTOP-EI069KK
- [ ] `ssh work whoami` returns `desktop-ei069kk\admin` when SSH is involved
- [ ] `rdp work` uses `DESKTOP-EI069KK\Admin` when RDP is involved
- [ ] Transfers arrive at the requested path
- [ ] Important transferred files have matching hashes
- [ ] No credential, private key, or raw access-bundle secret was printed

