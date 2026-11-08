## Why

The database page spreads each responsibility across `ui`, `model`, `lib` and `api`, so changing WAL monitoring requires navigating four directory trees. Its model also separates small related types into many files; bringing each responsibility together will make changes easier to locate and review without changing application behavior.

## What Changes

- Organize the existing `pages/databases` slice into local areas: `wal-monitoring`, `physical-backups`, `logical-backups`, `verification`, `healthcheck`, `backup-config`, `settings`, `create-wizard` and `restores`. Keep page composition and genuinely shared page-local presentation outside these areas.
- Move each area's requests, model, supporting logic, components and existing tests together. Preserve the slice's public entry point and direct internal imports.
- Consolidate related generated-schema aliases in domain-named contract files. Keep small view types with their owning unit and enum label mappings with their corresponding enum rather than imposing one large type file on every area.
- Complete relocation and type consolidation as separate implementation passes so reviewers can distinguish path changes from declaration grouping. Preserve runtime statements and test scenarios.
- Document the database page's local layout in `docs/architecture.md`, without changing dependency rules or creating a new layer.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is a structural refactor with no spec-level behavior changes; `.openspec.yaml` declares `skip_specs: true`. Existing database screen, connection, request lifecycle, backup, restore, verification and healthcheck requirements continue to apply.

## Impact

- Implementation is confined to `frontend/src/pages/databases/`, references to its moved modules, and the relevant description in `docs/architecture.md`.
- The change follows root `AGENTS.md`, `frontend/AGENTS.md`, the frontend architecture, and the `feature-sliced-design` and `testing` skills.
- Internal import paths change and old files are removed; all repository consumers are updated together, without compatibility re-exports. The public `@/pages/databases` entry point remains unchanged.
- No HTTP contract, generated API schema, dependency, translation, URL or user-visible behavior changes.
- Verification covers frontend formatting, lint, type checking, production build and existing database page tests. No new scenarios or file-count target are required.

## Out of Scope

- Extracting widgets, features or entities, or introducing checks between the page's internal areas.
- Refactoring request lifecycles, hooks, algorithms, forms, component props or permissions.
- Consolidating aliases or labels elsewhere in the frontend, or relocating existing entity-owned contracts into the page.
- Adding a generic `types.ts`, a barrel for every directory, empty technical subdirectories, or import compatibility shims.
- Changing agent instructions, the FSD skill, tooling configuration or product specifications.
- Committing, merging or pushing as part of implementation.
