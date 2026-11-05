Each task starts by adding its tests on the unchanged code, with responses declared on the real screen, and by seeing them fail for the reason the task names.

## 1. Shared schedule units

- [x] 1.1 Export `ScheduleDetailsFields` and `getNextCronRun` from `entities/schedules`, move `isScheduleChanged` there with its export, and update `isLogicalBackupConfigChanged`. Verification: `pnpm --dir frontend test src/entities/schedules src/pages/databases/model`.
- [x] 1.2 Add `isVerificationConfigComplete`, `isVerificationConfigChanged` and `buildVerificationConfigForSave` with node tests under `pages/databases/model/verification`, covering verification off, after a backup and on each schedule type, and a config acknowledged on an interval whose draft chose after-backup and turned verification off, which keeps the acknowledged `scheduleType`. Verification: `pnpm --dir frontend test src/pages/databases/model/verification`.

## 2. Editor and view

- [x] 2.1 In `frontend/src/pages/databases/ui/verification/EditVerificationConfigComponent.test.tsx`, add tests for: `0 0 31 2 *` shows the invalid message and no next run and keeps Save disabled; an invalid expression hidden by choosing after-backup or by turning verification off sends the acknowledged schedule; the fields read "Weekday" and "Time of day"; a failed load offers a retry and Cancel, and a successful retry shows the form; a refused save keeps the input and shows the reason; Cancel is disabled while a save is pending; a reverted edit and the two hidden edits disable Save; a save after changing a notification trigger and the schedule choice and turning verification off sends the acknowledged values; an answer held until after unmount or a session change shows nothing; and a save held while the component rerenders with another database leaves the section untouched with no feedback when it answers. Rewrite the editor over `acknowledgedConfig` and `draft`, using the shared fields and the model units, with inline load and save failures, a guard on the session and the database, and semantic roles. Delete its cron parsing and labels, and remove its entries from `style-baseline.json`. Verification: `pnpm --dir frontend test src/pages/databases/ui/verification/EditVerificationConfigComponent.test.tsx` and `mise run frontend:lint`.
- [x] 2.2 In `ShowVerificationConfigComponent.test.tsx`, add tests where a failed load offers a retry without `alert`, a successful retry shows the config, a late answer after unmount shows nothing, and the fields read "Weekday" and "Time of day". Rewrite the view with the shared schedule labels, `getNextCronRun`, an inline failure, a retry, 150-pixel labels and semantic roles, and remove its entries from `style-baseline.json`. Verification: `pnpm --dir frontend test src/pages/databases/ui/verification/ShowVerificationConfigComponent.test.tsx` and `mise run frontend:lint`.
- [x] 2.3 Delete `entities/intervals`. Verification: `rg -n 'IntervalType|entities/intervals|INTERVAL_TYPE_LABEL_KEYS' frontend/src` finds nothing and `mise run frontend:lint` passes.

## 3. Text

- [x] 3.1 Under the frontend-localization skill, create the `schedules` group in all six dictionaries, move `intervals.periods` to `backups.logical.retentionPeriods`, update every consumer and delete the old keys. Verification: `rg -n "intervals\.(types|weekdays|periods)|backups\.config\.schedule|verification\.config\.(weekday|dayOfMonth|timeOfDay|cron)" frontend/src` finds nothing, `rg -n '^  intervals: \{' frontend/src/shared/i18n/locales` finds nothing, and `mise run frontend:lint` and `pnpm --dir frontend test src/shared/i18n src/entities/schedules` pass.

## 4. Browser and handover

- [x] 4.1 Add `frontend/browser/scenarios/verificationConfig.spec.ts`: on a mocked API, edit a schedule in a long locale, save, and read it again after the view reloads. Verification: `pnpm --dir frontend test:browser --project=ui`.
- [x] 4.2 Search for every renamed key group and removed path in `docs`, `.agents` and `frontend/README.md`, and update what they say. Mark this item done in `temp/refactor/frontend-plan.md`. Verification: `rg -n 'intervals\.(types|weekdays|periods)|entities/intervals' docs .agents frontend/README.md` finds nothing, and `rg -n '\[x\] \*\*7\.3\.' temp/refactor/frontend-plan.md` finds the line.
- [x] 4.3 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend test`. Verification: all three pass.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
```
