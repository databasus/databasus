## Context

See proposal.md for scope. temp/refactor/plan.md:49 and :52 leave milestones 13 and 13c unchecked, while openspec/changes/archive/2026-10-30-legacy-teardown/tasks.md is complete.

The unused middleware is declared at backend/internal/modules/users/module.go:71, forwarded at internal/module/module.go:60 and implemented at internal/httpapi/access.go:62. Runtime routing uses GetAccessMiddleware. Stale helper comments appear in databases/module.go and backups/module.go, and stale verification claims appear in backups/internal/logical/logical_backup.go. Equivalent transitional comments remain in databases/internal/userdatabase/database_repository.go and workspaces/deletion_test.go. The deleted-helper rationale in users/internal/signin/session.go is already absent.

docs/architecture.md:303 states an unconditional test-dependency ban. backend/tools/archlint/production_dependencies.go:19 and openspec/specs/backend-code-checks/spec.md:138 already define two reasoned third-party exceptions.

Follow backend/AGENTS.md, root documentation consistency rules and the testing skill. This change has skip_specs: true because it changes no requirement.

## Goals / Non-Goals

**Goals:** remove the dead auth surface and make current documentation match runtime wiring and checks.

**Non-Goals:** broaden authentication cleanup or delete working APIs whose comments happen to be obsolete.

## Decisions

### Remove only the unused session-only adapter

Confirm there are no runtime callers with gopls references, remove all three GetSessionMiddleware declarations/implementations, and rewrite the access comments around the active contract guard. Retain GetAccessMiddleware, Authenticate and every HTTP access mark.

Rejected: keep a deprecated alias for a caller that no longer exists. Rejected: change auth behavior while doing dead-code cleanup.

### Update helper comments according to present use

Confirm that the old-session-test-helper rationale is absent and describe logical verification as current verification. For DeleteDatabase and SaveBackupConfig, remove stale helper claims while preserving methods still called by handlers or module wiring. Remove the database repository's obsolete cross-module cascade claim and the workspace tests' obsolete route descriptions. Search the migrated modules for related transitional comments.

Rejected: mechanically delete any method whose comment names old helpers; comments do not establish whether an API is dead.

### Describe the existing rclone exception accurately

In the architecture's automatic-check table, qualify the test-dependency restriction with a link to the exact allowlist, and explain that repository-owned test support is excluded from both binaries while the server intentionally retains Files.com SDK and Proton Drive bridge importers. Do not modify archlint's accepted importers or storage registry.

Rejected: delete the backends to make the prose true; the user explicitly retains rclone. Rejected: weaken the rule to accept any external importer.

### Keep the temporary plan's history intact

Check the archived tasks, then change the two completion boxes. Leave historical phase descriptions alone. The temporary plan is ignored and may remain an uncommitted Russian working document under the root language rules.

Rejected: rewrite the entire refactor analysis; the requested repair is bookkeeping.

## Risks / Trade-offs

- An unobserved middleware caller would stop compiling -> use gopls references, backend lint and the users/application API tests.
- A comment-only edit accidentally removes an active method -> preserve runtime signatures except the explicitly unused middleware.
- Overbroad documentation language permits new test imports -> name the two existing exceptions and retain all other checks.

## Migration Plan

No migrations, contract generation or behavior changes. Run affected package tests and backend formatting/lint, then leave the change ready for review. Spec generation is deliberately skipped.
