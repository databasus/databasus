## 1. Configuration

- [x] 1.1 Create `internal/lib/config` with a typed `Config` structure, the log settings included, and a `Load(spec)` function per design decision 5. The function collects the process environment, `.env` through `godotenv.Read` without overriding set variables, and the published connection string into a map, drops empty values and parses the map with `github.com/caarlos0/env/v11`. It does not change the process environment, requires the connection string only with `spec.IsDatabaseRequired`, and returns a load report instead of logging. Tests:
  - a missing or empty required variable (`DATABASE_DSN=`) is named in the error;
  - `.env` does not override a variable from the environment;
  - an empty `SMTP_PORT` does not break loading;
  - an empty `DATABASE_DSN` with a published connection string gives the published string;
  - a variable set empty in the environment is filled from `.env`;
  - `os.Environ()` is unchanged after `Load`, and neither the variables from `.env` nor the published connection string appear in it;
  - loading passes without a connection string with `IsDatabaseRequired: false`.

  Check: `just test-package ./internal/lib/config/...` in `backend/`.
- [x] 1.2 Move `internal/config` to the bridge per design decisions 1 and 3. `EnvVariables` embeds `lib/config.Config` and keeps the test fields, `Install` takes the value from `app`, `GetEnv` returns it. Add `loadWithoutApp` in two files: without the `testhelpers` tag it panics, with the tag it loads the test configuration through `config.Load` and claims a slot only under `testing.Testing()`. Remove `isTestProcess`, `isStorageProbeProcess`, `unsetEmptyEnvVars`, the `os.Setenv` of the published connection string, the `--test-storage` parsing, the client tool check, the logging at load time and the `cleanenv` dependency. The `StorageProbeFlagName` constant stays until task 7.4, while the old `cmd/main.go` uses it. In `internal/features/email/email_test.go`, rewrite the `EnvVariables` literals for the embedded `Config`. Check: `just test-package ./internal/config/... ./internal/features/email/...` in `backend/`.

## 2. Logs

- [x] 2.1 Move the implementation from `internal/util/logger` to `internal/lib/logging` with its tests and without changing behavior: the fan-out, the stdout, file and OTLP sinks, the masking, the request attributes from `request_context.go`. The settings come from `Config`, not from the environment and not from its own `.env`. Check: `just test-package ./internal/lib/logging/...` in `backend/`.
- [x] 2.2 Keep in `internal/util/logger` only the bridge per design decision 6: `Install`, `GetLogger`, `GetAuditLogger`, `ExitAfterFlush`, `FlushAndCloseSinks`, `Shutdown`, `Redact`, `ContextWithRequestID`, `GetRequestID`, `ContextWithUserID` and `GetUserID`, which forward the call to `lib/logging` or return the installed logger, and `loadWithoutApp` in two files by the `testhelpers` tag, as in task 1.2. Check: `just test-package ./internal/util/logger/... ./internal/features/audit_logs/... ./internal/features/storages/files/...` in `backend/`.

## 3. Database and migrations

- [x] 3.1 Create `internal/lib/database` with `Open(ctx, dsn)` and the existing pool settings from `internal/storage/storage.go`: 10 open and 10 idle connections. Move `storage.GetDb()` to the bridge: `Install` from `app` and `loadWithoutApp` by the tag, as in task 1.2, where the test version opens the pool through `database.Open`. Check: `just test-package ./internal/lib/database/... ./internal/features/users/...` in `backend/`.
- [x] 3.2 Add a Go file to `backend/migrations` that embeds the SQL through `embed.FS`, and `database.Migrate(ctx, db, files)` on `github.com/pressly/goose/v3` at the version in `mise.toml`, with out-of-order allowed and a PostgreSQL session lock. Each test works on its own database, which it creates and drops through `lib/testdb`. Tests:
  - an empty database gets every migration, and a second call changes nothing;
  - a missed migration with a lower number is applied;
  - a failing migration returns an error with its name, the migrations before it are recorded as applied, and the tables and columns it created before the failing statement are absent from the database;
  - a second `Migrate` after the failing migration is fixed applies it;
  - two concurrent `Migrate` calls on one database both finish without an error, and each migration is recorded in the version table exactly once.

  Add an `archlint` rule (`tools/archlint`) that rejects a migration with `-- +goose NO TRANSACTION`, with a fixture test.

  Check: `just test-package ./internal/lib/database/... ./tools/archlint/...` in `backend/`.

## 4. Shutdown and background goroutines

- [x] 4.1 Create `internal/lib/background` with `Group` per design decision 10: `Go(ctx, name, fn)` runs the function, turns a panic into an error with `stack` in the log and counts the goroutine, `Wait(ctx)` waits for every goroutine or the end of `ctx`, and the number of running goroutines is available for checks. Tests: a panic does not crash the process and reaches the log; `Wait` returns after every function has finished and when `ctx` expires. Check: `just test-package ./internal/lib/background/...` in `backend/`.
- [x] 4.2 Create `internal/lib/shutdown` per design decision 12: `ListenForStopSignal(ctx)` and `Drain(ctx, DrainSpec)`, which runs the steps in parallel within the drain limit, then flushes the logs within a separate limit. The package does not import `internal/features` and does not call `context.Background`. The tests carry over the scenarios of `cmd/shutdown_test.go`: shutdown fits the limit, a step that is late does not delay the exit, a second signal does nothing. `cmd/shutdown.go` and its test stay until task 7.4. Check: `just test-package ./internal/lib/shutdown/...` in `backend/`.

## 5. HTTP

- [x] 5.1 Create `internal/lib/httpserver` per design decision 11: the server with the timeouts from `cmd/main.go:353`, a synchronous `Listen` followed by `Serve`, the panic recovery in handlers, gzip, CORS in development mode only, and the unknown path handler (a JSON 404 under `/api/`, the frontend page for the rest). Move the scenarios of `cmd/http_server_test.go` here; the file itself stays until task 7.4. Do not touch `internal/middleware`. Check: `just test-package ./internal/lib/httpserver/...` in `backend/`.

## 6. Lazy objects of the old code

- [x] 6.1 Move every package-level object in the 48 `di.go` files and both `init()` functions (`internal/features/disk/di.go`, `internal/features/backups/backups/download/di.go`) to `sync.OnceValue` without changing the getter signatures. Do the same for the global broker in `internal/util/pubsub/global.go`, which calls `logger.GetLogger()` at import time. Replace the two package-level variables `log = logger.GetLogger()` outside `di.go` (`internal/config/config.go`, `internal/storage/storage.go`) with a call at the point of use. Check: `just lint` in `backend/` and `just test-package ./internal/features ./internal/features/disk/... ./internal/features/backups/backups/download/... ./internal/features/databases/... ./internal/features/storages/... ./internal/features/notifiers/...` in `backend/`.

## 7. Composition root and entry point

- [x] 7.1 Create `internal/app` with `LoadEnvironment(ctx, spec)` and `New(ctx, environment)` per design decisions 5 and 8. `New` opens the database, applies the migrations, creates the directories, fills the bridge, moves the key from the database to its file and calls the 14 `SetupDependencies` from `internal/app/wiring.go` in the order of `cmd/main.go:420`. Tests without a build constraint:
  - after `New` no public getter of the old code panics, hangs or returns `nil`;
  - `New` on a database with a failing migration returns an error with the migration's name and does not fill the bridge;
  - `LoadEnvironment` with an empty required variable returns an error that names it.

  Check: `just test-package ./internal/app/...` in `backend/`.
- [x] 7.2 Add `(*App).Serve(ctx, address)` per design decisions 8 and 10-13. `Serve` checks the client tools, cleans the temp directory, opens the port and registers the routes of the former `setUpRoutes` with `internal/middleware`. Then `Serve` starts the 13 background loops and the HTTP server through `background.Group`. On shutdown it closes `config.BeginShutdown`, cancels the loops and passes the steps to `shutdown.Drain`: HTTP, the task registry, the group. Tests:
  - `Serve` on a free port answers 200 on `/api/v1/system/version`, and `/api/v1/no-such-route` gives 404;
  - `/api/v1/docs/swagger/index.html` and `/api/v1/docs/swagger/doc.json` give a 404 with a JSON body with both `ENV_MODE=production` and `ENV_MODE=development`;
  - a busy port returns an error before the loops start;
  - after the context is cancelled, `Serve` returns within 8 seconds and every loop has finished.

  Check: `just test-package ./internal/app/...` in `backend/`.
- [x] 7.3 Add `(*App).RunConsoleCommand(ctx, command)` for the password reset, the account creation, disabling the second factor and the administrator list, and `app.ProbeStorage(ctx, environment)` without the database. The logic and the output texts move from `cmd/main.go:220-349` and `cmd/storage_command.go` unchanged. Tests:
  - each command changes the database as the `host-console-commands` spec describes and writes the audit records for the account creation, the password change and disabling the second factor;
  - an unknown address returns an error, and no account or setting changes;
  - after each command no goroutine runs in a `background.Group` and the server port is not open;
  - `ProbeStorage` passes without a connection string.

  Check: `just test-package ./internal/app/...` in `backend/`.
- [x] 7.4 Move `main` and `healthcheck` with its test to `backend/cmd/databasus`. `main` parses the existing flags and the `healthcheck` subcommand, refuses incompatible flags before calling `app`, calls `LoadEnvironment` (without a connection string for `--test-storage`), then `New` and `Serve(ctx, ":4005")`, a console command or `ProbeStorage`, prints the result and picks the exit code. Delete `backend/cmd/main.go`, `shutdown.go`, `storage_command.go`, their tests, `http_server_test.go` and the `config.StorageProbeFlagName` constant, leaving only `databasus`, `seed_dev` and `cleanup_test_db` in `backend/cmd`. A test without a build constraint checks that each set of flags selects its command and that `--create-user` is checked before `--new-password`. Check: `just test-package ./cmd/databasus/...` and `just lint` in `backend/`.
- [x] 7.5 Add a test to `cmd/databasus` that builds the binary without the tag (`GOFLAGS=` and `go build`) and runs it as an operator. The binary runs in a temporary working directory with an empty `go.mod`, so its data directory and `.env` belong to the test. The test checks:
  - `healthcheck` with an empty environment does not panic, and its exit code and message follow whether a server answers on `:4005`: on a shared machine another agent's `just run` may hold that port, so the test cannot assume it is free;
  - `--test-storage` without a connection string saves and deletes the probe file in the test's data directory and exits with code 0;
  - on a new database the test creates, `--create-user`, `--new-password`, `--disable-2fa` and `--list-admins` exit with code 0, print the expected text and do not open the server port;
  - a password reset for an unknown address exits with a non-zero code;
  - with `ENV_MODE` set, an empty `DATABASE_DSN` and no `.env` or published connection string, the server exits with a non-zero code, and the output says that no `.env` file was found and names `DATABASE_DSN` as a missing required variable;
  - `--create-user --email=<address> --new-password=<password>` exits with a non-zero code, the output names `--password`, and no account appears or changes in the database;
  - `--new-password=<password>` without `--email` and without a connection string exits with a non-zero code, and the output names `--email` and does not mention the connection string;
  - while the server runs on the same database, a password reset by the binary exits with code 0, and the server answers 200 on `GET /api/v1/system/version` before and after the command. The test runs the server in its own process through `app.New` and `Serve` on a free port, because the binary always listens on `:4005`, which another agent's `just run` may hold on a shared machine;
  - on a new database where the test has already created a table that the first migration creates, the server exits with a non-zero code, the output names that migration, and the port does not open; after the table is dropped, the next run of a command applies the migration and exits with code 0.

  Check: `just test-package ./cmd/databasus/...` in `backend/`.
- [x] 7.6 Remove `./cmd` from the `archlint` entry points (`tools/archlint/archlint.go:18`) and move the fixtures in `tools/archlint/archlint_test.go` to `cmd/databasus`. Check: `just test-package ./tools/archlint/...` and `just lint` in `backend/`.

- [x] 7.7 Remove swag from the code: the import `_ "databasus-backend/swagger"`, the Swagger UI route and the generation at startup (none of them moves to `app`), the `backend/swagger` directory, and every `// @...` annotation in the controllers of `internal/features` together with the `// FunctionName` line that opens each block. Other comments above the handlers stay. Remove `github.com/swaggo/swag`, `github.com/swaggo/gin-swagger` and `github.com/swaggo/files` from `go.mod` through `go mod tidy`. Check: `rg -n '^\s*// @' -g '*.go'` in `backend/` finds nothing, `rg swaggo backend/go.mod` finds nothing, `just lint` in `backend/`.

## 8. Build and environment

- [x] 8.1 In the `Dockerfile`, build `./cmd/databasus` into the existing `/app/main`; remove from the `backend-build` stage the `RUN` steps that build `goose` and install swag and the `swag init` step with the comment above it; remove from the runtime image the `COPY` of the `goose` binary and of the `migrations` directory, which only the external `goose` read. Check: `just lint` at the root and the test of task 7.5, which builds `./cmd/databasus`.
- [x] 8.2 In `backend/.air.toml`, build `./cmd/databasus` and remove `swagger` from `exclude_dir`. Remove `swag init` from `scripts/dev.sh`, the swag installation and documentation generation steps from `.github/workflows/ci-release.yml`, swag from `mise.toml`, the API documentation line from `process-compose.yaml` and the `swagger` lines from `backend/.gitignore`. In `scripts/test.sh` and CI, run `cleanup_test_db` with `-tags testhelpers`. Check: `just lint` at the root.

## 9. Documents

- [x] 9.1 In `docs/architecture.md`, remove the paragraph about `runMigrations` and the external `goose` (line 172) and replace "A second signal exits immediately" (line 238) with the rule from the `application-shutdown` spec: a second signal during shutdown neither kills the process nor starts the shutdown again. In the root `AGENTS.md`, remove the `swag init` line and add to the section "Until the migration finishes" a line saying that the old code gets the configuration, the database and the logger through `config.GetEnv()`, `storage.GetDb()` and `logger.GetLogger()`, which `internal/app` fills, and that in a test build with the `testhelpers` tag they load themselves. Find with `rg` every mention of `cmd/main.go`, `go run ./cmd` and `go build ./cmd` outside the OpenSpec archive and `temp/` and bring it to the new path. In the "Migrations" section of `docs/architecture.md`, state that each migration runs in one transaction and `NO TRANSACTION` is forbidden, and add the `archlint` rule to the "Automatic checks" table. In the Code section of `docs/glossary.md`, add the term "Console command | console command | An operation of the Databasus binary that the operator runs on the host or in the container instead of starting the server", stating that the `healthcheck` subcommand is not one: it only asks the running server. Remove the Swagger section from `backend/README.md` and find with `rg -i swag` the remaining mentions outside the OpenSpec archive and `temp/`. Check: `just lint` at the root.

- [x] 9.2 Translate `proposal.md`, `design.md`, `tasks.md` and every delta spec of the change into English, keeping the requirements, the scenarios and the SHALL keywords. The translation happens before packing, because the delta specs enter the main specs `openspec/specs` on archiving, and everything committed is written in English per the root `AGENTS.md`. Check: `openspec validate app-skeleton --strict`.

## Change verification

```sh
just test-package ./internal/lib/config/... ./internal/lib/logging/... ./internal/lib/database/... ./internal/lib/background/... ./internal/lib/shutdown/... ./internal/lib/httpserver/... ./internal/config/... ./internal/util/logger/... ./internal/util/pubsub/... ./internal/app/... ./cmd/databasus/... ./tools/archlint/...
just test-package ./internal/features ./internal/features/email/... ./internal/features/audit_logs/... ./internal/features/storages/files/... ./internal/features/users/... ./internal/features/disk/... ./internal/features/backups/backups/download/... ./internal/features/databases/... ./internal/features/storages/... ./internal/features/notifiers/... ./internal/features/tasks/...
just test-package ./internal/features/backups/backups/backuping/logical/... ./internal/features/restores/restoring/... ./internal/features/verification/config/... ./internal/features/verification/runs/... ./internal/util/encryption/...
```
