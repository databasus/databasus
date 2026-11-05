## 1. Monthly schedule rule

- [x] 1.1 Add month-end scenarios to `TestIsDueForMonthlySchedule` in `backend/internal/lib/schedule/schedule_test.go` from the "A monthly schedule has a slot in every month" requirement: day 31 at 08:00 on November 30 at 07:59, at 08:00 and at 10:00; after a run on the November slot, not due on December 1 or December 30 and due on December 31 at 08:00; a missed November slot not due on December 1; a manual run on November 5 still due on November 30 at 08:00; day 30 in February of a common and a leap year; and day 31 in a month that has it. Run `mise run backend:test-package ./internal/lib/schedule` on the unfixed code and record that the due scenarios fail because November and February are skipped.
- [x] 1.2 Make `isMonthlyDue` use the smaller of the schedule's day and the days in `now`'s month through a private `getDaysInMonth`, per decision 1 of `design.md`, and extend the `Schedule` doc comment. Verify with `mise run backend:test-package ./internal/lib/schedule` and `mise run backend:lint`.

## 2. Contract

- [x] 2.1 Load the api-design skill, change the `dayOfMonth` description in `backend/api/openapi.yaml` per decision 3 of `design.md`, and run `mise run generate-api`. Verify that the generated diff holds only that description, with `mise run backend:lint`, `mise run frontend:lint` and `pnpm --dir frontend typecheck`.

## 3. Day conversion in the interface

- [x] 3.1 Add `frontend/src/shared/time/utils.test.ts`, a node test over UTC, Europe/Moscow and America/New_York with the clock fixed in October, November and February. It covers the scenarios of "The interface keeps a monthly day whatever the current month is", including round trips of every day from 1 to 31 and a round trip in America/New_York on the daylight saving switch day 2026-11-01, with UTC day 10 at 04:30. Run `pnpm --dir frontend test src/shared/time/utils.test.ts` on the unfixed helpers and record that it fails with the day overflow values from `design.md`.
- [x] 3.2 Add screen scenarios to `frontend/src/pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx` at a fixed November clock and a UTC zone: a loaded monthly config on day 31 shows 31, choosing day 31 on a config with another day submits 31, and changing only the time of a config on day 31 submits 31. Run `pnpm --dir frontend test src/pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx` on the unfixed helpers and record that the new scenarios fail with the wrong day.
- [x] 3.3 Rewrite `getLocalDayOfMonth` and `getUtcDayOfMonth` in `frontend/src/shared/time/utils.ts` per decision 2 of `design.md`, keeping their signatures and registering the `utc` and `customParseFormat` dayjs plugins in that file. Verify with `pnpm --dir frontend test src/shared/time/utils.test.ts src/pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx`.

## 4. Plan and handover

- [x] 4.1 Update `temp/refactor/frontend-plan.md`: add items 6.1 to 6.4 under milestone 6 with this change as 6.1, and mark as already done the `onSaved` fix (only its regression tests remain), the storage composition and the restore form slot. Record the agreed decisions for the later changes: logical backups and restores move to the page and their features are deleted, the schedule editor splits into a full editor and its detail fields so verification can reuse them, schedule fields use shared labels, schedule validation requires a time and a five-field cron, and the real API E2E for the config is replaced by a mocked browser round trip because creating a database connects to it. Also record the defects found for 6.4: the wizard draft reset, Continue disabled after Back, the restore dialog crash on a vanished backup, a late save closing another section or restoring another database, an alert on each failed restore poll, a failed config load shown as disabled backups, and an empty editor after a failed load. Verify with `rg -n "fix-monthly-schedule-day" temp/refactor/frontend-plan.md`.
- [x] 4.2 Run `mise run frontend:lint` and `pnpm --dir frontend build`, then the commands of the Change verification section, and record their results.

## Change verification

```sh
mise run backend:test-package ./internal/lib/schedule
pnpm --dir frontend test
```
