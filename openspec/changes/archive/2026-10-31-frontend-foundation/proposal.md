## Why

The frontend refactor needs checks that prevent new dependency violations and tests that expose unexpected requests and stale list responses. The existing typed client, screen tests and paging controller provide that foundation, but gaps in their isolation and error handling can conceal regressions during the later domain moves.

## What Changes

- Establish the frontend architecture and its rationale in `docs/architecture.md`, keep local conventions in `frontend/AGENTS.md`, and add the upstream FSD skill with its license, attribution and pinned revision. Update the architecture review prompt to read the new source of architectural rules.
- Add a separate FSD check to `frontend:lint`, with fixtures proving import enforcement and a finite baseline of existing violations. **BREAKING:** rename `entity` to `entities`, update every internal consumer, and remove the old import paths without an alias or compatibility barrel. Configure `@/` consistently across tools; defer domain restructuring to later milestones.
- Strengthen the existing screen renderer and typed network mocks: real configurable providers, explicit request declarations, request inspection, empty/error/network/delayed responses, independent unexpected-request failure and cleanup between tests. Replace blanket pending handlers in both database screen test files.
- Remove unused `apiHelper`, `RequestOptions`, `RateLimiter`, `useTemporaryVisibility` and URL validation exports after mapping useful transport guarantees to the active client. Keep email validation. **BREAKING:** remove the unused exports outright, with no wrappers.
- Normalize active-client network and malformed-response failures without automatic mutation replay. **BREAKING:** remove the free-text API error fallback; unknown or missing codes show a translated general message. Preserve technical diagnostics explicitly allowed by known contract errors.
- Share cursor collection across workspace, database, storage and notifier listing operations, and repair the existing paging lifecycle for reset, reload, refresh, next-page requests and unmount. Preserve explicit domain requests, deduplication and the loaded tail where it remains valid.
- Document frontend testing, align frontend CI with the mise-pinned toolchain, inventory hook lint rules, and prepare the review gate for the planned isolated browser UI command without introducing browser tests in this change.
- Reproduce and fix bugs discovered during this work, including flaky tests and insufficient timeouts, in the same change. Update affected artifacts and verification; do not weaken assertions, hide failures with retries or defer discovered defects.

## Capabilities

### New Capabilities

- `frontend-code-checks`: enforce frontend dependency boundaries and a shrinking baseline without disrupting existing lint, type and generated-code checks.
- `frontend-screen-testing`: make screen-test providers, network declarations and per-test cleanup reliable and independently verifiable.
- `frontend-api-requests`: handle successful, refused, malformed and unreachable HTTP requests consistently without implicit retries or import-time request machinery.
- `frontend-list-loading`: collect complete cursor lists and keep visible paged lists consistent across concurrent requests and lifecycle changes.

### Modified Capabilities

- `frontend-localization`: replace backend-message fallback with translated general errors while preserving explicitly permitted technical diagnostics.
- `backup-list-paging`: define when a first-page refresh retains the loaded tail and when an unalignable, empty or complete page replaces it.
- `ci-pipeline`: use the repository-pinned frontend runtime and package manager for frontend lint, build and tests.
- `implementation-review`: accept the bounded future browser UI verification command and read frontend architecture from its authoritative document.

## Impact

The change answers to root `AGENTS.md` and `frontend/AGENTS.md`. It affects frontend source and tests, frontend configuration and dependencies, `mise.toml`, the frontend jobs in `.github/workflows/ci-release.yml`, architecture/testing documentation, the FSD/testing skills, `.agent-hooks/architecture-prompt.md`, review-gate command parsing and its tests, and `openspec/config.yaml`. Existing action versions remain pinned to full commit SHAs. The HTTP contracts, generated schema content and backend business operations do not change.

Add Steiger and its FSD plugin only after a local compatibility check against the pinned toolchain, and add Testing Library user-event. Keep Oxlint, Vitest/MSW, node tests and happy-dom screen tests. A supporting `coverage.md` maps retained, moved, replaced and deliberately removed test scenarios; the historical 595-test count is an inventory, not a target that preserves obsolete assertions.

### Out of scope

- Domain screen moves, flattening existing nested domain groups, session/navigation redesign, new routes and new product features.
- Theme token redesign, general form patterns, typography, layout changes and translation-workflow restructuring.
- Playwright installation, browser suites, screenshots, accessibility scans and real-backend browser orchestration; only review-command preparation belongs here.
- Backend implementation, HTTP operation/schema changes, engine matrix runs and changes to the website CI runtime.
- A replacement UI kit, router, query library, global state store, CRUD factory, generated HTTP client or broad toolchain upgrade.

These exclusions do not defer bugs actually exposed while implementing or checking this change; a necessary repair is reproduced, implemented, reflected in the artifacts and verified in the same change.
