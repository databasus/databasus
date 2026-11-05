# Workspace resource screen coverage

## Prerequisites and baseline

Both prerequisite changes are implemented and archived on 2026-10-31: `fix-google-drive-storage-oauth` and `fix-sftp-storage-credentials`. Their coverage inventories remain authoritative for provider transport and credential policy. Baseline `pnpm --dir frontend test`: 124 files, 1109 tests passed.

## Ownership map

Paths below are relative to `frontend/src` unless otherwise stated. Existing scenarios are retained. No test scenario was removed.

| Existing surface | Final owner | Retained coverage |
| --- | --- | --- |
| `entities/{workspaces,storages,notifiers}/api/*.test.ts` | Unchanged entity facades | Request shapes, authorization boundary, provider secrets and domain operations |
| `features/workspaces/ui/WorkspaceMembersComponent.test.tsx` | `pages/workspace-settings/ui` | Existing user, invitation, duplicate membership refusal and role policy |
| Workspace creation exercised by `app/ui/*.test.tsx` | `app/ui/CreateWorkspaceDialogComponent.tsx` | Creation policy, acknowledged selection and session isolation |
| `features/storages/ui/StoragesComponent.test.tsx` | `pages/storages/ui` | List selection, live SFTP draft during list refresh and acknowledged settings |
| `features/storages/ui/StorageComponent.test.tsx` | `pages/storages/ui` | Detail rendering, password/key/dual metadata and deletion refusal |
| `features/storages/ui/StorageTransferDialogComponent.test.tsx` | `pages/storages/ui` | Destination policy, storage transfer and migration progress |
| `features/notifiers/ui/{NotifiersComponent,NotifierComponent}.test.tsx` | `pages/notifiers/ui` | Selection, retained dirty creation, obsolete detail responses and refusals |
| `features/notifiers/ui/show/notifier/ShowTelegramNotifierComponent.test.tsx` | `pages/notifiers/ui/show/notifier` | Literal saved provider values |
| `features/notifiers/ui/edit/*.test.tsx` and provider tests | `features/edit-notifier/ui` | Submitted acknowledgment, persisted ID, explicit delivery, secrets, revisions, locale/theme, reset/session/unmount and replay |
| `features/storages/ui/edit/EditStorageComponent.test.tsx` | `features/edit-storage/ui` | Saved-type locking, S3 classes, secret omission and SFTP credential lifecycle |
| `features/storages/ui/edit/GoogleDriveAuthorization.test.tsx` | `features/edit-storage/ui` | Independent attempts, expiry, exact callback identity, obsolete exchange/test responses and retained draft |
| `pages/storage-oauth/ui/StorageOAuthPage.test.tsx`, `app/ui/App.test.tsx` | Unchanged pages/app | Callback validation, query clearing, acknowledgment, timeout and existing route |
| `features/databases/ui/DatabaseTransferDialogComponent.test.tsx` | `pages/databases/ui` | Target policy, storage move/copy, nested notifier creation and retained caller |
| `features/databases/ui/DatabaseConfigComponent.test.tsx` | Retained feature with page composition coverage | Notifier selection, configured draft and explicit nested creation |
| `features/backups/{logical,physical}/ui/Edit*BackupConfigComponent.test.tsx` | Retained config features with page composition coverage | Real selection, replacement confirmation and accepted switch progress |
| `entities/backups/config/api/backupConfigApi.test.ts` | Unchanged entity | Config HTTP contract |
| `browser/scenarios/{googleDriveStorage,sftpStorage}.spec.ts` | Existing production browser scenarios | Actual nested editor, callback channels, credentials, keyboard and 320px containment |
| Backend storages API and provider tests | Unchanged backend | Encryption, omitted credentials, method selection, identity and isolation; no backend change planned |

## Additions and verification

The additions below cover workspace recovery and acknowledgment, moving-scope resource selection, shared audit recovery, composed caller lifetime and storage revision/acknowledgment. Production browser scenarios exercise provider presentation and real-API workspace isolation.

No existing scenarios have been removed.

## Workspace and shared mechanics

- `pages/workspace-settings/ui/WorkspaceSettingsComponent.test.tsx` reproduces the initial failure and newer-name overwrite on the original implementation, then checks retry, acknowledged rename callbacks, confirmed 403/404 versus ordinary 500 recovery and viewer restrictions. Both original regressions failed before repair and passed afterward.
- `shared/hooks/useScopedSelection.test.tsx` covers preference-key replacement without remount, invalid remembered IDs, failed/unready versus authoritative reads, refresh removal, mobile initial/back/removal and unavailable browser storage.
- `pages/workspace-settings/ui/WorkspaceAuditLogsComponent.test.tsx` checks retry filters/page size, literal content/author, column policy and stale success/error after a workspace switch under StrictMode. Existing instance/user audit coverage is retained.
- Intermediate verification: 82 files, 578 tests passed across workspace/resource/app/shared/audit and database/backup consumers.
- Cross-feature integration tests for database details/config and logical/physical storage replacement now live in `pages/databases/ui`, where production resource composition is available. They continue to exercise the retained feature implementations through public APIs; no scenarios were removed.

## Editor and composition regressions

- `features/edit-storage/ui/edit/StorageLifecycle.test.tsx` covers created-ID retention with newer name input, a second update of that ID, returning to the acknowledged name, stale success/error after settings A-to-B-to-A, omitted secret acknowledgment with and without a newer replacement, current callbacks and independent concurrent editors. Running the first regressions against the pre-repair moved editor reproduced the newer-input/completion and clean-baseline failures. The A-to-B-to-A tests already passed under the prerequisite fix and remain guards.
- Existing storage tests retain saved-type restrictions, failed operations, SFTP metadata/secret/method policy, context/reset/session/unmount rejection, concurrent test/save ordering and Google authorization lifecycle. Existing notifier tests retain the equivalent submitted-snapshot, secret, callback, presentation and explicit-delivery guarantees.
- `pages/databases/ui/BackupStorageSelection.test.tsx` checks delayed creation with newer input followed by updating the same ID, and late creation after cancel, workspace replacement or unmount. `NotifierComposition.test.tsx` retains newer draft input and persisted choices in both selection and transfer parents. `DatabaseTransferDialogComponent.test.tsx` also rejects a storage creation completing after the target workspace changes. Logical/physical config tests retain real storage selection, destructive replacement confirmation and accepted progress. Existing Google browser scenarios check nested cancel/create/select while retaining the caller's config draft.
- Both resource detail test files add delayed rename acknowledgment with newer input. Workspace membership adds stale successful/failed list rejection after a workspace switch under StrictMode. `app/ui/WorkspaceResources.test.tsx` distinguishes failed reads from empty lists, retries reads and checks app reconciliation callbacks after confirmed access loss.

## Production browser coverage

- `browser/scenarios/{storageResources,notifierResources,workspaceSettings}.spec.ts` checks eight storage variants and seven notifier variants in both themes, with French/Russian labels, keyboard type selection, accessible names, retained input, 320px layout and actual 200% text. Storage modal Escape restores the invoking control. Additional scenarios exercise workspace read recovery, viewer restrictions with saved testing retained, and deletion refusal without losing the selected resource.
- Existing SMTP reference images are unchanged. Their normal-size geometry and all existing locale/theme, validation, explicit test/save, notification lifetime and focus scenarios remain covered. Browser checks exposed and repaired action-label clipping, unnamed switches, SFTP help contrast, Webhook help/header overflow and modal focus restoration.
- `browser/e2e/workspaceResources.spec.ts` creates two uniquely named workspaces and a local storage through the production UI, reads storage list/detail through the real API, reloads workspace/resource preferences and proves resource isolation. The separate sign-in/sign-out scenario is retained. It retains credential-safe evidence and isolated test-application cleanup; no external provider account is used.

## Documentation and migration boundaries

`docs/architecture.md`, `docs/frontend-design.md`, `docs/testing.md` and `frontend/README.md` describe the final owners, acknowledgment policy, composition contracts and verification. The old workspace/resource feature paths and cross-feature editor imports are gone. FSD baseline occurrences decrease from 30 to 19; style baseline occurrences decrease from 860 to 603. Remaining entries belong to later migrations; no exception was added. HTTP contracts and backend behavior are unchanged.

## Final verification

- `pnpm --dir frontend test`: 131 files, 1262 tests passed.
- `pnpm --dir frontend test:browser --project=e2e`: production browser scenario and gateway tests passed with the real API.
- `mise run frontend:lint`: formatting, lint, architecture, semantic styles, types and generated API freshness passed.
- `pnpm --dir frontend build`: passed; the existing large-chunk warning remains outside this change.
- `pnpm --dir frontend test:browser --project=ui`: 83 scenarios passed, including unchanged SMTP references and the browser harness probes.
- `pnpm --dir frontend exec vitest run --config vitest.browser-e2e.config.ts`: all seven supervisor lifecycle checks passed, including deliberate failures, interruption, concurrent real-API runs, credential-safe evidence and owned-resource cleanup.
- `pnpm --dir website lint` and `pnpm --dir website build`: contributor guide checks and static export passed.
- All implementation checks are complete. Mandatory review acceptance is required before declaring milestone 4 complete.

## Mandatory review repairs

- Startup keeps a remembered workspace preference until the accepted workspace list validates it. Reconciliation preserves a newer explicit selection if it remains accessible. `app/ui/WorkspaceNavigation.test.tsx` covers non-first reload, acknowledged navigation labels, deletion with/without a fallback, delayed reconciliation and failed-refresh recovery through the real application composition and HTTP boundary.
- Notifier requests capture the authentication snapshot and pending work is invalidated on account replacement without relying on unmount. Completion compares the merged draft with its acknowledged baseline, so a name changed away and back can complete creation. Six lifecycle regressions cover these cases and normalized-setting acknowledgment alongside newer name input without replacing the HTTP boundary or editor implementation.
- Workspace audit tests now exercise desktop tables and mobile cards, short timestamps, author visibility and workspace-specific empty copy. Stale storage-failure matrices assert current inline feedback rather than relying only on the obsolete window.alert channel.
- The ownership warning uses contrasting surface/status roles; production browser accessibility checks cover both themes. Shared selection names its generic entity `selectedItemId`, and acknowledgment/selection tests separate logical steps with blank lines.
- Two existing multi-step notifier-creation tests now allow 15 seconds overall. The full concurrent suite exposed the previous five-second limit (one scenario took 5.219 seconds); conditions, assertions and request-count checks are retained, with no retry. The full suite verifies the repaired deadline.

- Saved-notifier tests clear obsolete delivery/API failures on the next explicit activation. Regression coverage checks failure followed by success and delivery outcomes completing after an acknowledged rename; the result updates only delivery metadata on the current resource. Failed storage deletion closes its confirmation so translated recovery stays visible while the acknowledged storage remains available.

- Resource detail screens derive rename dirty state from the acknowledged name. Both screens reproduce and repair the redundant Save after changing a name away and back.
- Saved storage/notifier tests are invalidated when settings editing supersedes their context. Actual detail-screen regressions hold the old saved-resource test, test and persist new settings, then release old success, provider failure or HTTP failure. The obsolete result cannot add feedback or overwrite delivery metadata; separate rename-overlap cases preserve acceptance across name-only changes.
- Workspace rename/deletion 404 responses clear obsolete workspace content and request app reconciliation. Ordinary rename 403/500 responses retain the current draft and local recovery. Eighteen StrictMode cases hold reads, renames and deletions across workspace replacement, authentication replacement and unmount, then release success or failure and verify current content, inline feedback and callbacks. Owned deletion confirmations are destroyed when the workspace lifetime ends.
- Membership screen cases activate successful/refused role changes and removals through HTTP, check submitted member IDs and roles, acknowledged rows and translated refusals, and exercise owner/admin/self restrictions.

- Database transfer keeps its submitted workspace and resource choices locked while the HTTP request is pending, prevents duplicate submissions and retains accepted storage-switch progress. Delayed success/failure screen regressions reproduced the unlocked-target defect before repair; they verify one submitted operation, the original workspace/storage IDs, accepted cleanup progress and retry availability after failure.

- Nested backup storage creation supersedes pre-creation list reads while retaining the acknowledged option and selection. Controlled success/failure regressions reproduce the initial-read race and verify that late lists cannot remove the created choice or hide it behind obsolete errors.
- Both resource lists now have controlled old-workspace success/failure coverage after the replacement workspace and explicit selection are accepted under StrictMode. Assertions retain current cards, selected ID, preference and feedback, and reject obsolete reconciliation callbacks.
- Real-API workspace resources have their own E2E scenario, separate from sign-in/sign-out and its supervisor probes. Contributor guides point to the extracted editor features and page-owned provider displays.

- Post-creation refresh reconciles server choices with acknowledged resources instead of discarding the initial read without replacement. Backup selection regressions verify both existing and created choices remain selectable after late success/failure; a failed fresh read retains the acknowledged choice and offers retry. Transfer regressions hold creation, retain newer unsaved input, refresh existing choices and then release a pre-creation target read before canceling the editor. Both the persisted storage and existing storage remain available; notifier persistence uses the same target-read acceptance policy.

- Acknowledged workspace creation, rename and deletion supersede pending access-reconciliation reads. Access reads have a separate revision from application startup, so obsolete success and failure cannot overwrite acknowledged navigation or publish a global error. Six real-application controlled-response cases reproduce and repair all three mutation orderings without replacing domain APIs.

- Database notifier composition merges acknowledged resources before refreshing, retains them through refresh failures and preserves newer input until explicit completion. Selection and transfer now exercise completion with automatic selection; their actual database-update/transfer requests include the new ID alongside the caller's previous choices. Cancellation cases retain both successful-refresh and failed-refresh coverage.
- Membership initial failures render recovery without successful empty-list content. Desktop and mobile regressions verify failure followed by retry and an accepted empty result. Browser scenarios are separated by storage, notifier and workspace-settings screens, with shared navigation and access-policy setup; the scenario inventory and SMTP references are retained.

- Storage save acknowledgment invalidates tests started during saving even when they have already completed. A controlled SFTP regression reproduces approval against the previous stored password while a replacement is pending, then requires another explicit test after acknowledgment while retaining newer path input.
- Both resource pages exercise successful deletion and transfer with a remaining fallback and with the last resource. The actual HTTP mutation must finish before selected cards and scoped browser preferences reconcile.
- Twelve nested notifier-creation cases cover both database parents under StrictMode. Pending success/failure released after closing the editor, replacing its workspace or unmounting cannot add choices, replace the current selection, refresh the caller or publish obsolete feedback.

- Database transfer completes nested storage creation without a manual picker action, retains the target workspace and notifier-transfer option, then submits the created storage ID through the actual transfer HTTP operation.

- Workspace member/audit read refusals propagate through the settings page to app-owned access reconciliation. Six controlled real-application cases release child 403/404/500 responses after settings load: confirmed access loss clears old input and selects an accessible workspace; ordinary server failure preserves the workspace and draft. A stale audit 404 cannot reconcile the replacement workspace.

- The Add member dialog associates its visible Email label with a unique control ID for both administrator autocomplete and ordinary input. Both role variants are queried through their accessible names, with one matching control per label.

- Saved-resource tests show translated failure feedback for valid unsuccessful responses that omit the optional diagnostic. Both storage connection and notifier delivery screens reproduce the silent-failure bug and retain the acknowledged resource and explicit retry action after repair.

- Storage and notifier detail screens coordinate name/settings editing as exclusive modes. Controlled rename regressions reproduce the overlap and verify that subsequent settings requests retain the acknowledged name and newer provider settings.
- `browser/resourceScrollScenarios.ts` registers four production-browser scenarios with overflowing lists/details at 320px and 1280px, check their independent scroll positions, final controls, mobile back navigation and absence of competing page scrolling. They exposed short-desktop sidebar overflow; the sidebar now scrolls within its allocated height, keeping the final navigation control reachable.

- Explicit workspace selection supersedes pending access reconciliation. Two real-application controlled-response cases reproduce an obsolete list removing the newer workspace and an obsolete failure unmounting its settings draft; both retain the selected workspace, scoped preference and newer input after repair.

- Google Drive's failed-save scenario retries saving without authorizing again. The successful second update must retain the newly authorized tokens and reentered client secret; request counts prove that no additional popup or token exchange was needed.

- Provider browser matrices assert each provider's named controls and retain provider-specific input through narrow layout and enlarged-text checks. The fieldless local provider retains its disk-space warning. Removing a provider form now fails its scenario instead of passing through generic Name-field assertions.
- Physical backup configuration opens and cancels nested storage creation after changing the backup-enabled draft. The previous storage and unsaved flag remain; the eventual configuration request and completion callback retain both values.
- `browser/scenarios/databaseTransfer.spec.ts` reproduces target-picker overflow and clipped action labels at 320px with 200% French/Russian text. It checks reflow, opens/cancels both nested editors and retains target workspace/storage choices without submitting a transfer. Resource controls stack at narrow widths, buttons wrap and the multiple-picker placeholder has room to wrap.
