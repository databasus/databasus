## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/internal/` unless they start with another top-level directory. `UL` stands for `features/backups/backups/usecases/logical` and `RU` for `features/restores/usecases`.

**Backup use cases.** `UL/create_backup_uc.go:23-75` switches on `database.Type` and calls one of four use cases. Each `Execute` does three things:
- reads the decrypted connection settings from the databases module;
- opens the tunnel through its engine source and measures the raw size, plus the TimescaleDB version for PostgreSQL;
- calls its own `streamToStorage`: `UL/postgresql/create_backup_uc.go:121`, `UL/mysql/create_backup_uc.go:168`, `UL/mariadb/create_backup_uc.go:168` and `UL/mongodb/create_backup_uc.go:124`.

The four `streamToStorage` functions share one skeleton:
1. They derive the backup context with `NewBackupContext` (`features/backups/backups/core/logical/backup_interruption.go:19`), which ends on shutdown and after 23 hours.
2. They write the credential files: `.pgpass` and the certificates through `pgshared.WriteCredentialFilesToTempDir`, or a `.my.cnf` through `createTempMyCnfFile` (`UL/mysql/create_backup_uc.go:312`, `UL/mariadb/create_backup_uc.go:313`). MongoDB puts the URI into the arguments.
3. They start the tool with `exec.CommandContext` and `WaitDelay = DumpToolOutputWaitDelay` (10 s): PostgreSQL at `UL/postgresql/create_backup_uc.go:146`, MySQL at `:201`, MariaDB at `:202`, MongoDB at `:140`. PostgreSQL also calls `exec.LookPath` (`:427`).
4. They build the writer chain `[zstd ->] encryption -> io.Pipe -> storages.StartBackgroundWrite`. Only MySQL and MariaDB compress in Databasus (`UL/mysql/create_backup_uc.go:231-237`). `pg_dump -Fc --compress` and `mongodump --gzip` compress in the tool.
5. They copy stdout through `copyWithShutdownCheck`, which runs every `Write` in a goroutine of its own so that a blocked write cannot outlive the context (`UL/postgresql/create_backup_uc.go:264`).
6. They peek at the storage error, check the context, close the writers and pick the error to report.

PostgreSQL re-checks the context in its final switch (`UL/postgresql/create_backup_uc.go:241-255`), and the other three do not. The progress listener gets the bytes written into the encryption writer, which is the stored format before encryption in all four.

**Restore use cases.** `RU/restore_backup_uc.go:24-79` switches the same way. Each engine opens the backup through the storage reader and wraps it in two `io_utils.FailureTrackingReader`, one on the storage read and one on the decoded stream. It decodes the salt and IV from base64 and builds `encryption.NewDecryptionReader`. MySQL and MariaDB add a zstd decoder. Then the engine runs the tool:
- PostgreSQL streams into `pg_restore` through stdin (`RU/postgresql/restore_backup_uc.go:154-350`, with a bare goroutine at `:296`). With parallel jobs, an extension filter or the user-mapping filter, it downloads the archive into `config.GetEnv().TempFolder` through a fifth `copyWithShutdownCheck` (`:594`, defined at `:833`) and runs `pg_restore -l` to build a filtered TOC (`:888-968`). The disk space check before the restore (`features/restores/service.go:352-405`) covers parallel jobs and the extension filter, but not the user-mapping filter.
- TimescaleDB backups are wrapped in the engine's pre- and post-restore steps (`:124-152`).
- `IsPreexistingPublicSchemaOnly` (`RU/postgresql/preexistingpublicschema.go:15-64`) turns a failure whose only error is the existing `public` schema into success, and `resolvePgRestoreOutcome` (`:728-830`) classifies the rest.
- MySQL and MariaDB pipe into `mysql` or `mariadb` with a `.my.cnf`, and MongoDB into `mongorestore --archive --gzip`.

`restores_core.GetBackupStreamFailure` (`features/restores/core/stream_failure.go:20`) reports a broken storage read before a broken decode, and either before the tool's own error. Each engine sets its own 23-hour timeout.

**Encryption.** `features/backups/backups/encryption` writes a 64-byte header (`PGRSUS01`, a 32-byte salt, a 12-byte nonce, 12 reserved bytes) followed by 1 MiB chunks sealed with AES-256-GCM. The key is PBKDF2-SHA256 over the master key concatenated with the backup ID, with 100 000 iterations (`encryption/utils.go:12-36`). `NewDecryptionReader` refuses a header whose salt or nonce differs from the ones stored with the backup (`encryption/decrypting_reader.go:91`). The package is imported by:
- the four backup use cases and the four restore use cases;
- the physical code in `usecases/physical/postgresql/{encryption,stream,timeline,wal_upload,create_incremental_uc}.go`, which passes the master key as a string (`encryption.go:21`);
- `download/restore_stream/artifact.go`, `dto/logical/dto.go` and `services/logical_service.go`.

**Storage write.** `storages.StartBackgroundWrite` closes the pipe's read end with the upload's error when the upload fails, and calls `OnFailure` with it (`modules/storages/internal/files/background_write.go`). A producer blocked on the pipe therefore unblocks when the storage fails.

**`lib/process`** (`lib/process/process.go`) starts a tool in its own process group with `Pdeathsig`, kills the group when the context ends and reports the context's error from `Wait`. Only the client tool checker uses it today.

**Tests.**
- `UL/*/create_backup_uc_test.go` test the private argument builders and drive `Execute` through `core/logical/testing.go` (`AssertStorageTimeoutReportedAsStorageFailure`, `AssertDumpToolErrorReported`).
- `RU/*/restore_backup_uc_test.go` drive each engine with shell tool stubs from `restores/core/testing.go`.
- The compression probe tests need a live server.
- `features/tests/logical/{postgresql,mysql,mariadb,mongodb}` back up and restore real servers through the old HTTP API, including SSL, SSH tunnels, mTLS, TimescaleDB and the dump scope.

**Rules in force.**
- `docs/architecture.md`, "Engines and providers": an engine dumps into an `io.Writer` and restores from an `io.Reader`.
- "Long-running work": external tools run through `lib/process`, and goroutines start only through `lib` helpers.
- "Backup data flow": streaming with backpressure, and per-file encryption with PBKDF2-SHA256 and AES-GCM. Its exceptions name `pg_restore -j` as the only restore that writes the archive to disk, which the filtered restore contradicts today.
- "Shared code" places stream encryption in `lib/crypto`.
- The engines import no GORM or Gin (`depguard`) and start processes only through `lib/process` (`forbidigo`). Both checks already cover `internal/engines` and `internal/lib`.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Every logical dump and restore tool starts from `internal/engines` through `lib/process`.
- One backup pipeline and one restore pipeline hold the streaming, encryption, progress and error ordering, which are the same for every engine.
- The behavior and the messages of backups and restores stay as they are, except the PostgreSQL missing-password message of decision 1, and the unchanged end-to-end tests show it.

**Non-Goals:**

- Moving the pipelines into the `backups` module, or replacing the type switches with a registry.
- Changing the restore request, the disk space check, the cancellation registry, the shutdown signal or the time limits.
- Sharing the restore code with the verification agent.

## Decisions

### 1. An engine dumps and restores through its source

Each logical engine source gains two methods:

```go
// pglogical; mysql, mariadb and mongodb follow the same shape with their own specs.
type DumpSpec struct {
	ClientTools      pgshared.ClientTools
	Version          pgshared.Version
	IncludeSchemas   []string
	ExcludeTables    []string
	ExcludeTableData []string
	Logger           *slog.Logger
}

func (s *Source) Dump(ctx context.Context, spec DumpSpec, output io.Writer) error

type RestoreSpec struct {
	ClientTools         pgshared.ClientTools
	Version             pgshared.Version
	SourceDatabaseName  string
	ParallelJobs        int
	IsTimescaleDB       bool
	IsRestoreOwnership  bool
	IsRestorePrivileges bool
	IsExcludeExtensions bool
	IsSkipUserMappings  bool
	TempDir             string
	Logger              *slog.Logger
}

func (s *Source) Restore(ctx context.Context, spec RestoreSpec, input io.Reader) error
```

The source of a dump is the backed-up database seen through its tunnel. The source of a restore is the target database seen through its tunnel. Opening the tunnel stays with the caller, as it does today, because the caller also measures the size and detects the version through the same tunnel.

The engine owns everything that depends on the engine:
- the arguments, built from the spec;
- the credential files, removed when the call returns;
- the environment of the tool;
- the stored format the tool produces or expects;
- the classification of the tool's exit status and error output into today's messages, kept word for word, because they are the fail messages users see and the tests match. The one exception is the PostgreSQL dump message about a missing password: it pointed users at `PGPASSWORD` and a `.pgpass` file, while the password reaches the tool through the password file the engine writes, so it now names that file;
- for PostgreSQL, the choice between stdin and a temporary file in `spec.TempDir`, the filtered TOC, the TimescaleDB steps and the tolerated `public` schema error.

A dump writes the tool's stdout into `output` until the tool exits. A restore feeds `input` into the tool's stdin, or for the file path into the temporary file first.

*Rejected: give the engine the storage, the encryption and the backup row.* The engine would then know the storages module and the backup model, which `docs/architecture.md` forbids, and the verification agent could not call it.

*Rejected: one `Dumper` interface in `engines` that every engine implements.* `docs/architecture.md` says a consumer that serves several engines declares the interface it needs. The pipelines are the consumers, so they declare it (decision 3).

### 2. Compression is declared by the engine and applied by the pipeline

PostgreSQL and MongoDB compress inside the tool. MySQL and MariaDB send uncompressed SQL, which Databasus compresses with zstd level 5. The engine states which applies: `mysql` and `mariadb` export a zstd level for the stored format, and the other two need none. The pipeline wraps the dump output in a zstd writer and the restore input in a zstd decoder when the engine asks for it.

This keeps the two failure trackers of a restore in one place. The pipeline tracks the storage read and the decoded stream, as `GetBackupStreamFailure` does today, so a corrupt zstd frame is still reported as a broken backup stream, not as the syntax error `mysql` prints about the half statement it received.

*Rejected: compress inside the engine.* The engine would need a third tracker for its decoder, and a way to report it ahead of the tool's own error. The message about the broken stream would then live in two packages.

### 3. One backup pipeline and one restore pipeline

`UL` keeps one package. It holds the pipeline and four small adapters, one per database type, which implement the interface the pipeline declares:

```go
type logicalDumpSource interface {
	GetRawSizeMb(ctx context.Context) (float64, bool)
	GetTimescaleDBVersion(ctx context.Context) string // empty for every engine but PostgreSQL
	GetStoredCompression() storedCompression
	Dump(ctx context.Context, output io.Writer) error
	Close()
}
```

An adapter reads the connection settings from the databases module, opens the tunnel, builds the engine's `DumpSpec` and binds it. The dispatcher at `UL/create_backup_uc.go:23-75` chooses the adapter by database type, as the old code does today. `RU` does the same for restores with a `logicalRestoreTarget` interface.

The backup pipeline runs these steps:
1. It derives the backup context with `NewBackupContext`.
2. It builds the chain `[zstd ->] counting -> encryption -> pipe`, starts the background write and calls `Dump`, which starts the tool.
3. It reports progress from the counting writer every megabyte and once at the end, as the four copies do.
4. It picks the error in the order of `UL/postgresql/create_backup_uc.go:206-256`. An early storage error comes first. Then an ended backup context, described by `DescribeBackupInterruption`. Then a failed close of the writers, then the tool's error, then the copy error, then the storage error. The tool, copy and storage errors are each checked against the context first.

Today the tool starts before the background write (`UL/postgresql/create_backup_uc.go:180-182`). A storage that fails at once cancels the backup context, and with the old order the tool's start error, `context canceled`, was returned before the storage error was looked at. The pipeline cannot start the tool first, because `Dump` starts and waits for it in one call. The order of step 4 makes the start order irrelevant: when the storage fails before the tool starts, `Dump` returns the start error, and step 4 picks the early storage error, or the ended context, whose cause the storage write set. `DescribeBackupInterruption` turns that cause into `save to storage: <storage error>`, the message the old order gave. The pipeline test of an immediate storage failure in decision 9 pins that message.

MySQL, MariaDB and MongoDB lack the context check in the last step today. That check only matters when the context ends between the earlier check and the final switch, so taking PostgreSQL's order changes no observable outcome.

The restore pipeline runs these steps:
1. It opens the file through the storage reader under a 23-hour timeout and wraps it in the storage-read tracker.
2. It decrypts, then decodes zstd when the engine asks for it, and wraps that in the decoded-stream tracker.
3. It calls `Restore`.
4. It reports a stream failure from `GetBackupStreamFailure` before the engine's error. A cancelled context gives "restore cancelled", as today, and the restorer still classifies the outcome by the context's state.

These helpers are deleted:
- `cleanupOnCancellation`, `closeWriters`, `setupBackupEncryption` and the `writeResult` type (four copies each);
- `copyWithShutdownCheck` (five copies: four backups and the PostgreSQL restore's download into a temporary file);
- `containsIgnoreCase` (seven copies);
- `createTempMyCnfFile` (four copies), and the inline salt and IV decoding (five copies).

*Rejected: keep a backup and a restore use case per engine and only extract helpers.* The skeleton would still exist four times, and each copy would keep its own small differences in error order, which is how the missing context check came about.

### 4. Backpressure and cancellation without a goroutine per write

The pipeline passes its writer chain to the engine, and the engine hands it to `lib/process` as `Stdout`, so `exec` copies the tool's output into the pipe. A slow storage blocks that copy, and the tool blocks on its own stdout. That is the backpressure `backup-streaming` asks for.

Two events must unblock the copy:
- **A failed upload.** `StartBackgroundWrite` closes the pipe's read end with the upload's error and cancels the backup context through `OnFailure`. The pending write returns that error, and `lib/process` kills the tool's group.
- **An ended context** (a cancellation, a shutdown, the time limit). The pipeline registers `context.AfterFunc(backupCtx, ...)`, which closes the pipe's write end with `context.Cause(backupCtx)`. A write blocked in `io.Pipe` returns as soon as either end closes, and the upload sees a failed read and leaves the pending deletion in place.

`copyWithShutdownCheck` exists today only to guarantee these two exits. Both now hold without it.

The PostgreSQL restore through a file has a third copy: `pglogical.Restore` writes `input` into the temporary file before it starts `pg_restore`. Its old `copyWithShutdownCheck` checks the context before each read. The engine copies through a private reader that returns `context.Cause(ctx)` before each read, which keeps that check without a goroutine. A read blocked inside the storage ends too, because the pipeline opened the storage reader with the same context.

*Rejected: keep a context-aware copy loop in `lib`.* It would spawn a goroutine per write to escape a write that cannot block for longer than the context lives anyway.

`internal/util/io` stays where it is. The engines take a plain `io.Writer` and `io.Reader`, and the pipelines that use its counting writer and failure-tracking reader are old code, which may import `util`.

### 5. How the tools are started

Every dump, restore, `pg_restore -l` and compression probe runs through `lib/process`:
- `StopSignal` is nil, so the group gets `SIGKILL` when the context ends. That is what `exec.CommandContext` does today, now applied to the whole group.
- `KillAfter` is 10 seconds, the value of `DumpToolOutputWaitDelay` and `ToolOutputWaitDelay`. It bounds how long `Wait` reads output that a stray child keeps open.

Both constants move into the engine code as one unexported constant per engine package. `exec.LookPath` before `pg_dump` and `pg_restore` goes away. The startup check of the client tools already refuses to start without the PostgreSQL bundle, and a missing executable now fails with `lib/process`'s start error, which names the path.

*Rejected: `SIGTERM` with a grace period.* A dump that is being cancelled has nothing to flush, and `pg_restore` aborts its transaction either way. A grace period would only delay the cancellation.

### 6. MySQL and MariaDB share their runner in `mysqlfamily`

After renaming, MySQL and MariaDB differ in five flags and in the SSL line of `.my.cnf`. `mysqlfamily` holds what is the same:
- the `.my.cnf` writer, in the OS temp directory with modes 0700 and 0600 (the reason in `UL/mysql/create_backup_uc.go:313-314` stays);
- running the dump and the restore tool;
- the classification of connection errors;
- the probe loop of network compression.

`mysql` and `mariadb` pass their executables, their flags, their SSL settings and their compression candidates. `mysql` keeps `--set-gtid-purged=OFF` and the version-dependent candidates. `mariadb` keeps `IsExcludeEvents`, `--init-command=SET SESSION wsrep_on=OFF` and the Galera switch.

### 7. Stream encryption in `lib/crypto`

```go
type StreamHeader struct {
	Salt  []byte
	Nonce []byte
}

func NewEncryptingWriter(output io.Writer, masterKey *MasterKey, fileID uuid.UUID) (*EncryptingWriter, StreamHeader, error)
func NewDecryptingReader(input io.Reader, masterKey *MasterKey, fileID uuid.UUID, header StreamHeader) (*DecryptingReader, error)
```

The format, the PBKDF2 parameters and the chunking stay byte for byte, so every stored file of an earlier release decrypts. A test decrypts a fixture file that the current code encrypted, with a fixed master key and ID kept beside the test. If the format ever drifts, that test fails, not a user's restore.

The salt and nonce are bytes in the API. The base64 form stored on the backup row is the backup code's business, encoded and decoded once in each pipeline. The physical callers pass their `*crypto.MasterKey` instead of its secret string.

*Rejected: keep the package where it is until the physical code moves.* The physical code would then import `features/backups/backups/encryption` from code that milestone 10 moves into the module, and the engines would need it too for the verification agent in milestone 12.

### 8. A filtered PostgreSQL restore is a file-based restore

The restore that excludes extensions or skips user mappings needs the archive twice: `pg_restore -l` lists it, and `pg_restore -L` restores the filtered list. It therefore downloads the archive into a file, as the parallel restore does, and the move keeps that. The documents say otherwise. `docs/architecture.md` and the `backup-streaming` spec name only the parallel restore, and the old disk space check misses the user-mapping filter, so such a restore can fill the host's disk.

The change brings the three in line with the code. The old check also needs the archive's room when the source database skips user mappings, which `RestoreBackupWithAuth` already reads as `source`. "Backup data flow" and the `backup-streaming` delta name every restore that uses a file. `logical-backups-module` keeps the same three conditions in its own check.

*Rejected: stream the filtered restore through stdin.* The listing and the restore each read the whole archive, so streaming would download it from the storage twice and double the restore's transfer.

### 9. Tests

- **Engine tests** in each engine package:
  - `Dump` and `Restore` against a stub tool that records its arguments and stdin, which replaces the tests of the private `build*Args` and `createTempMyCnfFile`;
  - the error classification against stubs that write a given stderr and exit, moved from `RU/*/restore_backup_uc_test.go`;
  - the compression probe tests, moved with the probe, which keep their live server.

  The engines get no round trips of their own against real servers. The unchanged end-to-end tests of `features/tests/logical` already back up and restore every engine, and a second copy would only add servers to the run.
- **Pipeline tests** in `UL` and `RU`. The cases of `core/logical/testing.go` and `RU/*_test.go` that pin how a storage timeout, an immediate storage failure, a tool failure before any data, a truncated stream and a cancellation are reported now run once per engine adapter against the pipeline. The immediate storage failure asserts the message `save to storage`.
- **Encryption tests** of the pipelines, one per scenario of the `backup-encryption` spec, with stub tools and a local storage:
  - an encrypted backup of a stub dump holding a marker stores a file without the marker, and its restore feeds the stub restore tool the whole dump;
  - two encrypted backups of one stub dump store different files with different header salts;
  - a restore of a stored file with one changed byte, and of a file cut inside a chunk, records the failure about the backup stream from storage;
  - a restore of a file swapped for another backup's file fails before the stub restore tool reads any byte.
- **`lib/crypto`** gets the tests of the old package and the fixture test of decision 7, which is the test of the spec's previous-format scenario.
- **The disk space check** gets a test in `features/restores` that refuses a single-job restore of a database that skips user mappings when the disk lacks room.
- **`features/tests/logical/**`** does not change. It is the check that the move kept every engine's behavior, SSL, SSH tunnels, mTLS, TimescaleDB and the dump scope included.

The tests in the target layout use no `time.Sleep`. The shell stubs that wait before they fail keep `sleep` inside the script, which the Go linter does not see and which is the behavior they model.

## Risks / Trade-offs

- [A message changes by a word and a test or a user's alert rule stops matching] → The classifiers move as whole functions with the one message decision 1 corrects, and the end-to-end and moved tests match today's substrings.
- [Killing the group also kills a child that used to survive] → That is the purpose. No tool child holds work that must finish after the parent is cancelled.
- [The pipeline order differs from three of the four engines in one branch] → Decision 3 explains why the branch is unreachable apart from a race, in which the context's cause is the more accurate answer.
- [About 20 files change imports for `lib/crypto`] → The changes are mechanical, and the compiler finds every one of them.
- [A restore that skips user mappings and ran on a nearly full disk is now refused] → It used to fill the disk and fail midway, so the refusal comes earlier with a message that says how much space is needed.
- [The verification agent keeps its own restore copy until milestone 12] → Nothing changes for it now. Its tolerated errors are a superset of the backend's, which milestone 12 must reconcile.

## Migration Plan

Nothing to migrate: no schema, contract, configuration or file format changes. The only behavior change is the disk space check of decision 8. A release with this change reads every backup that earlier releases wrote.
