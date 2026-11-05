## 1. Reproduce the missing warning

- [x] 1.1 Add `frontend/src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.test.tsx` using the existing screen renderer. Add the scenario `warns that skipping host key verification reduces security when advanced settings are expanded`: open advanced settings while the skip option is unchecked and assert that the warning is visible and describes the checkbox. Before editing production code, run `pnpm --dir frontend test src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.test.tsx` and confirm it fails because the warning is absent.

## 2. Restore accessible localized help

- [x] 2.1 Render the existing `storages.edit.sftp.skipHostKeyTooltip` translation through `Form.Item help`, give it a component-unique ID and associate the checkbox through `aria-describedby`. Preserve the checkbox behavior and disclosure state. Verify that the regression passes with `pnpm --dir frontend test src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.test.tsx`.
- [x] 2.2 Extend the same screen test to cover immediate visibility when skipping verification is already enabled and parameterize visible warning text and accessible-description checks across all six locales using their dictionaries. Run `pnpm --dir frontend test src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.test.tsx` and verify every case passes.

## 3. Check the frontend integration

- [x] 3.1 Format only the changed frontend files with `pnpm --dir frontend exec oxfmt src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.tsx src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.test.tsx`. Verify formatting, lint, types and the production build with `mise run frontend:lint` and `pnpm --dir frontend build`.

## 4. Repair the verification failure

- [x] 4.1 Reproduce the hourly WAL statistics failure deterministically by moving the manual application clock two minutes behind its initial value before the existing wall-time assertions. Verify the intended failure with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestGetWalStreamStatistics/returns_sixty_consecutive_minute_buckets_when_the_caller_requests_an_hour' ./internal/modules/backups/physicalapitests/streamstatistics` from `backend/`, after sourcing `../scripts/fedora.sh` and calling `apply_libedit_shim`.
- [x] 4.2 Compare the last minute bucket with the manual application clock and retain the other series assertions. Verify the whole package with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/physicalapitests/streamstatistics` from `backend/`, with the same Fedora shim.
- [x] 4.3 Format the changed backend test and run `mise run backend:lint` to verify backend formatting, lint and contracts.
- [x] 4.4 Build a uniquely tagged candidate image from this checkout, then run the upgrade stage that the failed full run did not reach with `DATABASUS_IMAGE=<candidate-image> mise run backend:test-upgrade` and verify it passes.

## Change verification

```sh
mise run test
```
