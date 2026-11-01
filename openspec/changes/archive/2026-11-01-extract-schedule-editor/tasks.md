## 1. Backend rule for a cron expression with no date

- [x] 1.1 Load the testing skill. Add scenarios to `TestValidate` and `TestIsDueForCronSchedule` in `backend/internal/lib/schedule/schedule_test.go`: `0 0 31 2 *` and `0 0 31 2,4 *` fail validation with `ErrInvalidCronExpression`, `0 0 29 2 *` passes, `0 0 31 2 *` is not due with a last run or without one, and `TZ=Europe/Moscow 0 2 * * *` stays valid and becomes due. Run `mise run backend:test-package ./internal/lib/schedule` on the unfixed code and record that the `0 0 31 2 *` and `0 0 31 2,4 *` scenarios fail because the expressions validate and count as due.
- [x] 1.2 Implement decision 8 of `design.md` in `backend/internal/lib/schedule/schedule.go` and update the `Schedule` doc comment and the `IsDue` comment, whose "A schedule that never ran is due at once" no longer holds without an exception. Verify with `mise run backend:test-package ./internal/lib/schedule` and `mise run backend:lint`.
- [x] 1.3 Add API scenarios that return 400, name the field the spec states and keep the stored config for `0 0 31 2 *`: a logical config and `0 0 31 2,4 *` in `backend/internal/modules/backups/backup_config_test.go`, a physical full and a physical incremental schedule in `backend/internal/modules/backups/physical_config_test.go`, and a verification config in `backend/internal/modules/verification/config_test.go`. Add a scenario to `backup_config_test.go` that saves a logical config with `0 0 29 2 *` and sees it stored. Add three scenarios to `physical_config_test.go` that place the expression with direct SQL: a stored incremental expression reads back as the built daily schedule under a weekly full schedule, reads back as no schedule under an hourly full schedule, and a stored full expression makes a backup type change return 400 with the database and config unchanged. If that change does not answer 400, map the refusal so it does. Add scenarios to `backend/internal/modules/backups/physicalapitests/backuping/scheduler_test.go` that store the expression with direct SQL and run a scheduler pass: the built daily schedule starts an incremental, and an hourly full schedule starts nothing. Add a case to `scheduler_decision_test.go` that a stored full expression schedules no first full, and a scenario to `backend/internal/modules/verification/scheduler_test.go` that a stored expression creates no verification. Add one scenario to `TestLogicalBackupScheduler` in `backend/internal/modules/backups/logical_backup_schedule_test.go` that places the expression with direct SQL, because the API now refuses it, runs a scheduler pass and sees no backup. Verify with `mise run backend:test-package ./internal/modules/backups ./internal/modules/backups/internal/physical/backuping ./internal/modules/backups/physicalapitests/backuping ./internal/modules/verification`.

## 2. Contract

- [x] 2.1 Load the api-design skill, change the `cronExpression` description in `backend/api/openapi.yaml` per decision 9, and run `mise run generate-api`. Verify that the generated diff holds only that description, with `mise run backend:lint`, `mise run frontend:lint` and `pnpm --dir frontend typecheck`.

## 3. Weekly conversion

- [x] 3.1 Add weekly scenarios to `frontend/src/shared/time/utils.test.ts` from the "The interface keeps a weekly day through its conversion" requirement, over UTC, Europe/Moscow, America/New_York and Africa/Cairo on 2026-10-27, including round trips of every weekday at several times and weekday `0`. Also pin a schedule saved as Thursday at 21:00 UTC shown as Thursday at 23:00 in Africa/Cairo after the offset change. Run `pnpm --dir frontend test src/shared/time/utils.test.ts` on the unfixed code and record that the Africa/Cairo round trip fails.
- [x] 3.2 Rewrite the weekly functions per decision 4 of `design.md`, keeping their signatures. Verify with `pnpm --dir frontend test src/shared/time/utils.test.ts`.

## 4. The schedules entity

- [x] 4.1 Load the feature-sliced-design skill. Create `frontend/src/entities/schedules` per decision 1. Move `Schedule`, `ScheduleType` and `ScheduleTypeLabelKeys` from `entities/backup-configs`, `Weekday` and `WeekdayLabelKeys` from `entities/intervals`, and `shared/time/utils.ts` with its test to `model/scheduleDayConversion` without bridges. Add `getUserClockFormat` to `shared/time`, delete the boolean `getUserTimeFormat`, and delete `features/backups/physical/monitoring/lib/clockFormat.ts` with its two callers reading `getUserClockFormat().format`, per decision 11. Point every consumer at the new public APIs. The verification editor and show view use `getLocalScheduleParts` and the `withLocal*` functions, so the weekly requirements hold there too. Verify with `pnpm --dir frontend typecheck`, `mise run frontend:lint`, `pnpm --dir frontend test src/entities/schedules` and `pnpm --dir frontend test src/features/backups/physical/monitoring`.
- [x] 4.2 Add `model/getNextCronRun` with a node test whose table is the one in `design.md`, plus `0 0 ? * *` and `*/5 1-3 1,15 JUL mon-fri`, per decision 6, including `0 0 31 2,4 *` refused and `0 0 29 2 *` accepted. Verify with `pnpm --dir frontend test src/entities/schedules`.
- [x] 4.3 Move `isScheduleFilled` to the entity, rename it `isScheduleComplete` and make it follow decision 7, with a node test of every schedule type, weekday `0`, `null` and the cron cases. Verify with `pnpm --dir frontend test src/entities/schedules` and `pnpm --dir frontend typecheck`.
- [x] 4.4 Add `model/localSchedule` per decision 5, with node tests of a day chosen before a time at UTC+3 with the clock at 01:00 local, for weekly and monthly schedules, and of a time change that keeps the local day. Verify with `pnpm --dir frontend test src/entities/schedules`.

## 5. Editor and copy

- [x] 5.1 Load the frontend-localization skill. Delete `backups.logical.config.{weekday,dayOfMonth,timeOfDay}` from all six dictionaries per decision 10 and point the logical show view at `backups.config.schedule`. Verify with `pnpm --dir frontend typecheck` and `pnpm --dir frontend test src/shared/i18n`.
- [x] 5.2 Load the frontend-design skill. Add `ui/ScheduleEditor`, `ui/ScheduleDetailsFields` and `ui/CronExpressionField` per decisions 2 and 12, with only `ScheduleEditor` exported through `index.ts`. Add `ui/ScheduleEditor.test.tsx` covering the whole returned value, kept fields after a type switch, only the type choice with `null`, the next run and the invalid message for the cron table, 12- and 24-hour time, and labels found by accessible name. Verify with `pnpm --dir frontend test src/entities/schedules` and `mise run frontend:lint`.

## 6. Consumers

- [x] 6.1 Use `ScheduleEditor` in `EditLogicalBackupConfigComponent`, with the `gfsHours` rule of decision 3, and delete its inline schedule form. Update `EditLogicalBackupConfigComponent.test.tsx` to the new labels and accessible queries. Add scenarios that a type change to daily under GFS sets `gfsHours` to 24, that a time change keeps it, and that Save is disabled for an incomplete schedule. Add a weekly scenario that finds "Weekday" and "Time of day" by accessible name and sees no "Backup weekday" or "Backup time of day". Add a config loaded with `TZ=Europe/Moscow 0 2 * * *`: the expression shows as invalid and Save stays disabled, then Save enables after replacing the expression, and separately after turning backups off. Verify with `pnpm --dir frontend test src/pages/databases/ui/backup-config`.
- [x] 6.2 Use `ScheduleEditor` twice in `EditPhysicalBackupConfigComponent` and delete `PhysicalIntervalEditor` and `mergeSchedule`. Add scenarios to `frontend/src/pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx`: two weekly schedules keep separate labelled fields, an empty incremental schedule offers only the type and keeps Save disabled, and an edit to one schedule leaves the other unchanged in the submitted config. Verify with `pnpm --dir frontend test src/pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx`.
- [x] 6.3 Use `getLocalScheduleParts` in `ShowLogicalBackupConfigComponent` and `ShowPhysicalBackupConfigComponent`, and add show-view scenarios for weekday `0` and for the "Weekday" and "Time of day" names of a weekly logical config. Verify with `pnpm --dir frontend test src/pages/databases/ui` and `pnpm --dir frontend test src/features/backups/physical`.
- [x] 6.4 Bring `frontend/style-baseline.json` in line per decision 12 and remove the now-empty exports from `entities/backup-configs` and `entities/intervals`, deleting the now-unused `Interval` and keeping `IntervalType` and `INTERVAL_TYPE_LABEL_KEYS` for verification. Verify with `mise run frontend:lint`.

## 7. Specs, plan and handover

- [x] 7.1 Change the Purpose of `openspec/specs/recurring-schedules/spec.md` so that it covers weekly days, cron schedules and the completeness rule as well as monthly schedules. Verify with `sed -n 3,5p openspec/specs/recurring-schedules/spec.md | rg -i 'weekly' && sed -n 3,5p openspec/specs/recurring-schedules/spec.md | rg -i 'cron'`, `openspec validate recurring-schedules --type spec --strict` and `openspec validate extract-schedule-editor --strict`.
- [x] 7.2 Update item 6.3 of `temp/refactor/frontend-plan.md`: record the additions to the plan (backend fix, stricter cron rule, weekly conversion, `isScheduleComplete`, `getUserClockFormat` instead of a predicate) and the move of all schedule copy at milestone 7, then mark it done. Verify with `rg -n '^\s*- \[x\] \*\*6\.3\.' temp/refactor/frontend-plan.md` and `rg -n 'getUserClockFormat' temp/refactor/frontend-plan.md`.
- [x] 7.3 Search the repository for `PhysicalIntervalEditor`, `shared/time/utils`, `isScheduleFilled`, `getClockFormat` and the deleted logical keys, and fix any remaining mention. Run `mise run backend:lint`, `mise run frontend:lint` and `pnpm --dir frontend build`, then the commands of the Change verification section, and record their results.

## Recorded results

Reproduced on the code before each fix:

- Task 1.1: `mise run backend:test-package ./internal/lib/schedule` failed in `TestValidate` for `0 0 31 2 *` and `0 0 31 2,4 *` with `Expected error with "invalid cron expression" in chain but got nil`, and in `TestIsDueForCronSchedule` with and without a last run with `Should be false`.
- Task 1.3: with `schedule.go` temporarily restored to its committed version, "schedules nothing before the built daily slot when the stored cron has no date to run on" failed because a backup started. It passes on the fixed code.
- Task 3.1: `pnpm --dir frontend test src/shared/time/utils.test.ts` failed for Africa/Cairo on 2026-10-27, both in the round trip and in Friday at 00:00, with `expected 4 to be 5`. The other zones passed.

Final checks of task 7.3 on the finished code:

- The search for stale names found no mention outside `openspec/` and `temp/`.
- `mise run backend:lint`, `mise run frontend:lint` and `pnpm --dir frontend build` passed.
- `pnpm --dir frontend test` passed 1647 tests in 148 files.
- The backend command below passed all five packages.
- `openspec validate extract-schedule-editor --strict` passed.

## Change verification

```sh
mise run backend:test-package ./internal/lib/schedule ./internal/modules/backups ./internal/modules/backups/internal/physical/backuping ./internal/modules/backups/physicalapitests/backuping ./internal/modules/verification
pnpm --dir frontend test
```
