## Why

Frontend source mixes relative imports with the existing `@/` alias, so contributors have no single convention to follow. Migrate the existing source first, then make frontend lint enforce that convention without changing application behavior or dependency boundaries.

## What Changes

- Use `@/` for every literal module reference from frontend application source to a target within the application source root, including sibling imports, type imports, re-exports, dynamic imports and side-effect stylesheet imports.
- Replace existing relative references mechanically before introducing enforcement. Preserve each reference's target, imported symbols and runtime evaluation order.
- Reject new relative references within that scope through the existing frontend lint command, with diagnostics identifying the file and rule.
- Preserve relative references in tooling outside the application source root, references from source tests to tooling outside that root, CSS resource URLs and module-relative resource URLs.
- Define the coding convention once in `frontend/AGENTS.md`; update `docs/architecture.md` to explain its reason and link to that rule.
- Keep the migration and enforcement as separate reviewable phases suitable for the requested two commits. Git operations remain outside the implementation task list.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-code-checks`: Frontend lint requires the configured source-root alias for internal module references while retaining dependency checks and the existing verification surface.

## Impact

- Governed by root `AGENTS.md` and `frontend/AGENTS.md`.
- Changes application source imports and re-exports, frontend lint configuration or its existing source scanner, lint regression tests, and the convention's documentation.
- Reuses the configured alias and installed tooling; no new runtime or development dependency is required. Make the existing source alias available to the tooling TypeScript project, isolated test runs and browser-test loader because they consume source modules.
- **BREAKING for source contributions:** relative references within application source stop passing lint. The mixed convention is removed rather than retained behind a compatibility option. Application behavior, HTTP contracts, stored data and external consumers do not change.

## Out of Scope

- Backend and marketing website imports.
- Adding aliases for tooling modules or rewriting the infrastructure's own imports. Loaders that consume source modules receive the existing source alias where needed.
- Moving modules, redesigning public entry points, changing layer dependencies or converting internal references into imports through their own barrel.
- Changing CSS resource URLs, module-relative resource lookup, generated API output or test-fixture source strings that intentionally exercise relative imports.
- Changing product behavior, translation copy, import extensions or unrelated lint rules.
