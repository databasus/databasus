## Context

See `proposal.md` for the motivation. Paths are under `frontend/src`. Line numbers come from the checkout before `move-physical-and-verification-screens`, where the files sit under `features/verification/config/ui`.

- `EditVerificationConfigComponent.tsx:137-143` accepts any non-empty cron expression, and `:291-319` parses cron a second time for its preview. `:186-201` builds the select from `IntervalType`, and `:60-76` casts the value. `:211-345` repeats the fields of `entities/schedules/ui/ScheduleDetailsFields.tsx`.
- `:103` and `:114` use `alert`, and `:122` returns an empty `<div />` after a failed load. `ShowVerificationConfigComponent.tsx:37` uses `alert` too, and `:108-140` repeats the cron preview.
- `backend/internal/modules/verification/internal/configs/service.go:78-82` validates the schedule only when verification is enabled and on an interval. `config.go:17-31` returns a disabled AFTER_BACKUP default with a weekly schedule, and `config.go:13` makes the schedule non-null.
- `entities/schedules/ui/ScheduleEditor.tsx` is the type select plus `ScheduleDetailsFields`, which is not exported yet, because `extract-schedule-editor` left that for this milestone.
- `entities/schedules` reads `backups.config.schedule.*` and `intervals.{types,weekdays}.*`. `pages/databases/model/RetentionPeriodLabelKeys.ts` reads `intervals.periods.*`, the group's only other consumer.

The approach is bound by the root `AGENTS.md` (naming, logical units, no shims) and `frontend/AGENTS.md` (component structure, form conventions, localization skill). It follows "Keep state near its lifecycle" in `docs/architecture.md`. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- One implementation of schedule fields, cron validation and schedule text.
- The same failure behavior as the logical config section.

**Non-Goals:**

- Moving "After backup" into the schedule domain, and changing what the server validates.

## Decisions

### 1. The editor takes the details fields and keeps its own select

`entities/schedules` exports `ScheduleDetailsFields` and `getNextCronRun`. The verification editor renders one select with "After backup" and the five schedule types, and it renders the detail fields only when the config runs on a schedule. Choosing a type keeps the rest of the stored schedule, as `ScheduleEditor` already does.

Alternative rejected: giving `ScheduleEditor` an extra option through props or adding "After backup" to `ScheduleType` would make the schedule entity know about verification, and that value is not a schedule in the contract.

### 2. Two small model units state the save rule

`pages/databases/model/verification/isVerificationConfigComplete.ts` returns true when verification is off or runs after a backup, and `isScheduleComplete` otherwise. `isVerificationConfigChanged.ts` compares what the form shows: the switch always, the schedule choice and the notification triggers only while verification is on, and the schedule only while it runs on a schedule. The editor holds `acknowledgedConfig` and `draft`.

Alternative rejected: a shared comparison with `isLogicalBackupConfigChanged` would tie two different configs together for three lines of schedule comparison. `isScheduleChanged` moves from that file to `entities/schedules` so both use one schedule comparison.

### 3. A save sends the acknowledged values for what the form hides

`buildVerificationConfigForSave({ acknowledgedConfig, draft })` takes the schedule from `acknowledgedConfig` while verification is off or runs after a backup, and the schedule choice and the notification triggers from it while verification is off. Otherwise a user could type `0 0 31 2 *`, choose "After backup" and save, and because `backend/internal/modules/verification/internal/configs/service.go:78-82` skips validation in that state, the invalid expression would be stored.

Alternative rejected: sending the draft as it is keeps edits the user can no longer see, and it makes a later switch back to a schedule show values nobody confirmed.

### 4. All schedule text moves once

The new top-level `schedules` group holds `types`, `weekdays`, `weekday`, `dayOfMonth`, `cronExpression`, `timeOfDay`, `nextRun`, `invalidCron` and `cronHelp`. `backups.config.schedule`, `intervals.types`, `intervals.weekdays` and `verification.config.{weekday,dayOfMonth,timeOfDay,cron}` are deleted from all six dictionaries. `intervals.periods` moves to `backups.logical.retentionPeriods`, next to `backups.logical.retentionPolicyTypes`, because only logical retention reads it, so the `intervals` group disappears.

Alternatives rejected: keeping `backups.config.schedule` leaves a domain-neutral entity reading backup keys. Keeping the verification copies preserves their slightly different sentences, at the price of translations that must stay in step for no user benefit.

### 5. The view shows the next run through the same helper

`ShowVerificationConfigComponent` uses `getNextCronRun` through the public API of `entities/schedules` and the `schedules.nextRun` text, and it shows nothing extra for an expression with no next run. The view keeps its own load with a retry, because it remounts after a save and so shows the stored values without any coordination.

Alternative rejected: moving the load to `DatabaseConfigComponent`, as the logical view does, would make the parent refresh this config after each save and pass it down, while this section's view and editor are the only readers. The logical view has a parent load because the parent needs that config for its own decisions.

### 6. Failures render inline, as in the logical section

The editor shows `Alert` with a retry and Cancel after a failed load, and `Alert` above its actions after a refused save. The session guard, a check that the request's database is still the shown one and a load flag drop late answers, including late failures, and a late success does not close the section. Cancel stays disabled while a save is pending.

Alternative rejected: a context-bound toast cannot offer a retry and disappears before a user reads it.

### 7. The labels align at the shared width

The shared fields fix labels at 150 pixels, so the verification form moves from 180 to 150, including its switch and notification rows. The read-only view moves too, as the logical view already has, so opening the section does not shift the labels.

Alternative rejected: a width prop on the shared fields adds an option whose only purpose is keeping a difference nobody chose.

## Risks / Trade-offs

- A saved schedule that passes the server's parser but not the stricter interface check, such as a `TZ=` prefix, blocks Save while verification stays on that schedule. `recurring-schedules` already accepts this for backups, and the user can replace it or choose another option.
- Renaming keys touches all six dictionaries in one diff. Their typed keys and the dictionary checks catch a missed reference at lint time.
- Long labels in some locales have less room at 150 pixels. They wrap as on the backup forms, and the browser scenario checks a long locale.
- Rollback: reverting the commit restores the old text and editor. No stored data changes.
