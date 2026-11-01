## Context

See `proposal.md` for the motivation and the delta spec for the required behavior. All paths are under `frontend/src`. The defects were found by reading the code; each task reproduces its defect before the fix.

- `pages/databases/ui/CreateDatabaseComponent.tsx:101` calls `createDatabase` on every completion. `:104`, `:110`, `:107` and `:113` then save the configuration and start the backup, and `:119` reports any failure through a toast and returns to the notifier step. `:248` shows `databases.create.creating` for the whole chain.
- `:117-118` calls `onCreated` and `onClose` only after everything succeeded. `pages/databases/ui/DatabasesComponent.tsx:292` passes `onCreated` to `acceptAddedDatabase` (`:122`), which adds the database, selects it and refreshes the list. A database created by a partial failure reaches the list only through a later background refresh.
- `DatabaseComponent` and its children read the backup configuration once when they mount (`pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx:122-144`, `features/backups/physical/ui/PhysicalBackupsComponent.tsx:230-237`), and the server's default has backups off (`backend/internal/modules/backups/internal/config/defaults.go:27`). Selecting the database before the configuration is saved would show scheduled backups as disabled.
- `backend/internal/modules/backups/internal/config/storage_switch_acceptance_service.go` skips a storage switch only when the previous storage is empty, so a first save is a plain save and a later save with another storage is an accepted switch.
- `CreateDatabaseComponent.tsx:86` compares only `DatabaseType`. `features/database-connection/ui/EditPostgresqlPhysicalConnectionComponent.tsx:61` changes `backupType` on the connection step, and `CreateDatabaseComponent.tsx:215` then seeds the physical editor with the configuration of the old backup type.
- `backend/internal/modules/backups/internal/config/physical_config.go:115-228` refuses that configuration. For FULL it requires policy `FULL_BACKUPS`, `chainsCount` 0, no incremental schedule and WAL lag 0.
- `shared/auth/useSessionGuard.ts` turns false after unmount or a session change.

The approach follows the root `AGENTS.md` (names, logical units, no shims) and `frontend/AGENTS.md` (component structure, localization skill). It follows `docs/architecture.md`: "Database pages compose reusable editing" says each container owns its draft, its saving and its navigation, "Keep state near its lifecycle" says cleanup invalidates late results, and "Storage switches" says the interface warns before one. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- One database per wizard, whatever fails after the creation.
- A retry that does only the unfinished work and that the user can correct first.
- A screen that shows the saved values once the wizard closes.

**Non-Goals:**

- Server-side atomic creation, and cleanup of a database the user abandons.
- The physical editor's restructuring, which `refactor-physical-backup-config` does.

## Decisions

### 1. The wizard keeps the created database and whether its configuration is acknowledged

`CreateDatabaseComponent` holds `createdDatabase` and `isBackupConfigAcknowledged` in state. Completion runs one function: create if `createdDatabase` is absent, update the database if `notifierIds` differ from it, save the configuration if it is not acknowledged, then start the first backup, then call `onClose`. Each step records its result before the next one starts. After each await the function checks `useSessionGuard`, so an unmounted wizard or an old session starts no later step, shows no error and calls no `onClose`. `onCreated` follows the create response after that check, so a closed wizard never calls it.

Alternatives rejected: closing the wizard after a partial failure and sending the user to the settings is simpler, but it discards the entered configuration and leaves a database whose backups were never set up. The proposal's review lists this option for the user with its saving. A new server operation that creates a database with its configuration would be atomic, but it needs a contract change for a problem one state field solves. Comparing the configuration by value would let the step stay editable after acknowledgment, but it brings back the storage switch from decision 4.

### 2. Notifier changes go through the existing update operation

A retry sends `databaseApi.updateDatabase(createdDatabase.id, { ...createdDatabase, notifierIds })`, the same request the notifier section sends today. The created database comes from the server response, so it holds no secret and the omission rule keeps the stored credentials. The response replaces `createdDatabase`.

Alternative rejected: locking the notifier step after creation removes one request but leaves the user unable to fix a selection, and it needs its own explanation in the interface.

### 3. The host remembers the database on creation and accepts it on close

`onCreated` fires once and only remembers the database in a ref of `DatabasesComponent`, because the completion function that later calls `onClose` began before the database existed and holds the older render's callback. Every way of closing the wizard, its `onClose` and the modal's cancel, goes through one `closeWizardAndAcceptCreatedDatabase`, which hides the modal, clears the ref and passes a remembered database to the unchanged `acceptAddedDatabase`, which adds it, selects it and refreshes the list silently. A create answer that arrives after its wizard closed reaches nobody, so the periodic refresh lists that database, and closing a wizard opened later selects nothing from it. `DatabaseComponent` therefore mounts after the requests have finished, so it reads the saved configuration, notifiers and first backup, and the list card shows the storage.

Alternatives rejected: selecting at creation mounts the screen behind the modal, and its single configuration read then says backups are disabled until the user reselects. Selecting at creation and reloading at the end needs a new reload channel into `DatabaseComponent`, while nobody can see the screen behind the modal anyway. Adding to the list at creation needs a second callback and a split of `acceptAddedDatabase`, for a list that sits behind the same modal.

### 4. The way back ends in two stages

After creation, the logical step receives no `onBack`, and the physical editor receives `isShowBackButton={false}`, so the limited user and connection steps become unreachable. After the server acknowledges the configuration, the notifier step's Back disappears too. Only the first backup start and a changed notifier selection can remain, and neither needs that step. `CreateLogicalBackupConfigStep`'s `onBack` becomes optional, and the button appears only when it is passed.

Alternatives rejected: sending connection edits through an update would create a second edit path inside the wizard, with its own connection test and secret rules. Keeping the configuration step editable after acknowledgment requires the storage-switch warning and progress that `docs/architecture.md` demands, or an exception to that rule, for a state that a failed backup start does not need.

### 5. The error stays on the screen and progress names the right work

A failure after creation is stored and shown as an `Alert` above the notifier and backup configuration steps until the next completion attempt. The new key `databases.create.createdButNotFinished` in all six dictionaries takes the translated reason. A toast disappears, and the user must still see that the database exists when they move between the steps. A failed creation keeps its current toast, because nothing has changed on the server. While a retry runs, the wizard shows a new `databases.create.finishing` text, because `databases.create.creating` would read as a second creation.

Alternative rejected: only a toast with the same text, because the user could read the retry as harmless creation.

### 6. The backup type is compared on Continue

`CreateDatabaseComponent` records `acceptedPhysicalBackupType` when it stores the physical configuration. On Continue from the connection step, a different backup type clears `physicalBackupConfig`. This mirrors how `acceptDatabaseType` treats a different database type, including that returning to the original value keeps the input.

Alternatives rejected: clearing on every radio change loses input when the user returns to the original value before continuing, against the same rule for database types. Fitting the configuration as `physical_type_coercion.go` does would keep storage and encryption, but it copies a server policy into the frontend for a rare path.

## Risks / Trade-offs

- A retry after an ambiguous network failure of the backup start can start a second backup. The retry is an explicit user action, and no request repeats on its own.
- A configuration save the server applied but whose response was lost leaves the step editable. A retry with another storage then starts a storage switch with no warning or progress, against "Storage switches" in `docs/architecture.md`. It is an accepted exception, because the new database has no backups yet to move.
- A user who closes the wizard after a failure keeps a database with the server's default configuration. They can finish it in the settings or delete it, and the error has told them it exists.
- A user who closes the wizard while a configuration save is in flight can see the screen's earlier read until they reselect the database, because the screen mounted before the save answered. The window is one request long and the values on the server are right.
- A database whose create answer arrives after the wizard closed is not selected and reaches the list through the periodic refresh, at most 30 seconds later (`DATABASES_REFRESH_INTERVAL_MS`).
- The wizard shows no "Backups enabled" switch on either path, and its defaults have backups on, so a user cannot reach the disabled case through the screen today. The delta spec still requires it, and `pages/databases/model/startFirstBackup.ts` holds that rule. Its own test covers both paths through the HTTP boundary, because our modules are never mocked and a screen test has no way to turn backups off. The first attempt and a retry run the same line, so the closing of the wizard is covered by the enabled cases.
- Rollback: reverting the commit restores the old behavior. No stored data changes.
