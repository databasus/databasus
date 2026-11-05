## Context

See `proposal.md` for the motivation and `specs/recurring-schedules/spec.md` for the required behavior. The current state:

- `backend/internal/lib/schedule/schedule.go:179` builds this month's slot with `time.Date(now.Year(), now.Month(), *s.DayOfMonth, ...)`. Go normalizes day 31 of November to December 1. All of November is then before the slot, and on December 1 the slot becomes December 31, so November never runs. The backup, physical and verification schedulers share this rule through `IsDue`.
- `frontend/src/shared/time/utils.ts:34` uses `dayjs.utc().date(utcDom)` and line 40 uses `localTime.date(localDom)`. Both overflow in the same way. A throwaway run of these two helpers with the fixed clock set to each month gave these results:

| Zone and month | UTC day 31 at 10:00 shows as | Local day 31 at 10:00 saves as | Local day 1 at 02:00 saves as |
|---|---|---|---|
| UTC, November | 1 | 1 | 1 |
| UTC, October | 31 | 31 | 1 |
| UTC, February | 3 | 3 | 1 |
| Europe/Moscow, November | 1 | 1 | 31 |
| Europe/Moscow, October | 31 | 31 | 30 |
| Europe/Moscow, February | 3 | 3 | 31 |

- The six consumers call the two helpers only for display and for the edit callbacks. For example `frontend/src/features/backups/logical/ui/EditLogicalBackupConfigComponent.tsx:282`, `:414` and `:518`, and `frontend/src/features/backups/physical/ui/PhysicalIntervalEditor.tsx:57`, `:114` and `:203`. The editors already hold the time of day conversion separately, using the current UTC offset.
- `backend/api/openapi.yaml:5767` documents `dayOfMonth` as "The day of a monthly schedule". The range 1 to 31 is validated at `schedule.go:66`.

The approach is bound by the root `AGENTS.md` (logical units, names, no compatibility shims), `backend/AGENTS.md` and `frontend/AGENTS.md`. No architecture rule changes: the schedule rule stays in `internal/lib/schedule`, and the helpers stay in `shared/time` until the later schedule unit moves them. The api-design skill governs the description edit.

## Goals / Non-Goals

**Goals:**

- No monthly schedule is skipped because of the length of a month.
- The length of a month never changes the day the interface shows or saves, and a day survives an edit. A daylight saving change can still move the shown day together with the shown time, as the spec states.

**Non-Goals:**

- Exact local-day semantics in every month, which needs a stored time zone. See Risks.
- Moving the helpers or merging the schedule editors. The next schedule change does that.
- Correcting days the old interface already saved wrongly. Nothing records what the user meant.

## Decisions

### 1. A day past the end of a month means that month's last day

`isMonthlyDue` uses the smaller of the schedule's day and the number of days in `now`'s month. A small private `getDaysInMonth` sits next to `getStartOfMonth`. Once the slot is inside the month, the existing comparison already gives one slot per month and catch-up within the month. A first run and manual runs keep their existing behavior.

Together with the interface rule below, this also gives the right run for "the 1st at 02:00" at UTC+3. It is stored as UTC day 31 at 23:00, and the last day of any month at 23:00 UTC is the 1st at 02:00 local.

Rejected:

- Keeping the skip and warning in the interface. Backups would still be missed, only with a notice, and users at UTC+3 who chose the 1st would have no way to get monthly runs.
- Limiting the day to 28 in validation. This breaks saving every existing config with a later day, and "the last day" could no longer be expressed.
- Rewriting saved days in a migration. A saved 30 or 31 might be a real choice or the old interface's error, and the database cannot tell them apart.

### 2. The interface converts a day on a 31-day cycle by integer arithmetic

Both helpers compute the day shift from the time of day and the UTC offset, and wrap the result on a 31-day cycle:

- `getLocalDayOfMonth(utcDom, utcTime)` adds `dayjs.utc(utcTime, 'HH:mm').local().utcOffset()` to the time in minutes.
- `getUtcDayOfMonth(localDom, localTime)` subtracts `localTime.utcOffset()` from the time in minutes.
- `Math.floor(minutes / 1440)` gives a shift of -1, 0 or +1, and `((day + shift - 1 + 31) % 31) + 1` wraps the day.

Their signatures and the consumers stay the same, so all six consumers are fixed at once, and the logic has one owner. The editors derive the shown time from the same `dayjs.utc(timeOfDay, 'HH:mm').local()` value (`EditLogicalBackupConfigComponent.tsx:271-273`), and `getUtcDayOfMonth` subtracts the offset of the time it receives. The day and the time therefore use one offset, and a day round-trips even on a daylight saving switch day, when the current moment's offset can differ from the offset at that time of day. A test covers that day.

The format-string parse needs the dayjs `customParseFormat` plugin, which the app gets today only as a side effect of Ant Design's date picker, and `dayjs.utc` needs the `utc` plugin that `main.tsx` registers. `shared/time/utils.ts` registers both itself with `dayjs.extend`, which is safe to repeat, so the helpers do not depend on what else has loaded and work in a node test. Replacing the parse with `dayjs.utc().hour(h).minute(m)` was rejected: it takes its date from the UTC date while the editors' value takes it from the local date, so near midnight on a switch day the two offsets can differ.

The helpers stay in `shared/time/utils.ts`, whose name breaks the rule on file names, because renaming them now would touch all six consumers. The later schedule unit moves the helpers and this test together and names them after the concept.

Rejected:

- Clamping to the current month, so day 31 shows as 30 in November. The saved value would then change on any edit made in a short month, which is the corruption this change removes.
- Calculating on a fixed reference month such as January. That applies January's offset to the day and today's offset to the time, so the two disagree during daylight saving time.
- Storing a time zone in the schedule. It is the only exact answer, but it needs a contract, a migration and a scheduler change, and the plan does not ask for it.

### 3. The contract states the rule in the existing description

The `dayOfMonth` description becomes "The day of a monthly schedule. In a month without this day, the schedule runs on the month's last day." Scripts read the contract, and this is the only place they can learn the rule. Field names and ranges stay. `mise run generate-api` updates the comments in the generated code. `mise run backend:lint` checks that the Go output is fresh, and `mise run frontend:lint` checks the generated types. The `Schedule` doc comment at `schedule.go:23` says the same.

Rejected: leaving the contract alone. The rule would then be invisible to API users, and the description would be incomplete as soon as this change lands.

### 4. The rule is tested as pure logic, plus one screen

`TestIsDueForMonthlySchedule` gains month-end scenarios, because `IsDue` is pure logic with many date combinations and `docs/testing.md` allows unit tests for this. The frontend helpers get a node unit test over zones and months, setting the zone with `vi.stubEnv('TZ', zone)`. A screen test in `pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx` runs at a fixed November clock and checks the shown day and the saved request.

Rejected: a scheduler API scenario with a moved clock. It needs a real source database for a run that only repeats the pure rule, and it would add `./internal/modules/backups` to the verification without covering anything more. Passing the offset to the helpers as a test parameter was also rejected, because no production caller needs it.

## Risks / Trade-offs

- [A local day near a month boundary can land a day off in a shorter month. At UTC-5, day 30 at 22:00 is stored as UTC day 31 at 03:00, which in November runs on November 30 at 03:00 UTC, the local 29th.] → Accepted without a stored time zone. It still runs once in the month, which is the guarantee this change gives. The spec states the 31-day cycle explicitly.
- [Schedules on days 29 to 31 start running in short months, so they produce more backups than before.] → This is the intended fix. Retention by count, period and GFS keep working as they do for any other run.
- [The last day's slot leaves a short window for catch-up. An outage from the slot to midnight misses that month.] → Unchanged in kind: any missed monthly slot is caught up only within its month today, and real day 31 slots already have this window.
- [A day the old interface saved wrongly now shows its true value, so a user may notice a day they did not choose.] → Correct and visible, and the user can change it. The summary of the change names this.
- [Daylight saving can move the local time across midnight, because a schedule stores a UTC time.] → An existing limit of every schedule type. This change neither improves nor worsens it.

## Migration Plan

No data migration. The new rule applies from the first scheduler pass after an upgrade. A schedule on day 31 that already ran in October simply becomes due on November 30. It does not cause a burst, because each config still has one slot in its month. Rollback is reverting the commit, and saved configs need no change.
