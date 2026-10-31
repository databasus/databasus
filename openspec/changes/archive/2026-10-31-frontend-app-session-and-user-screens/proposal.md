## Why

Application startup, workspace selection, session handling and user screens have overlapping owners. This makes ordinary changes risky: a late authentication failure can end a newer session, responses can overwrite newer form input, and the existing browser suite cannot establish that the built frontend signs in against the real backend.

Milestones 1 and 2 established architecture checks, shared presentation and browser UI fixtures. Milestone 3 uses that foundation to complete application composition and the user-facing account area in one change.

## What Changes

- Give app ownership of startup, active session, selected workspace, screen selection and version polling. Extract an app shell that receives content and actions rather than importing screens.
- Move authentication, OAuth callback, profile, users and instance settings into page slices. Preserve current tabs and the `/`, `/auth/callback` and `/storages/google-oauth` routes. Move the storage callback into a page slice mechanically; its OAuth redesign belongs to milestone 4.
- Separate session infrastructure from user-domain operations. Bind unauthorized-response handling to the originating session, make subscriptions safe to dispose, and clear session-owned screen state on logout or expiry without a reload loop.
- Bind GitHub and Google sign-in callbacks to an unpredictable, short-lived, single-use attempt in the initiating tab. Prevent duplicate code exchange and stale completion.
- **BREAKING:** OAuth sign-in attempts started before this update, including callbacks carrying the old provider-name state, are rejected with a way to restart sign-in. Delete constant-state handling; do not retain a compatibility path. Existing HTTP operations, persistent account data and routes are unchanged.
- Preserve current profile/settings drafts across saves and current user-search results across overlapping requests. Adopt the existing theme, localization, accessible-control and responsive-layout conventions in the migrated area.
- Share audit-event presentation between instance and user views while keeping authorization, filters, paging and scrolling at their existing owners. Reuse `useAuditEventPages`; workspace-view migration remains in milestone 4.
- Establish a real-API browser pilot using the production frontend and an isolated Go-owned test application. Provide a separate local `e2e` entry point and release-gating CI check, with verified failure propagation and cleanup.

## Capabilities

### New Capabilities

- `frontend-session`: Observable session transitions, isolation from obsolete requests, and browser-bound OAuth sign-in attempts.
- `frontend-application`: Recoverable startup, existing navigation and capability rules, stable language/theme switching, responsive shell and bounded version checking.
- `frontend-user-screens`: Account-flow continuity, profile/settings save acknowledgement, and current-generation user management with preserved audit views.

### Modified Capabilities

- `implementation-review`: Permit bounded UI and real-API browser verification commands while retaining read-only validation and rejection of unknown projects or extra arguments.

- `frontend-screen-testing`: Add an independently runnable production-build browser suite against an isolated real backend, including lifecycle-failure verification.
- `ci-pipeline`: Add the browser E2E check, preserve exhaustive non-overlapping test selection, and require its success before release.

## Impact

The change answers to root `AGENTS.md`, `frontend/AGENTS.md` and `backend/AGENTS.md`. Frontend changes cover app composition, the shell, account pages, shared session/transport infrastructure, audit-event UI, dictionaries, test ownership and shrinking migration baselines. Delete superseded internal paths and exports and update their callers together; keep no re-export shims.

Backend changes are test infrastructure under `internal/app`, reusing `apptest` and `testdb`; no production test endpoint or contract change is planned. Tooling changes cover the browser runner, backend test selection, `scripts/test.sh`, CI, review-command validation and its tests. Update architecture/testing/design/localization documentation and command references where their described behavior changes. Add the OAuth sign-in attempt to `docs/glossary.md`, distinguishing it from the server-side pending sign-in for emailed-code verification. `coverage.md` records preserved and added scenarios.

### Out of scope

- New deep links, a router/UI-kit replacement, a new visual identity, a global application store or a query-library migration.
- Rewriting workspace, storage, notifier, database, backup or verification workflows. Existing verification-agent UI stays reusable and is composed from the instance-settings page.
- Google Drive OAuth security/contract changes, SFTP credential semantics and workspace audit-view migration; these remain in milestone 4.
- Changing server-side account, password, two-factor, global-role or workspace-role policy, token persistence strategy, or introducing session revocation endpoints.
- Live third-party OAuth login in CI, a browser backup-engine matrix, broad toolchain upgrades, deployment, publishing or Git operations.
