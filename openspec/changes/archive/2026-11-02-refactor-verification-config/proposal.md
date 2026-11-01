## Why

The verification config editor is the last screen with its own copy of the schedule fields, its own cron parsing and the handwritten `IntervalType`. Because of that copy, it accepts any non-empty cron expression, so Save stays available and the server refuses the value afterwards. The text of schedules is also spread across three dictionary groups, one of them named after backups even though the shared schedule entity reads it.

The editor and the view also report failures through blocking `alert`. A failed editor load leaves an empty section with no way to retry or close it. `frontend-database-screens` already forbids this for the logical backup configuration section, and this change adds the verification section.

## What Changes

- The verification editor uses the shared schedule fields. "After backup" stays a choice in the editor's own select, and the other values come from the shared schedule type.
- Save requires a complete schedule by the same rule as backups while automatic verification is on and runs on a schedule. A cron expression the server refuses is shown as invalid and blocks Save.
- `IntervalType`, `INTERVAL_TYPE_LABEL_KEYS` and `entities/intervals` are deleted, along with the editor's cron parsing and next-run code.
- All schedule text moves once to a `schedules` group in the six dictionaries. `backups.config.schedule`, `intervals` and the schedule keys under `verification.config` disappear. Retention periods from `intervals.periods` move to `backups.logical.retentionPeriods`.
- A failed load in the editor offers a retry and Cancel, and a failed load in the view offers a retry. A refused save keeps the input and shows its reason inline. No `alert` remains in either.
- Save follows the values the form shows against the acknowledged config, so reverting an edit disables it again. A save sends the acknowledged values for whatever the form hides, so an edit the user hid afterwards is never stored.
- Cancel stays disabled during a pending save, and that rule is now specified.
- The style baseline entries of the two components, 15 occurrences, are removed.

Visible text changes: the verification form's cron help and next-run sentence become the shared ones, and its labels align to the same width as the backup config forms. Nothing breaks an API, a schema or a config.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `recurring-schedules`: the verification schedule follows the same completeness rule and field names.
- `frontend-database-screens`: the verification section's loads and saves end in content or a recoverable failure.

## Impact

This is the third change of frontend milestone 7, after `move-physical-and-verification-screens`, and its paths are the ones that change creates. The root `AGENTS.md` and `frontend/AGENTS.md` apply, with the frontend-design, frontend-localization and testing skills.

Affected areas:

- `frontend/src/pages/databases/ui/verification/{Edit,Show}VerificationConfigComponent.tsx` and new units in `pages/databases/model/verification`;
- `frontend/src/entities/schedules`, which exports `ScheduleDetailsFields`, `getNextCronRun` and `isScheduleChanged` and reads the new keys, and `entities/intervals`, which is deleted;
- `frontend/src/pages/databases/model/RetentionPeriodLabelKeys.ts`, the logical and physical config views for their key paths, and the six dictionaries in `shared/i18n/locales`;
- `frontend/style-baseline.json` and a new browser scenario under `frontend/browser/scenarios`;
- `docs/frontend-localization.md` and `docs/glossary.md` only if they name a renamed group, and `temp/refactor/frontend-plan.md`.

There is no backend change, migration or new dependency.

### Out of Scope

- The verification list, details and agents, which `refactor-verification-list-and-agents` handles.
- The server's rule that validates a schedule only for an enabled interval config.
- A shared "After backup" schedule type.
- Cancel and pending rules of the notifier and healthcheck sections.
