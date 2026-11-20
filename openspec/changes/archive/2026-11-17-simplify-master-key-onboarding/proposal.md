## Why

On their first visit, root admin gets a master-key dialog that offers a password, an emailed code, a host-file copy, an HTTP warning, an "I saved the master key" button and a "Remind me later" button that leaves a floating reminder behind. Asking for the password right after Databasus starts protects nothing, because nobody could have taken over the account yet. Deferral lets an installation run without a saved key, and that key is the only way to decrypt its backups.

## What Changes

- Onboarding becomes a full-screen step shown instead of the application, like the sign-in page: the logo, the language and theme control, and the key card in the middle. The application opens only after root admin confirms saving the current key. The step has no "Remind me later" or sign-out action. No floating master-key reminder remains.
- The step explains the key in one paragraph: Databasus encrypts backups with it, and after losing the server the owner needs it to decrypt the backups, including the backup of Databasus itself.
- The step offers two ways to save the key: copy it to the clipboard, next to a preview that shows only part of the key, or download it as `databasus-master-key.txt`. The "I've saved it in a safe place" button stays disabled until root admin copies or downloads the key. Pressing it records the acknowledgment for the current key and opens the application.
- Neither the export nor the recovery upload warns about unencrypted HTTP any more. Root admin knows how they reach their instance, and the key can be copied without a network transfer warning getting in the way.
- When the instance does not require two-factor sign-in, the step fetches the key without any identity confirmation.
- When the instance requires two-factor sign-in, the export requires an emailed code. Password confirmation is no longer accepted for the export.
- If the code cannot be sent, the step shows the sending error and explains that the key can be copied from the server by hand, with the key's path inside the container. The step stays in place.
- The host-copy instructions move from the onboarding to the master-key panel in the Databasus backups settings section. The panel keeps the saved status and lets root admin copy or download the key again after an explicit action.
- **BREAKING (API):** In `MasterKeyExportCreate`, `isInsecureTransferAcknowledged` and the `proof` object are deleted, and an optional `code` replaces them. The server requires `code` when the instance requires two-factor sign-in, and a password can no longer be sent. `InstanceBackupStatus` gains `isMasterKeyExportCodeRequired`. `PUT /master-key-acknowledgments` stays unchanged and is now the only writer of the acknowledgment. No shim keeps the old shape.
- **BREAKING (API):** `InstanceRecoveryUpload` loses `isInsecureTransferAcknowledged`, and the server no longer refuses an upload over unencrypted HTTP. The middleware that recorded whether a request arrived over an encrypted connection had no other reader and is deleted.
- The floating reminder to set up Databasus backups can no longer be put off: its "Remind me tomorrow" action and its close button are removed, so once the instance holds a database or a storage, root admin sees it until a destination is selected.
- **BREAKING (API):** `PUT /instance-backup-reminders` and the `InstanceBackupReminderUpdate` schema are deleted, and `InstanceBackupReminder` loses `deferredUntil`. A migration drops the `setup_deferred_until` column from `instancebackups_configs`.
- Accepted risk: when the instance does not require two-factor sign-in, anyone holding root admin's session can obtain the key. Requiring two-factor sign-in closes this.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `databasus-backup-ui`: the setup reminder offers no deferral or dismissal.
- `databasus-backup-alerts`: stopped protection no longer mentions a setup deferral to bypass.
- `databasus-recovery`: the upload no longer warns about or requires an acknowledgment for unencrypted HTTP.
- `master-key-export`: who may export and with which confirmation, the removal of the HTTP warning, and blocking onboarding that offers copying or downloading the key and records the acknowledgment through an explicit button.

## Impact

- API contract `backend/api/openapi.yaml` and the generated servers, clients and frontend types.
- Backend: `instancebackups` (key export service and handler, status, the setup reminder and a migration of its config table), and identity proofs in `users`. The change answers to `backend/AGENTS.md`.
- Frontend: the `export-master-key` feature, the `instance-backup-status` widget, the master-key panel on the instance settings page, and every interface dictionary. The change answers to `frontend/AGENTS.md`.
- Browser end-to-end helpers in `e2e/browser/scenarios`.
- Documentation: `docs/architecture.md` and the `api-design` skill, which both describe the export's identity proof and the acknowledgment.

## Out of scope

- Recovery apply: its identity proof and `identityProofMethods` stay as they are.
- Two-factor sign-in itself, its instance setting and the console `--disable-2fa` command.
- Any way to skip or defer the step, including after a failed code delivery.
- Key rotation and any change to how the key is stored on the server.
- The floating reminders for backup incidents.
