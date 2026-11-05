## Why

The backend assembles itself while its packages are imported. 110 package-level variables in 48 `di.go` files create services before `main` starts. As a result the configuration loads itself on first use, test mode is guessed from the binary's name, and a second application instance cannot be built in one process. Milestone 3 of the backend rewrite plan builds `apptest` and a database per test on top of this composition root, so the skeleton is needed now, before the first module moves.

## What Changes

- The server gets one entry point, `backend/cmd/databasus`. `main` and `healthcheck` move there; argument parsing, output and exit codes stay in `cmd`. The shutdown logic moves to `lib/shutdown`, the HTTP server test to `lib/httpserver`, and the console commands and the storage probe to `internal/app`. The other `backend/cmd/*.go` files are deleted; `cmd/seed_dev` and `cmd/cleanup_test_db` stay. The `healthcheck` subcommand and the flags `--test-storage`, `--new-password`, `--email`, `--create-user`, `--password`, `--role`, `--name`, `--disable-2fa` and `--list-admins` do not change. In the image the binary stays available as `/app/main` and as `databasus` on `PATH`.
- `internal/app` appears:
  - `app.LoadEnvironment` loads the configuration and builds the logging;
  - `app.New` opens the database, applies the migrations, hands the configuration, the database and the logger to the old code through the bridge and calls the 14 `SetupDependencies` functions in one explicit order;
  - `(*App).Serve` registers the routes, starts the background loops and the HTTP server, and runs the shutdown;
  - `(*App).RunConsoleCommand` runs a console command without routes or background work.
- `internal/lib` packages appear:
  - `config` reads the environment variables once, does not change the process environment and does not load at import time;
  - `logging` grows out of `util/logger` and takes its settings from `config`;
  - `database` opens the connection pool and applies the embedded migrations;
  - `shutdown` gives the only signal handler and drains the shutdown steps within a limit;
  - `background` gives `Group` with `Go` and `Wait`: a goroutine whose panic is recovered and which shutdown can wait for;
  - `httpserver` holds the server with its timeouts, the panic recovery in handlers and the 404 answer for an unknown path under `/api/`;
  - `testdb` gives a test an empty database of its own; it starts with one function and grows in milestone 3.
- A bridge for the old code. `config.GetEnv()`, `storage.GetDb()`, `logger.GetLogger()` and the shutdown signal return what `app` handed over. Every object in `di.go`, both `init()` functions (`features/disk`, `features/backups/backups/download`) and the global broker of `util/pubsub` become lazy and are built on the first getter call.
- Test mode of the old code is set by the `testhelpers` build tag instead of the binary's name. In a test build, access points that `app` did not fill load themselves, as now. In the production build such a call panics with a clear message.
- Migrations are embedded in the binary and applied by the goose library with out-of-order allowed and with a lock held while they are applied.
- **BREAKING** for the image: the `backend-build` stage no longer builds `goose`, and the runtime image no longer copies the `goose` binary or the `migrations` directory.
- **BREAKING** for developers: `go build ./cmd` and `go run ./cmd` no longer build the server; `./cmd/databasus` does. `go run ./cmd/cleanup_test_db` needs the `testhelpers` tag. The Dockerfile, `.air.toml`, `scripts/*.sh`, CI and `AGENTS.md` are updated in this change, and no old path remains.
- **BREAKING** for anyone who opened the Swagger UI of their instance: swag is removed entirely. The server no longer serves the Swagger UI or an API description in any mode. Removed: the generation at server start, the `swag init` steps in the `Dockerfile`, CI and `scripts/dev.sh`, the generated `backend/swagger` package, the `swaggo` dependencies and the `// @...` annotations in 26 controllers. The API description will appear in the contract `backend/api/openapi.yaml`: milestone 3 builds the generation pipeline, and each module's operations enter the contract when the module moves in milestones 4-13. Until then there is no API description.
- Console commands get tests: each command runs the way the binary runs it, and the test checks the result in the database, the exit code and the output.
- `docs/architecture.md` follows the `application-shutdown` spec: a second signal during shutdown does nothing.

## Capabilities

### New Capabilities

- `internal-database-migrations`: how the application applies the migrations of its internal database at startup by itself, including missed ones and ones applied by two processes at once, and what happens when a migration fails.
- `host-console-commands`: which console commands the binary has and under which paths it is available, what each command does and does not do (it starts neither the server nor background work), exit codes, running inside a running container, and the storage probe without the internal database.

### Modified Capabilities

- `api-http-server`: the application serves neither an API documentation interface nor an API description in any mode.
- `runtime-configuration`: an empty value of any application setting means it is not set, so the environment file fills a variable set empty in the process; the tools the application starts get the operator's variables unchanged and none of the values the application loaded for itself.

## Out of scope

- Moving features into `internal/modules` and the bodies of the `SetupDependencies` functions. Listeners stay in their features and become events when their module moves.
- `background.Runner` with passes, `job_id` and a health report, and `logging.With` for operation attributes in the context. Both appear in milestone 4; the old loops move to `Runner` with their module.
- `Conn(ctx, db)` and `Transaction(ctx, fn)` in `lib/database`: they appear in milestone 4 with the first repositories.
- Moving `internal/middleware` to `lib/httpserver`. The access log knows about verification agents, so it moves in milestone 4, when the actor attributes arrive through `logging.With`.
- `lib/clock`, `lib/apierror` and the `Problem` format appear in milestone 3, `lib/process` in milestone 8.
- Moving the cancellation registry `features/tasks/cancellation` to `lib`.
- `apptest`, migrated databases in `testdb` and dropping the shared per-slot metadata databases: milestone 3.
- Changes to the database schema, the console flags and the HTTP API, except removing the Swagger UI and the API description.

## Impact

- Code: `backend/cmd` (moves to `cmd/databasus`), new `backend/internal/app` and `backend/internal/lib/{config,logging,database,shutdown,background,httpserver,testdb}`, `backend/internal/config`, `backend/internal/storage`, `backend/internal/util/logger`, `backend/internal/util/pubsub`, all 48 `di.go` files, `backend/migrations` (a Go file with the embedded migrations), `backend/cmd/cleanup_test_db`, `backend/tools/archlint` (entry points and the migration transaction rule), the annotations in 26 controllers of `internal/features`, the deletion of `backend/swagger`.
- `go.mod` dependencies: `github.com/pressly/goose/v3` and `github.com/caarlos0/env/v11` are added; `github.com/ilyakaznacheev/cleanenv`, `github.com/swaggo/swag`, `github.com/swaggo/gin-swagger` and `github.com/swaggo/files` are removed. goose 3.27.3 requires newer minor versions of `pgx`, `moby/moby/api`, `moby/moby/client`, `go-sql-driver/mysql` and `klauspost/compress`, which `go mod` raises.
- Build and environment: `Dockerfile`, `backend/.air.toml`, `backend/.gitignore`, `scripts/test.sh`, `scripts/dev.sh`, `.github/workflows/ci-release.yml`, `mise.toml` (the swag version), `process-compose.yaml` (the API documentation link), `docker/start.sh` (a comment naming the configuration file).
- Documents: `docs/architecture.md` (the paragraph on the external goose, the second signal at shutdown, the migration transaction rule), `docs/glossary.md` (console command), the root `AGENTS.md` (section "Until the migration finishes", the `swag init` line), `backend/README.md` (the Swagger section), `temp/refactor/plan.md`.
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
- Running instances: the schema does not change. Only the way migrations are applied changes, and migrations already applied do not run again.
