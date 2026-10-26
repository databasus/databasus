## Why

What Databasus knows about a database engine lives inside the GORM models of the old `features/databases`. The struct that is saved to `postgresql_logical_databases` also connects to the server, inspects privileges, creates a read-only user (in one 449-line method) and detects the version. Its siblings do the same for physical PostgreSQL, MySQL, MariaDB and MongoDB. That code depends on `features/sshtunnel`, `util/encryption`, `util/tools` and GORM, so nothing outside the old feature can use it without the tables, and the verification agent, which may import only `lib` and `engines`, cannot use it at all. The client tool lookup in `util/tools` finds its directory through a package-level `sync.OnceValue` that panics, keeps the `--version` verdicts in a package-level `sync.Map` and exits the process itself. External tools run through bare `exec.CommandContext`, so a cancelled tool can leave its children running.

`docs/architecture.md` puts this code into `internal/engines/<engine>` and the tool runner into `lib/process`. Milestone 8 of the backend rewrite plan is split in two. This change, the first half, builds the engine layer under the old code without changing the schema, the API or the frontend. `databases-module`, the second half, then moves the feature into a module on top of it.

## What Changes

- `internal/engines/postgresql/{logical,physical,shared}`, `internal/engines/mysql`, `internal/engines/mariadb`, `internal/engines/mongodb` and `internal/engines/mysqlfamily` appear. Each engine takes plain connection settings without GORM or JSON tags, with secrets in clear text, and offers connecting and testing a connection, version and cluster detection, the database size, privilege inspection, read-only and replication user provisioning, SSH tunnels, and for physical PostgreSQL its replication slots and inspection connection. The code moves from `features/databases/databases/*` and keeps its behavior, apart from the two refinements below.
- `CreateReadOnlyUser`, `CreateReplicationOnlyUser` and the `ShouldSuggest*` family are split into steps inside their engine. Creating a user retries only when the generated name is already taken, recognised by the driver's error code; today four engines retry three times on any error. Where a driver gives no code for a failure, the text comparison is collected into one function per engine.
- `internal/engines/clienttools` holds what every engine's client tools share: the bundle descriptions each engine registers, the release-line comparison and a checker that runs each client's `--version` once at startup and keeps its verdicts. Each engine declares its bundles and finds its executables in a directory it is given. `util/tools` is deleted, and `util/namelist` moves to `lib/namelist`, because three engines share its list helpers.
- `internal/lib/process` runs an external tool in its own process group, bound to the operation's context, and kills the whole group when the context ends. Its first user is the client tool check. The dump and restore tools move to it in milestone 9, `pg_basebackup` and `pg_receivewal` in milestone 10.
- The SSH tunnel code moves from `features/sshtunnel` to `internal/lib/sshtunnel` and takes its secrets in clear text and its logger as a required argument. `features/sshtunnel` keeps only the stored tunnel settings of the old models.
- The password generator moves from `util/encryption` to `lib/crypto`.
- `internal/app` resolves the client tool directory at startup, runs the client tool check and refuses to start when a PostgreSQL bundle fails, instead of `util/tools` exiting the process. It hands the checker to the old system health check, which keeps reporting the startup verdict with a fresh look at the files.
- The old engine models in `features/databases` keep their tables, JSON shape, validation and encryption, and delegate every engine operation to `internal/engines` with decrypted settings. Callers in backups, restores, health checks, verification and telemetry keep calling the old models. They change the imports of the moved helpers, and three call sites outside the feature pass their context to the methods that start from `context.Background()` today: `restores/service.go`, `restores/restoring/restorer.go` and a physical backup test helper.
- The engine tests move with the code into `internal/engines`. The physical PostgreSQL tests start their own containers instead of the shared compose servers, because a test in the target layout reads no configuration.
- Two lint rules become machine checks: production code starts external processes only through `lib/process`, and the engines import neither an HTTP framework nor an ORM.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backend-code-checks`: the lint rejects production code that starts an external process outside `lib/process`, and an HTTP framework or ORM in the engine drivers.

## Out of scope

- The `databases` module, its contract, its frontend, its migration and its events. They are `databases-module`.
- A registry of engines and the capability interfaces it consumes. They arrive with their consumer in `databases-module`.
- Moving dump and restore into engines, and moving the dump, restore, `pg_basebackup` and `pg_receivewal` launches onto `lib/process`. Milestones 9 and 10.
- The verification agent and its own restore code. Milestone 12.
- Moving `internal/util/testing/containers` to `lib/testcontainers`.
- Removing the reset of the shared test servers before a full run. The old tests of `databases` and `backups` still use them.
- Any change to which server versions are accepted, which client a version uses, or what a read-only or replication user may do.

## Impact

- Backend: new `backend/internal/engines`, `backend/internal/lib/process` and `backend/internal/lib/sshtunnel`; `backend/internal/lib/crypto`; `backend/internal/features/databases` and its `databases/*` subpackages reduced to the old models and their delegation; `backend/internal/features/sshtunnel`; `backend/internal/features/system/healthcheck`; `backend/internal/app`; the old callers of the moved helpers in `features/backups`, `features/restores`, `features/tests` and `internal/util/testing/bastion`; `backend/internal/config` and its test loader for the client tool directory; `backend/.golangci.yml`; `backend/cmd/cleanup_test_db` if the moved tests change slot names.
- Deleted: `backend/internal/util/tools`, the password generator in `backend/internal/util/encryption`, and the engine behavior and SSH code that moves.
- No schema, contract, frontend or user-facing change. A read-only or replication user whose creation fails for a reason other than a taken name now fails on the first attempt instead of after three.
- Documents: `docs/architecture.md` (the `clienttools` package beside the engines, the two new automatic checks), `docs/glossary.md` (client tool bundle, engine source), `backend/AGENTS.md` (one sentence that the linter enforces the `lib/process` rule), `temp/refactor/plan.md` (milestone 8 split into 8a and 8b, the temporary relaxations this change adds).
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
