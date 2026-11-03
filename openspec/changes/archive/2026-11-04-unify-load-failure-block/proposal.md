## Why

The "failed to load, retry" block is written by hand 39 times across the frontend and has drifted: some copies lack the icon, some use a different button size, and only some let the button text wrap. Four API facades also copy a cursor loop that `collectCursorPages` already provides, one screen repeats `useBackupConfigLoad`, and the database screen keeps React state only to choose a corner radius.

## What Changes

- Add one shared `LoadFailure` block and use it in 37 files, about 250 fewer lines in total.
- Replace the cursor loops in the restore, backup config, workspace member and verification agent API facades with the existing `collectCursorPages`.
- Make `DatabaseConfigComponent` load its backup configuration through `useBackupConfigLoad`.
- Remove `isDirectlyUnderTab`, `isHealthcheckVisible` and `onVisibilityChange`, and let CSS choose the corner of the card directly under the tabs.

Small visible differences follow, and no requirement in `openspec/specs` changes:

- The auth page, the backup storage selection and the database notifier fields gain the error icon.
- The WAL monitoring retry button becomes normal size.
- The users page retry button loses its `disabled` state. The block disappears as soon as a reload requests its page, and `retryUsers` ignores any click during a load.
- While the healthcheck configuration loads, the backups card is fully rounded. It gets its square corner if the healthcheck turns out to be disabled.
- A failed healthcheck configuration load shows its error directly under the tabs with a square top-left corner.

Nothing here is **BREAKING**: no API, schema, configuration or stored data changes.

### Out of scope

- Converting imports to one style and the lint rule that would enforce it. The user will do this as a separate change.
- Save errors and other failures that are not a load with a retry.
- `StorageSwitchHistoryComponent`, because its button has its own "Retry status" text.
- `DatabaseTransferDialogComponent`, because its retry starts four loads and is disabled during a transfer.
- The cursor loop in `HealthcheckAttemptsComponent`, because it stops early when a load has gone stale.
- Cancellation support in `collectCursorPages`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. Failed loads keep ending in translated feedback with an explicit retry, as `frontend-database-screens` and `frontend-application` already require, so the change sets `skip_specs: true`.

## Impact

- Code: `frontend/src` only, in `shared/ui`, `app`, `entities` and `pages`.
- Tests: a new `LoadFailure` test, API tests for the four facades, screen scenarios for the behavior the block must keep, and a browser scenario for the corner. Existing screen tests keep their assertions, with one exception: the healthcheck test that rerenders with a new `onVisibilityChange` is renamed and narrowed to the language change, because the prop no longer exists.
- Rules: the change answers to the root `AGENTS.md` and `frontend/AGENTS.md`.
- Documentation: none, because no rule changes.
