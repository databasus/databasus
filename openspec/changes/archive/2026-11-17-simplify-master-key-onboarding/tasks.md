## 1. Contract

- [x] 1.1 In `backend/api/openapi.yaml`:
  - keep `PUT /master-key-acknowledgments` and `MasterKeyAcknowledgment` as they are on `HEAD`, restoring them where this change deleted them;
  - replace the body of `MasterKeyExportCreate` with an optional `code` and drop `isInsecureTransferAcknowledged`;
  - add the required `isMasterKeyExportCodeRequired` to `InstanceBackupStatus`;
  - rewrite the descriptions of `requestMasterKeyConfirmation`, `createMasterKeyExport`, `downloadMasterKey`, `acknowledgeMasterKey` and `isMasterKeyAcknowledged` to the new rules: the export needs a code only when the instance requires two-factor sign-in, the download spends its token and records nothing, and only the acknowledgment operation records that the current key is saved.

  Then run `mise run generate-api`. Verify with `git diff --stat` that only the contract and the generated files changed, and that `rg -n "isInsecureTransferAcknowledged" backend/internal/modules/instancebackups/internal/httpapi/api.gen.go frontend/src/shared/api/schema.gen.ts` finds only the recovery field. The backend compiles again only after group 2.

## 2. Backend

- [x] 2.1 Add `IsTwoFactorAuthRequired(ctx)` to `users.Module` in `module.go`, reading the instance setting through the `users` settings service. Verify with `go -C backend build ./internal/modules/users/...`. The predicate has no HTTP operation of its own in `users`, so its behavior is pinned through the status test in 2.3, which reads `isMasterKeyExportCodeRequired` before and after two-factor sign-in is switched on.
- [x] 2.2 Rework the export in one step, so the backend compiles at the end:
  - in `instancebackups/internal/keyexport`, make the export require root admin, plus a non-empty code passed to `VerifyIdentityProof` as a code-only proof when `IsTwoFactorAuthRequired` is true, and drop the HTTP check;
  - make the download spend its token, check the token's user is still root admin and write its audit event without recording an acknowledgment;
  - keep `Acknowledge` and `configs.AcknowledgeMasterKey` as on `HEAD`: root admin only, the digest of the current key and one audit event in the same transaction;
  - delete `ErrInsecureTransferNotAcknowledged`;
  - update `internal/httpapi/master_key_handler.go`, keeping the acknowledgment handler;
  - fill `isMasterKeyExportCodeRequired` in the status service and its handler from the same predicate.

  Verify with `go -C backend build ./...` and `mise run backend:lint`.
- [x] 2.3 Update `backend/internal/modules/instancebackups/master_key_test.go` and verify that `go -C backend test -count=1 ./internal/modules/instancebackups/ -run MasterKey` passes. Name every new top-level test `Test...MasterKey...` so the filter matches it. The tests cover:
  - without two-factor sign-in, export needs no code, and the downloaded bytes equal the installed key under `attachment; filename=databasus-master-key.txt`, with the installed key unchanged;
  - with two-factor sign-in, a missing, wrong, expired or reused code, a code issued for a recovery, and a password-only body are each refused with no token issued; a valid code works;
  - another administrator gets 403 for the export and for the acknowledgment;
  - a reused token is refused;
  - a spent token leaves the key unacknowledged;
  - the acknowledgment marks the current key saved in another session;
  - a replaced key is unacknowledged again;
  - the status reports `isMasterKeyExportCodeRequired` false and then true when the instance switches two-factor sign-in on.

  Remove the tests of the HTTP check.
- [x] 2.4 Update `exportMasterKey` in `backend/internal/modules/instancebackups/e2e/source_installation_test.go` to the new request body. Verify with `go -C backend test -count=1 -timeout 20m ./internal/modules/instancebackups/e2e/...`.
- [x] 2.5 Search with `rg -n "IsInsecureTransferAcknowledged|DownloadAndAcknowledge|RecordMasterKeyAcknowledgment" backend --glob '!*.gen.go'` and verify that only recovery's own insecure-transfer field remains. Then verify that `go -C backend test -count=1 ./internal/modules/users/... ./internal/modules/instancebackups/` passes.

## 3. Frontend

- [x] 3.1 Rework the export UI and its tests in one step, so the type check runs after every file it reads is updated:
  - in `frontend/src/entities/instance-backups/api/masterKeyApi.ts`, make `createExport` take an optional code, add a fetch of the key text by token, and keep `acknowledge`;
  - rewrite `frontend/src/features/export-master-key/ui/MasterKeyExport.tsx`: obtain the key at once or after the code and "Continue"; show a preview with only the first and last four characters; offer "Copy" through `shared/lib/BrowserClipboard` and "Download" as a `Blob` named after the export's `fileName`; report each successful copy or started download through a callback; on a failed code request, show the translated error with `MasterKeyHostInstructions` and the key's path; no HTTP warning or password field;
  - export `MasterKeyHostInstructions` from `frontend/src/features/export-master-key/index.ts`;
  - replace `MasterKeyOnboardingDialog.tsx` with a full-screen `MasterKeyOnboardingScreen` in the sign-in page's frame: logo, language and theme control, one paragraph about what the key is for, the export, and "I've saved it in a safe place", disabled until the export reports a copy or a download, which calls the acknowledgment and refreshes the status; no "later" or sign-out action;
  - in `frontend/src/app/ui/ApplicationScreen.tsx`, render that screen instead of `AppShell` while the status reports root admin's key unacknowledged, with the usual loader until the status is known; drop the dialog and the credential-expiry state from `InstanceBackupStatusWidget.tsx`;
  - delete `model/masterKeyOnboardingDeferral.ts` and the `master-key` kind in `selectFloatingNotice.ts` and `FloatingBackupNotice.tsx`, keeping the rule that an unacknowledged key suppresses the setup reminder;
  - in `MasterKeyPanel.tsx`, show the saved status, `MasterKeyHostInstructions`, and `MasterKeyExport` only after "Copy or download the key", through the feature's public API;
  - update every dictionary in `frontend/src/shared/i18n/locales/` following the `frontend-localization` skill: delete the keys for the password, the HTTP warning, the "later" action and the master-key floating notice; add the copy, preview, close and "Copy or download the key" copy and the failed code delivery; replace the onboarding description with the single paragraph;
  - rewrite `frontend/src/widgets/instance-backup-status/ui/MasterKeyOnboarding.test.tsx` to cover: the onboarding step replaces the application and offers no way to skip it, and no floating master-key notice appears; without two-factor sign-in the key is obtained without a code and the preview hides the middle of the key; "I've saved it in a safe place" is disabled until a copy or a download; a copy puts the whole key on the clipboard; a download saves `databasus-master-key.txt`; the close button sends the acknowledgment and opens the application; the code branch appears with two-factor sign-in; a failed code request shows the error, the path and the copy instructions while the step stays in place;
  - update the master-key panel test in `frontend/src/pages/instance-settings/ui/instance-backups/InstanceBackupsSection.test.tsx`: saved status, copy instructions with the path, and no export request before "Copy or download the key";
  - update `FloatingBackupNotice.test.tsx` and the status fixtures in `RecoveryPauseNotice.test.tsx`, `WorkspaceNavigation.test.tsx` and `InstanceBackupsSection.test.tsx`;
  - update the UI browser fixtures and scenarios in `frontend/browser/instanceBackupFixtures.ts` and `frontend/browser/scenarios/instanceBackups.spec.ts`, so the onboarding scenario checks that the step replaces the application, copies the exact key without the Clipboard API and opens the application after "I've saved it in a safe place".

  Verify with `pnpm --dir frontend typecheck`, `pnpm --dir frontend lint`, `pnpm --dir frontend test -- src/widgets/instance-backup-status src/features/export-master-key src/pages/instance-settings src/app/ui/WorkspaceNavigation.test.tsx` and `pnpm --dir frontend test:browser --project=ui`.

## 4. Browser end-to-end

- [x] 4.1 In `e2e/browser/scenarios/application_screens.py`, replace `defer_master_key_onboarding` and `acknowledge_master_key` with one helper that downloads the key from the onboarding step, presses "I've saved it in a safe place" and waits for `isMasterKeyAcknowledged`. Update `test_critical_path.py`, and change the export call in `test_databasus_recovery.py` to the new body. Verify with `mise run python:lint` and `mise run e2e:browser`.

## 5. Documentation

- [x] 5.1 Update the "Master-key export" bullet in `docs/architecture.md` and the master-key exception in `.agents/skills/api-design/SKILL.md`. Both say that the export needs root admin's session, plus an emailed code when the instance requires two-factor sign-in, that the download only spends its token, and that the acknowledgment is a separate explicit operation. Search with `rg -n -i "identity proof|acknowledg" docs .agents/skills/api-design README.md assets/readme`. Verify that no text describes a password for the export or a download that records the acknowledgment.

## 6. Setup reminder without deferral

- [x] 6.1 In `backend/api/openapi.yaml`, delete `PUT /instance-backup-reminders` and `InstanceBackupReminderUpdate`, drop `deferredUntil` from `InstanceBackupReminder` and update the status description; run `mise run generate-api`. In `instancebackups/internal/configs`, delete `UpdateSetupReminder`, `updateSetupDeferredUntil`, `SetupDeferredUntil` and the deferral constant, and remove the handler. Add a goose migration that drops `setup_deferred_until` from `instancebackups_configs` and adds it back in its down step. Remove the deferral tests from `backend/internal/modules/instancebackups/setup_reminder_test.go` and keep the eligibility tests. Verify with `go -C backend build ./...`, `mise run backend:lint` and `go -C backend test -count=1 ./internal/modules/instancebackups/`.
- [x] 6.2 In the frontend, remove `instanceBackupApi.updateReminder`, the deferral mutation in `InstanceBackupStatusWidget.tsx`, the deferral check in `selectFloatingNotice.ts`, the "Remind me tomorrow" button and the close button in `FloatingBackupNotice.tsx`, and `instanceBackupNotices.remindTomorrow` from every dictionary. Update the status fixtures and `FloatingBackupNotice.test.tsx` so it checks that the setup reminder has neither action. Verify with `pnpm --dir frontend typecheck`, `mise run frontend:lint` and `pnpm --dir frontend test -- src/widgets/instance-backup-status`.

## 7. Recovery upload without the HTTP warning

- [x] 7.1 In `backend/api/openapi.yaml`, drop `isInsecureTransferAcknowledged` from `InstanceRecoveryUpload` and its mention from `createInstanceRecovery`; run `mise run generate-api`. Remove the acknowledgment field and the insecure-transport refusal from `instancebackups/internal/recovery/upload_receiver.go`, the transport flag from `upload_service.go`, and delete `lib/httpserver/transport_security.go` with its registration in `internal/app/server.go`. Update the recovery upload tests, `cmd/databasus/main_test.go` and `instancebackups/e2e/file_import_test.go` so no upload sends the field, and remove the tests of the refusal. Verify with `go -C backend build ./...`, `mise run backend:lint` and `go -C backend test -count=1 ./internal/modules/instancebackups/... ./cmd/databasus/` without the end-to-end packages.
- [x] 7.2 In the frontend, remove the field from `instanceRecoveryApi.createRecovery`, delete `isInsecureTransfer`, remove the warning and its checkbox from `InstanceRecoveryImport.tsx` and `instanceRecovery.form.insecure*` from every dictionary. Update `InstanceRecoveryImport.test.tsx`, the recovery scenarios in `frontend/browser/scenarios/instanceBackups.spec.ts` and `_upload` in `e2e/browser/scenarios/test_databasus_recovery.py`. Verify with `pnpm --dir frontend typecheck`, `mise run frontend:lint`, `mise run python:lint` and `pnpm --dir frontend test -- src/pages/instance-settings`.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
mise run e2e:browser
```
