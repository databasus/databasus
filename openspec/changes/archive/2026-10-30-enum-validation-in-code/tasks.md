## 1. Reproduce the upgrade failure

- [x] 1.1 Add migration and storage API regressions for previously accepted COLD and DEEP_ARCHIVE classes, and an edit that changes only the storage name. Run `mise run backend:test-package ./migrations ./internal/modules/storages` on the unfixed code and record failures caused by the closed SQL/API lists.

## 2. Remove enum constraints from unpublished migrations

- [x] 2.1 Audit the migration set added after v3.64.0, remove scalar, array and fixed-discriminator enum checks, and update dependent drop/rename statements in both directions. Keep structural constraints and application-required repairs. Revise migration assertions accordingly. Verify fresh application of the chain and migration behavior with `mise run backend:test-package ./migrations`.

## 3. Preserve provider-defined classes end to end

- [x] 3.1 Change the S3 class contract to free text, remove provider membership validation and the closed domain enum, retain empty/default semantics and exact forwarding, then regenerate with `mise run generate-api`. Verify with `mise run backend:test-package ./internal/modules/storages ./internal/modules/storages/internal/providers/s3` and `mise run backend:lint`.
- [x] 3.2 After task 3.1 removes the S3 restrictions, check application-owned enum validation through the affected public API surfaces, add missing representative rejection coverage only where validation was relying on SQL, and preserve all structural failure assertions. Verify with `mise run backend:test-package ./internal/modules/users ./internal/modules/workspaces ./internal/modules/notifiers ./internal/modules/storages ./internal/modules/databases ./internal/modules/backups ./internal/modules/verification ./internal/modules/healthcheck`.
- [x] 3.3 Add observable provider/API tests for omitted and empty classes selecting the bucket default without sending an explicit replacement class, exact COLD and DEEP_ARCHIVE forwarding, and a nonretryable InvalidStorageClass response that reports failure and leaves the saved class unchanged without sending another class. Use MinIO behind a recording/failure-injecting HTTP proxy; inspect actual outbound requests and read the saved settings through the API. Verify with `mise run backend:test-package ./internal/modules/storages ./internal/modules/storages/internal/providers/s3`.
- [x] 3.4 Allow custom classes in the storage editor, keep common suggestions and preserve an existing custom class on other edits. Add screen coverage using the typed network and verify with `pnpm --dir frontend test src/features/storages/ui/edit/EditStorageComponent.test.tsx` and `mise run frontend:lint`.

## 4. Check the released-image upgrade and finish

- [x] 4.1 Extend previous-release storage fixtures and current-release checks to cover a suggested class, COLD and DEEP_ARCHIVE through the old and new APIs. Build the candidate image with `docker build --tag databasus-candidate:local .`, export DATABASUS_IMAGE to that tag and DATABASUS_PREVIOUS_IMAGE to the newest published v* tag's databasus/databasus image in the shell that runs final verification, and run `mise run backend:test-package ./e2e/upgrade/...`. Confirm actual startup and exact preservation; do not count compilation as an upgrade test.
- [x] 4.2 Confirm the already-recorded backend enum instruction and architecture link agree with the final SQL, update remaining live documentation references without rewriting archives, and run `mise run backend:format`, `mise run frontend:format`, `mise run lint` and all Change verification commands with the upgrade image variables set. Resolve mandatory review findings and leave implementation, documentation and tasks ready to commit.

## Change verification

```sh
mise run backend:test-package ./migrations
mise run backend:test-package ./internal/modules/users ./internal/modules/workspaces ./internal/modules/notifiers ./internal/modules/storages ./internal/modules/databases ./internal/modules/backups ./internal/modules/verification ./internal/modules/healthcheck
mise run backend:test-package ./internal/modules/storages/internal/providers/s3 ./internal/modules/storages/internal/providers
mise run backend:test-package ./e2e/upgrade/...
pnpm --dir frontend test src/features/storages/ui/edit/EditStorageComponent.test.tsx src/features/storages/ui/StorageComponent.test.tsx
```


## Reproduction evidence

On the unfixed code, `mise run backend:test-package ./migrations ./internal/modules/storages` failed for both COLD and DEEP_ARCHIVE. Migration application violated `ck_storages_s3_s3_storage_class`; API creation returned 400 `invalid_request` with `settings.s3StorageClass: is not an S3 storage class`. The rename assertions remain behind creation and will run after the restriction is removed.

## Application validation audit

User roles and workspace roles already reject unsupported values through their API tests. Verification config tests reject unknown schedule types and notification triggers. Health-check and lifecycle statuses are produced by application transitions rather than accepted as editable request values. Provider registries reject unsupported discriminators. Added representative API rejection coverage for webhook methods, Azure authentication methods, PostgreSQL SSL modes and physical backup types, and backup config encryption, schedules, retention and notification triggers. All affected module packages passed after removing the SQL restrictions; no additional production validation was needed.

## Final verification results

- `mise run backend:format` and `mise run frontend:format` completed; `mise run lint` passed across all modules.
- All Change verification backend packages passed with race detection, including the provider registry package whose fixture changed. The migration package also verifies rollback and reapplication of the reversible refactoring chain.
- Both storage screen test files passed: 12 tests covering custom entry, rename preservation and exact display, including property-like names.
- `docker build --tag databasus-candidate:local .` completed. The latest published version tag was verified against the upstream tag list as v3.64.0.
- With `DATABASUS_IMAGE=databasus-candidate:local` and `DATABASUS_PREVIOUS_IMAGE=databasus/databasus:v3.64.0` exported, `mise run backend:test-package ./e2e/upgrade/...` passed: the upgrade package ran for 74.137 seconds and upgradecheck for 1.011 seconds. The old API read back STANDARD_IA, COLD and DEEP_ARCHIVE before the upgrade; the new API preserved each class exactly after actual startup. The suite also restored the saved backups.
- The final backend and frontend verification runs used those image variables. `openspec validate enum-validation-in-code --strict` passed.
