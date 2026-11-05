## Context

See `proposal.md` for the motivation, and `specs/recurring-schedules/spec.md` and `specs/backup-configs/spec.md` for the required behavior. The current state:

- The schedule form exists in `frontend/src/pages/databases/ui/backup-config/EditLogicalBackupConfigComponent.tsx:349`, `frontend/src/features/backups/physical/ui/PhysicalIntervalEditor.tsx:34` and `frontend/src/features/verification/config/ui/EditVerificationConfigComponent.tsx:201`. `PhysicalIntervalEditor` returns a partial patch, so `EditPhysicalBackupConfigComponent.tsx:85` needs `mergeSchedule`, which can build a schedule without the required `type`.
- `frontend/src/shared/time/utils.ts` converts a weekday through a calendar date in the current week and a monthly day through integer arithmetic. `getLocalWeekday` and `getUtcWeekday` therefore take the offset of another date, while the time of day takes today's offset. A run of a copy of these functions over all weekdays and half-hours gave these mismatches between the chosen and the shown day:

| Zone and date | Mismatches |
|---|---|
| UTC, 2026-10-28 | 0 of 336 |
| Europe/Moscow, 2026-10-28 | 0 of 336 |
| America/New_York, 2026-10-28 and 2026-03-07 | 0 of 336 |
| Africa/Cairo, 2026-10-27 | 10 of 336 |
| Asia/Beirut, 2026-10-23 | 2 of 336 |

- `PhysicalIntervalEditor.tsx:92` converts a day chosen before a time using the current time, while line 53 shows the raw value. Reading the code, a user at UTC+3 at 01:00 chooses Monday, and Sunday is stored and shown. A schedule can lack a time because the server's `HOURLY` schedule has none, as `backend/internal/modules/backups/internal/config/physical_type_coercion.go:73` builds it.
- `buildScheduleMoreOftenThan` at the same line returns `nil` for an hourly full schedule, so an empty incremental schedule reaches the form. `PhysicalIntervalEditor.tsx:183` then still shows the time picker.
- `frontend/src/entities/backup-configs/model/isScheduleFilled.ts` accepts any non-empty cron and treats `weekday: 0` as empty, although the contract allows it.
- The frontend and the backend parse cron with different libraries. A throwaway run of both gave:

| Expression | `cron-parser` | Backend |
|---|---|---|
| `0 2 * * *`, `0 0 ? * *`, `0 0 * * MON`, `*/5 1-3 1,15 JAN mon-fri` | accepts | accepts |
| `0 0 2 * * *`, `@daily`, `0 2 * *` | accepts | refuses |
| `0 0 L * *`, `0 0 * * 1#2`, `0 0 H * *`, `0 0 * * 7` | accepts | refuses |
| `TZ=Europe/Moscow 0 2 * * *`, `CRON_TZ=UTC 0 2 * * *` | refuses | accepts |
| `0 0 31 2 *` | refuses | accepts |
| `0 0 31 2,4 *`, `0 0 31 FEB,APR *` | accepts, next run 2054-03-18 | accepts, `Next` returns the zero time |
| `0 0 29 2 *` | accepts | accepts |

- `backend/internal/lib/schedule/schedule.go:72` validates a cron expression only by parsing it. For `0 0 31 2 *` the parser's `Next` returns the zero time, so the expression in `isCronDue` at line 191 is true at any moment. `IsDue` at line 89 also answers true without a last run. The logical scheduler (`backup_scheduler_service.go:119`), the physical scheduler (`scheduler_decision.go:42`) and the verification scheduler (`scheduler_service.go:93`) all go through `IsDue`.
- `logical_config.go:184` maps `ErrInvalidCronExpression` to the `cronExpression` field, and `verification/internal/configs/service.go:79` reports any schedule error on `schedule`.

The approach is bound by the root `AGENTS.md` (logical units, names, blank lines, no compatibility shims), `frontend/AGENTS.md` (component structure, design and localization skills) and `backend/AGENTS.md` (errors in `errors.go`, no import-time work). It follows the Frontend section of `docs/architecture.md`: an entity holds a domain model with real consumers, and slices on one layer do not import one another. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- One editor and one set of conversion and validation functions for logical and physical schedules.
- The client refuses what the server refuses wherever that is cheap to know, so the error appears before Save.
- A schedule with no date to run on is never due. Retries of a failed backup and forced requests do not consult the schedule and stay as they are.

**Non-Goals:**

- Matching the server's parser in every corner. The server stays the authority, and its `400` still reaches the form.
- Moving the verification form, or adjusting its label width of 180 pixels. Milestone 7 decides how the shared fields fit there.
- Rewriting the rest of the logical or physical editor.

## Decisions

### 1. The slice is an entity, with the editor inside it

`entities/schedules` holds `model/` and `ui/`, and reads its types directly from the generated schemas:

```text
entities/schedules
  model/ Schedule, ScheduleType, ScheduleTypeLabelKeys, Weekday, WeekdayLabelKeys,
         isScheduleComplete, getNextCronRun, scheduleDayConversion, localSchedule
  ui/    ScheduleEditor, ScheduleDetailsFields, CronExpressionField
```

The editor's consumers are a page and two features, and a feature cannot import another feature, so a `features/schedule-editor` slice could not serve physical and verification. `entities/backup-configs` already holds UI in the same way (`StorageSwitchHistoryComponent`). The entity imports no neighboring entity and knows nothing of backups or verification.

Rejected:

- Keeping the schedule in `entities/backup-configs`. The import would be allowed, but a schedule is not owned by the backup domain, and verification would depend on a backup entity for its own schedule.
- `shared/`. The editor holds domain labels and rules, and shared has no domain knowledge.

### 2. The editor is controlled and returns a whole schedule

`ScheduleEditor` takes `label`, `schedule: Schedule | null` and `onChange: (schedule: Schedule) => void`. `ScheduleDetailsFields` takes the same `schedule` and `onChange` without the type choice, so verification can add `AFTER_BACKUP` to its own Select at milestone 7. It stays private to the slice until that first outside consumer, and `index.ts` exports only `ScheduleEditor`. Switching type keeps the other fields in the returned value, as the forms do today, so switching back restores them. The server ignores the fields a type does not use. With `null`, only the type Select renders, and choosing a type returns `{ type }`.

Rejected: keeping partial patches. The parent form would need `mergeSchedule`, which is how a schedule without `type` appears today.

### 3. `gfsHours` reset compares the schedule types

The logical config keeps its rule that a daily, weekly or monthly type sets `gfsHours` to 24 under the GFS policy. Because `onChange` now fires for every edit, the handler applies the rule only when `next.type` differs from the current type.

Rejected: moving the rule into the editor, which would teach it about retention. Applying it on every `onChange` would silently reset the value when only the time changes.

### 4. One day shift for weeks and months

`scheduleDayConversion` holds one private `shiftDay({ day, minutesAfterStartOfDay, daysInCycle })`, and the weekly and monthly functions call it with 7 and 31. The offset comes from the schedule's time today, as it does for a monthly day. The public signatures stay: `getLocalWeekday(utcWeekday, utcTime)`, `getUtcWeekday(localWeekday, localTime)` and their monthly pair. Weekday `0` becomes `7` through the same arithmetic.

Rejected: keeping the calendar date in the weekly functions and adding tests in UTC, Moscow and New York only. Those zones pass on the sampled dates and the defect stays, so the tests also pin Africa/Cairo on 2026-10-27.

### 5. A pure local view of a schedule

`localSchedule` holds `getLocalScheduleParts(schedule)`, which returns the local time, weekday and day of month, plus `withLocalWeekday`, `withLocalDayOfMonth` and `withLocalTimeOfDay`. These are what the editor, the verification form and all the show views call. Without a time of day, a day is kept as chosen, and `withLocalTimeOfDay` converts it together with the time.

The show views keep their markup, because their label widths differ and verification waits for milestone 7.

Rejected:

- Filling in a default time when the type changes. The user would get a time they never chose, and the form could not tell it from a real one.
- A shared read-only component now. It is the natural next step, but it serves its third consumer only at milestone 7.

### 6. `getNextCronRun` is the single cron rule

`getNextCronRun(expression): Date | null` returns the next run in UTC, or `null` for an expression the form refuses. `isScheduleComplete` and the preview both call it, so they cannot disagree. The rule, in order: five whitespace-separated fields; no token outside digits, `*`, `,`, `/`, `-`, `?` and three-letter month or weekday names; no weekday number above 6; a successful `cron-parser` parse; and a next run within five years. Checking whole tokens is required because `JUL` and `THU` contain the letters `L` and `H`. The five-year limit matches the search of the server's parser and catches the expressions `cron-parser` answers with a false date, such as `0 0 31 2,4 *` with a next run in 2054. February 29 stays inside it.

The table in Context becomes a test table, so a change in either parser shows up.

Rejected:

- The plan's narrower rule, five fields and a parse. `0 0 L * *`, `0 0 * * 1#2`, `0 0 H * *` and `0 0 * * 7` would show a next run and fail after Save.
- Adding a request that validates on the server. A form field does not justify a new operation.

Saved `TZ=` and `CRON_TZ=` expressions fail the five-field rule, as the proposal records. Exempting an unchanged saved value would make the check remember the loaded value for a form that already shows such an expression as invalid.

### 7. `isScheduleComplete` follows the contract

The function moves anyway and has two consumers, so it takes the name of what it now checks: a fully typed but refused cron expression is not "unfilled", and the spec calls this rule completeness.

`HOURLY` needs nothing more. `DAILY`, `WEEKLY` and `MONTHLY` need a time, `WEEKLY` needs a weekday including `0`, `MONTHLY` needs a day, and `CRON` needs `getNextCronRun` to return a date. `null` and `undefined` are incomplete.

### 8. The server refuses an expression with no date, and `IsDue` ignores a stored one

`Validate` calls a private `isCronWithoutRunDate`, which asks the parser for the next run from a fixed start of 2024-01-01 UTC and reports a zero result. It wraps the existing `ErrInvalidCronExpression`, so `logical_config.go:184` and verification report it without new mapping. `IsDue` checks the same helper before its `lastRun == nil` branch, because a first run would otherwise still start. An unparsable stored expression keeps today's behavior.

The start is fixed rather than taken from a clock. Whether a day-and-month pair exists does not depend on the moment, `Validate` stays free of a clock, and 2024 is a leap year, so February 29 falls inside the parser's five-year search.

Rejected:

- A check in `isCronDue` only. A schedule that never ran would still start once at every pass until the first backup exists.
- A data migration. Nothing records what the expression was meant to do, and a member still sees what was entered.
- A new sentinel error. Nothing tells this case apart from any other invalid expression.

### 9. `cronExpression` documents the rule

The description becomes "The five-field cron expression of a cron schedule; an expression with no date to run on, such as February 31, is refused". `mise run generate-api` regenerates the comments. The 400 response and the field are unchanged, so no client code changes.

### 10. Schedule copy stays where it is until milestone 7

The editor reads the existing `backups.config.schedule` keys, and `backups.logical.config.{weekday,dayOfMonth,timeOfDay}` are deleted from all six dictionaries. `backups.logical.config.interval` stays, because the caller passes it as `label`. All schedule copy moves to its own group once, at milestone 7, when verification first reads it and `intervals` is removed. Dictionary typing makes a forgotten language fail the build.

Rejected: moving the twelve keys to a `schedules` group now. The user sees the same text, the entity would still read `intervals.types` and `intervals.weekdays`, and the same dictionaries would change again at milestone 7.

### 11. One time-of-day format in `shared/time`

`getUserClockFormat()` in `shared/time/getUserTimeFormat.ts` returns `{ use12Hours, format }` for a time without a date, using the private `getIs12HourFormat` already there. The six copies of `is12 ? 'h:mm A' : 'HH:mm'` call it, and the boolean `getUserTimeFormat` of `utils.ts` is deleted. `features/backups/physical/monitoring/lib/clockFormat.ts` is a seventh copy and is deleted too: `WalStreamChartsComponent.tsx:115` and `monitoringTextValues.ts:56` read `getUserClockFormat().format`.

Rejected: the plan's rename of that function to a predicate inside the entity. It would keep a second implementation of the same check.

### 12. The new slice meets the current design rules

`style-baseline.json` permits no new findings, so the editor uses `text-muted` and `text-danger` instead of `text-gray-600`, `text-red-500` and `color: gray`. The baseline entries of `PhysicalIntervalEditor` are removed, and the counts of `EditLogicalBackupConfigComponent` shrink to what remains. Visible labels become real `<label>` elements bound with `useId`, as `pages/profile/ui/ProfilePage.tsx:149` does, and `ScheduleEditor` is a `role="group"` named by its `label`. This satisfies "A visible label identifies the actual control" in `docs/frontend-design.md`, and two editors on one form stay distinct. The cron help uses the physical form's plain lines, without the logical form's bullets.

## Risks / Trade-offs

- [A saved `TZ=` expression blocks Save for other settings in that config] → The form shows the expression as invalid, and the user can rewrite it or turn backups off. The schedule keeps running.
- [A stored expression with no date silently stops running] → Before this change it started work at every pass, which is the defect. The form shows it as invalid. A notice to members is out of scope.
- [A stored full expression with no date blocks a backup type change] → `OnDatabaseBackupTypeChanged` at `event_handler_service.go:73` fits the config and validates it in the transaction of the change, and fitting replaces only the incremental schedule. The change is refused with `400` until a member rewrites the expression. Silently replacing a full schedule the member chose was rejected, and a physical config test pins the refusal.
- [A stored incremental schedule with no date is replaced] → `coerceForIncremental` at `physical_type_coercion.go:63` already replaces an invalid incremental schedule when it reads a config, and this expression now counts as invalid. A member sees the replacement in the form and the scheduler uses it. The exception is an hourly full schedule, reachable because `isRunningMoreOften` at `physical_config.go:231` lets any cron through: nothing can be built, `isIncrementalEnabled` at `backuping/scheduler.go:348` turns incremental backups off, and the form shows an empty incremental schedule until the full schedule is slowed down.
- [Turning backups off enables Save for states the server still refuses] → The server validates a config whether or not backups are on. A stored expression with no date, or an hourly full schedule with an empty incremental schedule, gets `400` after the click, and that refusal reaches the form. The member fixes the schedule first. Skipping validation for disabled configs was rejected, because it would store a schedule that fails the moment backups turn back on.
- [The client and server parsers still differ somewhere untested] → The server stays the authority, and its refusal still reaches the form.
- [A larger diff than a mechanical move] → Conversion, cron rule and backend fix each land with their own tests, and tasks reproduce every defect first.

## Migration Plan

No migration. Deploying changes the behavior as the proposal states, and reverting the commit restores it. Stored configs are not modified in either direction.
