## Context

See proposal.md for the failure and scope. The S3 migration closes storage classes over seven text values at backend/migrations/20261026120000_move_storages_into_module.sql:487. The same restriction exists in backend/api/openapi.yaml:4567 and backend/internal/modules/storages/internal/providers/s3/provider.go:67. Other unpublished migrations repeat enum checks for accounts, provider types, configs, backup catalogs, verification and health checks.

The enum rule has been recorded in backend/AGENTS.md under Migrations, and docs/architecture.md:167 now links to it. Structural data invariants remain in the schema. Implementation follows backend/AGENTS.md, frontend/AGENTS.md and the API-design/testing skills.

## Goals / Non-Goals

**Goals:** make the final unpublished schema follow the enum rule, retain valid provider-defined values across the real release upgrade, and keep application-owned enum validation effective.

**Non-Goals:** reinterpreting every historical value, changing structural constraints, or supporting intermediate unpublished schemas.

## Decisions

### Edit the unpublished migration chain, including dependent statements

Use the refactoring migration set introduced after v3.64.0, beginning with 20261024122431_drop_postgres_monitoring_tables.sql, through the final pre-fix migration. Inspect every Up and Down block for enum CHECK additions, drops, renames and inherited constraints; remove only value-domain checks. The released in-flight backup_type check disappears with its table during the existing merge migration; do not rewrite that released migration. Drop the inherited email security check in the notifier migration. Its Down block restores the released check under its original name so the earlier released rollback remains valid; the final upgraded schema has no such check. Keep composite foreign keys and structural checks for chain references, completed backup fields, numeric limits and required-field relationships.

Rejected: adding a new migration that drops constraints after the storage migration. Startup would still fail before reaching it. Rejected: removing every CHECK regardless of purpose; that would discard structural guarantees unrelated to enums.

### Keep normalization justified by application behavior

Review repairs next to each removed check. Retain conversions needed by the current application, including unknown application-owned status/retention handling and in-progress uniqueness repairs. Remove transformations whose sole purpose was satisfying a deleted enum constraint. Update migration tests to stop asserting rejection of arbitrary text by SQL, while retaining structural assertions and checking application rejection through existing module API surfaces. Adjust backup-configs upgrade wording to refer to application usability.

Rejected: deleting every repair with its constraint, which can leave unusable schedules or unsafe lifecycle state. Rejected: retaining closed S3 class normalization, which would change the selected remote storage policy.

### Treat S3 storage class as external text

Change S3StorageClass in the contract to an unrestricted string, allowing the existing empty/default meaning. Remove the provider's closed membership check and pass nonempty values unchanged. Replace the closed domain enum with a string representation; keep common choices as UI suggestions, not a validation list. The storage editor must retain custom values on rename and allow custom entry. Regenerate every affected server/client/type via mise run generate-api.

Rejected: adding COLD and DEEP_ARCHIVE to the enum. Other S3-compatible services can use other classes. Rejected: silently using STANDARD or the bucket default; that changes behavior and potentially cost.

### Test the real upgrade as well as isolated migrations

Extend previous-release storage fixtures and current-release assertions for a suggested class, COLD and DEEP_ARCHIVE. Create the settings through the old API and assert startup and exact class preservation through the new API. Add API read/edit coverage and frontend preservation coverage, without requiring a remote service to accept those arbitrary classes.

Rejected: SQL-only reproduction, because it cannot prove that previously accepted API values survive the full image upgrade.

Verify the wire behavior separately with MinIO behind a recording/failure-injecting HTTP proxy. An omitted or empty class sends no explicit storage-class override; COLD and DEEP_ARCHIVE reach the provider unchanged. Inject a nonretryable InvalidStorageClass response, assert the failed connection-test/upload outcome, read the unchanged saved class through the API, and assert no request substitutes a default or another class. These tests cover the default and rejection delta scenarios rather than merely checking that saving text succeeds.

## Risks / Trade-offs

- SQL accepts unknown application enum strings after the change -> validate through the application and retain representative invalid-input API tests.
- A removed constraint name remains in a later migration or Down block -> audit the complete unpublished chain and run fresh-install and upgrade migration tests.
- External services reject a class at write time -> retain that provider error; never retry with another class.
- Editing unpublished migrations changes their content hash -> recreate test databases; intermediate unpublished application databases are outside the release upgrade contract.

## Migration Plan

Apply this change before storage-switch-cleanup so its new migrations follow the recorded enum rule. Fresh installations and upgrades from the newest published release run the edited SQL chain. Existing published migration files remain unchanged. Run the isolated migration/API/UI checks, build the candidate image and run the real upgrade suite. Follow the existing one-way upgrade policy; rollback after deployment requires the pre-upgrade data snapshot.
