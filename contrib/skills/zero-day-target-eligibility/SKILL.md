---
name: zero-day-target-eligibility
description: "Use before or during a vulnerability-research campaign governed by strict acquisition criteria: the bug must affect the latest stable product, have widespread deployment, remain non-public, and prioritize RCE, enterprise/server software, operating systems, browsers, sandbox or VM escapes, SCADA/IIoT, and security products. Provides fail-closed version, novelty, deployment, impact, and exclusion..."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: security
  upstream: C:\Users\Admin\.agents\skills\zero-day-target-eligibility\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\zero-day-target-eligibility\SKILL.md

# Zero-Day Target Eligibility

## Purpose

Fail closed before investing months in a technically valid but ineligible bug. Re-run the gate whenever the vendor ships an update, the finding becomes public, or impact changes.

This skill decides **eligibility**, not whether a crash is exploitable.

For Trend Micro **Zero Day Initiative (ZDI)** submissions — portal workflow, valuation
factors, exclusivity/IP assignment, disclosure SLAs, reward tiers, and **live product-interest
suspensions** — also load `zdi-researcher-guidelines` and re-check ZDI bulletins before submit.

## Hard Gates

A candidate remains active only when all are true:

1. **Current:** reproduced on the latest stable/GA product and all independently updated components as of the verification date.
2. **Deployed:** credible evidence shows widespread consumer, enterprise, infrastructure, OEM, cloud, or OS distribution.
3. **Novel:** no public source describes the same affected component, trigger, root cause, and security impact.
4. **Security boundary:** impact crosses a meaningful confidentiality, integrity, availability, privilege, sandbox, guest/host, tenant, or pre-auth boundary.
5. **In scope:** not merely XSS, DLL planting, a live-site configuration issue, ActiveX, consumer-only game software, pre-release-only behavior, or an already published bug.

If evidence is missing, mark `UNVERIFIED`; do not silently treat it as passing.

## Latest-Stable Gate

### Establish authoritative versions

Record, with retrieval date and URLs:

- vendor release channel and latest stable version/build;
- security and cumulative updates;
- component versions that update independently;
- browser engine/runtime build;
- security-product platform, engine, definitions, and management-server version;
- mobile OS and monthly security patch level;
- hypervisor and device-model package version;
- appliance firmware and hardware revision;
- cloud/server rolling release identifier where a downloadable version does not exist.

Prefer vendor release notes, signed package metadata, in-product version output, and update catalogs. Third-party version sites are corroboration only.

### Reproduction matrix

| Product/channel | Build/component | Patch date | Clean install or upgraded | Trigger result | Evidence |
|---|---|---|---|---|---|
| latest stable | | | | | |
| latest stable clean | | | | | |
| previous stable | | | | | |
| beta/pre-release | | | | | |

Rules:

- A beta-only result fails.
- A previous-version-only result fails even if still supported, unless the acquisition policy explicitly accepts it.
- A finding seen on beta passes this gate only after independent reproduction on latest stable.
- For SaaS/live services, do not test production tenants merely to satisfy freshness; use vendor-provided local, lab, appliance, or program-authorized environments.
- Re-run after every relevant release until submission.

Completion criterion: exact latest stable bytes/configuration are archived or hash-identified and the trigger reproduces there.

## Deployment Gate

Collect at least two independent signals where possible:

- bundled/default OS component;
- vendor install-base or active-device figures;
- enterprise market presence or standard infrastructure role;
- cloud marketplace/download/package telemetry;
- OEM preload or appliance fleet evidence;
- common dependency embedded in many products;
- default exposure in server, browser, mobile, hypervisor, or security deployments.

Do not equate GitHub stars with deployment. A low-profile parser embedded in a major product can qualify; a popular developer toy may not.

Classify:

- `PERVASIVE`: bundled/default or very large install base;
- `WIDESPREAD`: strong enterprise/consumer deployment evidence;
- `MATERIAL`: meaningful but uncertain deployment;
- `NICHE`: reject unless exceptional impact changes policy priority.

Completion criterion: deployment classification cites concrete evidence and identifies who runs the affected component.

## Novelty Gate

Search by all of:

- product and component names;
- function, service, protocol method, message type, file format, and error signature;
- root-cause pattern and sink API;
- patch diff and changed check;
- crash stack and distinctive strings;
- vendor advisories, CVE databases, security bulletins, issue trackers, commits, release notes, mailing lists, conference material, blogs, forums, PoC repositories, and exploit indexes.

Maintain a novelty ledger:

| Source/date | Closest public issue | Same component? | Same root cause? | Same trigger? | Same impact? | Distinction |
|---|---|---:|---:|---:|---:|---|

A known bug can seed a new variant. The candidate remains novel only if the affected instance or invariant failure is materially distinct and not publicly described.

Stop and reclassify if a matching disclosure appears. Preserve dates and hashes proving when research began, but do not call a public bug a zero-day.

Completion criterion: closest known issues are documented and the distinction is root-cause based, not cosmetic input variation.

## Impact and Priority Score

Score only after hard gates pass.

| Dimension | 0 | 1 | 2 | 3 |
|---|---|---|---|---|
| Reachability | local/admin | local low-priv | authenticated remote | unauthenticated remote |
| Result | low impact | DoS/disclosure | privilege/sandbox boundary | reliable code execution/host escape |
| Deployment | niche | material | widespread | pervasive/default |
| Enterprise value | consumer-only | mixed | enterprise common | core server/security/infrastructure |
| Interaction | complex user action | normal user action | low interaction | none |
| Reliability | theoretical | unstable | repeatable | deterministic |

Priority order when scores are close:

1. unauthenticated server-side RCE;
2. VM or sandbox escape with practical initial foothold;
3. browser or OS remote code execution;
4. security-product RCE/LPE/bypass in enterprise deployments;
5. SCADA/IIoT gateway, engineering workstation, or management-plane RCE;
6. mobile zero-click/one-click chains;
7. high-impact local privilege escalation in pervasive OS components.

## Exclusion Filter

Reject or deprioritize:

- reflected/stored/DOM XSS without an exceptional native or tenant boundary;
- search-order/DLL planting requiring attacker control already equivalent to impact;
- live website bugs rather than shippable product defects;
- ActiveX-only issues;
- beta, nightly, Insider, Canary, developer-preview, or unreleased hardware-only behavior;
- games and consumer-only software unless a widely deployed security/IoT component is actually affected;
- known, publicly posted, duplicate, or trivially rediscovered issues;
- crashes without a proven security boundary;
- unsupported/default-disabled modules with negligible deployment;
- products or classes currently suspended on the ZDI bulletins board (see `zdi-researcher-guidelines`).

Document the rejection reason so the campaign does not rediscover the same dead end.

## Campaign Card

Every active target gets:

```text
Target/product:
Latest stable proof:
Independent component versions:
Deployment class/evidence:
Attack surface:
Expected boundary:
Preferred impact:
Closest public work:
Novelty distinction:
Exclusion risks:
Research environment:
Next falsifiable experiment:
Revalidation deadline:
Status: ELIGIBLE | UNVERIFIED | REJECTED | PUBLIC
```

## Revalidation Cadence

Re-run gates:

- at campaign start;
- after every vendor update affecting the component;
- before exploit engineering;
- before reporting/submission;
- immediately after a related public disclosure;
- after 30 days of inactivity.

Automate version checks where possible, but manually confirm package/component versions in the actual test environment.

## Common Pitfalls

1. Calling “latest supported” the “latest available” release.
2. Checking OS build but not an independently serviced engine or firmware component.
3. Treating a different crashing input as a novel root cause.
4. Using popularity proxies without deployment evidence.
5. Spending weeks weaponizing before checking exclusions.
6. Letting a stale VM remain the only reproducer.
7. Assuming a beta fix means stable is affected; test stable bytes directly.
8. Searching CVE titles only; public root causes often use different vocabulary.

## Verification Checklist

- [ ] Latest stable/GA release proven from authoritative sources
- [ ] All independently updated components recorded
- [ ] Clean latest-stable reproduction completed
- [ ] Deployment classified with evidence
- [ ] Novelty ledger covers closest public work
- [ ] Meaningful security boundary identified
- [ ] Exclusion filter passed
- [ ] Priority score calculated after hard gates
- [ ] Campaign card and revalidation deadline recorded
- [ ] Gate rerun immediately before reporting

