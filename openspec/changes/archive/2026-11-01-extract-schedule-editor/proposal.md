## Why

The schedule form is written three times: in the logical config, in the physical config and in the verification config. A fix in one copy does not reach the others, and each copy repeats the same conversion between UTC and local time. Validation is also too weak: an incomplete schedule or a cron expression the server refuses keeps Save enabled and fails only after the click.

Working through the shared form exposed defects a plain extraction would carry over:

- A weekly day can come back one day off when the user's UTC offset changes later in the current week. In Africa/Cairo on 2026-10-27, Friday 00:00 is saved as Thursday 21:00 UTC and is shown as Thursday.
- A cron expression with no date to run on, such as `0 0 31 2 *`, passes the server's validation and then counts as due at every scheduler pass, so a backup or a verification keeps starting.

## What Changes

**Interface**

- A new `entities/schedules` slice owns the schedule types, their labels, schedule validation, the conversion between UTC and local time, and the editor. The logical and physical config editors and show views use it. The verification editor and show view use its weekday and local schedule functions and keep their own form until milestone 7.
- `PhysicalIntervalEditor`, `mergeSchedule` and `shared/time/utils.ts` are deleted without re-export bridges. The editor takes the whole schedule and returns a whole schedule.
- A weekly day uses the same conversion rule as a monthly day, so it survives being shown and saved again.
- A day chosen before a time stays the day the user chose. A schedule without a type shows only the type choice.
- Save stays disabled until the schedule is complete. Daily, weekly and monthly schedules need a time. A cron expression must have five fields and a date to run on, and must avoid the forms the server refuses: the `L`, `H` and `#` tokens and weekday `7`. The next-run preview follows the same rule.
- The logical and physical forms share their labels. "Backup weekday", "Backup day of month" and "Backup time of day" become "Weekday", "Day of month" and "Time of day" in all six languages.
- **BREAKING** for users of a saved cron expression the server accepts but the interface now refuses, such as `TZ=Europe/Moscow 0 2 * * *`. The schedule keeps running, but Save stays disabled until the expression is rewritten as five UTC fields or backups are turned off. The contract already promises five fields. This default was recommended during exploration and is recorded here for review.

**Backend**

- **BREAKING** for API callers: saving a cron schedule with no date to run on is refused with `400` and `code` `invalid_request`. This applies to logical backups, physical backups and verification.
- A stored schedule with such an expression is never due, with or without a previous run. Backups or verifications that ran at every pass because of it stop. No migration touches the stored value, and a member sees the expression as invalid in the form.
- A stored incremental schedule with such an expression now fails validation, so the existing fitting of a physical config replaces it on read with the schedule the backend builds. With an hourly full schedule nothing can be built, so incremental backups stop and the form shows an empty incremental schedule.
- **BREAKING** for a physical database whose stored full schedule has such an expression: changing its physical backup type is refused with `400` until a member rewrites the expression. Today the same change succeeds.
- The `cronExpression` description in `backend/api/openapi.yaml` states the rule, and the generated code is regenerated from it. Fields and operations stay the same.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `recurring-schedules`: adds the rule for a cron expression with no date to run on, the weekly day conversion, a day chosen before a time, the completeness rule for Save and the shared field names.
- `backup-configs`: a physical backup type change is refused when the adjusted config still fails validation, such as a stored full schedule with no date to run on.

## Impact

This is item 6.3 of frontend milestone 6 in `temp/refactor/frontend-plan.md`. The root `AGENTS.md`, `frontend/AGENTS.md` and `backend/AGENTS.md` apply, with the feature-sliced-design, frontend-design, frontend-localization, api-design and testing skills.

Affected areas:

- `frontend/src/entities/schedules` (new), `entities/backup-configs` and `entities/intervals`, which lose what moves;
- `frontend/src/shared/time`, which gains one time-of-day format and loses `utils.ts`;
- the six editors and show views in `pages/databases/ui/backup-config`, `features/backups/physical/ui` and `features/verification/config/ui`;
- `features/backups/physical/monitoring`, whose `clockFormat.ts` is replaced by that shared format;
- the six dictionaries in `frontend/src/shared/i18n/locales`, which lose three keys, and `frontend/style-baseline.json`;
- `backend/internal/lib/schedule`, shared by the logical, physical and verification schedulers;
- `backend/api/openapi.yaml` and the code `mise run generate-api` produces from it, for the description only;
- `openspec/specs/recurring-schedules/spec.md`, whose Purpose grows with the capability, and `openspec/specs/backup-configs/spec.md`, through its delta.

There is no database migration and no new dependency. Reverting the commit restores the old rules, and stored configs stay untouched either way.

### Out of Scope

- The verification form itself and `IntervalType`, which milestone 7 replaces. The handwritten `Interval` type loses its last consumer here and is deleted.
- Renaming the `backups.config.schedule` and `intervals` dictionary keys. All schedule copy moves once at milestone 7, when verification first reads it.
- A shared read-only schedule component. The show views share a pure function and keep their own markup until milestone 7.
- The rest of the logical config editor and the defects listed for `refactor-logical-backup-screens` in 6.4, including the `window.alert` calls and the retained color entries of that form.
- Backend rules for the `TZ=` and `CRON_TZ=` prefixes. The server keeps accepting them although the contract promises five fields, and that gap stays for a separate decision.
- A notice to members whose stored expression stopped running.
- Storing a time zone in a schedule.
