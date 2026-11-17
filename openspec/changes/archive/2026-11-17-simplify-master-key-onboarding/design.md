## Context

See proposal.md for the motivation and specs/master-key-export/spec.md for the required behavior.

Current state:

- `backend/internal/modules/instancebackups/internal/keyexport/service.go:82` `CreateExport` requires root admin, rejects HTTP without `IsInsecureTransferAcknowledged` and verifies a password or code proof. It then issues a 60-second single-use token. `Download` (`service.go:123`) spends the token and returns the key, and it deliberately records no acknowledgment. `Acknowledge` (`service.go:160`) is the only writer of the acknowledgment and is reached through `PUT /master-key-acknowledgments`.
- `backend/internal/modules/instancebackups/internal/configs/service.go:317` `AcknowledgeMasterKey` takes a `users.Actor`, checks root admin, stores the key's SHA-256 and writes the audit event in one transaction. `keyexport.Service.IsAcknowledged` (`keyexport/service.go:174`) compares that digest with the installed key, so a replaced key asks again.
- `backend/internal/modules/users/internal/identityproof/service.go:181` `Verify` accepts either a password or a code. A code is bound to the session, the root admin and a purpose. `users.Module` (`backend/internal/modules/users/module.go:91`) exposes no read of whether the instance requires two-factor sign-in. That setting is read only inside sign-in (`backend/internal/modules/users/internal/signin/password_sign_in_service.go:66`).
- Frontend: `frontend/src/widgets/instance-backup-status/ui/InstanceBackupStatusWidget.tsx:60` opens the onboarding dialog unless a per-sign-in deferral sits in `sessionStorage` (`model/masterKeyOnboardingDeferral.ts`). `model/selectFloatingNotice.ts:24` turns a deferred onboarding into the `master-key` floating notice. `frontend/src/features/export-master-key/ui/MasterKeyExport.tsx` holds the HTTP warning, the password and code forms, the host instructions and the acknowledge button. The same component is used in the onboarding dialog and in `frontend/src/pages/instance-settings/ui/instance-backups/MasterKeyPanel.tsx`.

The approach follows `backend/AGENTS.md` and `frontend/AGENTS.md`:

- Module boundaries go through `module.go`. A handler holds no logic. A service checks access first and writes audit in the same transaction. Every error sits in `errors.go`.
- Frontend code follows the FSD layers. Copy goes through the dictionaries.
- `docs/architecture.md` "Databasus backups and recovery" describes the export. Its sentence about the identity proof changes with this change.

## Goals / Non-Goals

**Goals:**

- One server-side rule decides whether a code is needed. The status tells the interface the same answer, so the interface and a refusal never disagree.
- The acknowledgment is written only by root admin's explicit confirmation, which the interface unlocks after the key was copied or downloaded.

**Non-Goals:**

- Changing the identity proof used by recovery, or `identityProofMethods`, which only recovery reads after this change.
- Changing the download token's lifetime, its single use or its `no-store` attachment.

## Decisions

### The page holds the key, and both actions use that one copy

The onboarding step creates one export and fetches `GET /master-key-downloads?token=...` with `fetch` instead of following the link. The response body becomes the key in page memory. The preview shows the first and last four characters with the rest masked. Copy writes the key through `shared/lib/BrowserClipboard`, which falls back to `document.execCommand('copy')` when the page is served over plain HTTP and the Clipboard API is missing. Download builds a `Blob` from the same text and saves it under the `fileName` from the export response, `databasus-master-key.txt`. Neither action asks the server again, so one export serves both and the single-use token stays single-use.

The key still leaves the server only through the token-protected download, so the requirement "Exported key material is never an ordinary response field" holds: no JSON field, URL, log or browser storage carries it. What changes is that page script now sees the key, while before the browser saved the file without script access. Script on this page can already call the export as root admin, so this gives an attacker nothing new.

Alternatives rejected:

- Two exports, one per action. The preview needs the key before either action, so the first export happens anyway, and a second one would only add an audit event and a second code for two-factor sign-in.
- A preview built on the server, such as a fingerprint in the status. The user asked to see part of the key itself, and a key fragment in the status would break the rule that the status carries no key material.

### The acknowledgment is an explicit button

`PUT /master-key-acknowledgments` stays as it is in the contract and in `configs.AcknowledgeMasterKey`, which checks root admin, stores the key's digest and writes the audit event. The download goes back to only spending its token and auditing, so the master-key chain no longer needs the exception that let `GET` record state. "I've saved it in a safe place" calls the endpoint; the step enables it after a successful copy or a started download and gives way to the application once the refreshed status reports the key acknowledged.

The server does not check that a copy or download happened, as it did not with the old "I saved the master key" button. It cannot see a clipboard write or a local file, and a server-side "token spent" check would still not prove that the owner kept the key.

Alternative rejected: keeping the acknowledgment on the spent token. The page now fetches the key to show the preview, so the token is spent before root admin copies or downloads anything.

### When the key is fetched

The onboarding step fetches the key as soon as it appears when the instance does not require two-factor sign-in, and after root admin enters the code and presses "Continue" when it does. The settings panel fetches only after root admin presses "Copy or download the key", so opening the settings does not create an export and an audit event each time.

### The export takes an optional `code`, not a `proof` object

`MasterKeyExportCreate` becomes `{ code?: string }`. A password cannot be sent, so "password is not accepted" holds by the contract's shape rather than by a runtime check. `users.Module` gains one predicate, `IsTwoFactorAuthRequired(ctx)`. When it is true, the export requires a non-empty `code` and passes it to the existing `VerifyIdentityProof` as an `IdentityProof` that holds only the code; `identityproof.Service.Verify` (`identityproof/service.go:181`) already takes its code path for such a proof and already refuses wrong, expired, reused and wrong-purpose codes. A code sent while the instance does not require two-factor sign-in is ignored, because the session alone already authorizes the export.

Alternatives rejected:

- Keeping `proof` and rejecting `proof.password` at runtime would leave a field in the contract that is always refused.
- A new code-only verification method on `users.Module` would duplicate the existing code path and its tests.
- Reading the settings repository from `instancebackups` would cross a module boundary that `docs/architecture.md` forbids.

### The status carries `isMasterKeyExportCodeRequired`

The status service fills it from the same `users` predicate that the export checks. The interface decides from it whether to show "Send code" and the code field.

Alternative rejected: deriving it on the frontend from `identityProofMethods`. That list answers which methods are available for recovery, not whether the export demands one.

### A full-screen step instead of a dialog

`ApplicationScreen` reads the status for root admin before it renders the shell. While the status loads it shows the usual loader; while the key is unacknowledged it renders `MasterKeyOnboardingScreen` in place of `AppShell`. That screen reuses the sign-in page's frame, the logo with the language and theme control, and centers a card with the export and the confirmation. The shell, its pages and their requests start only after the acknowledgment, so nothing hides behind an overlay.

Alternatives rejected:

- A blocking dialog over the shell. It needs Escape, outside-click and focus handling only to stay unclosable, and it renders the application underneath for nothing.
- A route of its own. The step is not a place a user navigates to; a route would need guards against leaving it and a redirect back.

### The setup reminder loses its deferral entirely

Deferral is removed end to end rather than hidden in the interface: the operation, the `deferredUntil` field, the service method and the stored column go together, so nothing keeps a state no client can set. The eligibility rule (a database or a storage exists, no destination is selected) stays and alone decides whether the reminder shows. The migration drops `setup_deferred_until`; its down step adds the column back empty, because a deferral expires within a day and losing it is harmless.

Alternative rejected: removing only the buttons. The server would keep an operation and a field that nothing uses, which `AGENTS.md` forbids as unrequested compatibility.

### Code delivery failure keeps the step in place

When the code request fails, the step shows the translated API error, the key's path inside the container and the `docker cp` command. That failure is `identity_proof_unavailable`, a mail error or a rate limit. The step offers no way out, and no sign-out either, as the user decided.

### No HTTP warning on recovery either

The recovery upload loses its warning and its "Send them over HTTP anyway" checkbox, and the server stops refusing an upload over plain HTTP without the acknowledgment. Root admin chose how they reach their own instance, and the warning only added a click to every recovery on a local or private network. The server-side check existed only to back that checkbox, so it goes with it, together with the `RememberTransportSecurity` middleware in `lib/httpserver`, which had no other reader.

### Deleted code

Deleted without a shim:

- in the contract: `isInsecureTransferAcknowledged` and `proof` on the export, and `isInsecureTransferAcknowledged` on `InstanceRecoveryUpload`;
- in `recovery`: the acknowledgment field of the upload receiver, and in `lib/httpserver`: `RememberTransportSecurity` and `IsSecureTransport`;
- in `keyexport`: `ErrInsecureTransferNotAcknowledged`;
- on the frontend: `masterKeyOnboardingDeferral.ts`, the `master-key` floating notice kind, `isInsecureTransfer` and the recovery HTTP warning, with their dictionary keys.

`POST /master-key-confirmations` stays, because the export still needs codes when the instance requires two-factor sign-in. `PUT /master-key-acknowledgments` and `Acknowledge` stay as the only writer of the acknowledgment.

## Risks / Trade-offs

- [When the instance does not require two-factor sign-in, a stolen root-admin session obtains the key without a password.] → This is accepted by the user and written into the spec. Requiring two-factor sign-in is the protection, and the audit log still records every export and download.
- [The instance requires two-factor sign-in and mail delivery breaks while the key has no acknowledgment, so the step blocks the interface.] → The step explains how to copy the key from the server. Root admin can run `./main --disable-2fa` from the console (spec `host-console-commands`) and then obtain the key without a code. The step does not mention that command, as the user specified its content.
- [Root admin presses "I've saved it in a safe place" after a copy that the clipboard did not keep, or after a download the browser dropped.] → The key is marked saved without a copy on the owner's side, the same window as the old "I saved" button. The settings panel lets root admin copy or download the key again.
- [The key sits in page memory while the onboarding step or the panel shows it.] → It is dropped when the step gives way to the application or the panel's key view unmounts, and it never reaches browser storage.
- [Existing installations that deferred onboarding get the onboarding step on their next visit.] → This is intended, and the spec says it applies to existing installations.

## Migration Plan

The acknowledgment column and its digest semantics stay. The only migration drops `setup_deferred_until` and restores it on the down step. Older frontends do not exist separately from the server, because the frontend ships in the same image. Rollback is a redeploy of the previous image. An acknowledgment recorded by the button remains valid there.
