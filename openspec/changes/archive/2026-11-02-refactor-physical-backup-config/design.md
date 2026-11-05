## Context

See `proposal.md` for the motivation and the delta spec for the required behavior. Paths are under `frontend/src`. Line numbers of the physical editor and show view come from the checkout before `move-physical-and-verification-screens`, under `features/backups/physical/ui`, and those of `CreateDatabaseComponent.tsx` from before `fix-wizard-partial-create`.

- `EditPhysicalBackupConfigComponent.tsx:43-60` holds the mode flags, and `:207`, `:238`, `:281`, `:353`, `:362`, `:700` and `:706` branch on them. `CreateDatabaseComponent.tsx:51` builds `toUnsavedDatabase` for this editor and for `EditDatabaseNotifiersComponent`, whose `isSaveToApi`, `isShowBackButton`, `isShowCancelButton` and `isShowSaveOnlyForUnsaved` follow the same pattern (`:13-27`).
- `:122-125` keeps `backupConfig.settings.retention.fullBackups` and `fullBackupsRetention` side by side, `:194-200` fills both, and `:91-104` joins them at save. It forwards both the count and the GFS periods, while `backend/internal/modules/backups/internal/config/physical_config.go:37-60` refuses `LAST_N` with any GFS period and GFS with a non-zero count.
- `CreateDatabaseComponent.tsx:219` drops the draft on Back. `pages/databases/ui/backup-config/CreateLogicalBackupConfigStep.tsx` is the working pattern, and `fix-wizard-partial-create` records the accepted backup type when the wizard stores a configuration.
- `pages/databases/ui/backup-config/EditLogicalBackupConfigSection.tsx` holds the load with retry and Cancel, the inline save error, the backups switch, the footer, `StorageSwitchHistoryComponent` and the storage-switch confirmation. The physical section differs only in the conversion, the predicates, the fields and the `isPhysical` argument of `hasBackupHistory`.
- `:226` and `:299` use `alert`, and `:320` returns an empty `<div />` after a failed load. `ShowPhysicalBackupConfigComponent.tsx:137` uses `alert` and loads the config that `DatabaseConfigComponent.tsx` already loads into `backupConfig`.
- `physical_config.go:115-136` validates the full schedule and retention whatever `IsBackupsEnabled` says, while the editor's `:351-358` skips every check with backups off.
- `entities/backup-configs/ui/StorageSwitchHistoryComponent.tsx:13` takes `initialOperation`, which this editor passes at `:710` and `pages/databases/ui/DatabaseTransferDialogComponent.tsx:376` passes to show the switch a transfer accepted.
- `openspec/specs/frontend-database-screens/spec.md:148` states the exception this change removes.

The approach is bound by the root `AGENTS.md` (logical units, names, no shims, refactoring within a feature) and `frontend/AGENTS.md` (component structure, form conventions). It follows "Database pages compose reusable editing" in `docs/architecture.md`: each container owns its draft, its saving and its navigation. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- One set of fields for the wizard and the section, with the differences in small containers.
- One retention draft, and one place that knows the contract's shape.
- One tested flow for a backup configuration section.
- `toUnsavedDatabase` deleted.

**Non-Goals:**

- A common retention model with logical configs, and an editor for hidden settings.

## Decisions

### 1. A shared section and two physical containers

`EditBackupConfigSection<TDraft>` takes over the flow of `EditLogicalBackupConfigSection`: the load with retry and Cancel, the acknowledged and shown values, the save, the backups switch, the storage-switch warning and the history. Its props are `database`, `isPhysical`, `toDraft`, `toInput`, `isChanged`, `isComplete`, `renderFields`, `onCancel` and `onSaved`. `EditLogicalBackupConfigSection` and `EditPhysicalBackupConfigSection` become thin wrappers that supply these.

`CreatePhysicalBackupConfigStep` takes `workspaceId`, `backupType`, an optional `initialDraft`, and `onContinue` and an optional `onBack`, each receiving the draft. The wizard holds the draft in state, records `acceptedPhysicalBackupType` whenever it stores a draft, on Back as well as on Continue, and builds the input only for the request. `PhysicalBackupConfigFields` is controlled and split into `PhysicalRetentionFields` and `PhysicalNotificationTriggerFields`. The database's backup type reaches the fields as a prop, because it belongs to the database and not to the config.

Alternatives rejected: one component with a discriminated `mode` prop removes the booleans but keeps the wizard and the section in one file, and the logical split has already shown what they need. A second copy of the section would duplicate the flow and its tests.

### 2. One retention draft and one input function

`PhysicalBackupConfigDraft` differs from `PhysicalBackupConfigInput` in one field: `settings.retention.fullBackups` is never null, and it is filled with the defaults when the server returned nothing. `toPhysicalBackupConfigDraft` builds it, and `toPhysicalBackupConfigInputFromDraft` drops what the contract forbids. `CHAINS` sends `fullBackups: null`, and `FULL_BACKUPS` sends `chainsCount: 0`. Within the full-backup retention, `LAST_N` sends zero GFS periods and GFS sends `count: 0`. Everything else in the settings passes through, including `walLagThresholdBytes`. The name keeps the contract's word "input", because `toPhysicalBackupConfigInput` in `entities/backup-configs` already converts a `BackupConfig`.

Alternatives rejected: a shared `RetentionEditor` for logical and physical would merge period, count and GFS with chains and full backups, whose rules differ. Keeping `fullBackupsRetention` beside the config keeps the manual synchronization this change removes.

### 3. Save compares what the form shows

`isPhysicalBackupConfigChanged` compares only the fields the form shows for the database's backup type and policies, as its logical counterpart does. `isPhysicalBackupConfigComplete` applies the retention rules always, and the encryption and schedule rules only with backups on, as `isLogicalBackupConfigComplete` does. The wizard additionally requires a storage.

Alternative rejected: comparing the whole input treats a filled-in default of an unchosen policy as an edit, and requiring a schedule with backups off contradicts `recurring-schedules`, which keeps that rule even though a broken stored schedule then gets a server refusal.

### 4. The notifier step splits the same way

`CreateDatabaseNotifiersStep` takes `workspaceId`, the initial IDs, `onBack` and `onContinue`. `EditDatabaseNotifiersSection` takes `database`, `onCancel` and `onSaved`. Both render `DatabaseNotifiersFields`, which keeps the nested notifier creation and the acknowledged-notifier merge unchanged. This completes the removal of `toUnsavedDatabase`.

Alternative rejected: keeping the notifier component with a smaller `Pick<Database, ...>` input would delete the adapter but keep the mode flags, which the plan names for removal.

### 5. The settings screen passes the loaded config to the view

`ShowPhysicalBackupConfigComponent` receives the `BackupConfig` that `DatabaseConfigComponent` already loads, as its logical counterpart does, and it keeps its own load only for the storage, with an inline retry.

Alternative rejected: keeping two loads gives two errors and two retries for one failed server, which the load requirement's "exactly one" scenario excludes for logical.

### 6. The post-save history request goes away

`hasBackupHistory` after a save and `acceptedSwitch` are deleted, because the section closes as soon as the request answers and the view loads the history itself. `refactor-logical-backup-screens` made the same deletion. `initialOperation` stays with its test, because the transfer dialog still uses it.

Alternative rejected: keeping them preserves a state nobody sees, and it costs an extra request on every save.

## Risks / Trade-offs

- Applying retention rules with backups off can disable Save for a stored config the old editor could send. The server would refuse such values anyway, and the retention fields stay on screen, so the user can fix them.
- A stored schedule that is broken with backups off still gets a server refusal on Save, now shown inline. Fixing that needs a decision about showing hidden fields, and it is left as it is.
- The shared section changes a tested logical flow. Its tests stay and must pass before the physical wrapper is added.
- The notifier split touches a tested composition. `NotifierComposition.test.tsx` runs on the real editor and stays as the guard, and `coverage.md` maps every moved test.
- Rollback: reverting the commit restores the old editor. No stored data changes.
