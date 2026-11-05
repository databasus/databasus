## Context

See `proposal.md` for the motivation. The facts below shape the approach.

**A start records a request.** `POST /physical-backups` writes a timestamp into `force_full_requested_at` or `force_incremental_requested_at` of the database's physical config (`backend/internal/modules/backups/internal/physical/backup_request_service.go:48`, `:76`, `:90-102`). The scheduler, a background job that runs every second, reads the timestamp, creates the `backups_physical` row and starts the process (`physical/backuping/scheduler.go`, `scheduleBackup` and `claimAndInsert`). While an earlier backup of the database still runs, the timestamp waits.

**The queue has other users.** WAL streaming requests a full backup itself (`physical/walstreaming/manager.go:428`). A storage switch sets the same timestamp (`config/service.go:491`). The public field `isFullBackupPending` reads it (`physical/wal_stream_statistics_service.go:78`), and the WAL monitoring screen hides its full backup button by that field (`frontend/src/pages/databases/lib/wal-stream-monitoring/fullBackupAction.ts:32`). The `physical-backups/operations` spec requires a start right after a cancel to wait for the old `pg_basebackup` and then complete (`openspec/specs/physical-backups/operations/spec.md:176-179`).

**A start can be accepted with nothing to follow.** The request path never looks at `IsBackupsEnabled` or `StorageID`, while the scheduler only reads enabled configs (`config/service.go:170`). A timestamp written for a turned-off database stays until backups are turned on. For a config never saved, the `UPDATE` changes no row and its result is ignored (`config/service.go:314`, `config/config_repository.go:354`).

**A cancel already knows its outcome.** `Canceller.cancel` locks the row and marks it `CANCELED` in a transaction, and signals the process after the commit (`physical/backuping/canceller.go:44-58`). `BackupLifecycle.Cancel` fills the reason, message and completion time on the struct, but `markTerminal` leaves the struct's status as it was (`physical/catalog/backup_lifecycle_service.go:77`, `physical/catalog/backup_repository.go:382`).

**The list form of a backup comes from SQL.** The list query builds each row, with `COALESCE` on the LSNs and the size and `NULLIF(parent_backup_id, root_full_backup_id)` for the parent (`physical/catalog/catalog_repository.go`, `backupListItemsQuery`), and `rowToListItem` maps it.

**The screen offers a start with backups off.** "Back up now" depends only on the permission and on the list's first load (`frontend/src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx:188-199`), and a notice says scheduled backups are off (`frontend/src/pages/databases/ui/backup-config/BackupConfigNotice.tsx:27-29`). After this change a click would show the generic `invalid_request` text, which tells the user to reload the page.

The approach follows `docs/architecture.md` ("HTTP API": the contract comes first, a release may break the contract when the same change updates the frontend), `backend/AGENTS.md` and `frontend/AGENTS.md`, and it changes the "HTTP API" section of `docs/architecture.md`.

## Goals / Non-Goals

**Goals:**

- A cancel returns the same values the list shows right after it.
- A `202` from a physical start means a request is stored that the scheduler can serve.
- The deviations from the API rules are written down with their reasons.

**Non-Goals:**

- Changing how the scheduler chooses or starts a backup.
- Clearing a request that was accepted before backups were turned off.

## Decisions

### 1. A physical start stays a queued request, and the exception is recorded

The start keeps answering `202` with no body, and `docs/architecture.md` records why. A script finds the started backup with `status=IN_PROGRESS`, because a database has at most one running backup.

Rejected alternatives:

- *Create the row inside the request.* The queue must stay for WAL streaming, the storage switch and `isFullBackupPending`, so this adds a second way to start a backup beside it. A start during a running backup, or right after a cancel, would become `409 already_running`, which changes the spec scenario at `operations/spec.md:176-179`. It also needs a new `GET /physical-backups/{backupId}`, because the rule "the client follows progress with `GET`" has nothing to call today.
- *Answer `202` with a request object such as `{databaseId, kind, requestedAt}`.* The client learns nothing it did not send, and it still has no backup to follow.

### 2. A cancel answers `200` with `PhysicalBackupListRow`

The status is final when the response is sent, so the operation is finished for the client. `cancelVerification` already answers `200` with its resource. The row schema is reused so that a client reads one form of a backup.

Rejected alternatives:

- *Keep `202` and add the body.* `202` promises work still to come on the resource, and here only the process winds down. `cancelLogicalBackup` keeps `202` because its body still says `IN_PROGRESS` (`logical/service.go:144-167`).
- *A new `PhysicalBackup` schema without the WAL fields and the `WAL` type.* It would nearly copy the row and need its own mapping, and a client gains only a narrower enum.

### 3. The cancel reads its response through the list query

`catalog.BackupListFilter` gets an optional `ID` (one condition in `buildClause`, `physical/catalog/backup_list.go:59`). `Service.CancelBackup` calls the existing `s.chains.ListBackups` with that filter inside its outer transaction, after `CancelRunningBackup` returns true (`physical/backup_cancellation_service.go:32`), and maps the row through `rowToListItem`. `Transactor.Transaction` nests on the same transaction (`lib/database/transaction.go:37`), so the read sees the update while the row lock is held. `CancelRunningBackup` keeps its `(bool, error)` result. The response comes from the SQL the list uses, so its LSNs, size, parent and microsecond times cannot differ from the list. A test still compares the two.

This query reads all WAL segments of the database. That cost is acceptable here, because a cancel is a rare manual action and the open screen already runs the query every second.

Rejected alternatives:

- *A Go conversion from `catalog.Backup` to the list item, with the row read again in the canceller.* It copies the SQL rules for LSN, size, parent and gap into a second place that can drift, and it changes the canceller's signature.
- *Return the struct `Cancel` filled in.* Its status is stale, and its completion time has nanoseconds that PostgreSQL rounds to microseconds, so the response would differ from the list read right after it.
- *Read after the commit.* A concurrent delete could remove the row, and an accepted cancel would then fail.

### 4. One start path that checks the config before recording a request

`StartBackup` authorizes once, reads the physical config once, refuses a config with backups off or no storage, and then branches by kind. `RequestBackup`, `RequestFullBackup` and `RequestIncrementalBackup` are called only from `StartBackup` (`backup_request_service.go:121-125`), so they become private branches without their repeated authorization. The check runs before the chain check, so a turned-off database gets `400` and not `409 no_extendable_chain`. Authorization runs first, so an outsider gets `404` whatever the request says.

`errors.go` gets two field problems on `databaseId`, with the wording of `logical.ErrNoStorage` for the storage case. A config never saved reads as defaults with backups off (`config/defaults.go`), so it gets the same `400`.

The answer is `400 invalid_request`, although the `api-design` table gives `409` to a state that forbids an action. `startLogicalBackup` already answers `400` for a config without a storage (`logical/errors.go:20`), and the two starts should refuse the same target the same way. The `api-design` skill records this as a rule, so the table and the operation agree.

The internal requests of WAL streaming and the storage switch go straight to `config.Service` and are not affected.

Rejected alternatives:

- *Take `LockBackupConfig` during the check.* It also takes the `physical-backup` advisory lock (`config/storage_switch_service.go:11`) that `backend/internal/modules/backups/physicalapitests/walstreaming/last_backup_time_test.go:32-46` holds across a start, and the lock adds nothing. A save that turns backups off right after the read gives the same result as a start accepted just before that save, which is out of scope.
- *Serve a manual start while backups are off, as logical backups do.* Turning physical backups off cancels the running backup before the save is answered (`operations/spec.md:174`) and stops the WAL stream, so a backup taken then would have no WAL to continue with.
- *Check in `config.Service.RequestFullBackup`.* WAL streaming and the storage switch call it too, and they do not need this rule.
- *Answer `409` with a code of its own.* It follows the table, but a physical and a logical start would then refuse the same missing storage with different answers, and changing the logical answer is out of scope.

### 5. The screen disables the start and says what it needs

`PhysicalBackupsComponent` disables "Back up now" when `backupConfig` is loaded and has backups off or no `storageId`, and it shows one translated sentence saying that a start needs backups turned on and a storage chosen in the backup configuration. No start control exists while the config loads, because the component shows only a spinner (`PhysicalBackupsComponent.tsx:140`). When the config failed to load, the button keeps its current behavior, and the server decides.

The existing notice stays as it is. `backups.list.scheduledBackupsDisabled` speaks only of scheduled backups, logical backups share it while still allowing a manual start, and it does not appear for a config with backups on and no storage (`BackupConfigNotice.tsx:28`, reachable because `config/service.go:514` accepts a nil storage).

Rejected alternatives:

- *Disable the button with no explanation.* A member with backups on and no storage would see a dead primary button and no reason.
- *A new `ProblemCode` such as `backups_turned_off`, translated for the refused request.* The screen no longer sends such a request, and scripts get a clear `detail` from the field problem.
- *Leave the button active.* The user would see "reload the page and try again", which does not help.

### 6. `totalUsageMb` stays on the page

The field stays required, and `docs/architecture.md` records the decision. The screen shows the figure above the list, and a separate operation would add a request to a poll that runs every second (`PhysicalBackupsComponent.tsx:29`). The bigger cost of a page is likely the list query itself, which reads all WAL segments of the database (`catalog_repository.go`, `committed_wal`). Neither query has been measured.

Rejected alternatives:

- *A separate operation.* It costs one operation and one request per refresh, and it would save at most the `SUM` on each page.
- *Send the field only on the first page.* The poll refreshes the first page, so the saving is small, and the field becomes optional for every client.

### 7. The enlarged-text check measures the final layout

`browser/scenarios/databaseTransfer.spec.ts:128` failed in one of three browser runs. AntD gives a select's value `transition: all` over 0.3 s (`antd/es/select/style/single.js:40`), and `assertAccountReflow` measured about 0.1 s after `enlargeText`, so it usually saw text that had not reached its final size. Waiting for the transitions made five scenarios fail every time, which showed two separate causes in the final layout:

- The storage type row kept a fixed 250 px select, so at 320 px with enlarged text the logo left the dialog's content box. The row now takes the available width, the select shrinks up to a 250 px maximum, and the logo keeps its size.
- A long select value ends with an ellipsis, as AntD intends, and the check counted that as overflow. The check now exempts only `.ant-select-selection-item`, because the value's `title` and the select's options keep the full text.

Rejected alternatives:

- *Retry the test or add a delay.* A retry hides the race, and it would keep the check passing on a layout it never measured.
- *Make the select value wrap.* A small select has a fixed height, so wrapped text would be cut off vertically.
- *Keep the check strict and shorten the translated labels.* User-chosen workspace names reach the same select value, so short labels would not settle it.

## Risks / Trade-offs

- [A script cannot tell a manually started backup from a scheduled one that began at the same moment] → Accepted. The exception in `docs/architecture.md` states this limit.
- [A database with backups off that relied on a remembered start no longer gets it] → The `400` says so at once, and the proposal marks this as BREAKING.
- [A test starts a backup on a database with backups off] → The tests that do so now fail with `400`. The tasks repair them by turning backups on in their setup, if the test is not about the refusal.

## Migration Plan

No migration. Timestamps already stored for turned-off databases stay and run if backups are turned on, as they do today. Rollback is a revert of the change together with the regenerated code.
