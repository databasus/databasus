# Architecture

This document states how Databasus is built and why. The architecture review hook checks every plan and every diff against it. Terms follow the [glossary](glossary.md), and [testing.md](testing.md) describes the test setup. Product behavior lives in the OpenSpec specs under `openspec/specs`. Rules that a linter or `archlint` enforces are listed in [Automatic checks](#automatic-checks).

Go code under `backend/` and `agent/` outside the layout block in [Backend layout](#backend-layout) predates this document and is being moved into that layout. New code follows this document. A change to the old code keeps that code's existing structure, unless the change is the one that moves it.

## System

Databasus backs up and restores the user's PostgreSQL, MySQL, MariaDB and MongoDB databases into storages the user picks, and reports results through notifiers.

```
            browser (SPA)                         verification agent (user's VM)
                 │ /api/v1                                  │ agent API
                 ▼                                          ▼
 ┌──────────────── Databasus container ────────────────────────────┐
 │  backend process: HTTP API, background jobs, static SPA          │
 │  embedded PostgreSQL 17: the internal database                   │
 │  client tools of every supported engine version (assets/)        │
 └───────┬───────────────────────────┬──────────────────┬───────────┘
         ▼                           ▼                  ▼
  user databases               storages            notifiers
  (dump, replication)   (S3, local, SFTP, ...)  (email, Slack, webhook, ...)
```

- **One image, one container.** The image holds the backend, the built SPA, an embedded PostgreSQL and the client tools. Installing is one `docker run`, and upgrading is pulling a newer image. We considered two alternatives. A separate database service made users manage two images and two volumes, and they never upgraded the database. A single static binary would have made users install a dozen client tool versions themselves.
- **The internal database is PostgreSQL, not SQLite.** It is never external, and the image pins its version.
- **Client tools are vendored** under `assets/`, per engine, major version and architecture. The image copies them in instead of installing them from package repositories. The build stays reproducible, works offline and does not depend on a third-party mirror. Adding and refreshing the binaries follows the `client-tool-bundles` skill.
- **One process per installation.** The supplied deployments start exactly one backend process (spec `single-instance-runtime`). That process owns all in-memory state: the cache, rate-limit counters, task cancellation and the WAL streamers. There is no separate worker process and no shared cache service.
- **Backups run remotely.** Databasus connects to the user's database over the network or through an SSH tunnel to a jump host. It runs the dump or replication tools from its own container, and nothing of ours is installed on the database host. Managed databases forbid host installs, and an agent on the database host would double the release and test surface. We tried one and removed it.

## Backend layout

```
backend/
  api/                         openapi.yaml, agent.yaml and the package that embeds them
  cmd/databasus/               the server and its console commands
  cmd/verification-agent/      the verification agent binary
  internal/app/                composition root: config, database, modules, routes, jobs, event wiring, shutdown
  internal/modules/<name>/     domain modules
  internal/engines/<engine>/   database engine drivers
  internal/agent/              the verification agent's code
  internal/lib/<name>/         shared code without domain logic
  migrations/                  goose SQL migrations, embedded in the binary
  e2e/upgrade/                 the upgrade test and its comparison helper upgradecheck
  tools/archlint/              the architecture linter
```

Dependencies point one way: `cmd → app → modules → engines → lib`. `internal/app` is imported only by `cmd`. `lib` knows nothing of engines, modules or `app`. `engines` know nothing of modules, tables or HTTP. The agent imports only `lib`, `engines` and `lib/agentclient`, the agent API client generated from `agent.yaml`; modules and `app` never import the agent.

`e2e/upgrade` holds only test files and drives the images of two releases from outside. It imports only `lib`, `api` and its own `upgradecheck`, and nothing imports it, so none of it reaches the production binary.

No package-level variable holds a dependency. Each module and each `lib` service gets its dependencies through `Deps` in `New(deps Deps)`, and `internal/app` builds the whole graph explicitly in one place. There are no `init()` functions and no work at import time. `app` reads the configuration at startup, so it does not depend on which package happens to be imported first.

## Modules

A module owns one domain area and its tables.

```
internal/modules/<name>/
  module.go     the public contract
  internal/     handlers, services, repositories, models, split into subpackages by meaning
```

`module.go` is the only non-test file of the module's root package. It declares everything other modules may use:

- the `Module` interface and any other interfaces offered to other modules;
- the types those interfaces take and return;
- the events the module publishes;
- `Deps` and `New`.

Go's `internal/` rule makes the rest of the module unreachable from outside.

| Module | Owns |
|---|---|
| `users` | accounts, sign-in (password, GitHub and Google OAuth, email codes), password reset, instance settings, global roles, the request's `Actor`, the instance admin area including the global audit log |
| `workspaces` | workspaces, memberships, workspace roles, invitations, `Authorize`, the workspace audit log |
| `notifiers` | notification channels and sending a notification |
| `storages` | storages, the file lifecycle (write receipts, pending deletions and the deletion worker) |
| `databases` | connections to user databases, connection tests, read-only and replication users, the last backup time and health status shown in the list |
| `backups` | backup configs, logical backups, physical backups and WAL, restores, downloads and restore streams, retention |
| `verification` | verification agents, verification configs and verification runs |
| `healthcheck` | scheduled availability checks of user databases |
| `telemetry` | anonymous usage statistics |

The dependency graph between modules has no cycles. From the lowest module to the highest, the order is: `users`; `workspaces`; `notifiers` and `storages`; `databases`; `backups`; `verification` and `healthcheck`; `telemetry`. A module imports only modules lower in this order.

How modules talk:

- Only through the interfaces in `module.go`. Data of another module is read through its methods. To read a set of records, the consumer calls a batch method with their IDs and joins the results in Go, instead of calling a method once per record in a loop.
- The request's user is an `Actor` from `users`: the user ID and the global role. A module never receives a `users` model.
- Only `workspaces.Authorize(ctx, actor, workspaceID, action)` checks access to a workspace and to anything inside it. A module first looks the object up in its own table and answers 404 when it is missing. Then it calls `Authorize`, which answers 404 when the actor is not a member of the workspace and 403 when the actor's role does not allow the action.
- A lower module that needs something done by higher ones publishes an event declared in its `module.go`, for example `workspaces.WorkspaceDeleting`. The event's handlers run synchronously, in the publisher's transaction. They run in the order in which `internal/app` registers them in its single wiring file, and nothing else sets that order. Effects outside the database, such as cancelling a running process or dropping a replication slot, are registered as after-commit handlers.
- If two areas keep needing a SQL JOIN of each other's tables, the boundary is wrong and they belong in one module.

Inside a module's subpackage the code has three roles:

- **Handler** decodes the request through the generated strict server, calls one service method and encodes the response. It never touches the database or the service's dependencies.
- **Service** holds the rules: validation, `Authorize`, transactions, audit events, notifications. It writes no SQL.
- **Repository** is an unexported type that holds `*gorm.DB` and runs every query of its tables through `database.Conn(ctx, r.db)`.

Files follow the role: `handler.go` or `*_handler.go`, `service.go` or `*_service.go`, `repository.go` or `*_repository.go`, a model in `<entity>.go`, all errors of the package in `errors.go`. An enum lives next to the entity whose field it types. A long-running loop gets its own file.

## Engines and providers

Databasus has three families of interchangeable implementations. Each family has a registry keyed by type, and code outside the family never switches on the type.

- **Engines** live in `internal/engines/<engine>`: `postgresql`, `mysql`, `mariadb` and `mongodb`. `postgresql` is split into `logical`, `physical` and `shared`, and `mysqlfamily` holds what MySQL and MariaDB share.

  An engine knows how to:

  - connect and test a connection;
  - inspect privileges and provision a read-only or replication user;
  - dump into an `io.Writer` and restore from an `io.Reader`;
  - find its client tool for a server version.

  It exposes these as small capability interfaces, because a physical PostgreSQL source and a logical one support different sets. The backend's restore and the verification agent call the same engine code, so a verification restores exactly the way a restore does.
- **Storage providers** (`storages/internal/providers/<name>`) save, read and delete files and test a connection.
- **Notifier providers** (`notifiers/internal/providers/<name>`) send a notification and test a channel.

Each provider owns its behavior, its repository and one table with its settings as explicit columns. The common fields and the type sit in a parent row, in `storages`, `notifiers` or `databases`. Every other table references the parent, never a provider table. The parent repository saves the parent row and, in the same transaction, hands the provider's settings to that provider's repository. A list loads the parents first, then the provider rows in one batch per type.

Adding a provider takes a package, a migration with its table, a registry entry and the contract schema. Nothing else changes.

Settings stay in columns rather than in a JSON document for two reasons. The database keeps enforcing `NOT NULL`, `CHECK` and types. And when a field is added, existing rows get its value from `DEFAULT` in the migration instead of a zero value in Go.

## Shared code

`internal/lib` holds code without domain logic:

| Package | Holds |
|---|---|
| `database` | the database pool and transactions |
| `config`, `logging` | configuration, logging |
| `httpserver` | the HTTP server and its middleware |
| `apierror` | `Problem` errors |
| `background`, `shutdown` | background jobs, cancellation and shutdown |
| `process` | external processes |
| `clock` | the clock |
| `crypto` | the master key, field encryption and stream encryption |
| `audit` | audit events |
| `mailer` | email |
| `cache`, `ratelimit` | the in-memory cache and rate limiter |
| `schedule` | schedules |
| `sshtunnel` | SSH tunnels |
| `diskusage` | disk usage |
| `iox` | stream helpers |

Test support lives in `lib` too: `testdb`, `testwait`, the contract-checking test clients `apiclient` and `agentclienttest`, and `testcontainers` with the containers of every engine and storage.

Audit events are written through `lib/audit` in the transaction of the operation they describe. Each event stores a snapshot of its actor's ID and email. The event stays readable after the user is deleted, and reading it needs no join with `users`. `users` serves the global audit log and `workspaces` serves a workspace's audit log, because each of them owns the access rule for its view.

## Data

- **Ownership.** Each module reads and writes only its own tables, through its repositories. A table is named after its module. The module's main table carries the module name itself (`storages`), and every other table starts with it (`storages_s3`, `backups_logical`). A `lib` package may own a table named `lib_<name>_*`, such as `lib_audit_events`, and modules use that table only through the package.
- **References between modules.** A foreign key to another module's table is declared `ON DELETE RESTRICT`. `CASCADE` is allowed only between tables of one module. The reason is files: a backup row describes a file in a storage, and a cascade would delete the row without asking for the file's deletion. With `RESTRICT`, a missing cleanup fails loudly instead of leaving orphan files. The database itself also refuses to delete a storage, notifier or database that is still referenced, and the service maps that refusal to 409 `in_use`.
- **No SQL JOIN with another module's tables.** Data of another module comes through its interface.
- **Invariants live in the schema** when the schema can express them:
  - `NOT NULL` and `DEFAULT`;
  - `CHECK` on every enum column;
  - `UNIQUE` and partial unique indexes for "at most one" rules;
  - a composite foreign key `(id, type)`, so a provider row cannot attach to a parent of another type;
  - for a row that references one of two kinds of entities, two nullable foreign keys and `CHECK (num_nonnulls(a_id, b_id) = 1)`, never an untyped `(kind, id)` pair.
- **Types.** Lists are `text[]`. JSON is stored as `JSONB`, and only for data we treat as opaque, such as an OAuth token or webhook headers. Time is `TIMESTAMPTZ`. All time is UTC: in Go, in the database session, in the API and in logs. Identifiers are UUIDs with `DEFAULT gen_random_uuid()`. A row that continues another row of the same module takes that row's ID as its primary key.
- **Transactions travel in the context.** A service opens one with `Transaction(ctx, fn)` of the `*database.Transactor` it takes through `Deps`, and every repository call made with that context, in any module, joins it, because a repository runs its queries through `database.Conn(ctx, r.db)`. Transactions are never passed as parameters. The transactor is built in `internal/app` from the pool, so a service never holds the pool itself.
- **Repository methods** are named after what their SQL does. When a lookup finds no row, or a conditional `UPDATE` touches none, the method returns `false` as its second value, as in `(Backup, bool, error)`. The service decides which error the client sees. A repository wraps a database error with the operation's name and never returns `gorm.ErrRecordNotFound`.
- **One column changes with one `UPDATE`.** A status, a timestamp or a counter is never written by re-saving the whole row.
- **Secrets** of providers and databases are encrypted per field with the master key before they are saved, and decrypted only where they are used. The cipher is AES-GCM, and an encrypted value starts with `enc:`. Account passwords and agent tokens are hashed, not encrypted.

### Migrations

- The schema changes only through goose SQL migrations in `backend/migrations`, embedded in the binary and applied at startup. `AutoMigrate` is never used.
- Migrations are applied with out-of-order allowed, so a migration merged after a later-numbered one still runs. A migration therefore never depends on the order of migrations within a release.
- Every migration runs whole in one transaction, so a migration that fails leaves no partial changes behind and the next start applies it again. `-- +goose NO TRANSACTION` is not allowed. `CREATE INDEX CONCURRENTLY`, the usual reason to drop the transaction, gains nothing here: migrations run before the server accepts requests, so nothing waits for the table lock.
- Every migration runs on any older valid state: a fresh install, an install upgraded through every release, and an internal database restored from an older Databasus dump. A migration that adds or tightens a constraint first repairs the rows that would violate it. Otherwise the upgraded instance does not start. A migration that moves encrypted values copies them as they are. SQL cannot decrypt them and does not need to.
- An upgrade is one-way. Migrations only go forward, and a binary refuses to start on a schema newer than its own: it compares the newest migration recorded in the database with the newest one it carries and stops before applying anything. An older binary on a renamed table would otherwise fail at the first request instead of at startup. A recorded migration the binary does not carry but that is older than its newest one does not stop it, because releases have shipped migrations that later releases deleted or renamed.
- The upgrade test from [testing.md](testing.md) runs every migration against data written by the previous release.
- Format: one statement per table, constraint and index, column types aligned, each constraint clause on its own line.

## Stored files

Backup code never calls a storage provider directly. Every write and delete goes through the file store of the `storages` module.

- `WriteFile` records a pending deletion before the first byte leaves and returns a write receipt. The transaction that publishes the backup spends the receipt, which keeps the file. Any other outcome leaves the pending deletion in place, and the deletion worker removes the file. That covers a provider error, a crash, a rolled-back publication and a cancellation. A terminal failure asks for the deletion in the same transaction that records the failure.
- Every write attempt gets its own object name, because a retry that reused a name would address an object a pending deletion already owns.
- A provider's delete removes every object it derives from the name, such as chunk objects, part manifests and incomplete multipart uploads. Only a confirmed absence counts as success.

## Backup data flow

Backup and restore data streams in bounded chunks with backpressure. The producer, such as a dump tool, `pg_basebackup` or a storage download, cannot run ahead of the consumer by more than a small window. Memory per operation does not depend on the database size, and a slow storage slows the backup instead of exhausting the host's memory.

Backup data does not touch the host's disk, with two exceptions:

- `pg_restore -j` needs a real file, so the restore first checks that the disk has room for it.
- `pg_receivewal` writes WAL segments into a local queue before they are uploaded.

Backups are encrypted per file. The key for each file is derived from the master key and the file's ID with PBKDF2-SHA256, and data is encrypted with AES-GCM in chunks.

## Physical backups

Physical backups of PostgreSQL use the native PostgreSQL 17 stack:

- `pg_basebackup` for full backups;
- `pg_basebackup --incremental` against the parent's manifest for incremental ones;
- `pg_receivewal` for the WAL stream;
- `pg_combinebackup` on restore.

PostgreSQL owns the backup format, and we do not implement block-level incrementals. Several product rules follow from this choice: PostgreSQL 17 and later only, WAL summarization required for incrementals, no custom tablespaces, and only completed WAL segments uploaded. The specs under `openspec/specs/physical-backups` state them.

- `pg_basebackup` runs with `--pgdata=- --format=tar --wal-method=fetch --compress=server-zstd:5 --no-manifest`. A single tar on stdout lets the artifact stream straight into storage. `fetch` keeps the WAL inside that one stream, and a per-backup replication slot pins the WAL until the backup ends.
- Server-side compression sends about a third of the bytes over the source link. For a terabyte database over a WAN, that turns a multi-day backup into hours. Client-side compression and local staging were rejected for exactly that reason. When the source was built without zstd, the codec falls back to gzip and then to none. The codec that worked is remembered per database.
- With server compression, `pg_basebackup` cannot embed `backup_manifest`. Databasus therefore rebuilds the manifest from the tar stream and stores it as a separate encrypted sidecar file, which incrementals pass to `--incremental`. The tests check every rebuilt manifest with `pg_verifybackup` from PostgreSQL 18, which can read a compressed tar. Any drift from PostgreSQL's format then fails in CI. Speaking the replication protocol directly would avoid the rebuild, but it would mean reimplementing and maintaining `pg_basebackup` ourselves.
- The physical backup catalog is the only code that changes the status of a physical backup. Each change is one of the transitions `start`, `complete`, `fail`, `cancel` and `markChainBroken`, executed as a conditional `UPDATE ... WHERE status = 'IN_PROGRESS'`. The scheduler, the backuper and the canceller ask the catalog for a transition and never write the row themselves.
- Full and incremental backups share one table with `kind` and `parent_id`. A `CHECK` requires a parent exactly for incrementals, and the parent reference is `RESTRICT`.
- The scheduler's choice between a full backup, an incremental and nothing is a pure function of the current state, the config and the time, tested on its own. Only the code around it reads and writes the database.
- The health of a WAL stream is computed in one place from stored samples. The Monitoring tab, the API and the notifications all read that one verdict.
- The user restores a physical backup from a downloaded restore set with the recovery script Databasus serves.

## HTTP API

- The contract comes before the code. `backend/api/openapi.yaml` describes the API that the SPA and users' scripts call, with paths relative to `/api/v1`. `backend/api/agent.yaml` describes what the verification agent calls. A new or changed operation starts in the contract. Then `mise run generate-api` regenerates the servers, the test clients and the frontend types. Generated files (`*.gen.go`, `*.gen.ts`) are never edited by hand. The rules for designing an operation are in the `api-design` skill.
- Each module implements, in `internal/modules/<name>/internal/httpapi`, the strict server generated for its own operations. Operations are assigned to modules by tag. The `system` tag is the one exception: its operations describe the running instance rather than a module's domain, so `internal/app/systemhttpapi` serves it. A route registered in the router without an operation in the contract answers 500, so a mounting mistake cannot expose an operation without its access checks.
- Errors use the `Problem` format (RFC 9457) with a machine-readable `code`. Every operation declares a `default` response with it. The text of an internal error never reaches the client. The frontend shows a translation chosen by `code`.
- A request is authenticated by the `users` middleware, which puts the `Actor` into the request context. Operations of the instance admin area are marked in the contract, and the middleware enforces the mark. Workspace access is checked by `Authorize`, as described in [Modules](#modules).
- Lists are paged with a cursor through `limit` and `after`. `OFFSET` is not used.
- A long operation, such as a backup, a restore or a verification, answers 202 with the resource in its in-progress state.
- A path under `/api` that matches no route answers 404. Every other path serves the SPA.
- A release may break backward compatibility of the contract, as long as the same change updates the frontend. Two paths never change: `/api/v1/system/version` and `/api/v1/system/verification-agent`. Verification agents that users have already installed update themselves through these paths. A running agent keeps calling the agent API of its own release until its next update check, so operations in `agent.yaml` stay compatible with the agent of the previous release.

## Long-running work

- **Background jobs** run through `background.Runner`. Each run of a job is a pass. The runner puts `job_name` and `job_id` into the context of each pass and never overlaps two passes of one job. When a pass returns an error or panics, the runner logs it and still runs the next pass on schedule. A second `Start` panics, because it is always a bug. The runner reports each job's last result to the health endpoint.
- **External tools** (`pg_dump`, `pg_basebackup`, `pg_receivewal`, `mysqldump` and others) run through the `lib/process` helper. The helper binds the process to the operation's context and kills the whole process group when the context ends.
- **Goroutines** are started only through `lib` helpers that tie them to a context and recover a panic into an error. In production code a bare `go` statement exists only inside `lib`; a test may start one directly to drive concurrent calls.
- **Cancellation.** A running backup, restore or verification is registered under its ID, and a cancel request ends its context.
- **Shutdown.** SIGTERM or SIGINT cancels the root context, and then:
  - the HTTP server stops accepting connections and waits for started requests;
  - the runner starts no new passes;
  - running backups and restores end with the status `FAILED` and the reason that Databasus was stopping, and their files are handed to the deletion worker;
  - WAL streamers stop and keep their replication slots, so the next start resumes them;
  - the process flushes the logs and exits before the embedded PostgreSQL stops (spec `container-shutdown`).

  A second signal during the shutdown neither kills the process nor starts the shutdown again (spec `application-shutdown`). Work still marked in progress at the next start is failed the same way.
- **Time** that drives behavior, such as schedules, deadlines and retention, comes from a clock in `Deps`. SQL receives the current time as a parameter instead of calling `now()`.

## Configuration

Settings come from environment variables, with `.env` as an optional local source. `app` parses them once at startup into a typed structure through `lib/config`. When a variable is missing or invalid, `app` refuses to start with a message that names it. Each module receives only the values it needs through `Deps`. Paths inside the data volume, such as backups, temp files and the master key, are derived from one data directory.

## Logs

- The backend logs through `log/slog` from `lib/logging`. Every record goes to stdout and to a rotating JSON file in the data volume. When an OTLP endpoint is configured, records also go to an OpenTelemetry collector, because OTLP is the one protocol every log backend speaks. When the collector is down, its records are dropped instead of blocking requests. The file is the forensic copy that survives a lost database, so audit events go to it as well.
- The first middleware gives every request a fresh `request_id` and returns it as `X-Request-Id`. An incoming `X-Request-Id` is ignored, so a client cannot stitch its actions into another trace.
- Attributes of an operation travel in the context: `request_id` from the middleware, `job_name` and `job_id` from the runner, and entity IDs added with `logging.With`. The log handler adds them to every record.
- A message is a constant lowercase phrase. Values go into attributes with `snake_case` keys, never into the message.
- Before any sink sees a record, the log handler masks the values of attributes whose names mark a secret (password, token, secret, key, authorization, cookie), secrets that match known patterns inside messages, such as a password in a connection string, and email addresses. Audit records identify the actor by user ID, so masking emails does not cost the file its forensic value.

## Verification agent

The verification agent is a CLI worker that runs on a machine the user provides. It sends capacity heartbeats, claims verification jobs, restores each backup into a disposable PostgreSQL container through the Docker or Podman API and reports the result. It has no HTTP server and no database schema. The agent ships inside the Databasus image and is downloaded from `/api/v1/system/verification-agent`. It replaces itself with the version of the backend it is connected to. Because both are built from one commit, the agent and the backend always share the contract and the engine code.

## Automatic checks

| Rule | Checked by |
|---|---|
| Layer direction, the module boundary through the root package, `module.go` as the only root file, the agent's allowed imports | `archlint` |
| File roles, errors in `errors.go`, a bare `go` statement only in `lib` | `archlint` |
| No `testing`, testify or testcontainers in the production binary | `archlint` (dependency graph of `cmd/databasus`) |
| A migration adds no `CASCADE` foreign key to another module's table | `archlint` (table prefixes) |
| A migration runs in its transaction: no `-- +goose NO TRANSACTION` | `archlint` |
| `*gorm.DB` and `database.Conn` only in repositories; no `time.Sleep`, `Eventually`, `fmt.Print*`, `os.Exit`, `os.Getenv` or `context.Background()` outside the places allowed | `forbidigo` |
| Log calls: attributes only, `...Context` methods, constant messages, `snake_case` keys | `sloglint` |
| Imports of test libraries, Gin and GORM only where allowed; fixed import aliases | `depguard`, `importas` |
| No package-level dependency variables, no `init()`; every `Deps` literal lists all fields | `gochecknoglobals`, `gochecknoinits`, `exhaustruct` |
| Exhaustive `switch` on enums | `exhaustive` |
| The contract follows the API rules; generated code is up to date | `vacuum`; `backend:check-generated`, which `backend:lint` runs, and `check:api`, which `frontend:lint` runs |

`mise run lint` runs all of them; `mise run backend:lint` runs all but the freshness check of the frontend types. The linters check test files too. The `archlint` rules on imports, the module root, file roles, errors and bare `go` check production files only: a module's API tests build the whole application through `apptest`, which sits above the modules, and tests start goroutines of their own to drive concurrent calls.

`gochecknoglobals` arrives once no package-level dependencies remain. Until a module moves into the layout above, its code keeps the golangci-lint checks it had before these rules, and its test files are not linted; `backend/.golangci.yml` names the target layout. Only the production-binary rule, the testhelpers build tag rule and the rule against direct provider writes in the old backup code of `archlint` reach that code.
