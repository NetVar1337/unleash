---
name: deslopify
description: "De-slop pass for any text. Scans for the statistical fingerprints of AI writing (negative parallelism / 'not X but Y', em-dash density, rule-of-three, false ranges, puffery vocabulary, uniform cadence, hedged both-sidesing), rewrites by meaning, then re-scans until the text is clean and sits in the right register for its genre: academic article, tweet, reddit post, email, blog, docs, marketing...."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: engineering
  upstream: C:\Users\Admin\.agents\skills\deslopify\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\deslopify\SKILL.md

# Deslopify

Strip the AI fingerprints out of a text and make it good in its genre. The target is prose that reads like one person wrote it for one audience about something they actually know. Detector scores are beside the point; text can score human and still be unreadable.

## Why this loops

The "not X but Y" family and its relatives are generative habits. Preference tuning rewards balanced, contrastive, comprehensive-sounding framing, so the contrast move sits deep in the model's priors and surfaces about once a paragraph. Two consequences shape this skill:

1. **You cannot see your own slop.** The priors that produce the pattern also make it invisible on re-read. So detection runs mechanically, as regex against a fixed catalog. "Does this look AI to me?" is not a detection method.
2. **Rewriting reintroduces slop.** Ask a model to remove "it's not just X, it's Y" and out comes "this is less about X than Y", the same move in a wig. Every rewrite therefore gets re-scanned, and the loop runs until a scan comes back clean.

Workflow: **Scan → Diagnose → Rewrite by meaning → Re-scan → (repeat) → Register check.**

## Phase 0: Fix the target

Before touching the text, establish:

- **Genre and venue.** Academic article, tweet, reddit post, LinkedIn, email, blog, docs, marketing. Ask when it isn't stated and isn't obvious from the text. Genre decides which tells are fatal and what "good" means; see [references/voices.md](references/voices.md).
- **Audience and stance.** Who reads it, and what the author actually claims. Slop fills the space where a claim should be, so you cannot remove it without knowing the claim.
- **Constraints.** Length limits, required citations, house style.

## Phase 1: Mechanical scan

Run the detection patterns from [references/tells.md](references/tells.md) against the text. If the text is in a file, or you can write it to a temp file, run the grep commands in that reference literally: the catalog is written as runnable `grep -Ein` patterns. Otherwise apply each pattern by hand, line by line.

Produce a finding list: line or sentence, matched pattern, tell category. Then run the two structural checks regex can't catch:

- **Cadence.** Flag any run of 3+ consecutive sentences within ±4 words of the same length, and any paragraph where every sentence has the same shape (subject, verb, elaboration).
- **Formatting.** Bold scattered through prose, emoji-decorated headers or bullets, "**Term:** definition" bullet lists, headers on a text too short to need them, a tidy intro-three-points-conclusion skeleton.

Report the findings to the user as a short table before rewriting: category, count, worst example. The user should see the diagnosis.

## Phase 2: Rewrite by meaning

Go finding by finding. The cardinal rule: never fix a pattern by paraphrasing the pattern. Decide what the sentence asserts, then assert that.

### The "not X but Y" family: three-way triage

Every negative parallelism gets exactly one of these treatments:

1. **Strawman negation** (nobody believes X). Delete the X half and assert Y directly, with whatever evidence the text has.
   - *"It's not just a tool, it's a fundamental shift in how teams work"* → *"Teams that adopted it stopped holding standups within a month."*
2. **Real contrast** (people genuinely hold X). Earn it: name who holds X, say concretely why Y beats it. A real contrast survives being made specific; slop doesn't.
3. **Empty claim** (the contrast decorates a sentence that asserts nothing). Delete the sentence. Most cases are this one.

Banned escape hatches, all the same move, all counted as new findings: "less about X than Y", "X matters, but Y matters more", "the real X is Y", "the question isn't X, it's Y", "X? Y." (the rhetorical-question variant), and the em-dash form "— not X, but Y".

### Everything else

- **Puffery and inflated vocabulary** (pivotal, seismic, testament, tapestry, landscape, delve…). Replace with the plain word, or with the concrete fact the puffery hides. "Plays a vital role in" → "does".
- **Rule-of-three lists.** Keep the strongest item, cut the rest. Where all three carry distinct information, keep them and break the rhythm with different lengths and different syntax.
- **False ranges** ("from X to Y"). If you can't name a meaningful midpoint between X and Y, name the two things plainly or cut one.
- **Hedged both-sidesing** ("it's worth noting", auto-counterpoints, "while X, it's also true that Y"). Commit. One opinion, stated, owned. A counterpoint stays only where the author genuinely concedes it.
- **Uniform cadence.** Vary deliberately. Follow a long sentence with a short one. Fragments are legal. Avoid formulas, since alternating long and short is its own tell; read the paragraph aloud and break wherever the rhythm goes metronomic.
- **Low specificity.** Replace "many companies" / "studies show" / "recent research" with actual names, numbers, and dates, drawn **only** from the source text, the conversation, or research you actually do. Never invent specifics. Where the author has to supply one, leave a marked placeholder: `[ADD: which study?]`.
- **Stock skeleton.** Kill throat-clearing openers ("In today's fast-paced world…"), summary conclusions ("In conclusion… Ultimately…"), and engagement-bait endings ("What do you think?"). Start where the point starts; stop when it's made.

### Overcorrection is also slop

- No fake typos, forced slang, or manufactured "voice". Humanizer-tool output is its own genre of slop.
- Em dashes stay legal. Humans use them. The tell is density, plus the contrast form "— not X, but Y". Budget: at most one em dash per ~150 words, never two in a sentence.
- Keep precision in academic and technical text. There, de-slopping means cutting puffery and committing to claims. Adding attitude makes it worse.
- Preserve the author's meaning, claims, and facts exactly. This is a style pass. Flag anything that looks factually wrong rather than silently fixing it.

## Phase 3: Verify loop

Re-run the full Phase 1 scan on your rewritten text. Expect the rewrite to carry fresh tells, because the model producing it has the same priors that produced the originals. Skipping this step is how slop survives the pass. Fix and re-scan until one pass returns zero pattern hits and the cadence check passes. Cap at 4 passes. If a pattern survives 4 passes, rewrite that sentence from scratch, starting from its bare claim: what fact or opinion is this sentence for?

## Phase 4: Register check

Check the clean text against its genre profile in [references/voices.md](references/voices.md): right length, right formality, right person, genre-specific tells gone. On reddit that means no bold and no bullet essay. In academic prose it means no first-person hot takes added. Then read it aloud. Anywhere you wouldn't say it to the actual audience, rewrite that sentence.

Deliver the rewritten text and a short change log: categories fixed, counts, and how many verify passes it took.

