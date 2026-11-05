## Context

See proposal.md for the motivation. The state that shapes the approach:

- **The backend builds its own command.** `buildRestoreArgs` in `internal/engines/postgresql/logical/restore.go:200-232` emits `-Fc`, then `-j N` only when the archive is a file, then the connection, `--verbose`, `--clean --if-exists` unless TimescaleDB, and `--no-owner` and `--no-acl` by the restore's settings. The table of contents list and the archive path follow. The job count is capped at 8, or set to 1 for TimescaleDB, in `restore.go:16-19` and `:43-49`. The password goes through a credential file (`restore.go:99`).
- **The agent builds another one.** `RunPgRestore` in `internal/agent/restore/restorer.go:40-86` always emits `-Fc --no-password --no-owner --no-acl`, the connection, `-j N` and `--clean --if-exists` unless TimescaleDB. It runs `pg_restore` from the container's `PATH` through `docker exec` and passes the password as `PGPASSWORD`. The runner caps the jobs at 8 and sets 1 for TimescaleDB again, in `internal/agent/runner/runner.go:35` and `:305-314`.
- **Two copies of one parser.** `isPreexistingPublicSchemaOnly` in `logical/preexisting_public_schema.go:15-33` and `IsToleratedErrorsOnly` in `agent/restore/toleratederrors.go:16-34` both find the last `errors ignored on restore: N`, collect the lines with `could not execute query: error:`, require exactly N of them and match each against a rule. The backend has one rule, the existing `public` schema. The agent also treats an extension it cannot create as an ignorable error (`toleratederrors.go:95-117`). `ExtractUnavailableExtensions` (`toleratederrors.go:38-62`) reuses the same line collector and the extension rule.
- **The agent judges a tail.** `Result.StderrTail` holds the last 8192 bytes (`restorer.go:74`), and the runner passes it to `IsToleratedErrorsOnly` (`runner.go:367`). The Docker exec already returns the whole error output in memory (`internal/agent/container/dockerengine.go:268-283`).
- **Two copies of the TimescaleDB SQL.** `logical/timescaledb.go:37-49` and `agent/restore/timescaledb.go:23-33` run the same `CREATE EXTENSION IF NOT EXISTS timescaledb`, `SELECT timescaledb_pre_restore()` and `SELECT timescaledb_post_restore()`. The backend leaves restoring mode in a `defer`, even after a failure (`logical/restore.go:70-96`). The agent leaves it only after a restore that succeeded or failed only with ignorable errors, before it collects statistics. After a failure it discards the container (`runner.go:396-410`). `Source.RunTimescaleDBPreRestore` and `RunTimescaleDBPostRestore` are exported but called only inside `logical`.
- **`logical` links the server frameworks.** `logical/read_only_user.go:11` imports `lib/crypto` for the password generator. `lib/crypto/legacy_master_key_repository.go` imports `lib/database`, which imports GORM, goose and `lib/apierror`, and `lib/apierror` imports Gin. `GOFLAGS= go list -deps ./cmd/verification-agent` finds none of them today.
- **No check would catch that.** `archlint` allows the agent to import `internal/engines` and `internal/lib` (`tools/archlint/layers.go:24`, `:110-112`). The agent's `depguard` rule denies only direct imports of Gin and GORM (`backend/.golangci.yml:114-122`). The production dependency check of `archlint` walks the whole graph of both entry points, but looks only for test libraries (`tools/archlint/production_dependencies.go:16`, `:35-50`; `archlint.go:18`).
- **A package-level duration in the runner.** `diskWatchInterval` is a mutable `var` (`runner/diskwatch.go:9`) that two runner tests overwrite and restore (`runner_test.go:146-147`, `:171-172`). The rewrite plan lists it as a relaxation that 12c and 12e remove; 12c touches the runner, 12e touches `api` and `upgrade`.

The change answers to the root `AGENTS.md` and `backend/AGENTS.md`. The rules that constrain it: engine code knows no tables and no HTTP (`backend/AGENTS.md:21`, `docs/architecture.md:48`); external tools run through `lib/process`, which the agent's `docker exec` does not need, because it starts no local process (`backend/AGENTS.md:57`); no package-level variable holds a dependency (`backend/AGENTS.md:29`); a file holds one unit and a unit test covers only pure logic (`docs/testing.md:30`).

## Goals / Non-Goals

**Goals:**

- The `pg_restore` command, the job count, the reading of its error output and the TimescaleDB SQL exist once, in an engine package that the agent can import without linking the server frameworks.
- The backend's restore behaves exactly as before. The agent behaves as before, apart from judging the whole error output, which the `backup-verification` spec states.
- The agent end-to-end suite from 12b and the backend's PostgreSQL restore tests pass without changes to their scenarios.

**Non-Goals:**

- One function that both runs the command and judges it. The backend and the agent run `pg_restore` in different places with different password handling, and the plan for 12c keeps running in each caller.
- Changes to the failure messages the backend writes. `classifyRestoreFailure` (`logical/client_tool_failure.go:156-217`) stays in `logical`.
- Moving the disk exhaustion check (`agent/restore/diskexhaustion.go`), the archive owner roles (`agent/restore/archiveownerroles.go`) or the backend's filtered table of contents (`logical/restore_toc_list.go`).

## Decisions

### 1. A leaf package `engines/postgresql/logical/pgrestore` that imports only the standard library

The shared code goes into a subpackage of `logical`. `logical` imports it, and the agent imports it without importing `logical`. The `physical/walmath` package is the precedent for a leaf package under an engine.

Alternatives rejected:

- **Put the functions into `logical` itself.** The agent would then link GORM, goose and Gin through `read_only_user.go` → `lib/crypto` → `lib/database`. No current check fails on that, which is why decision 6 adds one.
- **Move the legacy master key repository out of `lib/crypto` first.** That fixes the root cause for every user of the password generator, but it is a rework outside 12c, and the agent would still link `lib/sshtunnel`, `x/crypto/ssh` and the client tool bundles through `logical` for nothing.
- **Put them into `postgresql/shared`.** `shared` holds what logical and physical PostgreSQL share, such as the client tools, credentials and versions. `pg_restore` belongs only to the logical side.

### 2. One argument builder with explicit fields

`BuildArgs(ArgsSpec) []string` takes the host, port, user, database, the job count, `IsTimescaleDB`, `IsRestoreOwnership`, `IsRestorePrivileges`, `IsVerbose`, the table of contents path and the archive path. It returns the arguments without the program, because the backend runs a bundled binary and the agent runs `pg_restore` from the container's `PATH`. It emits `-j N` only when N is above 1, `--clean --if-exists` unless TimescaleDB, and `--no-owner` and `--no-acl` when ownership and privileges are not restored. It omits the archive path when the archive comes from standard input. The order is the backend's current order (`restore.go:202-231`); `pg_restore` does not depend on the order of options.

`GetParallelJobs(requestedJobs int, isTimescaleDB bool) int` returns 1 for TimescaleDB and otherwise clamps the request to 1–8. The comment explaining why TimescaleDB restores with one job moves there from `restore.go:16-18` and `runner.go:305-311`. The backend's `RestoreSpec.getParallelJobs` and the runner's `maxParallelRestoreJobs` call it instead of keeping their own copy. The runner keeps choosing the job count, so its tests that record the count keep their shape.

What changes on the command line:

- The agent emits `-j` only for more than one job. `-j 1` and no `-j` restore the same way.
- Two backend restores from a file lose `-j 1` for the same reason. One is a filtered restore with one requested job, which takes the file path because it excludes extensions or skips user mappings. The other is a TimescaleDB restore with more than one requested job: `NeedsArchiveFile` sends it to a file (`restore.go:39-41`), and `getParallelJobs` gives it one job (`restore.go:43-49`, `:204-206`). Restores with standard input never had `-j`, and their job count is always 1, because `NeedsArchiveFile` sends any requested count above 1 to a file.
- Nothing else. The backend passes `IsVerbose: true` and its ownership and privilege settings; the agent passes `IsVerbose: false` and false for both.

Alternatives rejected:

- **Two presets, `ForRestore` and `ForVerification`.** The engine would then know its callers, and the differences between the two would hide inside the presets instead of being visible at each call.
- **`--verbose` for the agent as well.** The agent logs the last 8 KiB of the output. With `--verbose` that tail would hold only progress lines, so an operator reading the log would lose the errors. This is the reason the field exists.

### 3. Ignorable errors are judged by rules the caller passes

An ignorable error is an error that `pg_restore` skipped and that still lets the restore count as successful. `HasOnlyIgnorableErrors(stderr string, rules ...IgnorableErrorRule) bool` holds the shared parser: the last marker, the count, the error lines, the requirement that every line matches one of the rules. The package defines two rules as constants of `IgnorableErrorRule`, `PreexistingPublicSchema` and `UnavailableExtension`, and `GetUnavailableExtensions(stderr) []string` for the agent's log line. The backend passes only the first rule. The agent passes both. The comment on why a missing extension is an ignorable error of a verification stays in the agent, because the reason is the agent's: its container does not carry the source's extensions. The comment on why that is unsafe for a real restore goes with the backend's call.

`IgnorableErrorRule` is an integer type whose matcher is an unexported method, so a caller can only choose among the rules the package defines, and as constants they cannot be reassigned, for the same reason decision 5 returns the SQL from functions.

Alternatives rejected:

- **`HasOnlyRestoreIgnorableErrors` and `HasOnlyVerificationIgnorableErrors`.** The engine would own the verification's business reason, and adding a rule for one caller would mean a third function.
- **Leaving `ExtractUnavailableExtensions` in the agent**, which was the plan's default when no second consumer exists. It uses the same line collector and the extension rule, so the agent would need private copies of both, and the agent would keep its own parser of `pg_restore` output. The plan's completion criterion forbids exactly that.

### 4. The agent judges the whole error output

`Restorer.RunPgRestore` computes the verdict from the full `ExecResult.Stderr` and returns it in `Result`: `HasOnlyIgnorableErrors bool` and `UnavailableExtensions []string`. `StderrTail` stays for the log. The runner keeps the order of its checks: an exec failure, then the disk ceiling with `IsDiskExhausted(StderrTail)`, then the ignorable errors. Its fake restorer sets the two new fields directly.

Alternatives rejected:

- **Keep judging the tail.** The tail exists only to keep the log line short. The count guard in `toleratederrors.go:14-15` exists because of the tail, so the tail turns long outputs made only of ignorable errors into rejected backups. The verdict needs the whole text, and the agent already has it in memory.
- **Return the whole error output in `Result` and judge in the runner.** The runner would carry an unbounded string only to hand it to a parser, and the restorer already sees it.
- **Return no error for a failure made only of ignorable errors.** The runner logs such a run with the job's logger and reports the non-zero exit code with the success. Moving that into the restorer would move the job's logging with it.

### 5. Only the TimescaleDB SQL is shared

`GetTimescaleDBPreRestoreStatements()` and `GetTimescaleDBPostRestoreStatements()` return the SQL, with the explanation of the database-level GUC that the separate `pg_restore` connection inherits. They are functions, not variables, so no caller can change the shared slices. The backend runs them through `Source.execStatements` and keeps its `defer`. Its two methods become unexported, because no caller outside the package uses them. The agent runs them through `agent/restore.execStatements`, which `EnsureArchiveOwnerRoles` also uses, and keeps running the post step only on the success path.

Alternative rejected: **a shared `WithRestoringMode(ctx, exec, restore)`.** The agent checks its disk ceiling and its context between the steps and reports each step's failure differently. A wrapper would also run the post step after a failed agent restore, which adds work on a container the agent discards anyway.

### 6. `archlint` forbids the server frameworks in the agent binary

The production dependency check gets a list of forbidden packages for each entry point. Both entry points keep the test libraries. `cmd/verification-agent` also forbids `gorm.io`, `github.com/gin-gonic/gin` and `github.com/pressly/goose`. A violation names the package that imports the forbidden one, the way a test library violation does. The check already loads the whole dependency graph without build tags, so the addition is a list and a lookup.

Alternatives rejected:

- **`depguard`.** It sees only the direct imports of a file, so it cannot see a dependency that arrives through `logical`.
- **A layer rule that forbids the agent to import `logical`.** It would encode the current dependencies of `logical` instead of the property that matters. If `lib/crypto` loses GORM later, the rule would block a harmless import, and a new heavy dependency elsewhere would pass.

### 7. `diskWatchInterval` becomes a field of the runner

`NewRunner` sets the field to 3 seconds, and `newDiskWatcher` takes the interval from the runner. The tests set the field on the runner they build, the way `backupDownloadBackoffFn` already works (`runner.go:50`, `:78`). This removes the runner's part of the relaxation in the rewrite plan; `api` and `upgrade` stay for 12e. The alternative was to leave it for 12e. The runner would then be changed twice, and the two tests would keep overwriting a package variable.

### 8. Physical cancellation tests wait for replication slot cleanup

Change verification exposed a race in `backupTeam.assertNoRunningBackup`: cancellation makes a catalog row terminal before `pg_basebackup` exits and its replication slot is dropped. `apptest.WaitForPhysicalBackups` waits for terminal catalog rows, so the slot assertion must independently wait for the source to have no backup slots. It uses `testwait.Until` and retains the zero-slot requirement. A regression suspends the real process before HTTP cancellation; it fails with the immediate assertion.

Alternative rejected: **wait for the application's shared backup group.** The group also owns continuous WAL streamers, so waiting for it blocks fixtures with live physical loops. The source's slot count is the observable cleanup condition these tests need.

## Risks / Trade-offs

- [The backend passes the agent's rule by mistake and starts treating missing extensions as ignorable errors in real restores.] → The engine's restore test with a stubbed `pg_restore` that reports only an unavailable extension expects a failure, which pins the `database-tool-failure-output` scenario.
- [A different argument order or a dropped `-j 1` changes a restore.] → The backend's restore tests compare the full argument lists for the stream and file cases. The table test of `BuildArgs` covers both callers' combinations. The engine e2e restores on every PostgreSQL version and the agent e2e from 12b exercise the real tool.
- [Judging ignorable errors by the whole output lets a verification pass that failed before.] → The rules do not change. Only verifications whose listed errors all match a rule, with an exact count, change their verdict. The restored-size check on the backend stays as the remaining safety net.
- [A long error output costs memory in the agent.] → Nothing new is held in memory: the Docker exec already buffers the whole output. Without `--verbose` the output is a few lines per failed object.
- [The forbidden list misses another heavy dependency.] → The list names what the agent must never have: tables, migrations and an HTTP server. Other dependencies are reviewed in the diff as before.

## Migration Plan

Nothing to migrate: no schema, configuration or protocol change. A new agent release starts judging by the whole error output after the agent updates itself. Rolling back the release restores the tail-based verdict.
