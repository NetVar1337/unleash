---
name: grill-me
description: "Use to stress-test a plan one question at a time (\'grill me\', \'challenge my plan\')."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: coding
  upstream: https://github.com/JuliusBrussee/skills
---

> Bundled with Unleash skills pack. Upstream: https://github.com/JuliusBrussee/skills

# Grill Me

Interview the user until the plan is clear, defensible, and ready for action.

This is not hostile debate. It is calibrated pressure. First find the user's knowledge level and desired intensity, then ramp questions to match.

## Core Rules

- Ask one question at a time.
- Give a recommended answer for every question.
- If the answer can be found by reading files, code, docs, issues, or logs, inspect those first instead of asking.
- Keep track of unresolved decisions, assumptions, risks, and dependencies.
- Do not over-grill domain basics when the user is still learning the topic. Teach the missing frame briefly, then ask the next useful question.
- Do not under-grill confident experts. If they know the terrain, pressure-test tradeoffs, edge cases, failure modes, and reversibility.
- Let the user change intensity any time with "softer", "harder", "teach more", or "skip basics".

## Phase 1: Frame The Target

Identify what should be grilled before asking about comfort. If the topic is not clear, ask:

> What plan, design, or decision should I grill?
>
> Recommended answer: give me the concrete goal, current approach, constraints, and what decision you need to make.

If context already contains the plan, summarize it in 3-6 bullets and ask for correction:

> I think target is: [...]
>
> Recommended answer: "Yes, grill that" or "Adjust: ..."

## Phase 2: Calibration

Before grilling the topic, ask a short calibration question unless the user's level is already obvious from context.

Ask:

> Before I grill the plan: what is your current comfort with this topic, and how hard do you want the pressure?
>
> Recommended answer: "I know the basics of [topic], but I want standard pressure. Explain missing concepts briefly, then keep pushing."

Use the user's answer to set two dials:

### Knowledge Level

- **New** - user lacks core vocabulary or model of the domain.
- **Working** - user understands basics and can discuss tradeoffs.
- **Expert** - user knows domain deeply and wants sharper critique.

### Pressure Level

- **Light** - clarify goals, constraints, and missing context.
- **Standard** - challenge assumptions, tradeoffs, and execution path.
- **Hard** - probe failure modes, edge cases, incentives, reversibility, and second-order effects.

If the user does not answer calibration, default to:

- Knowledge: **Working**
- Pressure: **Standard**

## Phase 3: Build The Decision Map

Create a private decision map while asking questions one at a time:

- Goal - what success means.
- User or customer - who this affects.
- Constraints - time, money, stack, team, policy, risk.
- Options - obvious alternatives and why current option wins.
- Dependencies - what must be true first.
- Risks - what breaks, gets expensive, or becomes irreversible.
- Validation - how user will know it worked.
- Rollback - how to undo or recover.

Do not dump the full map unless user asks. Use it to choose the next question.

## Phase 4: Question Ladder

Move through this ladder. Stop early if the plan becomes clear enough or user asks to stop.

### 1. Goal Fit

Questions:

- What outcome matters most?
- What would make this not worth doing?
- What problem are we solving, and for whom?

### 2. Constraint Reality

Questions:

- What hard constraint cannot move?
- What resource bottleneck decides the plan?
- What assumption would kill the plan if false?

### 3. Option Pressure

Questions:

- What are the top two alternatives?
- Why this approach over the boring one?
- What are you optimizing for: speed, quality, learning, cost, control, or upside?

### 4. Execution Path

Questions:

- What is the smallest useful version?
- What has to happen first?
- What can be deferred without harming the goal?

### 5. Failure Modes

Questions:

- How does this fail in production or real use?
- What edge case would embarrass the plan?
- What part is hardest to observe once it breaks?

### 6. Validation

Questions:

- What test, metric, screenshot, demo, or user behavior proves this works?
- What would you check before trusting it?
- What does done mean in observable terms?

### 7. Reversibility

Questions:

- What decision here is hardest to undo?
- What backup, migration, rollback, or escape hatch exists?
- What should be logged as an ADR or explicit tradeoff?

## Pressure Adaptation

### If Knowledge Is New

- Define one missing concept in 2-4 sentences before asking.
- Avoid jargon unless you define it.
- Ask fewer branching questions.
- Focus on goals, constraints, and first principles.
- Recommended answers should model good reasoning, not only give answer text.

### If Knowledge Is Working

- Ask normal tradeoff questions.
- Surface alternatives.
- Push for validation and smallest useful version.
- Challenge vague words like "simple", "scalable", "good", "clean", or "fast".

### If Knowledge Is Expert

- Skip basics.
- Ask sharper counterfactuals.
- Probe hidden costs, adverse incentives, migration paths, and long-term maintenance.
- Ask what evidence would change their mind.

### If Pressure Is Light

- Keep questions clarifying.
- Use supportive framing.
- Stop after top ambiguities are resolved.

### If Pressure Is Standard

- Challenge assumptions and tradeoffs.
- Keep moving until implementation path is concrete.

### If Pressure Is Hard

- Be direct.
- Name weak reasoning.
- Ask about unpleasant edge cases.
- Demand observable validation.
- Still ask one question at a time.

## Recommended Answer Format

Every question includes:

```text
Question: ...
Recommended answer: ...
Why it matters: ...
```

Keep "Why it matters" to one sentence.

## When To Stop

Stop grilling when one of these is true:

- User says stop.
- Plan has clear goal, constraints, chosen approach, validation, and next step.
- Missing information can only come from external research or code exploration.
- User's knowledge gap blocks useful grilling; switch to brief teaching and propose next learning question.

End with:

- Final decision or current best plan.
- Remaining open questions.
- Next concrete action.
- Risks to watch.
