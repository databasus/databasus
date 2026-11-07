---
name: frontend-design
description: Apply Databasus design conventions when changing frontend UI layout, styling, controls or visible copy. Covers theme ownership, accessible interaction and responsive form presentation.
---

# Frontend design

Read [frontend-design.md](../../../docs/frontend-design.md) before changing presentation and follow the [UI composition and import rules](../../../frontend/AGENTS.md#ui-kit-and-icons). Inspect the existing shared control and its callers before introducing a focused component in `shared/ui`. Theme values stay in code, and the design document explains their roles.

Read the applicable official Mantine skill using the [frontend skill-loading rules](../../../frontend/AGENTS.md#mantine-skills). Project conventions take precedence over upstream examples. Use the [documentation lookup guidance](../../../docs/frontend-design.md#official-mantine-documentation) for version-matched API details.

Use semantic presentation and preserve labels, keyboard focus, pending/error states and narrow layouts. Update every caller when a shared control contract changes.

Use the localization skill when copy or language-dependent output changes, FSD for placement/dependencies and the testing skill for checks. Verify touched behavior with screen tests; browser checks establish geometry, focus, font loading and reference images. The commands and environment are in [testing.md](../../../docs/testing.md#browser-ui-tests).
