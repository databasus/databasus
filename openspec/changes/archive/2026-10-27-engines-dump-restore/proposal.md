## Why

A logical backup and a restore each run an engine's client tool, and that code lives in eight use cases of the old `features/backups` and `features/restores`, one per engine and direction. Each backup use case repeats the same streaming pipeline in its own `streamToStorage`: credential files, `exec.CommandContext`, a pipe into the storage write, encryption, progress and the cancellation cleanup. `setupBackupEncryption`, `cleanupOnCancellation`, `closeWriters` and the `writeResult` type exist four times each. `copyWithShutdownCheck` exists five times, once more in the PostgreSQL restore, and `containsIgnoreCase` and `createTempMyCnfFile` appear in both directions. The tools start without a process group, so a cancelled `pg_restore -j` or `mongodump` can leave its children running. The stream encryption of backup files sits in `features/backups/backups/encryption`, which `docs/architecture.md` places in `lib/crypto`.

Milestone 9 of the backend rewrite plan moves logical backups and restores into the `backups` module. It is split in three, and this change is the first part, like `engines-layer` was for milestone 8. It moves dump and restore into `internal/engines` under the old code, without changing the schema, the API or the frontend. The end-to-end tests that back up and restore real PostgreSQL, MySQL, MariaDB and MongoDB servers through the old API stay unchanged and prove that the move kept the behavior. The change lands first because `logical-backups-module` must rewrite those tests against the new API. If the dump code were rewritten in the same change, no unchanged test would be left to compare against.

## What Changes

- Each engine gains dump and restore:
  - `pglogical`, `mysql`, `mariadb` and `mongodb` dump a database into an `io.Writer` and restore one from an `io.Reader`, through their engine source and the shared `mysqlfamily` code.
  - The engine owns the tool's arguments, its credential files, the stored format (compression by the tool for PostgreSQL and MongoDB, zstd in Databasus for MySQL and MariaDB) and the meaning of the tool's exit status and error output.
  - Every launch goes through `lib/process`, so ending the operation's context stops the tool and every child it started.
  - The MySQL and MariaDB network compression probe moves into the engines with the dump.
  - The parser of the errors `pg_restore` may ignore and the TimescaleDB pre- and post-restore steps around a restore move into `pglogical`.
- One backup pipeline and one restore pipeline replace the per-engine use cases:
  - The backup pipeline encrypts what the engine writes, counts progress and feeds the storage's background write.
  - The restore pipeline reads the file from the storage reader, decrypts it and hands it to the engine.
  - Each pipeline reaches the engines through an interface it declares itself.
  - The old dispatchers keep choosing the engine by database type until `logical-backups-module` replaces them with a registry.
  - Every duplicated helper listed above is deleted. A storage failure or a cancellation now unblocks the pipeline by closing its pipe, so the copy needs no goroutine per write.
- The stream encryption of backup files moves to `lib/crypto` with the same file format, key derivation and chunking, so every file written by an earlier release still decrypts. Physical backups, the physical restore stream and the logical download switch to it.
- A PostgreSQL restore that skips user mappings downloads the archive into a file, like a parallel restore or one that excludes extensions, but today it skips the disk space check. The old check now covers it too, and `docs/architecture.md` and the `backup-streaming` spec name every restore that needs a file.
- The engine tests move with the code. The tests of private argument builders become tests of `Dump` and `Restore` against stub tools, and the unchanged end-to-end tests keep covering the real servers. The pipeline tests that pin how a storage failure, a cancellation and a shutdown are reported stay beside the pipelines, and new pipeline tests pin what the `backup-encryption` spec promises.

## Capabilities

### New Capabilities

- `backup-encryption`: how a backup file is encrypted with a key of its own, and that files written by earlier releases keep decrypting.

### Modified Capabilities

- `backup-streaming`: a PostgreSQL restore that filters the archive's table of contents needs the archive as a file, like a parallel one, and gets the same disk space check.

A dump and a restore keep what they produce and how they report a failure, as the specs `logical-backup-outcome` and `database-tool-failure-output` state.

## Out of scope

- The `backups` module, its contract, its frontend and its migration. They are `backup-configs-module` and `logical-backups-module`.
- The registry of engines keyed by database type. It arrives with its consumer in `logical-backups-module`.
- Moving `pg_basebackup`, `pg_receivewal` and `pg_combinebackup` onto `lib/process`. Milestone 10.
- The verification agent and its own copy of the restore code and the tolerated `pg_restore` errors. Milestone 12 merges it with the engine code.
- The restore request types in `features/restores/core`, the disk space check apart from its user-mapping case, the restore scheduler and the credential cache. `logical-backups-module` replaces them.
- Passing the MongoDB password some other way than inside the `--uri` argument.
- Any change to which tool version a server version uses, to the tool arguments, or to what a backup or restore produces.
- Limiting how much of a tool's error output is kept. The whole output stays buffered, because the check for errors `pg_restore` may ignore reads lines from the start of a verbose output.

## Impact

- **Backend, new and changed:**
  - `backend/internal/engines/postgresql/logical`, `mysql`, `mariadb`, `mongodb` and `mysqlfamily` (dump and restore);
  - `backend/internal/lib/crypto` (stream encryption).
- **Backend, reduced:**
  - `backend/internal/features/backups/backups/usecases/logical`, down to one pipeline and its engine dispatcher;
  - `backend/internal/features/restores/usecases`, the same;
  - `backend/internal/features/restores/service.go`, whose disk space check covers the user-mapping filter.
- **Deleted:** `backend/internal/features/backups/backups/encryption`.
- **Import changes:** the physical backup code, `download/restore_stream`, `services/logical_service.go`, `dto/logical` and the test helpers.
- **Not affected:** the schema, the contract, the frontend and every user-visible behavior apart from the disk space check of a restore that skips user mappings.
- **Documents:** `temp/refactor/plan.md`, which lives only in the main checkout (milestone 9 split into 9a, 9b and 9c, and the relaxation rows this change touches). `docs/architecture.md`, "Backup data flow": the exception for restores that need a file names the filtered restore beside the parallel one. It already places dump and restore in the engines and stream encryption in `lib/crypto`.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
