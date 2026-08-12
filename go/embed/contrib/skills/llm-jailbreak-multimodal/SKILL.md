---
name: llm-jailbreak-multimodal
description: "Multimodal jailbreaks: typography-in-image, adversarial pixels, screenshot policy text, PDF polyglots, audio stego."
version: 1.0.0
license: GPL-3.0-or-later
metadata:
  package: unleash-skills
  category: llm-redteam
  author: NetVar1337/unleash
  triggers:
    - "multimodal jailbreak"
    - "image prompt injection"
    - "typography attack"
    - "PDF injection"
---
# Multimodal jailbreaks

## Channels
- Vision-language: render instructions as image text / screenshots
- Adversarial perturbations (research stacks with white-box access)
- Documents: PDF with hidden text, tiny font, white text
- Audio: TTS instructions, spectrogram stego (if supported)
- Video frames / OCR pipelines

## Practical playbook
1. Confirm modality path actually reaches the model (not stripped).
2. Typography attack first (high reliability).
3. Combine image instruction + short benign user text.
4. For indirect: host image/PDF where RAG/fetch will ingest.

## Evaluation
OCR the image yourself; if you can't read it, model may not either.
