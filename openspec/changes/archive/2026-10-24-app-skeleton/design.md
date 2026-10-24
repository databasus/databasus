## Context

Why the milestone is needed is in proposal.md, section Why. This document covers only what shapes the approach.

**Assembly at import time.** The objects of the old code are created by the initializers of package-level variables in 48 `di.go` files (110 variables) and by two `init()` functions: `internal/features/disk/di.go:8` and `internal/features/backups/backups/download/di.go:25`. For example, `internal/features/databases/di.go:17` calls `notifiers.GetNotifierService()`, `logger.GetLogger()` and `encryption.GetFieldEncryptor()` at import time. Go runs these expressions before `main`, so everything they need has to be able to assemble itself.

**Access points that load themselves.**

- `config.GetEnv()` (`internal/config/config.go:114`) does several things on its first call:
  - looks for `go.mod` upward from the working directory and reads `.env` with `godotenv.Load`, which writes the values into the process environment;
  - adopts the connection string from `/dev/shm` with `os.Setenv` (`config.go:314`);
  - removes empty variables from the process environment (`config.go:322`);
  - checks the client tools (`config.go:213`);
  - decides from `os.Args[0]` whether this is a test (`config.go:176`, `config.go:358`), and in a test process claims a metadata database slot through an advisory lock (`config.go:194`);
  - parses the `--test-storage` flag itself (`config.go:339`), because flags are parsed later.
- `logger.GetLogger()` (`internal/util/logger/logger.go:50`) reads `.env` itself (`settings.go:244`) and five of its own environment variables, bypassing the configuration.
- `storage.GetDb()` (`internal/storage/storage.go:19`) opens the pool on its first call.

Production code calls `storage.GetDb()` 145 times, `config.GetEnv()` 40 times and `logger.GetLogger()` 83 times. Besides `GetLogger`, the old code calls `GetAuditLogger`, `ExitAfterFlush`, `FlushAndCloseSinks`, `Shutdown`, `Redact`, `ContextWithRequestID`, `GetRequestID`, `ContextWithUserID` and `GetUserID` from `util/logger`. Among the callers are `internal/features/users/middleware/middleware.go`, `internal/features/storages/files/worker.go` and `internal/middleware/request_id.go`.

**Back links.** The 14 `SetupDependencies` functions are `sync.OnceFunc` values in `di.go`. `cmd/main.go:420` calls them in a hand-written order. They are called 88 more times elsewhere: 85 calls are in tests and test helpers, 23 of them in `internal/features/test_once_protection_test.go`, and 3 calls are made by the console commands (`cmd/main.go:263`, `295`, `321`).

**Background loops.** `cmd/main.go:459` starts 13 loops with bare `go` and `runWithPanicLogging`. The shutdown (`cmd/shutdown.go:62`) waits for the HTTP server and the task registry, but not for the loops. Tests start loops directly through `Get*Scheduler()` and helpers such as `StartSchedulerForTest` (`backuping/logical/testing.go:247`).

**HTTP.** `cmd/main.go:123-147` builds Gin with the middleware from `internal/middleware`. Its access log (`internal/middleware/access_log.go:12`, `48`) imports `internal/features/verification/agents` to add `agent_id`. An error from `ListenAndServe` is only logged (`cmd/main.go:366`).

**Migrations.** `runMigrations` (`cmd/main.go:551`) runs the external `goose` on `./migrations` without out-of-order. The `goose` binary is built by `RUN` steps inside the `backend-build` stage (`Dockerfile:24-36`) and copied into the runtime image (`Dockerfile:163`). All 89 migrations are transactional; none has `NO TRANSACTION`.

**Console commands.** `cmd/main.go:82-119` parses the flags. Then it applies the migrations, creates the directories, moves the key from the database to its file and runs the first matching command, which exits the process itself. The services behind these commands are covered by tests in `internal/features/users/services` (`console_account_test.go`, `admin_listing_test.go`, `disable_two_factor_test.go`). The `cmd` layer (flag parsing, start order, exit codes, output) has no tests. The storage probe (`cmd/storage_command.go`) is covered by `cmd/storage_command_test.go`. The user documentation calls the commands as `docker exec -it databasus ./main ...` (`README.md:267`, `275`, `283` and five translations). The startup script calls `/app/main --test-storage` (`docker/start.sh:351`) and exports `DATABASE_DSN` for the server itself (`docker/start.sh:555`).

**Rules already in force.** `cmd/databasus`, `internal/app` and `internal/lib` belong to the target layout (`backend/.golangci.yml:131`, `tools/archlint/archlint.go:25`). Code that lands there has to pass every linter and `archlint` at once:

- `fmt.Print*` and `os.Exit` are allowed only in `cmd` and `tools`;
- `os.Getenv` is allowed only in `lib/config`;
- `context.Background` is allowed only in `cmd` and `app`;
- bare `go` is allowed only in `lib`;
- logs carry attributes only and use the `...Context` methods;
- a file with the `testhelpers` tag, test files included, is forbidden in the target layout (`tools/archlint/build_tags.go`).

`archlint` does not stop new code from importing old code (`layerNone`, `tools/archlint/layers.go:36`). But `docs/architecture.md` forbids `lib` to know about modules and domain logic, so in this change `lib` imports no `internal/features` package. The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- The production binary builds nothing at import time. Everything is assembled in `internal/app`, in an order visible in one file.
- The configuration is read once, inside `app`, before the logger and the database, and does not change the process environment.
- Shutdown waits for HTTP requests, registered tasks and background loops within the existing 8-second limit.
- The old tests pass without changes to their logic.
- The console commands behave as now, and tests of the `cmd` layer and of the real binary pin that behavior.

**Non-Goals:**

- Removing the package-level variables of the old code. The bridge keeps them; moving the modules removes them.
- Giving tests an application instance of their own. That is `apptest` in milestone 3; the skeleton only makes it possible.

## Decisions

### 1. The bridge hands only `lib` to the old code; `app` does not build the feature graph

`app.New` opens the database pool and hands the configuration, the logger, the pool and the shutdown signal to the old access points: `config.Install`, `storage.Install`, `logger.Install`. The feature objects are still created by their `di.go`, on the first getter call.

*Rejected: `app` builds all 110 objects and the getters return them.* Go cannot keep these objects in one shared package: the features would import it, and it would import the features. So a variable would stay in every package anyway, and `app` would get more than a thousand lines of wiring that milestones 4-13 throw away piece by piece. Besides, tests call the getters without `app` and would get `nil`. A `TestMain` that builds the application would be needed in 96 test packages, and a package's internal tests cannot import `app` because of the import cycle.

### 2. `di.go` becomes lazy through `sync.OnceValue`

Each `var x = &X{...}` becomes `var getX = sync.OnceValue(func() *X {...})`, and the getter returns `getX()`. Both `init()` functions become lazy values the same way, and so does the global broker of `util/pubsub`, the one package-level variable outside `di.go` that calls a getter. Getter signatures do not change, so callers and tests are not edited.

*Rejected: keep the assembly at import time and fill the access points before it.* Go has no code that is guaranteed to run before the initialization of imported packages. Only an `init()` in a package everyone imports first gives such a guarantee, and that is exactly the hidden dependency on import order that `docs/architecture.md` forbids.

The price: the compiler no longer finds an initialization cycle inside a package. A cycle between lazy values deadlocks on the first call. There are no cycles now, or the package would not compile. New cycles are caught by the test from decision 4 that calls every getter.

### 3. Test mode of the old code is set by the `testhelpers` build tag

The `config`, `storage` and `logger` access points get two files each with one function, `loadWithoutApp`:

- without the tag it panics with a message saying that the getter was called before `app.New`;
- with the `testhelpers` tag it loads the value as now: the test DSN instead of the working one, the slot claim, the fixture ports.

These files live in the old packages, where the tag is allowed. Every test command already passes the tag (`backend/Justfile`, `scripts/test.sh:59`, CI). `archlint` already checks that the production binary builds without it. The test fields of the configuration (`TestDatabaseDsn`, `TestLocalhost`, the fixture ports) stay in the old `config.EnvVariables` structure. It embeds `lib/config.Config`, so `config.GetEnv().DataFolder` keeps working. Only the `go test` binary needs the slot claim, not the `cleanup_test_db` tool. `testing.Testing()` inside the tagged file draws that line: such a file does not reach the production binary, so neither does the `testing` package.

*Rejected: keep detecting tests by `os.Args[0]`.* It breaks when the binary is renamed or a test runs under a debugger. Besides, a production binary named `*.test` would silently switch to the test database.

*Rejected: a `TestMain` in every test package.* That is 96 test edits for code that milestone 3 replaces with `apptest`.

One test changes mechanically: `internal/features/email/email_test.go` builds `config.EnvVariables` as a literal, and after the embedding the SMTP fields sit in the nested `Config`.

### 4. The production build is checked by running the real binary

A test in `cmd/databasus` builds the binary without the tag (`GOFLAGS=` and `go build`) and runs it the way an operator does. A panic instead of the expected output means some package called a getter at import time. Task 7.5 lists what it checks. The binary finds its data directory and `.env` through the `go.mod` above its working directory, so each run gets a temporary working directory with an empty `go.mod`, and nothing from the checkout leaks in.

The second test lives in `internal/app` and has no build constraint, because `archlint` forbids the tag in the target layout. The tag that `just test-package` passes turns on the test build for it. The test calls every public getter of the old code after `app.New`. It catches the cycles of lazy values from decision 2 and getters to which the bridge handed no dependency.

Three test packages need an empty database of their own: `lib/database`, `internal/app` and `cmd/databasus`. `lib/testdb.CreateDatabase(t)` creates it on the test server and drops it after the test. Milestone 3 extends this package with migrated databases for `apptest`.

*Rejected: rely on a linter.* `gochecknoglobals` stays off until milestone 13, and no linter expresses a ban on calls at import time.

### 5. Configuration: `lib/config` parses, `app` loads and reports

`app.LoadEnvironment(ctx, spec)` calls `config.Load(spec)` and builds the logging. `cmd` only parses arguments, prints the result and picks the exit code. So `internal/app` stays the only place that reads the configuration, as `docs/architecture.md` requires.

`config.Load(spec)` collects the values into a map without touching the process environment:

1. Takes the process environment and drops its empty values.
2. Adds the values from `.env`, when the file exists, through `godotenv.Read`, also without empty ones, and does not override the variables left after step 1. A variable set empty already counts as unset at this point, so `.env` fills it.
3. Adds the connection string from the published file when `DATABASE_DSN` is still missing after steps 1 and 2. An empty `DATABASE_DSN` therefore does not hide the published string.
4. The process environment does not change: empty values are dropped from the map only.
5. Parses the map into the typed `Config` structure with `github.com/caarlos0/env/v11` through `Options.Environment`. When a required variable is missing or empty, the error names it.

`Config` also carries the log settings: `LOG_LEVEL`, `LOG_FILE_IS_ENABLED`, `OPEN_TELEMETRY_URL`, `OPEN_TELEMETRY_HEADERS`, `APP_VERSION`. `LOG_FILE_IS_ENABLED` keeps its rule: only an explicit `false` turns the file off. The paths in the data volume are derived from one data directory. `spec.IsDatabaseRequired` instead of parsing `os.Args` decides whether a connection string is needed: `--test-storage` does not need one. `spec.RootDirectory` and `spec.PublishedDatabaseDsnPath` let tests isolate the data directory, `.env` and the published file; empty values mean the production locations. The behavior is described in the `runtime-configuration` delta spec.

`Load` writes nothing to the log. It returns a report: the `.env` path, whether the file was loaded, whether the published connection string was used, and an error reading the published file. `LoadEnvironment` logs the report through the logger it built. When `Load` fails, `LoadEnvironment` logs the error through a stdout-only logger and returns it, and `cmd` exits with code 1. This keeps the messages the `runtime-configuration` spec requires.

The client tool check (`tools.LogAndExitIfClientToolsBroken`) moves from the configuration loading to `Serve`. Console commands do not need it.

*Rejected: `cleanenv`, as now.* It reads only the process environment. To make it see `.env` and the published connection string, they would have to be written into the environment with `os.Setenv`, and they would leak into the started tools.

*Rejected: a hand-written reflection parser.* That is one more piece of code with tests for what the library already does.

*Rejected: loading the configuration in `cmd`.* Then `cmd` would become a second composition root, and `docs/architecture.md` would have to change.

### 6. Logs: the code moves to `lib/logging`, `util/logger` stays as a bridge

These move to `lib/logging` without changing behavior:

- the fan-out handler;
- the stdout, file and OTLP sinks;
- the masking;
- the request attributes from `request_context.go`.

`lib/logging` takes its settings from `Config` instead of reading the environment itself. `logging.New(ctx, config)` returns an error for a malformed log export setting instead of exiting the process. `util/logger` keeps thin functions that forward the call to `lib/logging` or return the installed logging: `Install`, `GetLogger`, `GetAuditLogger`, `ExitAfterFlush`, `FlushAndCloseSinks`, `Shutdown`, `Redact`, `ContextWithRequestID`, `GetRequestID`, `ContextWithUserID` and `GetUserID`. The code that calls them and its tests do not change.

A shared `logging.With(ctx, ...)` for operation attributes does not appear in this change. Its first consumer is `job_name` and `job_id` from `Runner` in milestone 4.

*Rejected: rewriting the `util/logger` calls to a logger from `Deps`.* These calls live in old code, and moving the modules rewrites them anyway.

### 7. Migrations are embedded and applied by the goose library

The `backend/migrations` package embeds the SQL files through `embed.FS`. `lib/database.Migrate(ctx, db, files)` calls goose as a library (`github.com/pressly/goose/v3` at the version pinned in `mise.toml`) with out-of-order allowed and a PostgreSQL session lock held while it applies. The lock covers a console command and the server migrating one database at once. The file format and the `goose create` command for developers do not change. Each migration runs in its own transaction, as before. That is now a rule: a new `archlint` rule rejects a migration with `-- +goose NO TRANSACTION`, and `docs/architecture.md` states it in "Migrations". Without a transaction a failed migration would leave the database half changed, and the next start would trip over its own objects. `CREATE INDEX CONCURRENTLY`, the usual reason to drop the transaction, gains nothing here: migrations run before the server accepts requests, so nothing waits for the table lock. The behavior is described in the `internal-database-migrations` delta spec.

In this change `lib/database` holds only `Open` and `Migrate`. `Conn(ctx, db)` and `Transaction(ctx, fn)` appear in milestone 4 with the first repositories.

The external `goose` leaves the runtime image, and so does the copy of the `migrations` directory, which only the external `goose` read. The test slot databases are still migrated by the external `goose`: `scripts/test.sh` takes it from `mise.toml` (3.27.3), and CI installs its own copy with `go install` (3.27.1, `.github/workflows/ci-release.yml:436`). That is test infrastructure that `testdb` replaces in milestone 3, so this change does not touch it.

*Rejected: keep the external binary.* The image would keep building goose from source. The migrations would depend on the `./migrations` working directory, and the out-of-order rule from `docs/architecture.md:169` would stay unmet.

### 8. `app` is split into loading, core, server and console

- `app.LoadEnvironment(ctx, spec)` loads the configuration and builds the logging, see decision 5. It installs the logging into the bridge, so old code that logs works from then on.
- `app.New(ctx, environment)` runs the core steps in order:
  1. opens the database and applies the migrations;
  2. creates the data and temp directories;
  3. fills the configuration and database bridges;
  4. moves the key from the database to its file, which reads the database through the bridge;
  5. calls the `SetupDependencies` functions from the one file `internal/app/wiring.go`, which keeps the order of `cmd/main.go:420`.

  A failed migration returns before step 3, so the bridges stay unfilled.
- `(*App).Serve(ctx, address)` checks the client tools and cleans the temp directory. Then it opens the port, registers the routes, starts the background loops and blocks until `ctx` ends. `cmd` passes the existing address `:4005`; tests pass a free port.
- `(*App).RunConsoleCommand(ctx, command)` runs a console command and returns the text to print or an error. It does not touch routes or background loops; they exist only inside `Serve`.
- `app.ProbeStorage(ctx, environment)` runs the storage probe without `New`, that is, without the database. It installs only the configuration bridge, which local storage reads its paths from. A failure returns `StorageProbeError`, whose text is the message the operator saw before.

`cmd/databasus` only parses arguments, prints the result and picks the exit code. It refuses incompatible flags before calling `app`. The order of the flags stays: `--create-user` is checked before `--new-password`, so a mistyped flag cannot reset someone else's password. The telemetry notice is printed by `cmd` before `Serve`, because `app` does not print. `healthcheck` stays in `cmd`, because it is just an HTTP client. The other `backend/cmd/*.go` files are deleted; `cmd/seed_dev` and `cmd/cleanup_test_db` stay.

*Rejected: console commands in `cmd`, as now.* Then `cmd` would import feature services directly, bypassing `app`, and each command would have to know which `SetupDependencies` it needs. That is how it works now: `cmd/main.go:263`, `295`, `321`. A forgotten call there makes the command silently skip the audit record, which the `root-admin-account` and `two-factor-authentication` specs require.

### 9. `SetupDependencies` stays in the features

`wiring.go` calls all 14 functions in one explicit order, and the console commands get the same wiring as the server. The functions stay `sync.OnceFunc` in their packages, so the 85 calls from tests and test helpers work as before.

*Rejected: moving their bodies to `app`.* A package's internal tests cannot import `app`. The bodies disappear when a module moves and its listeners become events.

### 10. Background loops run through `background.Group`, and shutdown waits for them

`lib/background.Group` has two methods:

- `Go(ctx, name, fn)` runs the function in a goroutine, turns a panic into an error logged with a `stack` field, and counts the goroutine;
- `Wait(ctx)` waits until every goroutine has returned or `ctx` ends.

`app.Serve` starts the 13 loops and the HTTP server through the group. The group belongs to `Serve`, so `RunConsoleCommand` cannot start a loop.

*Rejected: moving the loops to `Runner` with passes.* That rewrites 13 loops and their tests. The WAL supervisor does not fit the pass model at all. `Runner` arrives in milestone 4.

### 11. HTTP: `lib/httpserver` takes only the shared parts, `internal/middleware` stays old code

`lib/httpserver` holds:

- the server with the timeouts from `cmd/main.go:353`;
- a synchronous `Listen`, after which `Serve` starts;
- the panic recovery in handlers (`ginRecoveryWithLogger` from `cmd/main.go:596`);
- gzip and CORS for development mode;
- the unknown path handler: a JSON 404 under `/api/` and the frontend page for the rest.

`internal/middleware` (request id, access log, `no-store`) stays where it is. `app` attaches it the way `main` does now.

The port opens before the background loops start. When it is busy, `Serve` returns an error and the process exits with code 1. Now the process keeps running without HTTP in that case, and the container looks alive but does not answer.

*Rejected: moving `internal/middleware` to `lib/httpserver` whole.* The access log imports `features/verification/agents`, and `lib` knows nothing of domains. The middleware moves in milestone 4, when the actor attributes reach the log through `logging.With`. This is recorded in `temp/refactor/plan.md`.

### 12. Shutdown: `lib/shutdown` knows only steps, `app` knows what to wait for

`lib/shutdown` gives two functions:

- `ListenForStopSignal(ctx)` registers the only `SIGTERM` and `SIGINT` handler and returns a context that ends on the signal. A second signal during shutdown does nothing.
- `Drain(ctx, DrainSpec{Steps, DrainTimeout, FlushLogs, FlushTimeout, Logger})` runs `func(ctx) error` steps in parallel within the limit. A step that has not finished by then is left behind. Then it flushes the logs within its own limit.

The drain context is built from `context.WithoutCancel` of the context `app` passed, so `lib` does not call `context.Background`. `lib/shutdown` does not know the task registry. `app` passes the steps itself:

1. `server.Shutdown`;
2. waiting for the task registry of `features/tasks/cancellation`;
3. `background.Group.Wait`.

Before handing over the steps, `app` closes `config.BeginShutdown` and cancels the loops' context. The limits stay: 6 seconds for the drain, 2 seconds for the logs.

`config.BeginShutdown` and `config.GetShutdownSignal` stay part of the bridge, because logical backups (`create_backup_uc.go:137` and three similar ones) and restores read this channel. `cmd/shutdown.go` lives until task 7.4, where it is deleted together with `main.go`.

`docs/architecture.md:238` says that a second signal exits the process at once. The `application-shutdown` spec requires the opposite: a second signal neither kills the process nor starts the shutdown again. The code follows the spec, and this change fixes the sentence in `docs/architecture.md`.

*Rejected: moving `cmd/shutdown.go` to `lib` as it is.* It holds `*task_cancellation.Registry` from an old feature and calls `context.Background()`. The first breaks the `lib` boundary, `forbidigo` forbids the second.

### 13. swag is removed entirely

The server mounts neither the Swagger UI nor the API description in any mode. Removed:

- the generation at startup (`cmd/main.go:524`) and the import `_ "databasus-backend/swagger"`;
- the swag installation (`Dockerfile:37`) and the `swag init` step with the comment above it (`Dockerfile:52-54`), the swag steps in CI (`.github/workflows/ci-release.yml:59-65`) and in `scripts/dev.sh:58`;
- the generated `backend/swagger` directory and its lines in `backend/.gitignore`;
- the dependencies `swaggo/swag`, `swaggo/gin-swagger`, `swaggo/files` and the swag version in `mise.toml`;
- all `// @...` annotations (about 1200 lines in 26 controllers) together with the `// FunctionName` line that opens each such block and carries nothing but the name.

The swag description is not kept in git, is built from comments the compiler does not check, and describes neither errors nor authentication. It already has protected routes without the mark and a reference to a nonexistent `BearerAuth` scheme. Neither the frontend nor the verification agent uses it. The API description will appear in the contract `backend/api/openapi.yaml`, from which the server, the clients and the types are generated. Milestone 3 builds the pipeline on one service operation, and each module's operations enter the contract when the module moves in milestones 4-13. Until then developers have no API description; that is the price of the decision.

A user who opened `/api/v1/docs/swagger/index.html` on their instance loses that page. The proposal records this as BREAKING.

*Rejected: keep the Swagger UI in development mode only.* Then `air` would have to run `swag init` before every build, and the annotations would need upkeep for eleven more milestones for a document developers do not trust. Each module move removes them anyway.

*Rejected: removing swag in milestone 13, as the plan said.* The skeleton rewrites `main.go`, the swag entry point, and the build in the `Dockerfile` and CI anyway. Moving the swag entry point to another file would be work thrown away later.

### 14. The binary keeps its name in the image

The Dockerfile builds `./cmd/databasus` into the existing `/app/main`. The `databasus` link, `docker/start.sh:351`, `docker/start.sh:671` and the `./main` calls in the READMEs do not change, as the `host-console-commands` spec requires.

*Rejected: renaming it to `/app/databasus`.* The startup script, the image healthcheck and six READMEs would have to change, and the milestone gains nothing from it.

## Risks / Trade-offs

- [Lazy objects deadlock on a cycle instead of failing to compile] → The test from decision 4 calls every getter. A cycle shows up on the first run.
- [A getter the bridge did not fill panics in production on the first request, not at startup] → The test from decision 4 checks that every getter answers after `app.New`. The binary test checks that nothing is called at import time.
- [Shutdown now waits for the background loops, and a slow loop eats the drain limit] → The limit is shared and does not grow. The process exits after 8 seconds, as before.
- [Out-of-order applies a migration that silently relied on the order] → `docs/architecture.md:169` already forbids migrations to depend on their order within a release. The migration package test applies a missed migration separately.
- [The test build hides a dependency not handed to the bridge] → The binary test builds without the tag and so sees the production behavior.
- [A console command and the server in one container apply migrations at once] → `lib/database.Migrate` holds a PostgreSQL session lock while it migrates, so the second process waits for the first. The test of task 3.2 checks this.
- [Values from `.env` no longer reach the process environment, so old code that reads a variable directly stops seeing it in development] → In production code only `internal/features/system/version/controller.go:32` (`APP_VERSION`) reads this way. In the image the image itself sets that variable, not `.env`.

## Migration Plan

The schema does not change. On a running instance the embedded migrations find every version already applied and do nothing. Rolling back to the previous image works: the previous version sees the same goose version table. The only difference is that the image has no `goose` binary, which only the server itself used at startup.
