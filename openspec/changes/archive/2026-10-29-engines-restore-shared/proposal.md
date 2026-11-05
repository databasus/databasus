## Why

The backend's restore and the verification agent each build their own `pg_restore` command, read its output with their own parser and keep their own copy of the TimescaleDB restore SQL. The two copies have already drifted apart in flags and in which errors they treat as ignorable, so a fix in one is easy to miss in the other. This is the third change of milestone 12. It lands under the agent end-to-end suite from `agent-e2e-on-apptest`, which stays unchanged, before `verification-module` and `agent-contract` rework both sides of the agent protocol.

## What Changes

- **A new leaf package `engines/postgresql/logical/pgrestore` holds what both callers share.** It imports only the standard library. It contains:
  - the `pg_restore` argument builder, with explicit fields for the connection, the parallel jobs, TimescaleDB, ownership, privileges, verbose output, the archive path and the table of contents list;
  - the parallel-jobs rule: TimescaleDB restores with one job, any other restore with at most 8;
  - the parser of `errors ignored on restore: N`, which accepts a failure only when every one of the N errors is accounted for;
  - two rules for ignorable errors, errors that `pg_restore` skipped and that still let the restore count as successful: the `public` schema that already exists, and an extension the target cannot create;
  - the list of extensions the target could not create, for the agent's log line;
  - the SQL of the TimescaleDB steps before and after the restore.
- **The backend's restore calls the package.** `logical.Source` keeps the credential file, `lib/process`, the temporary archive file, the filtered table of contents, the failure classification and the `defer` that always leaves TimescaleDB restoring mode. Its only ignorable error stays the existing `public` schema, as today. Its flags do not change, except that a restore from a file with one job loses `-j 1`: a filtered restore with one job and a TimescaleDB restore from a file.
- **The agent calls the package and still runs the command through `docker exec`.** Its flags stay `--no-owner --no-acl` without `--verbose`. Its own parser, its TimescaleDB SQL and its copy of the parallel-jobs rule are deleted. `IsDiskExhausted`, `EnsureArchiveOwnerRoles` and `StageBackupViaExec` stay in the agent, because only the agent needs them.
- **Behavior change in verification:** the agent judges the errors of `pg_restore` by its whole error output instead of the last 8 KiB. A verification whose errors are all ignorable but do not fit in 8 KiB now completes instead of failing. The set of ignorable errors does not grow, and a restore that ran out of the job's disk budget is still reported without the exit code, so the backend retries it.
- **Both callers pass `-j` only for more than one job.** `-j 1` and no `-j` restore the same way, so this changes nothing a user sees.
- **The architecture check forbids GORM, Gin and goose in the agent binary.** Today the agent links none of them. The `logical` package links all three through `lib/crypto` and `lib/database`, so a direct import of it from the agent would bring them in without failing any check.
- **`docs/architecture.md`** stops saying that a verification restores exactly the way a restore does. It says that the engine builds the restore command and judges its outcome, that each caller runs the command its own way, and that the agent imports only `pgrestore`. `docs/glossary.md` gets the term "ignorable error".
- **Verification repair:** the physical backup tests wait for replication slot cleanup after cancellation, because a terminal catalog status does not mean the process has stopped and dropped its slot. A regression exercises this ordering with a suspended real process.
- Nothing is **BREAKING**: no API, schema, configuration or protocol changes. `Source.RunTimescaleDBPreRestore` and `RunTimescaleDBPostRestore` become unexported, because nothing outside the package calls them.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: a new requirement states which `pg_restore` errors are ignorable errors of a verification and that it judges them by the whole error output.
- `database-tool-failure-output`: the restore requirement gets a scenario stating that a PostgreSQL restore fails when an extension cannot be created, because the verification rule now sits next to the restore rule in one package.
- `backend-code-checks`: a new requirement states that the verification agent binary links no ORM, HTTP framework or migration tool.

## Out of scope

- Moving the legacy master key repository out of `lib/crypto`. That would remove GORM from every user of the password generator, but it is a separate rework.
- `--verbose` for the agent. With it, the agent's log tail would fill with progress lines, and the change would need its own reasoning about log size.
- A shared TimescaleDB orchestration. The backend must leave restoring mode even after a failure, while the agent discards the container instead, and between the steps the agent checks its disk budget.
- Moving `EnsureArchiveOwnerRoles`, `IsDiskExhausted` or the backend's filtered table of contents into the shared package.
- How each caller passes the password: a credential file for the backend, `PGPASSWORD` inside the container for the agent.
- The `verification` module (12d) and `agent.yaml` with the generated agent client (12e).
- New end-to-end scenarios. The suite from 12b stays unchanged.

## Impact

- **Backend, new:** `internal/engines/postgresql/logical/pgrestore` with its tests; the agent dependency check in `tools/archlint`.
- **Backend, changed:** `internal/engines/postgresql/logical` (`restore.go`, `timescaledb.go`; `preexisting_public_schema.go` and its test are deleted); `internal/agent/restore` (`restorer.go`, `dto.go`; `toleratederrors.go`, `timescaledb.go` and their tests are deleted or reduced); `internal/agent/runner` (`runner.go`, `diskwatch.go`, and the tests `runner_test.go` and `fakes_test.go`).
- **Not affected:** the HTTP contract, the agent protocol, the database schema, the image, the frontend.
- **Running instances:** after agents update, a verification with a long list of ignorable errors completes. Nothing else changes.
- **Test harness:** `internal/modules/backups/physicalapitests/backuping/backup_team_test.go` and the cancellation regression in `shutdown_test.go`.
- **Documents:** `docs/architecture.md`, `docs/glossary.md`, `backend/AGENTS.md`; the rewrite plan `temp/refactor/plan.md` (the 12c line, the 12c section and the relaxation row about package-level durations in the agent, from which `runner` is removed).
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
