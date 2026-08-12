---
name: zdi-researcher-guidelines
description: "Use when planning, triaging, packaging, or submitting vulnerability research to Trend Micro Zero Day Initiative (ZDI). Encodes official submission criteria, valuation factors, disclosure timelines, exclusivity rules, current product-interest suspensions, report quality bar, and portal workflow. Pair with zero-day-target-eligibility before investing in a target."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: security
  upstream: C:\Users\Admin\.agents\skills\zdi-researcher-guidelines\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\zdi-researcher-guidelines\SKILL.md

# ZDI Researcher Guidelines

Operational rules for Trend Micro **Zero Day Initiative (ZDI)** acquisition and disclosure.
This skill decides **how to pick, package, and submit** to ZDI. For the generic fail-closed
eligibility gate used across 0-day campaigns, also run `zero-day-target-eligibility`.

Re-check live sources before every major campaign or submission:

| Source | URL |
|---|---|
| Submission criteria | https://www.zerodayinitiative.com/portal/criteria/ |
| Program bulletins / interest changes | https://www.zerodayinitiative.com/portal/bulletins/ |
| Disclosure policy | https://www.zerodayinitiative.com/advisories/disclosure_policy/ |
| Benefits / valuation / tiers | https://www.zerodayinitiative.com/about/benefits/ |
| FAQ | https://www.zerodayinitiative.com/about/faq/ |
| Researcher agreement | https://zerodayinitiative.com/documents/zdi_researcher_agreement.pdf |
| PGP key (email only) | https://www.zerodayinitiative.com/documents/zdi-pgp-key.asc |
| Published advisories | https://www.zerodayinitiative.com/advisories/published/ |
| Upcoming advisories | https://www.zerodayinitiative.com/advisories/upcoming/ |
| Contact | zdi@trendmicro.com |

Do **not** store portal cookies, session IDs, tax forms, government IDs, or payment details in skills, memory, git, or reports.

## 1. Program model (what ZDI is)

ZDI is a **vendor-agnostic vulnerability acquisition** program, not a vendor bug bounty:

1. Researcher finds a **non-public** vulnerability.
2. Submits via the **secure researcher portal** (one vulnerability per case).
3. ZDI validates and, at its sole discretion, makes a **USD offer**.
4. On accept, researcher **assigns all IP/rights** in the vulnerability information to Trend Micro / ZDI.
5. ZDI coordinates private vendor disclosure, may ship **encrypted/generic** customer protections first, then publishes a ZDI advisory after patch or policy deadline.
6. Researcher may take **public credit** (or stay anonymous) and may **republish only the public advisory text** after ZDI/vendor disclosure — no extra technical detail unless ZDI agrees in writing.

Key differences vs vendor bounties:

- ZDI buys **exclusivity** until coordinated disclosure.
- After offer acceptance you may **not** sell, discuss, leak, blog, tweet technical detail, or dual-submit.
- Ownership remains yours **until** an offer is accepted (or if declined / no offer).
- Offers are non-binding chatter until posted in portal/email and accepted there.
- ZDI will **not** bury a bought bug because a vendor refuses to fix it.

## 2. Hard acceptance gates

A candidate is only ZDI-viable when **all** hard gates pass. Fail closed.

### 2.1 Latest available version

- Must reproduce on the **latest available stable/GA** product build **as of verification day**.
- Beta / pre-release / Insider / Canary / nightly / unreleased hardware **fail**.
- End-of-life / end-of-support products **fail** (ZDI will not offer).
- Independently updated components (engines, firmware, definitions, browser runtimes, management servers) must also be current.
- Windows application impact is evaluated on **supported Windows 11** branches, not retired Windows 10.

### 2.2 Widespread deployment

- Target must have **credible widespread deployment** (enterprise, consumer-at-scale, infrastructure, OEM, cloud, OS-bundled, or critical role).
- Niche tools, lab toys, and low-install specialty software usually fail even if the bug is pretty.
- Prefer products that matter in real enterprise / infrastructure estates.

### 2.3 Novelty / non-public

- Must be **previously unpatched** and **not already publicly posted or otherwise known**.
- First researcher to give ZDI **verifiable** detail wins; later duplicates get nothing.
- Public root-cause / CVE / advisory / blog / exploit-db / patch diff describing the same issue kills eligibility.
- Variants are OK only when the **affected instance + root cause + impact** are materially distinct.

### 2.4 Security-relevant, reproducible impact

- Must cross a real confidentiality / integrity / availability / privilege / sandbox / guest-host / tenant / pre-auth boundary.
- Must be **reproducible** by ZDI analysts. Unreproducible submissions are not acquired.
- Crash-only reports without proven security impact are weak or rejected.

### 2.5 Geographic / identity eligibility

- Most countries allowed; FAQ states ZDI **cannot accept** researchers residing in **Cuba, Iran, North Korea, Sudan, or Syria** (US law).
- Identity is required for payment and ethics screening — no anonymous payout participation.
- Public advisory credit can still be anonymous/pseudonymous by preference.
- Trend Micro employees/consultants are ineligible.
- Follow employer IP / moonlighting policies before accepting payment.

## 3. Preference stack (what gets paid well)

Official preference order / emphasis:

1. **Remote code execution**
2. **Enterprise-affecting software**
3. **Server-side** flaws
4. **OS** (desktop or mobile)
5. **Browsers**
6. **SCADA / IIoT**
7. **Sandbox escapes**
8. **VM escapes**
9. **Security products**

Also accepted in principle (with lower or case-by-case interest): broad software including consumer apps, ICS/OT, cloud, and **Pwn2Own track** targets — subject to current bulletins and valuation.

### Valuation dimensions ZDI explicitly uses

Score a candidate on all of these before spending exploit-dev time:

| Factor | High value | Low value |
|---|---|---|
| Product deployment | pervasive / default / enterprise standard | niche / rare |
| Privilege gained | SYSTEM/root/host/pre-auth RCE | low-integrity info leak |
| Default exposure | default install / default config | optional obscure feature |
| Product importance | DB, mail, DNS, VPN, firewall, hypervisor, IdP, backup, security console | secondary utility |
| User interaction | none / drive-by / zero-click | complex social engineering |
| Exploit reliability | deterministic, default settings | racey, heisenbug, needs exotic heap state only |
| Attack surface | unauth remote / WAN | local only after admin foothold |

## 4. Common non-offers (exclusion filter)

ZDI states it does **not commonly offer** on:

- **XSS** (reflected/stored/DOM) without exceptional native/tenant boundary impact
- **DLL planting / search-order hijack** that already assumes attacker-controlled load path equivalent to the impact
- **Live websites** / SaaS-only config issues (shippable product defects only; do not poke production tenants just to satisfy freshness)
- **ActiveX-only** issues
- **Most consumer-only products**, including **gaming** (exceptions: widely used security products and some IoT)
- **Beta / pre-release** software
- Anything **already public or known**
- Products marked **EOL / EOS** by the vendor
- AI-slop / low-merit bulk submissions (currently clogging the queue; deprioritized)

If unsure about interest in a specific product: **email zdi@trendmicro.com before starting deep research**.

## 5. Current product-interest suspensions (check bulletins)

These were active on the portal bulletins as of last verification. **Always re-read bulletins** — interest shifts without individual notice.

### Suspended / do not target for ZDI acquisition

| Product / class | Bulletin | Notes |
|---|---|---|
| Oracle VirtualBox | 2026-07-02 | Acquisition pause |
| GIMP | 2026-03-31 | Suspended |
| QEMU | 2026-03-31 | Suspended |
| Krita | 2026-03-31 | Suspended |
| Ashlar-Vellum products | 2026-03-31 | Suspended |
| Embedded systems (general) | 2026-03-31 | Suspended until further notice |
| Digilent DASYLab | 2025-11-21 | Indefinite suspend |
| FontForge | 2025-11-21 | Indefinite suspend |
| Consumer networking equipment | 2025-11-21 | Any consumer networking gear |
| IrfanView products | 2025-07-08 PDF bulletin | No longer accepting |
| Foxit PDF Editor | 2025-07-08 | Not accepting (Reader may still be in scope with limits) |
| PDF-XChange Viewer | 2025-07-08 | Not accepting (Editor may still be in scope with limits) |
| Windows 10 retiring branches | 2025-09-15 | App vulns tested on Win11; retiring Win10 OS cases not accepted |
| Windows 7 / other EOL OS | historical | EOL never accepted |

### PDF reader interest snapshot (subject to change)

| Product | Accepting | Notes |
|---|---|---|
| Adobe Acrobat Reader DC | Yes — highest PDF payout | High-level bugs, JS engine ± sandbox escape, standalone escape, parsers; full chain not required |
| pdfforge PDF Architect Free | Partial | High-level + parser RCE with clear RCE evidence; no parser info-disclosure |
| pdfforge PDF Creator Free | Partial | High-level + non-Architect features only; **no** rendering/parser (send those to Architect) |
| Foxit PDF Reader | Partial | High-level, JS engine, parser RCE with evidence; no Chromium N-days; no parser info-disclosure |
| Foxit PDF Editor | No | Suspended |
| PDF-XChange Editor | Partial | High-level, JS, parser RCE; no parser info-disclosure |
| PDF-XChange Viewer | No | Suspended |
| PDFsam suite | Partial | High-level only; no rendering/parser |
| Soda PDF Desktop | Partial | High-level only; no rendering/parser |
| Other PDF apps | Unlikely | Case-by-case only |

## 6. Submission packaging rules

### 6.1 Case hygiene

- **One vulnerability per case.** Never bundle multiple root causes in one submission.
- Submit through the **portal**, not cleartext email. Email report bodies without PGP may be rejected; portal uploads need not be encrypted.
- If emailing ZDI about a case, encrypt to the current PGP key.
- Provide enough for an analyst who has never seen the target to reproduce on a clean latest-stable install.

### 6.2 Minimum technical package

Every case should include:

1. **Vendor / product / exact version / build / patch level / channel**
2. **Component** (service, module, parser, protocol, driver, etc.)
3. **Vulnerability class** and **root cause** (not just crash site)
4. **Attack vector** (remote/local, auth/unauth, interaction model)
5. **Impact** (RCE, LPE, SBX, VM escape, auth bypass, etc.) and resulting privilege
6. **Default-config reachability** statement
7. **Step-by-step reproduction** on clean latest stable
8. **PoC / exploit / crash sample / debugger notes** as appropriate
9. **Expected vs actual behavior**
10. **Suggested severity** (CVSS-like reasoning; ZDI still decides)
11. **Novelty statement**: closest public bugs and why this differs
12. **Test environment** details (OS build, deps, VM/hardware)
13. **Hashes** of PoC files and, when practical, affected binaries
14. **Discovery date** and confirmation it was not disclosed elsewhere

Format is flexible (write-up, annotated PoC, exploit). Quality and reproducibility beat prose volume.

### 6.3 Quality bar that maximizes offers

- Prefer **root-cause + reliable trigger** over “here is a crash”.
- For memory corruption: show **controlled crash, corruption primitive, or control-flow influence** sufficient to argue RCE potential when full exploit is not required.
- For “high-level” bugs (command injection, path traversal, LPE, auth bypass): full practical impact demo.
- State whether bug works on **default install** and **default settings**.
- Avoid AI-generated filler, unverified claims, or shotgun duplicate cases — queue is explicitly overloaded with low-merit AI submissions and those are deprioritized.
- Do not require ZDI to buy licenses they cannot get; prefer widely available enterprise/eval builds when possible, and document license needs early.

### 6.4 What not to put in a case

- Other researchers’ work presented as yours
- Employer confidential / stolen internal bug data
- Production customer data, live-site testing against third parties without authorization
- Extra unrelated bugs (split cases)
- Public exploit rebrands / N-day with no novel root cause
- Credentials, cookies, or personal payment data inside the vulnerability write-up

## 7. Process, SLAs, and researcher obligations

### 7.1 Lifecycle

```text
Discover → (optional interest email) → Portal submit (1 bug/case)
  → ZDI exclusive evaluation license while under review
  → Validate / reproduce / value
  → Offer | No-offer | Need-info
  → Accept offer (≤ 7 calendar days or offer rescinds)
  → IP assignment + payment (wire/check; often 2–3 weeks)
  → Vendor notify + coordination (standard 120 days)
  → Patch or policy deadline → ZDI public advisory (+ optional credit)
  → Researcher may republish the public advisory text only
```

### 7.2 Timing expectations

| Event | Expectation |
|---|---|
| Initial ZDI response (normal) | ~2 weeks average historically |
| Queue status HIGH (current bulletin) | **> 6 weeks** for most cases; critical cases prioritized |
| Offer validity | **7 calendar days**, then rescinded (may re-ask) |
| Payment after accept | ~2–3 weeks depending on method / tax forms |
| Vendor first ack window | 5 business days, then second try +5 |
| Vendor unreachable | possible limited public advisory ~15 business days after first contact |
| Standard patch window after vendor ack | **120 days** |
| Faulty/incomplete prior patch (critical + active/imminent exploit) | **30 days** |
| Faulty patch critical/high with partial protection, exploit not imminent | **60 days** |
| Other faulty-patch class | **90 days** |
| Extensions past 120 days | rare; ZDI sole discretion |

### 7.3 Exclusivity and confidentiality (post-accept)

After accepting an offer you **must not**:

- distribute, sell, assign elsewhere, discuss, or disclose vulnerability details
- publish write-ups, tweets, conference talks, or PoCs before ZDI/vendor public disclosure
- tip third parties or other programs

Violating exclusivity can mean **program ban** and loss of trust/payment remedies under the agreement.

During evaluation (pre-accept), ZDI has an exclusive license to evaluate; you still own the bug and may withdraw, but do not publicly burn it if you want an offer.

### 7.4 Credit and republish rights

- Choose credit name / anonymous / pseudonym in portal preferences.
- After **public** ZDI and/or vendor disclosure, you may republish **the public disclosure text only**, unmodified, unless ZDI agrees otherwise.
- Republishing early or modifying the advisory can forfeit credit.

### 7.5 Payment and account

- Methods: **wire transfer** or **mailed check** (set in portal).
- US persons: **W-9** before payment; 1099 as required.
- Non-US: **W-8BEN** / **W-8BEN-E** and government ID as requested; email forms to zdi@trendmicro.com (PGP preferred).
- Account must be **verified** before payout.
- Enable TOTP when available.
- Keep profile/payment data accurate; agreement requires prompt updates.
- Offers and acceptances are binding only via **portal/email offer + formal accept** (or manually signed writing) — phone chat is non-binding.

### 7.6 Reward tiers (points = USD paid, then multipliers)

| Tier | Points | One-time bonus | Future submission bonus | Point multiplier |
|---|---:|---:|---:|---:|
| Bronze | 15,000 | $2,000 | 10% | 10% |
| Silver | 25,000 | $5,000 | 15% | 15% |
| Gold | 45,000 | $10,000 | 20% | 25% |
| Platinum | 65,000 | $25,000 | 25% | 50% |

Example: Platinum base valuation $4,000 → paid $5,000 (25% bonus) and 6,000 points (50% multiplier).

Referral: referred researcher’s **first acquisition** grants referrer **2,500 points** (registration alone does not count).

Pwn2Own wins are purchased into the same disclosure machine under contest rules; TIP/targeted incentives may pay much higher for full working exploits on listed targets when active.

## 8. Pre-research decision checklist

Run before deep work. Any **FAIL** stops the campaign for ZDI purposes.

```text
[ ] Bulletins checked today — product not suspended
[ ] Latest stable/GA version identified from vendor authority
[ ] Not beta / pre-release / EOL / EOS
[ ] Widespread deployment evidence recorded
[ ] Impact in preference stack or clearly high enterprise value
[ ] Not XSS / DLL plant / live-site / ActiveX / consumer-game junk case
[ ] Novelty search clean (CVE, ZDI upcoming/published, vendor notes, git, blogs, exploit indexes)
[ ] Default-config remote or high-value local boundary plausible
[ ] Clean lab path exists (license, hardware, firmware)
[ ] Interest email sent if product is borderline
[ ] zero-day-target-eligibility campaign card opened
```

## 9. Pre-submit decision checklist

```text
[ ] Reproduced on latest stable within revalidation window
[ ] One root cause only in this case
[ ] PoC/repro steps work on clean install
[ ] Version/build/component table complete
[ ] Impact and privilege clearly demonstrated or evidenced
[ ] Novelty ledger attached
[ ] No prior public post / dual submission / conference abstract
[ ] Researcher agreement terms understood (assignment + exclusivity on accept)
[ ] Tax/payment profile ready if offer expected
[ ] Queue delay acceptable (HIGH may exceed six weeks)
[ ] Offer response plan ready (7-day clock)
```

## 10. Campaign card (ZDI-specific)

```text
ZDI target:
Product / channel / latest build proof:
Component / attack surface:
Boundary crossed:
Preference-stack fit (RCE/enterprise/server/OS/browser/SCADA/sbx/VM/security):
Deployment evidence:
Default-config reachable? 
User interaction:
Reliability:
Bulletin status (clear / suspended / PDF-limited):
Closest public work + distinction:
Lab recipe:
Report package paths:
Interest-email result (if any):
Submit-by date:
Case ID:
Status: RESEARCH | PACKAGING | SUBMITTED | NEEDINFO | OFFER | ACCEPTED | DECLINED | NO_OFFER | PUBLIC
Revalidation deadline:
```

## 11. Agent operating rules

When helping with ZDI work:

1. **Gate first** with this skill + `zero-day-target-eligibility` before exploit engineering.
2. **Re-fetch bulletins/criteria** if last verification is stale or user is about to submit.
3. Prefer targets in the **preference stack** with clear widespread deployment.
4. Reject or warn hard on suspended products and classic non-offer classes.
5. Package reports for **analyst reproducibility**, not blog aesthetics.
6. Never dual-track a bug to another buyer/program once ZDI offer is accepted.
7. Never place cookies, sessionids, W-8/W-9 content, government IDs, or bank details in repo files, skills, or memory.
8. After accept, treat all technical detail as **embargoed** until official public advisory.
9. For Windows targets, validate on **supported Windows 11**, not retired Win10 impact stories.
10. If queue is HIGH, prioritize **critical, high-clarity** cases over speculative low-impact spam.

## 12. Quick fail patterns

- “Works on last year’s build only”
- “Needs beta flag / debug SKU”
- “Consumer game trainer bug”
- “XSS in marketing site”
- “DLL plant from attacker-controlled directory already writable at same privilege”
- “Public CVE with different trigger string”
- “QEMU/VirtualBox/GIMP/embedded/consumer router after suspension dates”
- “Parser info-leak only on a PDF product that bars info-disclosure”
- “AI summary of possible bug without working PoC”
- “Three root causes in one portal case”

## 13. Verification before calling a target “ZDI-ready”

- [ ] Live criteria page still matches hard gates above
- [ ] Live bulletins do not suspend the product/class
- [ ] Latest-stable reproduction evidence exists
- [ ] Deployment + preference-stack story is credible
- [ ] Novelty ledger complete
- [ ] Report package meets section 6
- [ ] Exclusivity/payment consequences understood
- [ ] Campaign card updated

