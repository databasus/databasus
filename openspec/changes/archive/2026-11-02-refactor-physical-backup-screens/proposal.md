## Why

The physical backup list, its restore screen and the WAL stream monitoring tab have lifecycle defects, and each list action sits in one 640-line component. Found by reading the code, and to be reproduced before any fix:

- A failed backup configuration load leaves the whole tab on a spinner forever.
- A failed first page reads as "no backups" with a total usage of 0, after a blocking alert.
- A viewer sees "Back up now", and the server refuses the request.
- A late restore token answer can replace a newer one, so the commands can show a target time that does not match their restore set.
- Switching the language during a full backup request frees the button while the request is still running, because the request's controller is created again.
- Leaving the Monitoring tab during that request restarts the 30-second statistics refresh with nothing to stop it, and it shows feedback about a screen the user left.

These screens also hold 166 of the style baseline's remaining occurrences for this milestone.

## What Changes

**List**

- The list splits into a table, cards, actions, a status badge and an actions hook, like the logical list. Icon actions become keyboard buttons with names.
- The configuration load moves out of the logical list into one unit both lists use, with its inline failure and retry, and it no longer reads as disabled backups.
- A failed first page shows its reason with a retry instead of the empty text and the usage line, and the backup start stays disabled while that failure is shown. A failed next page keeps the rows and shows its reason once. A refused start, cancel or delete uses context-bound feedback. Leaving the screen or a session change ends this work without later feedback.
- "Back up now" appears only for users who may manage the database, because the operation requires it. Restore stays available to viewers, because their token request is allowed.
- A manual backup keeps the loaded rows and the scroll position. The fixed one-second wait and the reload go, and the 1-second poll shows the backup once the scheduler creates it.
- Polling intervals, page size, filters and the absence of automatic mutation retries stay as they are.

**Restore**

- The newest target time owns the screen: a late answer for an earlier target time changes nothing, and the commands appear only with the token of the shown target time. Clearing the picker means the latest restorable moment and needs its own token. Leaving the screen ends this work.
- The screen splits into a token hook and its environment panels, with the same commands and copy actions.

**Monitoring**

- The full backup request has one controller per database. A language switch keeps its busy state and sends its messages in the current language.
- Leaving the Monitoring tab ends the request's observable work: no feedback appears and no refresh starts. Hiding the browser tab is not leaving.
- Monitoring's calculations, controllers' schedules and charts stay as they are, apart from semantic colors for text, borders and surfaces. The WAL series palette stays in its owner.

Raw colors become semantic roles, which removes the entries of these screens from `frontend/style-baseline.json`. Nothing breaks an API, a schema or a config.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: the physical backup list ends its loads recoverably, a manual backup keeps the list, the backup start follows the user's permission, and the restore screen shows commands only for the token of the shown target time.
- `physical-backups/wal-stream-monitoring`: the full backup request survives a language switch and ends with the tab, and a failed request reports its error only while the tab is open.

## Impact

This is the fifth change of frontend milestone 7, after `refactor-physical-backup-config`. The root `AGENTS.md` and `frontend/AGENTS.md` apply, with the frontend-design and testing skills, and the frontend-localization skill if a control needs a new name.

Affected areas:

- `frontend/src/pages/databases/ui/physical-backups`, `ui/wal-stream-monitoring`, `lib/wal-stream-monitoring`, `lib/physical-backups`, `model/physical-backups` and the new `lib/formatBackupSize.ts`;
- `frontend/src/pages/databases/ui/logical-backups`, which gives up its configuration load, size formatter and icon action button to the shared units in `ui/backup-config`, `lib/formatBackupSize.ts` and `ui/BackupActionButton.tsx`;
- `frontend/style-baseline.json`, `frontend/browser/scenarios/wal.spec.ts` and a new browser scenario, and `temp/refactor/frontend-plan.md`.

There is no backend change, migration or new dependency.

### Out of Scope

- Splitting monitoring's components and rewriting its controllers, calculations or tests, apart from the one controller change here, which the milestone keeps stable on purpose.
- Changing any polling interval.
- Merging collection polling with tracking a single operation.
- Real physical backups in a browser test, which stay in the backend suites.
- The contract difference that lets a viewer start a logical backup but not a physical one.
