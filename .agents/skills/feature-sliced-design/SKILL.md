---
name: feature-sliced-design
description: Place or reorganize Databasus frontend code across layers and slices, define public APIs, and repair forbidden imports. Use for frontend architecture and ownership decisions; ordinary copy or style changes alone do not need this skill.
---

# Frontend boundaries

Read the Frontend section of [architecture.md](../../../docs/architecture.md#frontend) and the local conventions in [frontend/AGENTS.md](../../../frontend/AGENTS.md) before changing placement or dependencies. Those project rules take precedence over the upstream methodology, including its alternatives for domain API placement, widgets, cross-imports and decision records.

Use the vendored [FSD methodology](upstream/methodology.md) to investigate placement and dependency problems. Load its conditional references only when the task needs them. Start with the closest real consumer, identify the responsibility that changes independently, and choose the smallest boundary that owns it. Follow the project's architecture when an upstream alternative differs.

Check the affected consumers and public entry points before a move. Keep mechanical moves separate from changes in behavior, preserve scenario coverage, and run frontend lint, relevant tests and the production build. A forbidden dependency has no exception: repair it.

The upstream text and references are vendored verbatim from Feature-Sliced Design contributors. [upstream.json](upstream.json) records the source revision, license declaration and content hashes; [LICENSE](LICENSE) contains the MIT terms. Keep attribution and the full source revision when updating those files.
