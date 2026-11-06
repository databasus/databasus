# Architecture

Databasus backs up and restores the user's PostgreSQL, MySQL, MariaDB and MongoDB databases. It keeps the backups in storages the user picks, such as S3 or a local disk, and reports results through notifiers, such as Slack or email. Users run it themselves, as one Docker container.

A backup travels through the system like this. A schedule makes a background job start the backup. The backend connects to the user's database over the network or through an SSH tunnel, and runs the dump or replication tool from its own container. The data streams into the storage in small chunks. The backend then records the outcome and sends a notification. Later, a separate verification agent can restore the backup into a disposable container to prove that it works.

Three decisions shape the rest of this document: everything ships in one container, one backend process owns all in-memory state and nothing of ours runs on the database host.

This document explains the system's boundaries, data flow and the reasons behind them. Review checks plans and changes against these rules. Terms follow the [glossary](glossary.md), [testing.md](testing.md) describes the tests, and `openspec/specs` specifies product behavior, including its exact limits.

## System

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

- **One image, one container.** Installing is one `docker run`, and upgrading is pulling a newer image. A separate database service was rejected, because users managed two images and two volumes and never upgraded the database. A single static binary was rejected, because users would have to install a dozen client tool versions.
- **The internal database is PostgreSQL.** It always runs inside the container, and the image pins its version.
- **Client tools are vendored** under `assets/`, per engine, major version and architecture, so the build is reproducible and works offline. The `client-tool-bundles` skill covers adding and refreshing them.
- **One process per installation** (spec `single-instance-runtime`). It owns the cache, the rate-limit counters, task cancellation and the WAL streamers, so there is no worker process and no shared cache service.
- **Backups run remotely**, over the network or through an SSH tunnel to a jump host. Managed databases forbid host installs, and an agent on the host would double the release and test surface. We tried one and removed it.

## Backend layout

The backend is one Go module that builds the server and the verification agent. Its directories form layers, and dependencies between them point one way.

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
  e2e/upgrade/                 the upgrade test
```

- Dependencies point one way: `cmd → app → modules → engines → lib`. `internal/app` is imported only by `cmd`.
- `lib` knows nothing of engines, modules or `app`, and `engines` know nothing of modules, tables or HTTP.
- The agent imports only `lib`, `engines` and `lib/agentclient`, the client generated from `agent.yaml`. Modules and `app` never import the agent.
- `e2e/upgrade` holds only test files and imports only `lib`, `api` and `upgradecheck`. Nothing imports it, so none of it reaches the production binary.
- No package-level variable holds a dependency. Each module and `lib` service takes `Deps` in `New(deps Deps)`, and `internal/app` builds the whole graph in one place.
- There are no `init()` functions and no work at import time.

## Modules

A module owns one domain area and its tables, and the rest of the backend sees only its public contract. This keeps a domain change inside one directory.

```
internal/modules/<name>/
  module.go     the public contract
  internal/     handlers, services, repositories, models, split into subpackages by meaning
```

`module.go` is the only non-test file of the module's root package, and Go's `internal/` rule makes everything else unreachable from outside. It declares what other modules may use:

- the `Module` interface and any other interfaces offered to other modules;
- the types those interfaces take and return;
- the events the module publishes;
- `Deps` and `New`.

| Module         | Owns                                                                                                                                                                             |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `users`        | accounts, sign-in (password, GitHub and Google OAuth, email codes), password reset, instance settings, global roles, the request's `Actor`, the instance admin area              |
| `workspaces`   | workspaces, members, workspace roles, invitations, ownership transfer, `Authorize`, every audit log view: the global one, a user's and a workspace's                             |
| `notifiers`    | notifiers and sending a notification                                                                                                                                             |
| `storages`     | storages, the file lifecycle (write receipts, pending deletions and the deletion worker)                                                                                         |
| `databases`    | connections to user databases, the notifiers attached to a database, connection tests, read-only and replication users, the last backup time and health status shown in the list |
| `backups`      | backup configs, logical backups, physical backups and WAL, restores, downloads and restore streams, retention                                                                    |
| `verification` | verification agents, verification configs and verifications                                                                                                                      |
| `healthcheck`  | scheduled availability checks of user databases                                                                                                                                  |
| `telemetry`    | anonymous usage statistics                                                                                                                                                       |

The dependency graph between modules has no cycles. From lowest to highest, the order is: `users`; `workspaces`; `notifiers` and `storages`; `databases`; `backups`; `verification` and `healthcheck`; `telemetry`. A module imports only modules lower in this order.

### How modules talk

- Modules talk only through the interfaces in `module.go`.
- To read a set of records, the consumer calls a batch method with their IDs and joins the results in Go. It never calls a method per record in a loop.
- The request's user is an `Actor` from `users`: the user ID and the global role. A module never receives a `users` model.
- Only `workspaces.Authorize(ctx, actor, workspaceID, action)` checks access to a workspace and to anything inside it. A module first looks the object up in its own table and answers 404 when it is missing. `Authorize` then answers 404 for a non-member and 403 when the role does not allow the action.
- A lower module that needs something done by higher ones publishes an event declared in its `module.go`, for example `workspaces.WorkspaceDeleting`.
  - Handlers run synchronously, in the publisher's transaction, in the order that the single wiring file in `internal/app` registers them.
  - An effect outside the database, such as dropping a replication slot, is an after-commit handler. An effect that stays correct after a rollback, such as deleting files that already have a pending deletion, may run inside the handler.
- If two areas keep needing a SQL JOIN of each other's tables, they belong in one module.

### Roles inside a module

- **Handler** decodes the request through the generated strict server, calls one service method and encodes the response.
- **Service** holds the rules: validation, `Authorize`, transactions, audit events, notifications. It writes no SQL.
- **Repository** owns queries of the module's tables and joins the transaction from the caller's context.

[The backend rules](../backend/AGENTS.md#handlers-services-repositories) hold the details of each role, including what a repository returns when it finds no row.

## Data

Each module owns its part of the internal database. Integrity rules live in the schema where possible, so a bug in application code cannot break them.

- **Ownership.** Each module reads and writes only its own tables, through its repositories. Data of another module comes through its interface, never through a SQL JOIN.
- **Table names.** The module's main table carries the module name (`storages`), and every other table starts with it (`storages_s3`, `backups_logical`). A `lib` package may own a table named `lib_<name>_*`, such as `lib_audit_events`, and modules use it only through the package.
- **References between modules.** A foreign key to another module's table is `ON DELETE RESTRICT`, and `CASCADE` is allowed only inside one module.
  - A backup row describes a file in a storage, and a cascade would delete the row without asking for the file's deletion. With `RESTRICT`, a missing cleanup fails loudly and leaves no orphan files.
  - The database also refuses to delete a storage, notifier or database that is still referenced, and the service answers that refusal with 409 `in_use`.
- **Invariants live in the schema** when the schema can express them:
  - `NOT NULL` and `DEFAULT`;
  - `UNIQUE` and partial unique indexes for "at most one" rules;
  - a composite foreign key `(id, type)`, so a provider row cannot attach to a parent of another type;
  - for a row that references one of two kinds of entities, two nullable foreign keys and `CHECK (num_nonnulls(a_id, b_id) = 1)`, never an untyped `(kind, id)` pair.
- **Enum values** follow [the backend migration rules](../backend/AGENTS.md#migrations).
- **Types.** Lists are `text[]`. `JSONB` holds only data we treat as opaque and store in clear, such as webhook headers. Time is `TIMESTAMPTZ`, and all time is UTC. Identifiers are UUIDs with `DEFAULT gen_random_uuid()`, and a row that continues another row of the same module takes that row's ID as its primary key.
- **Transactions travel in the context** and are never passed as parameters.
  - A service opens one with `Transaction(ctx, fn)` of the `*database.Transactor` from `Deps`, so it never holds the pool itself.
  - Every repository call made with that context, in any module, joins the transaction through `database.Conn(ctx, r.db)`.
  - Background work that outlives its caller, such as a notification delivery, detaches with `database.WithoutTransaction(ctx)`.
  - An effect outside the database registers a hook with `database.AfterCommit(ctx, hook)`. The outermost transaction runs its hooks in order after it commits and drops them on rollback. The hooks get a context without the transaction.
- **Secrets** of providers and databases are encrypted per field with the master key and decrypted only where they are used. The cipher is AES-GCM, and an encrypted value starts with `enc:`. Account passwords and agent tokens are hashed.

### Migrations

The schema changes only through goose SQL migrations in `backend/migrations`, embedded in the binary and applied at startup. `AutoMigrate` is never used.

- **Order.** Migrations may apply out of order, so a migration merged after a later-numbered one still runs. A migration therefore never depends on the order of migrations within a release.
- **Transaction.** Every migration runs whole in one transaction, so a failed one leaves no partial changes and the next start applies it again. `-- +goose NO TRANSACTION` is not allowed. Migrations run before the server accepts requests, so they can lock tables and need no `CREATE INDEX CONCURRENTLY`.
- **Older data.** Every migration runs on a fresh install, on an install upgraded through every release and on an internal database restored from an older dump.
  - A migration that adds or tightens a constraint first repairs the rows that would violate it. Otherwise the upgraded instance does not start.
  - A migration that moves encrypted values copies them as they are, because SQL cannot decrypt them.
- **One-way upgrades.** A binary refuses to start when the database's newest migration is newer than its own. On a renamed table it would otherwise fail only at its first request. An unknown migration older than the binary's newest does not stop it, because releases have deleted or renamed migrations.
- **Testing.** The upgrade test from [testing.md](testing.md) runs every migration against data written by the previous release.

## Engines and providers

Databasus has three families of interchangeable implementations: database engines, storage providers and notifier providers. Each family has a registry keyed by type, and code outside the family never switches on the type.

### Engines

Engines live in `internal/engines/<engine>`: `postgresql`, `mysql`, `mariadb` and `mongodb`. An engine connects, inspects privileges, provisions limited users, dumps, restores and finds its client tool. PostgreSQL also streams base backups, rebuilds their manifests and receives WAL.

- `postgresql` is split into `logical`, `physical` and `shared`, and `mysqlfamily` holds what MySQL and MariaDB share. `clienttools` holds what the client tool sets of every engine share.
- An engine's sources offer small sets of methods, because a physical PostgreSQL source and a logical one support different sets.
- A consumer serving several engines through one shape declares the interface it needs, as `ConnectedDatabase` in `databases` does.
- That consumer maps each database type to its implementation in one registry of its own, as the logical backup pipeline of `backups` does. The provider registry of `databases` cannot hold those entries, because they open work `databases` knows nothing about, and a restore target has settings but no database row.
- Restore command construction and outcome handling are shared with the agent, but execution policy stays with each caller: verification differs in ownership, privileges and ignorable errors. The shared code must not pull the server's frameworks into the agent.

### Providers

**Storage providers** (`storages/internal/providers/<name>`) save, read and delete files and test a connection. **Notifier providers** (`notifiers/internal/providers/<name>`) send a notification and test a channel. [The backend rules](../backend/AGENTS.md#engines-and-providers) list the steps for adding one.

- Each provider owns its behavior, its repository and one table with its settings as explicit columns.
- The common fields and the type sit in a parent row, in `storages`, `notifiers` or `databases`. Every other table references the parent.
- The parent repository saves the parent row and, in the same transaction, the provider's settings. A list loads the parents first, then the provider rows in one batch per type.
- Settings stay in columns for two reasons. The database keeps enforcing `NOT NULL`, `CHECK` and types. Existing rows get a new field's value from `DEFAULT`.
- Backup configs have the same shape, with the backup method as the type: `backups_configs`, `backups_configs_logical` and `backups_configs_physical`.

## Shared code

`internal/lib` holds code without domain logic, one concept per package, so the directory listing tells what is available. Test support may live there too, but neither production binary may depend on it or on test libraries. The only exceptions are third-party dependencies required for rclone providers; the architecture linter's [explicit allowlist](../backend/tools/archlint/production_dependencies.go) records their reasons and rejects any other importer.

### Audit events

- Audit events are written through `lib/audit` in the transaction of the operation they describe, or right after an effect outside the database, such as a sent email.
- Each event stores a snapshot of its audit author, so it reads the same after the user changes or is deleted. Reading it needs no join with `users`.
- `workspaces` serves every audit log view, because only it can read workspace names and apply each view's access rule.

## HTTP API

Two contracts describe the API, and the servers, test clients and frontend types are generated from them. `backend/api/openapi.yaml` describes what the SPA and users' scripts call, with paths relative to `/api/v1`. `backend/api/agent.yaml` describes what the verification agent calls. The `api-design` skill holds the rules for designing an operation.

- **The contract comes first.** A change starts there, and `mise run generate-api` regenerates the code. Generated files (`*.gen.go`, `*.gen.ts`) are never edited by hand.
- **Implementation.** A module implements its generated strict server in `internal/modules/<name>/internal/httpapi`, or in `internal/agentapi` for `agent.yaml`. Operations are assigned to modules by tag. The `system` tag is the exception: it describes the running instance, and `internal/app/systemhttpapi` serves it.
- **Mounting mistakes.** A route registered without an operation in the contract answers 500, so a mistake cannot expose an operation without its access checks.
- **Errors** use the `Problem` format (RFC 9457) with a machine-readable `code`, and every operation declares a `default` response with it. An internal error's text never reaches the client, and the frontend chooses its translation by `code`.
- **Access to the user API.** The `users` middleware authenticates a request, puts the `Actor` into its context and enforces the instance admin mark from `openapi.yaml`. `Authorize` checks workspace access, as [Modules](#modules) describes.
- **Access to the agent API.** Its operations run on a separate route group behind their own contract guard, and the verification module authenticates each with the agent token.
- **Lists** are paged with a cursor through `limit` and `after`.
- **Long operations**, such as a backup, a restore or a verification, answer 202 with the resource in its in-progress state.
- **Starting a physical backup** is the one exception and answers 202 with no body.
  - It only records a request in the backup config, and the backup request job creates the backup on a later pass.
  - WAL streaming and a storage switch share that request, and a start must wait for an earlier backup's process. Creating the row in the request would add a second way to start a backup and refuse a start during a running backup.
  - A client finds the backup in the list and cannot distinguish simultaneous manual and scheduled starts.
- **Unknown paths.** A path under `/api` that matches no route answers 404, and every other path serves the SPA.
- **Compatibility.** A release may break the user API if the same change updates the frontend.
  - `/api/v1/system/version` and `/api/v1/system/verification-agent` never change, because installed agents update themselves through them.
  - A running agent calls the agent API of its own release until its next update check, so agent operations ordinarily stay compatible with the previous release.
  - An explicitly approved proposal may break an agent request shape while preserving discovery and downloads. An old agent may then become stale until it updates.

## Long-running work

Backups, restores and periodic jobs outlive any single request, so they follow one set of rules about starting, stopping and observing work.

- **Background jobs** run through `background.Runner`, and each run of a job is a pass.
  - The runner puts `job_name` and `job_id` into the pass's context and never overlaps two passes of one job.
  - A job can receive a wake signal to run before its next interval. A buffered signal received during a pass schedules another pass after it finishes.
  - A failed or panicked pass is logged, and the next pass still runs on schedule. Starting a runner twice panics, because it is always a bug.
  - Health reads use copied [job snapshots](glossary.md#code) and remain available while a pass blocks.
- **Instance health** is the public `/api/v1/system/health` operation (spec `instance-health`). Raw errors stay in logs. `/system/version` and agent downloads do not depend on it, and Helm probes target `/system/version`.
- **Permitted pass duration** is a monitoring budget that cancels nothing and changes no schedule, as the [glossary](glossary.md#code) defines.
- **Supervision** uses `background.Group` for the HTTP server and module work that must be drained at shutdown.
- **External tools** (`pg_dump`, `pg_basebackup`, `pg_receivewal`, `mysqldump` and others) run through `lib/process`. It kills the whole process group when the operation's context ends.
  - The one exception is the verification agent's detached daemon start, because the daemon must outlive the command that starts it.
  - After a tool exits, the limit on its standard output covers only waiting for more output. The writer's time to accept it does not count, so a slow storage cannot fail a finished dump.
- **Goroutines** start only through `lib` helpers that tie them to a context and recover a panic into an error. A bare `go` statement exists only inside `lib` and in tests.
- **Cancellation.** A running backup, restore or verification is registered under its ID, and a cancel request ends its context.
- **TimescaleDB restore cleanup** finishes before the result is recorded and the target tunnel closes. It has its own limit within the shared shutdown budget (spec `restores`). An archive failure remains the restore's cause of failure even if cancellation, timeout or shutdown follows.
- **Shutdown.** SIGTERM or SIGINT cancels the root context, and then:
  - the HTTP server stops accepting connections and waits for started requests;
  - the runner starts no new passes;
  - interrupted backups and restores end with the status `FAILED` and the reason that Databasus was stopping, and the deletion worker gets their files;
  - WAL streamers stop and keep their replication slots, so the next start resumes them;
  - the process flushes the logs and exits before the embedded PostgreSQL stops (spec `container-shutdown`).

  A second signal during the shutdown has no effect (spec `application-shutdown`). Work still marked in progress at the next start is failed the same way.

- **Time** that drives behavior comes from a clock in `Deps`. SQL receives the current time as a parameter and never calls `now()`.

## Backup data flow

Backup and restore data streams in bounded chunks with backpressure. The producer, such as a dump tool, `pg_basebackup` or a storage download, cannot run ahead of the consumer by more than a small window. Memory per operation therefore does not depend on the database size, and a slow storage only slows the backup.

Backup data does not touch the host's disk, with two exceptions:

- A PostgreSQL restore that runs `pg_restore -j`, excludes extensions or skips user mappings reads the archive more than once. It downloads the archive into a file after checking that the disk has room.
- `pg_receivewal` writes WAL segments into a local queue before they are uploaded.

Backups are encrypted per file. The key for each file is derived from the master key and the file's ID with PBKDF2-SHA256, and data is encrypted with AES-GCM in chunks.

## Stored files

A file in a storage must never lose its owner, even when a backup crashes halfway. Backup code therefore never calls a storage provider directly: every write and delete goes through the file store of `storages`, and every read goes through its storage reader.

- `WriteFile` records a pending deletion before the first byte leaves and returns a write receipt. The transaction that publishes the backup spends the receipt and so keeps the file.
- Any other outcome leaves the pending deletion in place, and the deletion worker removes the file. That covers a provider error, a crash, a rolled-back publication and a cancellation.
- A terminal failure asks for the deletion in the same transaction that records the failure.
- A committed deletion request wakes the deletion worker immediately. Full batches schedule another pass, and a finished write wakes cleanup again so files protected during the write can be removed. The periodic pass remains available for retries and recovery after a restart.
- Every write attempt gets its own object name, because a retry that reused a name would address an object a pending deletion already owns.
- A provider's delete removes every object it derives from the name, such as chunk objects, part manifests and incomplete multipart uploads. Only a confirmed absence counts as success.

### Storage switches

A storage switch activates the new storage immediately and cleans up the old files afterwards. Specs `backup-storage-assignment` and `storage-file-cleanup` hold the product rules and limits; the glossary defines the [cleanup queue and scope](glossary.md#code). The following rules keep concurrent writes and cleanup consistent.

- **Protect running writes.** A write holds a session advisory lock for its storage and file name, from before pending registration through provider completion and receipt preparation.
  - Uploads share one reserved connection per active file store. Deletion attempts try the matching transaction lock and skip protected files.
  - Cancellation keeps protection until the write returns. Session failure cancels writers and rejects receipts; process death releases locks but leaves durable deletion obligations.
- **A switch commits in one transaction:** the target config, the workspace transfer and notifier changes, the removal of the old backup catalog and scoped file-deletion requests.
- **Authorize the current state.** Config saves re-read and authorize the database under database and config locks. The workspace read there decides storage ownership and acceptance scope, so concurrent transfers cannot leave a stale decision.
- **Keep one queue.** `storages` owns each pending file deletion and retries until confirmed absence or explicit storage removal. A reassignment adds the database and accepting workspace as attribution; duplicate requests retain the original scope. These IDs have no foreign keys, so entity removal cannot erase obligations. Batch claims reserve rows; attempt counts increase only when a file deletion starts with a valid lease. Backup configs expose current counts from one batched scope read, with no operation history or copied cleanup state.
- **Delete storage explicitly.** Ordinary removal refuses outstanding queue entries, including protected uploads; live references take priority. Explicit abandonment deletes the storage and its queue entries atomically and records a bounded audit snapshot. Neither path contacts the provider. Workspace removal applies the ordinary guard to every storage and rolls back the whole deletion on refusal.
- **WAL stays consistent after cleanup.** After removing old physical FULLs, cleanup reconciles WAL and timeline history against the surviving completed FULLs. This includes WAL and history that ordinary cascade deletion skipped because a FULL was running.
- **Reject publication into replaced storage.** Every selection change advances a storage generation. Physical runs capture it and validate both generation and selection before publishing base, manifest, WAL or timeline-history rows or spending receipts.
  - Object names start with the database UUID, and a cleared config retains its last selected storage ID without a foreign key, so cleanup can still find unfinished writes.
  - After-commit cancellation drains old work while retaining its busy reservation. Separate WAL claim identities prevent an old exit or heartbeat from affecting a new owner.
- **Lock order.** Scheduling and physical config changes acquire the physical-history lock before database and config locks. Publication takes database and config locks without waiting for the physical-history lock.

## Physical backups

Physical backups copy a PostgreSQL cluster's files, using the native PostgreSQL 17 stack:

- `pg_basebackup` for full backups;
- `pg_basebackup --incremental` against the parent's manifest for incremental ones;
- `pg_receivewal` for the WAL stream;
- `pg_combinebackup` on restore.

PostgreSQL owns the backup format and the block-level incrementals. Product rules follow from this choice: PostgreSQL 17 and later only, WAL summarization required for incrementals, no custom tablespaces and only completed WAL segments uploaded. The specs under `openspec/specs/physical-backups` state them.

- **Stream one archive.** The base backup streams a tar archive, including its WAL, straight into storage. A per-backup replication slot retains the WAL until the backup ends.
- **Compress on the source.** Server-side compression reduces traffic over the source link, which client-side compression and local staging cannot do.
- **Codec fallback.** Without zstd on the source, the codec falls back to gzip and then to none. Every backup tries zstd first.
- **Manifest.** With server compression, `pg_basebackup` cannot embed `backup_manifest`, so Databasus rebuilds it from the tar stream and stores it as an encrypted sidecar file for `--incremental`.
  - Tests validate rebuilt manifests with PostgreSQL's own verifier, including compressed archives, so format drift fails the test.
  - Speaking the replication protocol directly would avoid the rebuild, but we would then maintain our own `pg_basebackup`.
- **Coordinate status and files.** One lifecycle owner commits each transition with its file obligations. Completion records database success before updating the backup, preserving database-before-backup lock order. Cancellation and deletion commit before the process stops; the database remains busy until its old run ends.
- **Reserve each database once.** Full and incremental backups share one table, with restrictive references to their parent and root full backup. A unique constraint permits only one in-progress backup per database; that row reserves the database.
- **Deciding what to run.** Two pure decision steps choose a full backup, an incremental or nothing, and the scheduler loads chain facts only when the second needs them.
  - A pass with nothing due ends before a transaction.
  - Otherwise the scheduler takes the advisory lock shared with orphan WAL cleanup, reloads the state, repeats the decision and inserts what it chooses.
- **Separate periodic scheduling from requests** (spec `physical-backups/background-load`). The full scheduler evaluates enabled configs with a fixed query count; the request job reads only pending requests, so manual starts stay fast. Polling committed configs avoids depending on every request writer to send a wake-up signal.
- **Separate reported health from live control.** API health comes from stored WAL samples. The streamer's live tracking sends notifications and controls receiver pauses, slot rebuilding and WAL rotation.
- **Restore.** The user restores from a downloaded restore set with the recovery script Databasus serves.

## Verification agent

The verification agent is a CLI worker on a machine the user provides. It proves that backups restore: it sends capacity heartbeats, claims verifications, restores each backup into a disposable PostgreSQL container through the Docker or Podman API and reports the result. It has no HTTP server and no database schema.

- **Distribution.** The agent ships inside the Databasus image and is downloaded from `/api/v1/system/verification-agent`. Both binaries are built from one commit, so after self-update the agent shares the backend's contract and engine code. Before that update its requests may be rejected.
- **New work** goes only to an agent with [assignment eligibility](glossary.md#product) (spec `backup-verification`). The verification service checks it after authentication and request validation, before locking any pending row. An ineligible agent gets the empty `204` of `claimVerification`, and the verification stays pending for another agent.
- **Heartbeats and claims run independently**, so a first claim may find no accepted heartbeat yet and retries on its next poll.
- **An agent whose eligibility has expired** stays listed, so an administrator can find and repair it. Work already running continues until the scheduler's stale-run pass requeues or fails it.
- **Failure diagnostics.** Failed reports carry a sanitized excerpt of the underlying error, keeping the beginning and end within 65,536 UTF-8 bytes. The agent masks known credentials and omits the unstructured stream from the first PostgreSQL sensitive section onward, because SQL bodies and row values can imitate diagnostic record prefixes. It also uses the [shared redaction rules](#logs); the server reapplies that policy before storage. The terminal failed outcome and diagnostic are written atomically on the verification row after checking its current owner under a row lock. Requeued attempts, completion, cancellation and server-detected failures clear the diagnostic. Authorized verification detail reads return it after the disposable container is gone; lists and notifications keep only the short failure summary.
- **Updates.** The agent periodically checks the server version and atomically replaces its executable, so its account needs write access to the executable's directory. The upgrade test exercises the [manual recovery procedure](testing.md#manual-agent-recovery) for a failed replacement.

## Configuration

Settings come from environment variables, with `.env` as an optional local source.

- `app` parses them once at startup into a typed structure through `lib/config`.
- When a variable is missing or invalid, `app` refuses to start with a message that names it.
- Each module receives only the values it needs through `Deps`.
- Paths inside the data volume, such as backups, temp files and the master key, are derived from one data directory.

## Logs

The backend logs through `log/slog` from `lib/logging`, and every record can be traced to its request or job.

- **Sinks.** Every record goes to stdout and to a rotating JSON file in the data volume. The file is the forensic copy that survives a lost database, so audit events go to it too.
- **OTLP.** An optional OpenTelemetry collector gets the records too, because OTLP is the one protocol every log backend speaks. When it is down, its records are dropped and requests keep running.
- **Request IDs.** The first middleware gives every request a fresh `request_id`, returned as `X-Request-Id`. An incoming `X-Request-Id` is ignored, so a client cannot stitch its actions into another trace.
- **Context.** `request_id`, `job_name`, `job_id` and entity IDs added with `logging.With` travel in the context, and the log handler adds them to every record.
- **Format.** Messages and keys follow [the backend logging rules](../backend/AGENTS.md#logging).
- **Masking.** Before any sink sees a record, the log handler masks three things: email addresses, attributes whose names mark a secret (password, token, secret, key, authorization, cookie) and known secret patterns inside messages. Audit records identify the author by user ID, so the file keeps its forensic value.

## Frontend

The frontend is a single-page application that the backend serves as static files. It is organized in the layers `app -> pages -> widgets -> features -> entities -> shared`.

Dependencies point only downward, and slices on one layer do not import one another. `app` and `shared` contain segments that may depend on each other. The owner of a screen or a reusable interaction is therefore visible from its directory, and domain changes do not spread through hidden cross-imports.

The [FSD skill](../.agents/skills/feature-sliced-design/SKILL.md) gives the placement methodology and its pinned upstream source. The boundaries below take precedence over the upstream alternatives.

### Where code lives

- **Start in the page.** A screen owns its local UI, requests, state and validation. Extract a feature for an independently reusable interaction and an entity for a domain model with real consumers. Similar-looking code alone is no reason for a new boundary.
- **Compose in app.** App owns providers, navigation and the choice of screen. A reusable shell in widgets receives children or slots and does not import pages. Tabs and URLs stay stable when responsibilities move between layers.
- **Use each slice's public API.** External consumers import its `index.ts`, including for types and dynamic imports.
  - A barrel cannot authorize an otherwise forbidden dependency.
  - Inside a slice, code imports its own modules directly.
  - Shared libraries and test infrastructure may expose focused entry points without a single shared-layer barrel.
- **Keep domain operations with their owner.** A domain API facade keeps its URLs, filters, request bodies and product policy in an entity, feature or page. Shared API holds transport, session-token plumbing, generated types and response normalization. Domain-independent cursor traversal may be shared.
- **An alias of a generated schema names a domain concept.** Keep aliases with their owner, and keep tiny shape aliases with the unit they describe.
- **File layout.** Every script module sits in a layer, apart from the [source-root exceptions](../frontend/AGENTS.md#architecture-and-skills). The [import-path convention](../frontend/AGENTS.md#import-paths) uses the `@/` source-root alias so references have one spelling regardless of the importing file's location. TypeScript, Vite, Vitest, the browser-test loader and architecture checks resolve that alias consistently, including when tooling consumes source modules. The spelling does not change dependency boundaries; architecture checks also recognize relative references in their fixtures.

### State and effects

- **Keep state near its lifecycle.** Form drafts, list selection and page-specific async work belong to their screen or interaction. Generic pagination machinery may live in shared. Application lifecycle hooks belong to app, and domain hooks stay with the domain.
- **Cleanup** invalidates late results, polling and subscriptions, and setup after cleanup must work again.

[Frontend effect rules](../frontend/AGENTS.md#effects-and-rendering) describe dependencies, current-value callbacks and lint exemptions.

### Session and startup

- **Shared session state owns account transitions** (spec `frontend-session`). It publishes a coherent token/user snapshot with a generation; only the active authentication interaction accepts a sign-in response. Requests belong to that snapshot, and a 401 invalidates only that session.
- **Account transitions reset account-owned state.** Logout and session replacement reset screens and workspace selection, dismiss programmatic confirmations and reject obsolete destructive callbacks. Language, theme and version-reload cooldown survive. Providers remain mounted, so password-change feedback survives the transition to sign-in.
- **Startup and shell have separate owners.** App accepts essential user, settings and workspace data together, derives capabilities and owns workspace selection and version polling. The shell widget receives slots and owns viewport and drawer presentation.
- **A sign-in attempt stays in its tab** and is consumed before code exchange. New authentication intent, unmount or session replacement prevents an older completion from activating a session.
- **Storage authorization stays with its editor.** The editing feature owns the attempt in memory, and a callback page returns control to it. Entered secrets and tokens stay in browser memory, which cannot protect them from same-origin code or extensions. [Google Drive storage authorization](google-drive-storage-authorization.md) describes the flow.
- **Theme infrastructure stays in shared.** It owns library configuration and derives application CSS roles from it. [Frontend design](frontend-design.md) explains presentation and its checks.

### Editors and resource pages

- **Keep response acceptance with the editor.** Shared request identity supplies only monotonic revisions. Storage and notifier editors own drafts, acknowledged baselines, persisted IDs and acceptance rules; profile and instance settings follow the same policy. A response may acknowledge submitted values without replacing newer input. Completion selects the resource and closes the editor only when the submitted snapshot is fully acknowledged.
- **Test approval belongs to the tested settings.** Any change invalidates approval, even a change away and back. The reusable database connection form owns approval; its containers own saving.
- **Resource pages own collection policy:** reads, retry, refresh and detail mutations. App recreates resource and workspace-settings screens when the workspace changes. Shared selection and list layout know nothing about resource APIs.
- **Database pages compose editing features.** One controlled connection form serves creation, settings and restore targets, whose containers own their drafts, saving and navigation. Shared draft types belong to entities and distinguish these scenarios, so a restore target cannot request a stored secret. Only pages compose storage and notifier editors; config editors own switch confirmation, and database pages show current cleanup beside Storage. The notice and read retry stay beside Storage during editing. A successful save acknowledges its returned config and cleanup summary. Polling updates only cleanup counts, independently of editing drafts, retains the latest known summary on read failure, and stops on zero or a read failure.
- **Screen tests reuse real providers** through shared test infrastructure, without importing app. Tests replace only HTTP, as [testing.md](testing.md#frontend-tests) describes.

## End-to-end suites

The root `e2e/` directory holds suites that use the application image only from outside.

- `e2e/browser` drives the image through its screens and imports nothing from `backend/` or `frontend/`. An internal rename therefore cannot break it, and a defect that exists only in the image can fail it.
- It tests a candidate built from the commit and the same `Dockerfile`. Publishing rebuilds the image with the version chosen after the checks, so the published image is not byte-identical to the candidate.
- `e2e/docker-storage` checks the image on the host filesystems users run.

## Automatic checks

Lint rejects dependency-boundary violations, code-convention violations and stale API generation. Frontend architecture checks have no list of tolerated findings. Test files are checked too, but production boundary rules allow test-only application composition and goroutines. The [backend](../backend/AGENTS.md) and [frontend](../frontend/AGENTS.md) rules hold coding conventions and suppression requirements. Commands are listed in the root [development guide](../AGENTS.md#development-environment); check implementations and configuration live with the code they check.
