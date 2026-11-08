## Why

Databasus styles and tests increasingly depend on Ant Design's internal markup, which makes responsive controls difficult to customize and screen tests expensive to maintain. Replace that dependency across the frontend while preserving the existing design, workflows and behavior.

## What Changes

- Complete the migration in one change, without a pilot release or a retained Ant Design implementation. Internal task ordering does not divide the agreed scope into optional stages.
- Use Mantine for interactive controls and overlays, semantic HTML and Tailwind for page layout, and focused shared components only for behavior or presentation reused by actual consumers.
- Preserve Databasus colors, typography, control density, responsive layout, translations, permissions, requests and editing-session policies.
- **BREAKING (frontend internal contracts):** remove `antd`, `@ant-design/icons`, their provider and theme contracts, library locale fields, imported component types, feedback APIs and internal CSS selectors. Update every consumer rather than keeping compatibility exports or wrappers. Replace standard icons with Lucide; retain provider, engine and navigation image assets.
- Replace library-internal test selectors with accessible names, roles and observable state. Retain meaningful behavior assertions, finite HTTP fixtures and real providers.
- Establish before/after coverage for every screen, provider editor, dialog and drawer. Supplement automated production-build comparisons with agent-operated browser inspection, including narrow layouts, enlarged text and keyboard behavior.
- After every migration evidence entry is complete, consolidate the added tests into maintainable regression coverage: retain unique behavior assertions and a small representative visual set, remove duplicate checks and migration-only capture infrastructure, and remove matrix IDs from executable tests and reference names. Keep the completed matrix and evidence mapping in this change.
- Vendor all three official Mantine skills and their referenced resources at a recorded revision, preserve attribution and the existing shared skill-discovery arrangement, and document when agents load each skill and consult the version-matched official documentation index.
- Update the frontend conventions, design and localization documentation, relevant test guidance and OpenSpec stack context together.

## Capabilities

### New Capabilities

None. This changes implementation and development tools without introducing a user-facing capability.

### Modified Capabilities

None. Existing requirements remain acceptance criteria, particularly `frontend-design-system`, `frontend-application`, `frontend-database-screens`, `frontend-database-connection`, `frontend-user-screens`, `frontend-workspace-screens`, `frontend-localization`, `frontend-session`, `frontend-screen-testing`, `frontend-code-checks`, `recurring-schedules`, `google-drive-storage-oauth` and `agent-workflow`. The change declares `skip_specs: true`; it must not invent behavior changes to justify the library replacement.

## Impact

- Governed by root `AGENTS.md` and `frontend/AGENTS.md`. No backend or marketing-site implementation change is planned. Browser end-to-end scenarios under `e2e/browser/` change only where their selectors depend on the removed UI library.
- Affects the application and test providers, shared theme and localization infrastructure, shared UI, shell, authentication, database workflows, storage and notifier providers, workspace and account screens, instance administration, frontend lint fixtures, screen tests and browser suites.
- Adds a matching Mantine package set and Lucide; removes `antd` and every `@ant-design/*` package from direct and transitive dependencies, manifests, lockfiles, imports and the generated build. The completed project must have no Ant Design dependency. Existing date/time and chart dependencies remain unless their actual consumers disappear during the migration.
- Adds official resources under `.agents/skills/`, retaining `.claude/skills -> ../.agents/skills` and Codex discovery from `.agents/skills/`. No personal or machine-global agent configuration is changed.
- Updates relevant `AGENTS.md`, repository-owned frontend skills, `docs/architecture.md`, `docs/frontend-design.md`, `docs/frontend-localization.md`, `docs/testing.md` and `openspec/config.yaml` references where the implementation changes their statements.

## Out of Scope

- A redesign, navigation or URL changes, altered product rules, API contract changes, backend refactoring, database migrations or new storage/notifier providers.
- A universal abstraction over every third-party component, replacement of ordinary HTML with layout components, a new generic table framework, or rewriting working domain state into a different form or request-state architecture.
- Live delivery through external providers for visual checks. Deterministic fixtures cover external authorization and delivery outcomes; existing real-backend end-to-end scenarios retain their own integration checks.
- Publishing, deployment, committing, pushing or modifying another agent's work.
