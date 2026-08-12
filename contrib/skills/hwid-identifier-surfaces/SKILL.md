---
name: hwid-identifier-surfaces
description: "Methods for retrieving PC unique identifiers used by anti-cheats: disk, SMBIOS, NIC, GPU, TPM, Windows product, volume, firmware tables."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: game
  triggers:
    - "HWID"
    - "MachineGuid"
    - "SMBIOS UUID"
    - "disk serial"
    - "GetSystemFirmwareTable"
---

# HWID / unique identifier surfaces

Research map of identifiers anti-cheats and ban systems commonly fingerprint. Operator-authorized lab use.

## Common ID families
| Family | Examples | Typical APIs / sources |
|---|---|---|
| Disk | serial, model, bus path | `IOCTL_STORAGE_QUERY_PROPERTY`, SCSI inquiry, NVMe identify |
| Volume | vol serial | `GetVolumeInformation` |
| SMBIOS/DMI | UUID, serial, baseboard | `GetSystemFirmwareTable('RSMB')`, WMI `Win32_*` |
| NIC | MAC, permanent addr | `GetAdaptersAddresses`, OID queries |
| GPU | BIOS/serial/UUID | DXGI/setupapi, vendor IOCTLs |
| CPU | microcode/IDs (weak alone) | CPUID leaves |
| TPM | EK/AK names, PCRs | TBS/TPM2.0 capability queries |
| Windows | MachineGuid, ProductId, SQM | registry `Cryptography\MachineGuid`, etc. |
| Monitor/EDID | display serials | setupapi monitor |
| USB | device instance IDs | setupapi |
| EFI | variable-backed IDs | firmware vars (when accessible) |

## Collection hygiene (research harness)
- Log **source + raw + normalized** form for each ID
- Note privilege level required (user vs admin vs kernel)
- Capture whether value survives reinstall / disk clone / VM move
- Never assume one ID is "the HWID" — ACs combine features

## Ban-stack relevance
- User-mode collectors vs kernel collectors
- Remote services may hash/salt server-side
- Spoofing one field while leaving correlated fields is often fingerprinted

## Pair with
`tpm-attestation-research`, `eac-ban-stack`, `anti-cheat-bypass`, `windows-internals`.

## Refs (public threads / topics)
- UC: methods retrieving unique identifiers / HWIDs
- UC: serials & EAC/FN ban discussions
