---
name: last-20-percent
description: "Use to finish the experiential last 20% agents skip (\'last 20%\', \'make it actually good\')."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: coding
  upstream: https://github.com/JuliusBrussee/skills
---

> Bundled with Unleash skills pack. Upstream: https://github.com/JuliusBrussee/skills

# The Last 20%

Agents reliably deliver the first 80% of a solution — the ingestion, the endpoints, the schema, the pipeline — and quietly skip the 20% that makes it something a person would love. Asked for an enterprise knowledge wiki, an agent builds the crawler, the embeddings, the LLM synthesis layer, and never once asks what a wiki page should *read like*, which pages should exist, or how someone finds an answer on day one. The result compiles, passes tests, demos fine, and is dead on arrival.

This skill exists to find that 20%, spec it as concretely as the plumbing, and finish it.

## Why the 20% gets skipped

The failure is structural, so the countermeasures must be too:

1. **Tasks come from the noun.** Decomposing "wiki" yields ingestion, storage, search, generation. The experience of *using* the wiki never appears in that decomposition — so it never becomes a task, so it never gets done.
2. **"Done" is defined by what's checkable.** A test can assert a 200 response. It cannot assert "this page answers the question in its first paragraph." Work without done-criteria silently drops off the plan.
3. **Taste feels like overreach.** Deciding what a page should read like is a product judgment. Models hedge: they build "flexible" machinery and defer every real decision to a user who asked precisely because they wanted those decisions made.
4. **The 20% is scheduled last, and last gets cut.** Even when the experiential work is named, it lands at the end of the plan — exactly where context, budget, and attention run out.

Each phase below counters one of these.

## The cardinal rule

**Never call a solution done until you have consumed its output the way the end user will — and it delivered the magic moment.** Reading your own code is not consuming the output. Passing tests is not consuming the output. Open the wiki and read three pages cold. Run the report and read the report. Load the dashboard and try to answer a real question with it.

## Phase 1: Write the magic moment

Before decomposing anything, write one concrete scene: a named person, a real task, what they see, and why they'd tell a colleague about it afterward.

Bad (a feature): "Users can search the wiki."
Good (a scene): "Sarah from legal types 'data retention' and lands on a page whose first paragraph answers her actual question, with a link to the policy doc it came from. She stops asking in Slack."

The scene is the destination for everything that follows. If you cannot write it, that gap *is* the missing 20% — stop and close it first. Propose two or three candidate scenes with a recommendation; never hand the user a blank page to fill in.

## Phase 2: Decompose the scene, not just the noun

Build two lists:

- **The noun list (the 80%).** What you get by decomposing the deliverable's name. Ingestion, auth, schema, API, deploy. Agents produce this list automatically; it needs no help.
- **The scene list.** Walk the magic moment step by step and write down everything that must be true for it to happen. What did Sarah type into, and how did she know to go there? What did the page look like when it loaded? Why was the answer in the first paragraph and where did the link come from? What would she have seen on day one, before ingestion finished?

**The last 20% is the scene list minus the noun list.** That residue is not random — it falls into recurring categories: content and voice, information architecture, first-run and empty states, opinionated defaults, the golden path, the failure experience, naming and microcopy. The full catalog, with probe questions and worked examples across domains, is in [references/last-20-catalog.md](references/last-20-catalog.md) — run the scene list against it to catch what the walk-through missed.

## Phase 3: Spec the 20% at plumbing fidelity

Every scene-list item gets done-criteria as concrete as "endpoint returns 200". Vague quality goals are how this work dies; convert each one into something checkable:

- "Pages should be readable" → "Every page opens with a 2–4 sentence direct answer before any detail; section headings are the questions readers actually ask; every claim links to its source document."
- "Good empty state" → "A search with zero results shows the three closest pages and a one-click 'request this page' action — never a bare 'no results'."

### Golden artifacts

When the system produces artifacts — wiki pages, reports, emails, dashboards, generated code — **hand-craft the best possible instance before building the generator.** Write the ideal wiki page yourself, with real content about a real topic from the actual corpus. That golden artifact becomes three things at once: the spec (this is what output looks like), the test fixture (compare generated output against it), and the quality bar (a generator that can't approach it isn't done).

A generator built without a golden artifact converges on generic filler, because "generate a wiki page" with no exemplar means "produce the statistical average of all wiki pages." And never use lorem ipsum or placeholder content in a golden artifact — fake content hides whether the structure actually works.

## Phase 4: Front-load, never append

In build mode, refuse to schedule the 20% last. Order the plan so the experiential layer survives:

1. Golden artifacts first — before the machinery that produces them.
2. Experience decisions (page types, defaults, voice, navigation) threaded through the build, decided at the moment the relevant plumbing is built, not batched for "polish week."
3. Keep the scene list visible as a **finish list** in the plan or tracker, each item with its done-criteria. Treat an unchecked finish-list item exactly like a failing test: the build is red until it's green.

## Phase 5: The walk

When the build claims to be done, become the end user — cold, no insider knowledge. Perform the magic moment literally: go where Sarah would go, type what she'd type, read what she'd read, all the way through.

Judge what you find against the golden artifacts and the finish list. Then write down every gap between the walk and the scene: pages that read like filler, a dead-feeling first run, a default that made you decide something the product should have decided. Fix, and walk again. The loop ends in one of two states:

- The walk delivers the magic moment → say so, show the evidence (the actual pages read, the actual flow taken), and stop.
- The remaining gaps need a human product decision → list them explicitly as decisions with a recommended option each, and stop there.

## Modes

- **Plan mode** — a build request just arrived. Run Phases 1–4 to shape the plan before or alongside the noun-list work, then Phase 5 before declaring done.
- **Audit mode** — a solution already exists ("is this actually done?", "why does this feel like a demo?"). Start with the walk (Phase 5) to gather evidence, backfill the magic moment and scene list (Phases 1–2), then spec and finish the gaps (Phases 3–4) and walk again.

## Boundaries

- **The last 20% is depth on the core scene, never breadth.** Adding features is the opposite failure wearing a finishing-touches costume. If an idea doesn't serve the magic moment, it goes to a someday list.
- **Bring a take.** Propose defaults, page structures, and voice with reasons. Reserve questions for decisions that are genuinely irreversible or genuinely business-strategic — and even then, attach a recommendation.
- **Don't gold-plate the plumbing.** This skill spends its budget on the experiential residue, not on rewriting working infrastructure to be prettier inside.
- **A solution is allowed to pass.** If the walk delivers the moment, say "this is done" and stop. Manufacturing gaps to look thorough is its own kind of slop.
