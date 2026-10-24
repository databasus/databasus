# Backend rules

Rules for writing Go code in `backend/`, which holds the server and the verification agent. [`docs/architecture.md`](../docs/architecture.md) describes how the backend is organized and why. Before changing the structure of some code, read the sections that cover it. The project-wide rules are in the root [`AGENTS.md`](../AGENTS.md). Tests follow the [testing skill](../.agents/skills/testing/SKILL.md), and HTTP operations follow the [api-design skill](../.agents/skills/api-design/SKILL.md).

## Commands

Run in `backend/` unless noted.

| Command | Purpose |
|---|---|
| `just lint` | Check the backend code, the contracts and the generated code without changing files |
| `just format` | Format the Go code |
| `just test-package <packages>` | Run the given test packages |
| `go test -race -tags testhelpers -count=1 -run 'TestName/scenario' <package>` | Run one test; packages in the target layout need no tag |
| `just generate-api` at the root | Regenerate servers, clients and frontend types after editing `api/*.yaml` |
| `goose -dir migrations create <name> sql` | Create a migration stub, then fill it by hand |

## Where code goes

- A domain rule goes into the module that owns the data, under `internal/modules/<name>/internal/`. Other modules reach it only through that module's `module.go`.
- Code that knows a database engine but no tables goes into `internal/engines/<engine>`.
- Code with no domain knowledge goes into `internal/lib/<name>`.
- Wiring goes into `internal/app` and nowhere else.
- A file holds one role, and its name says which: `service.go`, `*_repository.go`, `<entity>.go`, `errors.go`. Test files mirror the source split. Create a file only when it has content.

## Dependencies

- A module or a `lib` service takes its dependencies through `Deps` in `New(deps Deps)`. A `Deps` literal lists every field. `exhaustruct` fails the build when one is missing, so adding a dependency forces every call site to supply it.
- No package-level variable holds a dependency, and no package does work at import time.
- Configuration values arrive through `Deps`. A module never reads the environment.

## Handlers, services, repositories

- A handler implements the generated strict server interface: it takes the decoded request, calls one service method with the request context and returns the generated response type. It holds no logic, and it checks nothing beyond what the contract already enforces.
- A service checks access first: it looks the object up, answers "not found" when it is missing, then calls `workspaces.Authorize`. It opens transactions with `database.Transaction(ctx, fn)` and writes the audit event inside the same transaction.
- A repository runs every query through `database.Conn(ctx, r.db)`. A lookup that finds nothing returns `false` as its second value, never `gorm.ErrRecordNotFound`. A repository changes one column with one `UPDATE`, never by saving the whole row. It uses no GORM associations and no `Preload` across tables.

## Errors

- Every sentinel error and error type of a package lives in its `errors.go`.
- Wrap with context using `%w`, and compare with `errors.Is` or `errors.As`, never by the error's text.
- A service returns errors from `lib/apierror` with a stable `code` for anything the client must tell apart. The handler never builds an error body itself, and an internal error's text never reaches the client.

## Engines and providers

Code outside a family never switches on a database, storage or notifier type: only the family's registry maps a type to its implementation. To add a provider:

1. Add a package under the family with its settings struct, its repository and its behavior.
2. Add a migration with its table. Columns carry `NOT NULL`, `DEFAULT` and `CHECK` where they apply, and the row references the parent through the composite key `(id, type)`.
3. Register it in the family's registry.
4. Add its schema to the contract as one more variant of the type's `oneOf`.

## Long-running work

- A periodic job is a function given to `background.Runner`. Never write your own ticker loop or `hasRun` guard.
- Run external tools only through `lib/process`, so the process dies with its context.
- Start goroutines only through the `lib` helpers that tie them to a context and turn a panic into an error.
- Take the current time from the clock in `Deps`, and pass it into SQL as a parameter. All time is UTC.

## Stored files

Write and delete files in a storage only through the storages module's file store. A backup is published by spending its write receipt in the publishing transaction. The rules are in "Stored files" in [`docs/architecture.md`](../docs/architecture.md#stored-files).

## Migrations

- Follow "Data" and "Migrations" in [`docs/architecture.md`](../docs/architecture.md#data): module prefixes, `RESTRICT` between modules, constraints in the schema, upgrade safety.
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

- `just format` sets the formatting. Beyond that, separate logical blocks with a blank line: after declarations, between error handling and what follows, before the final `return`.
- `modernize` enforces current standard-library idioms. Use `omitzero` instead of `omitempty` for `time.Time`, `time.Duration`, structs, slices and maps.
- Every HTTP operation is described in the contract; Go code carries no API annotations.

## Verification agent

The verification agent's code lives in `cmd/verification-agent` and `internal/agent`. It runs on the user's machine, so it imports only `lib`, `engines` and the generated agent client. `depguard` keeps GORM, Gin and the modules out of it. The agent reads its settings from its own configuration file and flags, and stores the agent token with permissions `0600`. It talks to containers only through the Docker Engine API, which Podman also serves. Its end-to-end tests run it against a backend from the test harness.
