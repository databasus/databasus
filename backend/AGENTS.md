# Backend rules

Rules for writing Go code in `backend/`, which holds the server and the verification agent. [`docs/architecture.md`](../docs/architecture.md) describes how the backend is organized and why. Before changing the structure of some code, read the sections that cover it. The project-wide rules are in the root [`AGENTS.md`](../AGENTS.md). Tests follow the [testing skill](../.agents/skills/testing/SKILL.md), and HTTP operations follow the [api-design skill](../.agents/skills/api-design/SKILL.md).

## Commands

`mise run` tasks run from any directory; the other commands run in `backend/`.

| Command | Purpose |
|---|---|
| `mise run backend:lint` | Check the backend code, the contracts and the generated code without changing files |
| `mise run backend:format` | Format the Go code |
| `mise run test` | Check the work: the backend packages without the end-to-end ones and the frontend tests |
| `mise run test-full` | Run all tests for the usual Docker development environment; [testing.md](../docs/testing.md#checking-work) lists the included suites and the Podman/filesystem exceptions |
| `go test -count=1 -run 'TestName/scenario' <package>` | Repeat one failing test; [`docs/testing.md`](../docs/testing.md#commands) gives the full command |
| `mise run generate-api` | Regenerate servers, clients and frontend types after editing `api/*.yaml` |
| `goose -dir migrations create <name> sql` | Create a migration stub, then fill it by hand |

## Where code goes

- A domain rule goes into the module that owns the data, under `internal/modules/<name>/internal/`. Other modules reach it only through that module's `module.go`.
- Code that knows a database engine but no tables goes into `internal/engines/<engine>`.
- Code with no domain knowledge goes into `internal/lib/<name>`.
- Wiring goes into `internal/app` and nowhere else.
- Code is split into files per [Code is split into logical units](../AGENTS.md#code-is-split-into-logical-units). On the backend a role file is named after its role: `service.go`, `*_repository.go`, `<entity>.go`, and every error of a package sits in `errors.go`. Create a file only when it has content.

## Dependencies

- A module or a `lib` service takes its dependencies through `Deps` in `New(deps Deps)`. A `Deps` literal lists every field. `exhaustruct` fails the build when one is missing, so adding a dependency forces every call site to supply it.
- No package-level variable holds a dependency, and no package does work at import time.
- Read the environment only in `lib/config` and pass configuration values through `Deps`.
- Test libraries belong only to test files and test-support packages. Engines import neither Gin nor GORM; the verification agent's additional boundaries are below.
- Take the caller's context, or `t.Context()` in tests. Only commands and application composition create root contexts.

## Handlers, services, repositories

- A handler implements the generated strict server interface: it takes the decoded request, calls one service method with the request context and returns the generated response type. It holds no logic, and it checks nothing beyond what the contract already enforces.
- A service checks access first: it looks the object up, answers "not found" when it is missing, then calls `workspaces.Authorize`. It opens transactions with `Transaction(ctx, fn)` of the `*database.Transactor` from `Deps` and writes the audit event inside the same transaction, or right after an effect outside the database, such as a sent email.
- A repository runs every query through `database.Conn(ctx, r.db)`. A lookup that finds nothing returns `false` as its second value, never `gorm.ErrRecordNotFound`. A repository changes one column with one `UPDATE`, never by saving the whole row. It uses no GORM associations and no `Preload` across tables.
- Repositories are normally unexported and exposed through a `Repositories` bundle built in `module.go` with `NewRepositories`. Export a repository only when another package names its type or constructs it directly.

## Errors

- Every sentinel error and error type of a package lives in its `errors.go`.
- Wrap with context using `%w`, and compare with `errors.Is` or `errors.As`, never by the error's text.
- A service returns errors from `lib/apierror` with a stable `code` for anything the client must tell apart. The handler never builds an error body itself, and an internal error's text never reaches the client.

## Engines and providers

Code outside a family never switches on a database, storage or notifier type: only the family's registry maps a type to its implementation. A consumer that serves several engines through an interface of its own maps each database type to its implementation in one registry, and only there. To add a provider:

1. Add a package under the family with its settings struct, its repository and its behavior.
2. Add a migration with its table. Columns carry `NOT NULL`, `DEFAULT` and structural constraints where they apply, following [Migrations](#migrations), and the row references the parent through the composite key `(id, type)`.
3. Register it in the family's registry.
4. Add its schema to the contract as one more variant of the type's `oneOf`.
5. Add its conversion between that variant and its settings to the module's `httpapi`.

## Long-running work

- A periodic job is a function given to `background.Runner`. Never write your own ticker loop or `hasRun` guard.
- Run external tools through `lib/process`, so the process dies with its context. The verification agent's daemon start is the one exception, explained in [Long-running work](../docs/architecture.md#long-running-work). The linter rejects other `os/exec` calls in production code.
- Start goroutines only through the `lib` helpers that tie them to a context and turn a panic into an error.
- Take the current time from the clock in `Deps`, and pass it into SQL as a parameter. All time is UTC.

## Stored files

Write and delete files in a storage only through the storages module's file store. A backup is published by spending its write receipt in the publishing transaction. The rules are in "Stored files" in [`docs/architecture.md`](../docs/architecture.md#stored-files).

## Migrations

- Follow "Data" and "Migrations" in [`docs/architecture.md`](../docs/architecture.md#data): module prefixes, `RESTRICT` between modules, constraints in the schema, upgrade safety.
- Store enum values as `TEXT` and lists of enum values as `TEXT[]`. Application code validates their allowed values. Do not use PostgreSQL enum types or `CHECK` constraints to restrict enum values, including a provider discriminator fixed to one value. Keep structural constraints, such as foreign keys, uniqueness, numeric bounds and required-field relationships.
- One statement per table, constraint and index. Align column types and put each constraint clause on its own line.

```sql
CREATE TABLE storages_s3 (
    storage_id UUID NOT NULL,
    type       TEXT NOT NULL DEFAULT 'S3',
    bucket     TEXT NOT NULL,
    region     TEXT NOT NULL
);

ALTER TABLE storages_s3
    ADD CONSTRAINT fk_storages_s3_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
```

## Logging

- Log through the logger from `Deps` with the `...Context` methods, so `request_id`, `job_name` and `job_id` from the context reach every record.
- The message is a constant lowercase phrase without a trailing period. Values, IDs and errors are attributes with `snake_case` keys: `logger.InfoContext(ctx, "backup completed", "backup_id", backupID, "size_mb", sizeMb)`.
- Add an entity's ID to the context with `logging.With` where the entity becomes known, so later records carry it.
- Levels: Debug for routine steps, Info for completed actions and state changes, Warn for a degraded but recoverable state, Error for failures that need attention.
- Never log passwords, tokens, keys, full headers or request bodies. Masking happens in the log handler, not at call sites.

## Code style

- `mise run backend:format` sets the formatting. Group statements per [Blank lines separate logical steps](../AGENTS.md#blank-lines-separate-logical-steps).
- `modernize` enforces current standard-library idioms. Use `omitzero` instead of `omitempty` for `time.Time`, `time.Duration`, structs, slices and maps.
- Every HTTP operation is described in the contract; Go code carries no API annotations.
- A `switch` on an enum covers every value. Use the fixed import aliases declared in the lint configuration.
- Only commands and tools print directly or exit the process; application code logs through its logger and returns errors.
- A lint suppression names the specific rule and explains its reason. Immutable shared tables and process-level test fixture synchronization need such a suppression; application dependencies still belong to their owning instance.

## Verification agent

The verification agent's code lives in `cmd/verification-agent` and `internal/agent`. It runs on the user's machine, so it imports only `lib`, `engines` and the generated agent client. `depguard` enforces its direct import boundaries; `archlint` also keeps GORM, Gin and goose out of its whole production dependency graph. The agent reads its settings from its own configuration file and flags, and stores the agent token with permissions `0600`. It talks to containers only through the Docker Engine API, which Podman also serves. Its end-to-end tests run it against a backend from the test harness.
