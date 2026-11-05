---
name: frontend-design
description: Apply Databasus design conventions when changing frontend UI layout, styling, controls or visible copy. Covers theme ownership, accessible interaction and responsive form presentation.
---

# Frontend design

Read [frontend-design.md](../../../docs/frontend-design.md) before changing presentation. Inspect the existing shared control and its callers before introducing a wrapper; AntD already supplies the primitives. Theme values stay in code, and the document explains their roles.

Use semantic presentation and preserve labels, keyboard focus, pending/error states and narrow layouts. Update every caller when a shared control contract changes.

Use the localization skill when copy or language-dependent output changes, FSD for placement/dependencies and the testing skill for checks. Verify touched behavior with screen tests; browser checks establish geometry, focus, font loading and reference images. The commands and environment are in [testing.md](../../../docs/testing.md#browser-ui-tests).
