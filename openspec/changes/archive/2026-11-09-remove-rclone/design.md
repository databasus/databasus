## Context

See `proposal.md` for the problem and scope. The latest published release is [3.67.1](https://github.com/databasus/databasus/releases/tag/v3.67.1), which removes built-in Google Drive support. The recovery releases agreed for this change are 3.67.0 for Google Drive or both blockers, and 3.67.1 for rclone alone; they remain fixed when later releases appear.

Current implementation points that shape the change:

- `backend/internal/modules/storages/internal/providers/rclone/provider.go:10` imports every rclone backend. Registration is in `backend/internal/modules/storages/module.go:159`, and `backend/api/openapi.yaml:4628` includes its settings in the public union. `backend/api/openapi.yaml:4845` defines the variant.
- `backend/internal/app/app.go:74` checks storage upgrade eligibility before migrations at line 83. `backend/internal/modules/storages/internal/upgrade/eligibility_repository.go:23` inspects only Google Drive and returns immediately on that blocker at line 31.
- `backend/migrations/20251218123447_add_rclone_storages.sql:4` creates `rclone_storages`; `backend/migrations/20261026120000_move_storages_into_module.sql:73` renames it to `storages_rclone`. The existing Google Drive removal migration independently refuses its provider before dropping its settings tables.
- `backend/tools/archlint/production_dependencies.go:20` defines two third-party test-library exemptions, and line 94 applies them during production graph traversal. `docs/architecture.md:175` describes that exception.
- `backend/internal/modules/storages/internal/storage/service.go:128` and line 341 implement the rclone-specific admin restriction. `frontend/src/features/edit-storage/ui/edit/EditStorageComponent.tsx:214` renders its editor.
- `backend/e2e/upgrade/previous_release_storages_test.go:222` creates rclone in the successful upgrade fixture. `backend/migrations/storages_migration_test.go:196` also seeds it for a test that currently migrates to the latest schema. `backend/e2e/upgrade/google_drive_retirement_test.go:15` pins the 3.67.0 image; `backend/e2e/upgrade/retirement_snapshot_test.go:17` contains the reusable snapshot flow but selects that image directly.
- `README.md:82`, `website/app/(en)/storages/page.tsx:10` and `website/public/llms.txt:25` offer rclone and Dropbox. Comparisons also offer Dropbox without naming rclone, for example `website/app/(en)/pgdump-alternative/page.tsx:143`. The provider type list in `backend/internal/modules/storages/internal/providers/type.go:6` contains no independent Dropbox integration.

The design follows root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`. Module ownership and provider registration follow `docs/architecture.md` sections Modules and Providers; migration safety follows Migrations; API deletion follows HTTP API and the api-design skill. Frontend deletions retain slice ownership, existing design conventions and dictionary completeness. The only architectural policy change is removing third-party test-library exceptions, reflected in Shared code and its executable check. No instruction-file changes are needed.

## Goals / Non-Goals

**Goals:**

- Keep a refused installation usable by the indicated supporting release on its original volume.
- Detect every blocker from parent storage records without needing provider packages or decrypted settings.
- Remove both the production integration and its dependency exceptions; pin the result with executable checks.
- Preserve supported providers' secrets, streaming behavior and successful upgrade coverage.

**Non-Goals:**

- Generalize the two-provider recovery decision into a configurable retirement framework.
- Make direct, manually sequenced migrations provide the same rollback guarantee as normal application initialization.
- Require a fixed binary-size ratio or CVE count. Record comparable measurements without promising an unmeasured reduction.

## Decisions

### 1. Aggregate blockers inside the storages module before migrations

Extend the existing upgrade eligibility repository to read the presence of both parent types in one read-only query. Keep the public `storages.CheckUpgradeEligibility` entry point and the existing app call before the migration runner. Neither the generic migration runner nor unrelated modules need provider knowledge. No row is loaded through the runtime provider registry.

Use a small, explicit decision table and stable diagnostic order, Google Drive followed by rclone:

| Google Drive | rclone | Result |
| --- | --- | --- |
| absent | absent | Permit migrations |
| present | absent | Existing Google Drive guidance, 3.67.0 |
| absent | present | rclone guidance, 3.67.1 |
| present | present | Name both and require removal of both through 3.67.0 |

The combined diagnostic must distinguish the common recovery version from the individual removal history. It should say to roll back to 3.67.0, remove all Google Drive and rclone storages and retry, or continue using 3.67.0. It must not claim rclone disappeared in 3.67.1. Keep the existing Google Drive-only diagnostic unchanged; add the rclone-only and combined diagnostics in the upgrade package's `errors.go`.

Inspect every workspace, even when provider rows are absent or a storage is unused. No `storages` table means normal fresh initialization. A query error is a refusal, never permission to migrate. The shared app initializer continues to protect both the server and console commands. Existing newer-schema refusal stays in place.

**Alternatives rejected:** Sequential early returns hide the second blocker and lead to avoidable retries. Returning a generic unsupported-type error gives no usable recovery release. Moving the check into the last SQL migration can commit earlier pending migrations and make the requested return impossible. A configurable framework adds machinery for two fixed cases without improving behavior.

### 2. Give direct migrations an independent transactional guard

Add a new goose SQL migration. Before dropping any rclone settings table, check parent `storages` records for `RCLONE` and raise the rclone-only 3.67.1 recovery diagnostic. Drop both known settings-table names with `IF EXISTS`, without cascading to parent storages or other modules. Do not disable transactions. A down migration restores the empty module-owned settings schema and its matching constraints, not deleted credentials or compatibility with an older application.

Keep the historical create and rename migrations unchanged, along with the released Google Drive removal migration. Direct migration diagnostics remain provider-specific; only application initialization guarantees one combined diagnostic before all pending migrations. Clarify this distinction in the migration delta so preserving the released SQL does not contradict the combined-startup requirement.

Split migration coverage into historical transformation tests that stop at the appropriate boundary and full-upgrade tests that seed only supported providers. Preserve the historical encrypted-rclone-copy assertion in focused historical coverage instead of deleting that guarantee or weakening the removal guard.

**Alternatives rejected:** Automatically deleting parent rows destroys user data and bypasses cleanup rules. Leaving an empty settings table retains an unnecessary schema. Editing historical migrations breaks older upgrade paths. A schema-only refusal in the final migration is too late for normal startup rollback safety.

### 3. Delete the API variant and its whole implementation

Delete `RcloneStorageSettings` from the public union, discriminator mapping and schema, and remove rclone-specific permission text. Regenerate through `mise run generate-api`. Delete the provider package, settings conversion, registry entry, type constant, admin-only guard and provider-specific error. Remove provider-only tests, fixtures and parameters from cross-provider test matrices. Keep common streaming, retry, encryption, deletion and timing tests for remaining providers.

Use malformed-discriminator requests to test creation, update of a supported storage and the unsaved settings test through HTTP. The generated client can no longer describe rclone, so use the existing raw-request pattern rather than restoring a generated union member for tests. Assert `400 invalid_request`, unchanged supported records and no storage traffic for an authorized member and an instance admin. Authentication and ordinary workspace-access policies remain unchanged.

Remove the frontend variant, exports, defaults, validation branch, logo, editor and details. Remove dictionary entries in every existing interface dictionary. Update provider-choice expectations, resource fixtures and browser scenarios to cover the six remaining variants, including creation from backup settings. Keep current selection, secret omission, live language switching and read-only behavior.

**Alternatives rejected:** A hidden UI option leaves the API and dependency graph intact. A compatibility discriminator, external process or plugin retains integration maintenance and contradicts full removal. Keeping the admin restriction after removing the variant preserves dead policy.

### 4. Enforce dependency removal and eliminate exceptions

After deleting imports, run `go mod tidy` and inspect remaining consumers of dependencies that stay. Preserve account OAuth and every package still used by supported functionality. Do not remove dependencies by name alone or perform unrelated upgrades.

Delete the third-party test-importer allowlist and its conditional bypass. Replace the allowlist-positive fixture with rejection coverage. Extend production dependency checks for both server and verification-agent entry points to reject the rclone module namespace, including transitive imports, using the existing path-prefix matching. Add minimal fake-module fixtures for direct and transitive violations and verify that supported entry points still pass. These tests use synthetic modules and do not re-add rclone to the project dependency manifests.

Inspect the resulting production graphs and module build information. No production rclone packages or rclone-only adapter modules should remain. Build baseline and resulting server binaries under identical Go version, platform, flags and embedded frontend inputs; record their byte sizes. The dependency gate, rather than a comment or a strict size threshold, establishes removal. Run the existing Go vulnerability scan after cleanup without adding suppressions.

Update `docs/architecture.md` to state the unconditional production test-code rule. Keep the rclone namespace only where an executable prohibition needs it, not in a comment explaining history.

**Alternatives rejected:** Text search alone cannot prove a transitive dependency is absent. A comment can drift without failing a check. Leaving an unused allowlist preserves a security exception with no consumer. A hard binary-size threshold is sensitive to unrelated build inputs and does not identify the integration.

### 5. Use real release images for independent and combined recovery

Retain the Google Drive-only regression anchored to its existing 3.67.0 image. Add rclone-only coverage anchored to the actual 3.67.1 image, resolving and pinning its published image identity during implementation. Add combined coverage anchored to 3.67.0. General newest-release coverage seeds only the six supported storage variants.

Reuse the released API clients and retirement flow where they already fit. Extract shared setup only for repeated behavior, parameterizing snapshot helpers with the image that owns the volume rather than hardcoding 3.67.0. Use synthetic one-remote rclone settings accepted by the old API without running a connection test. The combined fixture can put the types in different workspaces to prove instance-wide detection.

For each scenario: create records through the released API, stop the old instance, capture records and migration history read-only, observe candidate refusal and non-zero exit, compare unchanged snapshots without printing secrets, restart the same release, sign in and verify records, delete unused blocking storages through its API, and start the candidate directly. Assert preserved supported records and encrypted credentials and absent obsolete schemas. The combined flow never starts 3.67.1. Retain unique resources, labels and unconditional cleanup, including failed starts.

**Alternatives rejected:** SQL-seeded upgrade fixtures do not prove the release can create or remove the resource. Following whichever release is newest loses the refusal scenario once it no longer accepts rclone. A combined-only test misses the 3.67.1 choice for rclone alone. Duplicating the snapshot infrastructure adds maintenance and can accidentally reopen a volume using the wrong image.

### 6. Replace public claims while retaining layout and current-state prose

Audit rclone, Dropbox, related configuration references and the broad provider-count claim across source and built public output. Replace unsupported names in prominent summaries with supported names, avoid duplicates within lists, and preserve surrounding element structure and approximately the original length. Update the English page and all six translations together, including FAQ text and its structured-data copy. Remove the standalone rclone entry from the storage-provider inventory rather than creating a second FTP or SFTP entry. No existing supported route needs removal or a redirect.

Cover home, MySQL and MongoDB landing pages, storage documentation, security examples, comparisons, metadata and `llms.txt`. Synchronize the English README and all six translated copies. Update architecture and glossary examples without inserting support-history or removal explanations. Preserve shared reader regression tests but remove their historical rclone comment.

Use automated source and static-output audits plus a focused frontend browser assertion that offers exactly the supported variants. Verify the rendered website summaries at narrow and wide widths during implementation; screenshots or rendered-size checks can be produced locally without making manual user actions a task dependency. Load the run-app skill if checking application pages.

**Alternatives rejected:** Removing only the name rclone leaves false Dropbox offers and provider counts. Adding deprecation notices contradicts the requested current-state documentation. Shortening whole sections unnecessarily can change landing-page layout. Rebuilding page structure or adding a replacement provider is outside this removal.

## Risks / Trade-offs

- **A supporting image cannot reopen the volume after refusal.** Keep the check before all migrations, prove byte-for-byte-equivalent persisted values and history through read-only snapshots, and restart the exact released image in each regression. Successful schema upgrades remain one-way.
- **Used storages cannot simply be deleted.** Recovery instructions require the operator to resolve references and cleanup through the supporting version. Do not bypass these rules or promise automatic backup transfer.
- **A dependency remains because another feature still uses it.** Verify actual consumers and production graphs after tidy; preserve account sign-in and supported providers. Record measured size rather than guessing a ratio or vulnerability reduction.
- **A translated or duplicated public claim survives.** Search all seven languages and static output, including Dropbox-only comparison text, metadata and structured data. Keep source and generated-public-output audits distinct from preserved OpenSpec history.
- **The original database is older than the selected supporting release.** The candidate still refuses without mutation; the operator then uses the indicated supporting release's own upgrade path. Do not claim that an older original binary can run after that recovery release has migrated its schema.

## Migration Plan

1. During implementation, capture the comparable binary baseline before deleting the provider. Add the pre-migration refusal tests and extend the check before removing runtime integration code.
2. Delete the API, backend and frontend variant, add the guarded schema removal and dependency enforcement, and update public materials. Run the tasks' targeted checks and the final verification commands against the finished implementation.
3. On normal startup, inspect both blocking types before applying any pending migration. If none exist, apply all pending migrations and start; no intermediate binary is launched.
4. If startup refuses, use the single recovery release named for all blockers, remove them through its interface or API, and retry the target directly. Refusal leaves persisted application data and migration history untouched.
5. After successful migrations, the ordinary newer-schema check continues to forbid starting an older binary. This change adds no post-upgrade downgrade path.

## Verification repair

The server command checks exposed a race in the existing healthcheck subprocess test: it queried the developer server before launching the subprocess, although that server could stop between the two requests. The test now verifies the exit code and diagnostic from one subprocess invocation together. The HTTP healthcheck tests continue to exercise successful, unsuccessful and unreachable servers deterministically.
