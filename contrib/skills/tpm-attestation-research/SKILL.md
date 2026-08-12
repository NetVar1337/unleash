---
name: tpm-attestation-research
description: "Remote TPM attestation / trust crypto / serial research: TPM2 keys, quote/PCRs, Windows health attestation surfaces, AC trust anchors."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: stealth
  triggers:
    - "TPM attestation"
    - "PCR quote"
    - "EK AK"
    - "fTPM"
    - "remote attestation"
---

# TPM attestation & trust crypto (research)

## Concepts
- **EK** (Endorsement Key), **AK** (Attestation Key), restricted signing
- **PCR banks** (SHA1/SHA256) reflecting boot measurements
- **Quote** operations binding PCRs to nonce
- Platform cert chains / manufacturer endorsements
- OS "health attestation" / measured boot consumers

## Why ACs care
Remote attestation can prove: real TPM present, boot chain measurements, device key continuity across sessions — harder than simple serial spoof.

## Research workflow
1. Inventory TPM presence (`Get-Tpm`, TBS APIs, `Windows.Devices.Tpm` stacks)
2. Enumerate capabilities / algs / NV indices (read-only first)
3. Capture what user-mode AC modules query vs kernel
4. Map any **remote** challenge-response (nonce → quote/cert → server verify)
5. Document failure modes (no TPM, fTPM, VBS, Hyper-V nested)

## Lab notes
- Distinguish **spoofable software serials** from **crypto-backed identity**
- Virtual TPMs / pass-through change the threat model
- Pair with hypervisor research carefully (measured boot interactions)

## Pair with
`hwid-identifier-surfaces`, `stealth-hypervisor`, `eac-ban-stack`, `windows-recovery-state-research`.

## Refs
- UC: remote TPM attestation / trust crypto / serial threads
