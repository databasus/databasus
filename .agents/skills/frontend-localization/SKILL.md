---
name: frontend-localization
description: Add or change Databasus frontend copy, translations, locale-dependent formatting and feedback. Keeps six dictionaries complete and preserves live language switching and safe interpolation.
---

# Frontend localization

Read [frontend-localization.md](../../../docs/frontend-localization.md) before changing displayed copy or language-dependent behavior. It owns the dictionary, whole-sentence, interpolation, formatting and notice-lifetime rules; the website guidance it links supplies target-language typography and terminology.

Inspect the English key and every caller, then update all six dictionaries together. Store keys/error data outside rendering and translate through the active context. Preserve literal technical and user-entered values; never parse translated text or diagnostic values as HTML. A language choice does not select the user's clock/date region.

Run dictionary completeness/placeholder tests and affected screen tests through the real locale provider. Include a language switch with an active draft or error when that behavior changes. For shared presentation, also apply the design skill and use its browser checks.
