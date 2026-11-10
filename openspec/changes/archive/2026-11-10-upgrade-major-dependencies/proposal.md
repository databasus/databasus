## Why

Several dependencies remain on older major lines, including the deprecated MongoDB Go driver v1, frontend build and test tooling, routing and release tools. A bounded migration plan will move them to suitable maintained lines while preserving existing application behavior and following `docs/how-to-upgrade-deps.md`.

## What Changes

- Use the completed dependency-maintenance guide, native locks and existing offline check. The implementing agent reviews release age, official migration guidance and lock diffs, and reports exceptions; this migration adds no dependency-selection or verification framework.
- Migrate the frontend build/test group to Vite 8, its compatible React plugin 6 and Vitest 5. Retain MSW 2 and its existing HTTP-boundary test harness; resolving the MSW 3 test-environment incompatibility is outside the accepted final scope.
- Migrate React Router 7 to 8, including compatible same-major React patches required by peer dependencies. Retain i18next/react-i18next 26/17. Preserve existing URLs, navigation, session resets, language switching and safe interpolation.
- Move website TypeScript to 6.0 and keep frontend TypeScript on one compatible 5.9 compiler. Preserve the classic compiler API required by repository-owned linters; frontend TypeScript 6 and native TypeScript 7 are outside this change.
- Migrate the MongoDB Go driver to v2 on a release that retains the accepted MongoDB 4.2 minimum, JWT v4 to v5, and the logging API/SDK to their coordinated stable OpenTelemetry line. Replace the pinned development gRPC build with an eligible stable release.
- Migrate pnpm 9 to 12 and Helm 3 to 4, update the website's ESLint 9 to 10 with a compatible same-major Next.js lint configuration, and align runtime type declarations with the selected Node line.
- Move Go to 1.27, coordinating the builder images, native locks and required development tools. Retain Node 24.21.0 and Python 3.14.7 with their existing runtime artifacts and Python tooling.
- Upgrade external CI actions with an eligible maintained successor major, and replace the archived release-creation action with the already pinned GitHub CLI while preserving release gates, generated notes and permissions.
- **BREAKING (development and library configuration):** remove superseded package imports, old package-manager metadata and migrated tool configurations. Contributors install the new pinned tools; no old-version aliases or parallel toolchains are retained. No public HTTP, URL, persisted-data or supported-server break is intended.

## Capabilities

### New Capabilities

None. This is dependency and tooling maintenance.

### Modified Capabilities

None. Existing navigation, session, localization, backup, logging and CI behavior must continue to satisfy the main specifications. `skip_specs: true` records that the migration changes implementation dependencies rather than requirements.

## Impact

Affected areas include frontend dependencies and tooling, the website's lint/type configuration, backend MongoDB and session-token integrations, log export, test clients and fixtures, mise pins and locks, Docker build inputs, browser environments, CI actions and Helm packaging.

The change answers to root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` and `website/AGENTS.md`, plus `assets/readme/AGENTS.md` if a published requirement changes. Implementation must retain module boundaries, the frontend compiler-based checks and the real-engine test matrices.

### Out of scope

- Node 26, Python 3.15, frontend TypeScript 6, MSW 3 and i18next/react-i18next 27/18. The user accepted completion with their existing compatible lines retained after reviewing the release-age and compatibility constraints in `design.md`.

- New dependency-discovery scripts, release-evidence stores, lock parsers or generic verification frameworks.
- New Helm, website-export or release-command test harnesses solely for this upgrade. Use native commands, existing tests and agent review; add focused regression coverage only for a concrete defect or an uncovered compatibility requirement.
- Upgrading every package merely because a newer version exists; same-major updates are included only when a migration requires them or resolves an exposed defect.
- TypeScript 7 and its compiler-API redesign, a router framework/SSR migration, a website redesign or a package-manager conversion of the npm-based website.
- Dropping MongoDB 4.2 or any other accepted database version to accommodate the newest driver.
- New database-server release lines, changing the embedded PostgreSQL major or rebasing the runtime distribution.
- Changing session-token format or invalidating existing sessions, weakening lint/security checks, publishing artifacts or requiring a push to verify the work.
