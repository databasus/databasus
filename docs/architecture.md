# Architecture

This document states how Databasus is built and why. The architecture review hook checks every plan and every diff against it. Terms follow the [glossary](glossary.md), and [testing.md](testing.md) describes the test setup. Product behavior lives in the OpenSpec specs under `openspec/specs`. Rules that a linter or `archlint` enforces are listed in [Automatic checks](#automatic-checks).

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

## Frontend

The SPA uses `app -> pages -> widgets -> features -> entities -> shared`. Dependencies point only downward. Slices on the same layer do not import one another; app and shared contain segments rather than slices, and their segments may depend on each other. This makes the owner of a screen or reusable interaction visible from its directory and prevents domain changes from spreading through hidden cross-imports.

- **Start in the page.** A screen owns its local UI, requests, state and validation. Extract a feature for an independently reusable interaction and an entity for a domain model with real consumers. Similar-looking code alone is not a reason to introduce a new boundary.
- **Compose in app.** App owns providers, navigation and the choice of screen. A reusable shell in widgets receives children or slots; it does not import pages. Existing tabs and URLs do not have to change when responsibilities move.
- **Use each slice's public API.** External consumers import its `index.ts`, including for types and dynamic imports. A barrel cannot authorize an otherwise forbidden dependency. Within a slice, use its own internal modules directly rather than importing its barrel back into itself. Shared libraries and test infrastructure may expose focused entry points without a single shared-layer barrel.
- **Keep domain operations with their owner.** A domain API facade holds its URLs, filters, request bodies and product policy in an entity, feature or page as appropriate. Shared API holds transport, session-token plumbing, generated types and response normalization. Domain-independent cursor traversal may be shared; a generic CRUD factory is not required to share a loop.
- **Keep state near its lifecycle.** Form drafts, list selection and page-specific async work belong to their screen or interaction. Generic pagination machinery may live in shared. Application lifecycle hooks belong to app; domain hooks stay with the domain. Cleanup invalidates late results, polling and subscriptions, and setup after cleanup must work again.
- **Shared session state owns account transitions.** `shared/auth` publishes a coherent token/user snapshot with a generation. `userApi` returns authentication responses; the active auth interaction alone accepts them. Transport binds an authorized request to its originating snapshot and invalidates only that session on 401. Logout and replacement reset account-owned screens and workspace selection while retaining language, theme and version-reload cooldown. Programmatic account confirmations are destroyed on transitions, and destructive callbacks and composed account operations reject an obsolete session. Providers remain mounted so password-change feedback survives the transition to sign-in.
- **Startup and shell have separate owners.** `app/model/useApplicationStartup.ts` accepts essential user/settings/workspace data together, derives capabilities and owns workspace selection. Disk and release information are supplementary. `widgets/app-shell` receives slots and owns the viewport boundary and drawer presentation. App composes the migrated `pages/auth`, `auth-callback`, `profile`, `users`, `instance-settings`, `workspace-settings`, `storages`, `notifiers` and `databases` slices and the remaining domain features through their public APIs. Workspace creation stays in app-local UI. Rename, deletion and confirmed access loss update app-owned workspace selection without reloading the document. Version polling belongs to app. The storage callback keeps its existing address in `pages/storage-oauth`.
- **Authentication attempts have bounded ownership.** `features/oauth-sign-in` creates a cryptographically random state and one tab-local `OAuthSignInAttempt`. Callback metadata must match provider, state, redirect URI and its ten-minute lifetime; consumption precedes exchange. The callback retains one exchange promise across StrictMode setup replay. A newer auth intent, parameter replacement, unmount or session replacement prevents an older completion from activating a session. Provider-name state values are rejected; a callback started before this format requires a fresh sign-in.
- **Storage authorization stays with its editor.** `features/edit-storage` owns an in-memory `StorageAuthorizationAttempt` bound to one editor, session, workspace and client settings. Its random state expires after ten minutes and is consumed before the browser exchanges the code with Google. The `/storages/google-oauth` page clears its query and relays only code/refusal and state over an attempt-specific same-origin `BroadcastChannel`, with a five-second acknowledgment deadline. Authorization updates the original draft; testing and saving remain explicit. Reloading the initiating tab or opening an old JSON-state callback requires a fresh attempt. Entered client secrets and tokens remain in browser memory, so this repair does not provide confidential-client security against same-origin code or browser extensions.
- **Theme infrastructure stays below app.** `shared/theme/getAntdTheme.ts` owns the AntD configuration; `ThemeCssVariables` derives the application CSS roles from its resolved tokens. App and the screen renderer compose that bridge under their own ConfigProvider. Document and portal styles share those roles without a shared-to-app dependency. Independent WAL series colors stay in the exact chart-palette owner. [Frontend design](frontend-design.md) explains presentation and migration checks.
- **Request identity is shared; acceptance policy is local.** `shared/lib/RequestRevision.ts` supplies only monotonic revisions. The storage and notifier editing features own their controlled drafts, acknowledged baselines, persisted IDs and session/settings/draft/request acceptance rules. Settings changes invalidate test approval even after a change away and back. A successful create publishes its assigned ID while newer input stays open for an update of that same resource. Omitted response secrets acknowledge only the submitted secret; a newer replacement remains in the draft. The database connection form has its own smaller policy in `useDatabaseConnectionTest`, which owns only test approval: a connection change withdraws it, a schema or table selection keeps it, and its containers own saving. Profile and instance-settings pages likewise acknowledge submitted snapshots, merge normalized values only into unchanged fields and retain newer edits. A response may acknowledge submitted values without replacing newer input or completing the editor. This keeps editor policy out of shared transport.
- **Resource pages own collection policy.** `pages/storages`, `pages/notifiers` and `pages/databases` own reads, retry, search, refresh and detail mutations. `shared/hooks/useScopedSelection` validates preferences only against an accepted list for the current scope; `shared/ui/ResourceListDetail` owns presentational slots and scrolling. Neither shared unit knows resource APIs. Workspace settings reuse the entity audit renderer and pagination interaction, including persistent read failure and retry.
- **Database pages compose reusable editing.** `pages/databases` owns the database list, details, settings sections and creation wizard, and reaches backups, monitoring and verification through their public APIs. The wizard's base-info and PostgreSQL backup type forms and the limited user form live in the page. `features/database-connection` holds one controlled connection form for the wizard, the settings section and the restore target. Each of them owns its draft, its saving and its navigation. `DatabaseCreateDraft`, `DatabaseEditDraft` and `RestoreTargetDraft` are entity types because the form, the page and restore all name them, while their instances stay in those containers. A draft's type states its scenario, so a restore target has no database identity and cannot produce a test request that reaches a stored secret. The page passes its backup-storage selection to the backup config features and the connection form to restore as render props. Only the page imports `features/edit-storage` and `features/edit-notifier`. Choices contain real IDs, and explicit create actions open nested editors. Persistence refreshes choices; completion selects the resource and closes the editor only after its submitted snapshot is fully acknowledged. Existing config confirmation and accepted storage-switch progress stay with their retained owners.
- **Generated aliases are public domain names.** A focused alias of a generated schema is not a duplicate DTO. Keep aliases with their owner, and do not split tiny shape aliases away from the logical unit they describe. Independently meaningful domain or view-state types follow the module's naming conventions.
- **Reuse existing providers in screen tests.** Shared test code composes locale, theme, library configuration and a memory router without importing app. Complete-application tests live above shared. Mock only the HTTP boundary; tests use the actual domain APIs and screen behavior.

`@/` resolves to `frontend/src` in TypeScript, Vite, Vitest and architecture checks; relative imports follow the same boundaries. The `lint:fsd` command checks imports in addition to ordinary frontend lint. Its configuration and finite migration baseline live under `frontend/`. The baseline identifies individual existing findings by rule, source, target or structural location, occurrence count, owner and removal milestone. It permits no new or multiplied finding, and a repaired finding requires removing its stale entry. Nested domain groups keep explicitly enumerated slice boundaries until their domain milestone flattens them; a group never exempts its descendants from import checks.

The baseline describes migration residue, not the target architecture. Moving screens and application hooks to their final owners happens in the corresponding domain milestones. The [FSD skill](../.agents/skills/feature-sliced-design/SKILL.md) provides placement methodology and links to its pinned upstream source; these project boundaries take precedence over upstream alternatives.

## Backend layout

```
backend/
  api/                         openapi.yaml, agent.yaml and the package that embeds them
  cmd/databasus/               the server and its console commands
  cmd/seed_dev/                development account, workspace and source seeding
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

| Module         | Owns                                                                                                                                                                             |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `users`        | accounts, sign-in (password, GitHub and Google OAuth, email codes), password reset, instance settings, global roles, the request's `Actor`, the instance admin area              |
| `workspaces`   | workspaces, members, workspace roles, invitations, ownership transfer, `Authorize`, every audit log view: the global one, a user's and a workspace's                             |
| `notifiers`    | notification channels and sending a notification                                                                                                                                 |
| `storages`     | storages, the file lifecycle (write receipts, pending deletions and the deletion worker)                                                                                         |
| `databases`    | connections to user databases, the notifiers attached to a database, connection tests, read-only and replication users, the last backup time and health status shown in the list |
| `backups`      | backup configs, logical backups, physical backups and WAL, restores, downloads and restore streams, retention                                                                    |
| `verification` | verification agents, verification configs and verification runs                                                                                                                  |
| `healthcheck`  | scheduled availability checks of user databases                                                                                                                                  |
| `telemetry`    | anonymous usage statistics                                                                                                                                                       |

The dependency graph between modules has no cycles. From the lowest module to the highest, the order is: `users`; `workspaces`; `notifiers` and `storages`; `databases`; `backups`; `verification` and `healthcheck`; `telemetry`. A module imports only modules lower in this order.

How modules talk:

- Only through the interfaces in `module.go`. Data of another module is read through its methods. To read a set of records, the consumer calls a batch method with their IDs and joins the results in Go, instead of calling a method once per record in a loop.
- The request's user is an `Actor` from `users`: the user ID and the global role. A module never receives a `users` model.
- Only `workspaces.Authorize(ctx, actor, workspaceID, action)` checks access to a workspace and to anything inside it. A module first looks the object up in its own table and answers 404 when it is missing. Then it calls `Authorize`, which answers 404 when the actor is not a member of the workspace and 403 when the actor's role does not allow the action.
- A lower module that needs something done by higher ones publishes an event declared in its `module.go`, for example `workspaces.WorkspaceDeleting`. The event's handlers run synchronously, in the publisher's transaction. They run in the order in which `internal/app` registers them in its single wiring file, and nothing else sets that order. Effects outside the database, such as cancelling a running process or dropping a replication slot, are registered as after-commit handlers. An effect that stays correct when the transaction rolls back, such as deleting files that already have a pending deletion, may run inside the handler.
- If two areas keep needing a SQL JOIN of each other's tables, the boundary is wrong and they belong in one module.

Inside a module's subpackage the code has three roles:

- **Handler** decodes the request through the generated strict server, calls one service method and encodes the response. It never touches the database or the service's dependencies.
- **Service** holds the rules: validation, `Authorize`, transactions, audit events, notifications. It writes no SQL.
- **Repository** is an unexported type that holds `*gorm.DB` and runs every query of its tables through `database.Conn(ctx, r.db)`.

Files follow the role: `handler.go` or `*_handler.go`, `service.go` or `*_service.go`, `repository.go` or `*_repository.go`, a model in `<entity>.go`, all errors of the package in `errors.go`. An enum lives next to the entity whose field it types. A long-running loop gets its own file.

## Engines and providers

Databasus has three families of interchangeable implementations. Each family has a registry keyed by type, and code outside the family never switches on the type.

- **Engines** live in `internal/engines/<engine>`: `postgresql`, `mysql`, `mariadb` and `mongodb`. `postgresql` is split into `logical`, `physical` and `shared`, and `mysqlfamily` holds what MySQL and MariaDB share. `clienttools` is not an engine: it holds what the client tool sets of every engine share, namely the bundle descriptions each engine registers, the release-line comparison and the check that every shipped client starts.

  An engine knows how to:

  - connect and test a connection;
  - inspect privileges and provision a read-only or replication user;
  - read a physical source's replication slots, identity, timeline history and WAL position;
  - read a `pg_receivewal` directory and calculate WAL positions and filenames through `postgresql/physical/walmath`;
  - dump into an `io.Writer` and restore from an `io.Reader`;
  - stream a physical base backup into an `io.Writer` and rebuild its manifest;
  - receive WAL into a directory;
  - find its client tool for a server version.

  An engine's sources offer small sets of methods, because a physical PostgreSQL source and a logical one support different sets. A consumer that serves several engines through one shape declares the interface it needs, as `ConnectedDatabase` in the `databases` module does, and maps each database type to its implementation in one registry of its own, and only there, as the logical backup pipeline of `backups` does. The provider registry of `databases` cannot hold those entries, because they open work that `databases` knows nothing about, and a restore target has settings but no database row. The engine package `postgresql/logical/pgrestore` builds the restore command and judges its outcome; each caller runs the command its own way. Verification deliberately differs from a restore in ownership, privileges and ignorable errors. The agent imports only `pgrestore`, which depends on the standard library, because the parent `logical` package links the server's frameworks.

- **Storage providers** (`storages/internal/providers/<name>`) save, read and delete files and test a connection.
- **Notifier providers** (`notifiers/internal/providers/<name>`) send a notification and test a channel.

Each provider owns its behavior, its repository and one table with its settings as explicit columns. The common fields and the type sit in a parent row, in `storages`, `notifiers` or `databases`. Every other table references the parent, never a provider table. The parent repository saves the parent row and, in the same transaction, hands the provider's settings to that provider's repository. A list loads the parents first, then the provider rows in one batch per type.

The backup configs of `backups` have the same shape, with the backup method in the role of the type: `backups_configs` holds the fields both methods share, and `backups_configs_logical` and `backups_configs_physical` hold the schedule and retention of each method.

Adding a provider takes a package, a migration with its table, a registry entry, the contract schema and its conversion in the module's `httpapi`, which turns the generated variant of the `oneOf` into the provider's settings and back. Nothing else changes.

Settings stay in columns rather than in a JSON document for two reasons. The database keeps enforcing `NOT NULL`, `CHECK` and types. And when a field is added, existing rows get its value from `DEFAULT` in the migration instead of a zero value in Go.

## Shared code

`internal/lib` holds code without domain logic:

| Package                  | Holds                                                                 |
| ------------------------ | --------------------------------------------------------------------- |
| `database`               | the database pool, transactions, constraint errors and keyset cursors |
| `config`, `logging`      | configuration, logging                                                |
| `httpserver`             | the HTTP server and its middleware                                    |
| `apierror`               | `Problem` errors                                                      |
| `background`, `shutdown` | background jobs, cancellation and shutdown                            |
| `process`                | external processes                                                    |
| `clock`                  | the clock                                                             |
| `crypto`                 | the master key, field encryption and stream encryption                |
| `audit`                  | audit events                                                          |
| `mailer`                 | email                                                                 |
| `cache`, `ratelimit`     | the in-memory cache and rate limiter                                  |
| `schedule`               | schedules                                                             |
| `gfs`                    | the grandfather-father-son selection of the backups a retention keeps |
| `sshtunnel`              | SSH tunnels                                                           |
| `diskusage`              | disk usage                                                            |
| `iox`                    | stream helpers                                                        |
| `filename`               | portable filename sanitization                                        |
| `namelist`               | the lists of object names a user enters, such as excluded tables      |

Test support lives in `lib` too: `testdb`, `testwait`, the contract-checking clients `apiclient` and `agentclienttest` with `contracttest`, `toolstub`, the source and provider containers in `testcontainers`, SSH settings in `testcontainers/bastion`, proxy and timeout behavior in `httpclienttest`, and mail assertions in `mailpittest`. Storage-provider-only support lives in `storages/internal/providers/providertest`. Production dependency checks keep test support out of both binaries.

Audit events are written through `lib/audit` in the transaction of the operation they describe. Each event stores a snapshot of its audit author's user ID, email and name as they were at the time of the action. The event reads the same after the user changes or is deleted, and reading it needs no join with `users`. `workspaces` serves every audit log view, the global one included: each view shows the name of the workspace an event belongs to, which only `workspaces` can read, and `workspaces` can apply each view's access rule.

## Data

- **Ownership.** Each module reads and writes only its own tables, through its repositories. A table is named after its module. The module's main table carries the module name itself (`storages`), and every other table starts with it (`storages_s3`, `backups_logical`). A `lib` package may own a table named `lib_<name>_*`, such as `lib_audit_events`, and modules use that table only through the package.
- **References between modules.** A foreign key to another module's table is declared `ON DELETE RESTRICT`. `CASCADE` is allowed only between tables of one module. The reason is files: a backup row describes a file in a storage, and a cascade would delete the row without asking for the file's deletion. With `RESTRICT`, a missing cleanup fails loudly instead of leaving orphan files. The database itself also refuses to delete a storage, notifier or database that is still referenced, and the service maps that refusal to 409 `in_use`.
- **No SQL JOIN with another module's tables.** Data of another module comes through its interface.
- **Invariants live in the schema** when the schema can express them:
  - `NOT NULL` and `DEFAULT`;
  - `UNIQUE` and partial unique indexes for "at most one" rules;
  - a composite foreign key `(id, type)`, so a provider row cannot attach to a parent of another type;
  - for a row that references one of two kinds of entities, two nullable foreign keys and `CHECK (num_nonnulls(a_id, b_id) = 1)`, never an untyped `(kind, id)` pair.
- **Enum values.** Their storage and validation follow [the backend migration rules](../backend/AGENTS.md#migrations).
- **Types.** Lists are `text[]`. JSON is stored as `JSONB`, and only for data we treat as opaque and store in clear, such as webhook headers. Time is `TIMESTAMPTZ`. All time is UTC: in Go, in the database session, in the API and in logs. Identifiers are UUIDs with `DEFAULT gen_random_uuid()`. A row that continues another row of the same module takes that row's ID as its primary key.
- **Transactions travel in the context.** A service opens one with `Transaction(ctx, fn)` of the `*database.Transactor` it takes through `Deps`, and every repository call made with that context, in any module, joins it, because a repository runs its queries through `database.Conn(ctx, r.db)`. Transactions are never passed as parameters. Background work that outlives its caller, such as a notification delivery, detaches from the caller's transaction with `database.WithoutTransaction(ctx)` and runs its queries on the pool. The transactor is built in `internal/app` from the pool, so a service never holds the pool itself. An effect outside the database, such as cancelling a process or dropping a replication slot, registers an after-commit hook with `database.AfterCommit(ctx, hook)`. The outermost transaction runs its hooks in order after it commits, with a context that no longer carries the transaction, and drops them when it rolls back.
- **Repository methods** are named after what their SQL does. When a lookup finds no row, or a conditional `UPDATE` touches none, the method returns `false` as its second value, as in `(Backup, bool, error)`. The service decides which error the client sees. A repository wraps a database error with the operation's name and never returns `gorm.ErrRecordNotFound`.
- **One column changes with one `UPDATE`.** A status, a timestamp or a counter is never written by re-saving the whole row.
- **Secrets** of providers and databases are encrypted per field with the master key before they are saved, and decrypted only where they are used. The cipher is AES-GCM, and an encrypted value starts with `enc:`. Account passwords and agent tokens are hashed, not encrypted.
- **Password invalidation.** The users module assigns each user a random UUID password version and rotates it in the transaction that replaces the password and records the audit event. Sessions and pending sign-ins carry that version and must match the user's current version exactly. Password creation time remains historical data; invalidation does not depend on clock precision or clock advancement.

### Migrations

- The schema changes only through goose SQL migrations in `backend/migrations`, embedded in the binary and applied at startup. `AutoMigrate` is never used.
- Migrations are applied with out-of-order allowed, so a migration merged after a later-numbered one still runs. A migration therefore never depends on the order of migrations within a release.
- Every migration runs whole in one transaction, so a migration that fails leaves no partial changes behind and the next start applies it again. `-- +goose NO TRANSACTION` is not allowed. `CREATE INDEX CONCURRENTLY`, the usual reason to drop the transaction, gains nothing here: migrations run before the server accepts requests, so nothing waits for the table lock.
- Every migration runs on any older valid state: a fresh install, an install upgraded through every release, and an internal database restored from an older Databasus dump. A migration that adds or tightens a constraint first repairs the rows that would violate it. Otherwise the upgraded instance does not start. A migration that moves encrypted values copies them as they are. SQL cannot decrypt them and does not need to.
- An upgrade is one-way. Migrations only go forward, and a binary refuses to start on a schema newer than its own: it compares the newest migration recorded in the database with the newest one it carries and stops before applying anything. An older binary on a renamed table would otherwise fail at the first request instead of at startup. A recorded migration the binary does not carry but that is older than its newest one does not stop it, because releases have shipped migrations that later releases deleted or renamed.
- The upgrade test from [testing.md](testing.md) runs every migration against data written by the previous release.
- Format: one statement per table, constraint and index, column types aligned, each constraint clause on its own line.

## Stored files

Backup code never calls a storage provider directly. Every write and delete goes through the file store of the `storages` module, and every read goes through its storage reader.

- `WriteFile` records a pending deletion before the first byte leaves and returns a write receipt. The transaction that publishes the backup spends the receipt, which keeps the file. Any other outcome leaves the pending deletion in place, and the deletion worker removes the file. That covers a provider error, a crash, a rolled-back publication and a cancellation. A terminal failure asks for the deletion in the same transaction that records the failure.
- Every write attempt gets its own object name, because a retry that reused a name would address an object a pending deletion already owns.
- A provider's delete removes every object it derives from the name, such as chunk objects, part manifests and incomplete multipart uploads. Only a confirmed absence counts as success.

### Storage switches

Running file writes hold PostgreSQL session advisory locks keyed by storage ID and file name, acquired before pending registration and released after the provider call and receipt preparation finish. Each file store reserves one connection while any of its writes run; uploads share that session instead of reserving a connection per file. Deletion claims and storage drains try the matching transaction lock and skip protected files across application instances. Caller cancellation keeps protection until the write returns. A detected session failure cancels its writers and rejects their receipts; process death releases the session locks while durable pending obligations remain.

A storage switch commits the target config, workspace transfer and notifier changes, removal of the old backup catalog, and an attributed deletion batch in one transaction. Clearing the storage preserves history. Reassigning storage after clearing removes retained history, including when the same storage is selected again. Ordinary edits that keep the storage do not create an operation. Physical switches preserve FULLs already in the target and the WAL those backups need; an enabled config requests a fresh FULL.

A config save re-reads and authorizes the database under its database and config locks. That locked workspace decides both storage ownership and operation acceptance scope, so a concurrent transfer cannot leave either tied to a stale workspace. After removing old physical FULLs, switch cleanup reconciles WAL and timeline history against the surviving completed FULLs, including when ordinary cascade deletion skipped them because a FULL was running.

The backups module owns `StorageSwitch` identities and the workspace at acceptance. Historical database, workspace and storage IDs do not hold foreign keys to live resources, so history survives database and storage deletion. The storages module owns `DeletionBatch` items, deadlines, attempt attribution and outcomes. Status reads compose these records directly through one batched storages read per page. There is no copied cleanup state or reconciliation job.

Each switch captures the old catalog's files and known unfinished writes. Its cleanup stops after five failed attempts per artifact or 24 hours after acceptance. Calls are bounded by the lease and remaining deadline. A live writer protects its files even at the deadline, when unresolved items become remaining files. Terminal counts and completion times never change. A later file from an old writer has its own deletion obligation and does not extend the captured set. Independent ordinary deletion obligations keep their unlimited retry policy, even for a file also attributed to a switch. Storage deletion drains both pending deletions and durable remaining batch artifacts, reports unresolved file names in the audit log, and settles unfinished batches before the pending rows and credentials disappear at commit. Terminal batch outcomes stay unchanged.

Batch failure accounting is independent of the pending queue's generation: another deletion request cannot erase a provider failure already attributed to a switch. A worker settles expired attempts before claiming again and cannot replace an active attempt's attribution. Queue generation still protects ordinary rescheduling and release of a completed pending obligation.

Physical runs capture the config's storage generation. Base, manifest, WAL and timeline-history publications validate that generation and the selected storage before committing catalog rows or spending write receipts. Every change of the physical storage selection advances the generation, including clearing it. Physical object names start with the database UUID, so durable unfinished writes remain attributable after their backup catalog row is deleted. The config retains its last selected storage ID when the current selection is cleared. Switch acceptance uses that historical ID to capture unfinished writes even without surviving catalog rows, while excluding unfinished files from earlier storage selections. The historical ID has no foreign key to a live storage. A switch records the next FULL request even while backups are disabled; the enabled-config filter prevents execution until they are enabled. After-commit cancellation drains old work while its busy reservation remains held. WAL ownership has a separate claim ID, so an old exit or heartbeat cannot release or update a newer owner. Scheduling and physical config changes acquire the physical-history lock before database and config locks; publication takes database and config locks without waiting for the physical-history lock.

Config saves that switch storage return 202 with `StorageSwitch` and a `Location` status URL. Ordinary saves, clearing storage and initial assignments without history return 200 with the config. `SWITCH_TO_STORAGE` transfers always return 202; `MOVE_WITH_DATABASE` returns 204. Individual status reads and cursor-paged workspace history use `workspaces.Authorize(ActionView)` against the workspace at acceptance, including the instance-admin bypass. The UI warns before submission and polls pending cleanup every two seconds. A read failure retains the last outcome and retries status without resubmitting the switch.

## Backup data flow

Backup and restore data streams in bounded chunks with backpressure. The producer, such as a dump tool, `pg_basebackup` or a storage download, cannot run ahead of the consumer by more than a small window. Memory per operation does not depend on the database size, and a slow storage slows the backup instead of exhausting the host's memory.

Backup data does not touch the host's disk, with two exceptions:

- A PostgreSQL restore that runs `pg_restore -j`, excludes extensions or skips user mappings reads the archive more than once, so it downloads the archive into a file. The restore first checks that the disk has room for it.
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
- Server-side compression sends about a third of the bytes over the source link. For a terabyte database over a WAN, that turns a multi-day backup into hours. Client-side compression and local staging were rejected for exactly that reason. When the source was built without zstd, the codec falls back to gzip and then to none. Every backup tries zstd first, so a source rebuilt with zstd support uses it on its next backup.
- With server compression, `pg_basebackup` cannot embed `backup_manifest`. Databasus therefore rebuilds the manifest from the tar stream and stores it as a separate encrypted sidecar file, which incrementals pass to `--incremental`. The tests check every rebuilt manifest with `pg_verifybackup` from PostgreSQL 18, which can read a compressed tar. Any drift from PostgreSQL's format then fails in CI. Speaking the replication protocol directly would avoid the rebuild, but it would mean reimplementing and maintaining `pg_basebackup` ourselves.
- `catalog.BackupLifecycle` owns status changes through `Start`, `Complete`, `Fail`, `Cancel`, `MarkChainBroken` and the startup sweep `FailLeftInProgress`. Its repository inserts a running row or conditionally updates a row still `IN_PROGRESS`. Each transition commits with its file obligations. Completion records database success before updating the backup, keeping the database-before-backup lock order. Cancellation records `CANCELED` before stopping the process after commit; deletion removes the rows before stopping the process after commit. The backuper keeps each database busy until its old run ends, including a cancelled or deleted run.
- Full and incremental backups share `backups_physical`, with `kind`, `parent_backup_id` for the direct parent and `root_full_backup_id` for the chain root. Both references are `RESTRICT`. Checks require both references exactly for incrementals, and the file name, LSN range and completion time for completed backups. The partial unique index `uq_backups_physical_database_in_progress` allows one `IN_PROGRESS` row per database; that row reserves the database for its backup.
- Two pure steps choose a full backup, an incremental or nothing. The first reads the newest backups, the config and the time; only when it needs chain facts does the scheduler load them for the second. A pass with nothing due ends before a transaction. Otherwise the scheduler takes the advisory lock shared with orphan WAL cleanup, reloads the state, uses the same decision functions and inserts what they choose.
- `streamhealth.GetVerdict` computes WAL stream health from stored samples for the API and the Monitoring tab. Notifications come from the WAL streamer's live tracking, which also pauses the receiver, rebuilds the slot and rotates WAL.
- The user restores a physical backup from a downloaded restore set with the recovery script Databasus serves.

## HTTP API

- The contract comes before the code. `backend/api/openapi.yaml` describes the API that the SPA and users' scripts call, with paths relative to `/api/v1`. `backend/api/agent.yaml` describes what the verification agent calls. A new or changed operation starts in the contract. Then `mise run generate-api` regenerates the servers, the test clients and the frontend types. Generated files (`*.gen.go`, `*.gen.ts`) are never edited by hand. The rules for designing an operation are in the `api-design` skill.
- Each module implements, in `internal/modules/<name>/internal/httpapi`, the strict server generated for its own operations. A module serving `agent.yaml` implements its generated strict server in `internal/agentapi`. Operations are assigned to modules by tag. The `system` tag is the one exception: its operations describe the running instance rather than a module's domain, so `internal/app/systemhttpapi` serves it. A route registered in the router without an operation in the contract answers 500, so a mounting mistake cannot expose an operation without its access checks.
- Errors use the `Problem` format (RFC 9457) with a machine-readable `code`. Every operation declares a `default` response with it. The text of an internal error never reaches the client. The frontend shows a translation chosen by `code`.
- The operations of `agent.yaml` run on a separate route group behind their own contract guard. The verification module authenticates each request with the agent token.
- A request to `openapi.yaml` is authenticated by the `users` middleware, which puts the `Actor` into the request context. Operations of the instance admin area are marked in `openapi.yaml`, and the middleware enforces the mark. Workspace access is checked by `Authorize`, as described in [Modules](#modules).
- Lists are paged with a cursor through `limit` and `after`. `OFFSET` is not used.
- A long operation, such as a backup, a restore or a verification, answers 202 with the resource in its in-progress state.
- A path under `/api` that matches no route answers 404. Every other path serves the SPA.
- A release may break backward compatibility of the contract, as long as the same change updates the frontend. Two paths never change: `/api/v1/system/version` and `/api/v1/system/verification-agent`. Verification agents that users have already installed update themselves through these paths. A running agent keeps calling the agent API of its own release until its next independent update check. Agent operations ordinarily stay compatible with the previous release; an explicitly approved breaking proposal may remove a request shape while preserving public discovery and downloads. Heartbeats require `maxRamMb`: gigabyte-only and missing-memory requests are rejected without changing capacity or last-seen time. Persisted megabytes remain unchanged. An old agent can become stale after five minutes until it updates.

## Long-running work

- **Background jobs** run through `background.Runner`. Each run of a job is a pass. The runner puts `job_name` and `job_id` into the context of each pass and never overlaps two passes of one job. When a pass returns an error or panics, the runner logs it and still runs the next pass on schedule. A second `Start` panics, because it is always a bug. The runner exposes copied, name-sorted [job snapshots](glossary.md#code) through `GetSnapshots`, with activation, running state, start/completion times and the last completed outcome and internal error. A separate metadata lock keeps reads available while a pass blocks; starting a pass retains its previous completed result. Production startup resets activation for jobs that have not run.
- **Instance health** is the public `/api/v1/system/health` operation. It checks the data volume, internal database, essential client tools, verification-agent freshness and every registered job. It returns every check and job on both 200 and 503, with wildcard CORS. An unhealthy response is a `HealthProblem` with `instance_unhealthy`, safe failure codes and a stale-agent count; raw errors remain in logs. Database and agent queries each have a five-second child context. `/system/version` and agent downloads remain independent of deep health, and Helm probes continue to target `/system/version`.
- **Permitted pass duration** is an application monitoring policy, distinct from an execution timeout; see the [glossary](glossary.md#code). It defaults to five minutes, with one minute for telemetry, thirty minutes for database probes and `max(5 minutes, ClaimBatchSize * AttemptLease + 1 minute)` for file deletion. Before the first completion the deadline is activation plus this duration; afterwards it is the last completion plus the interval plus the duration. A running pass also becomes stale at its own duration limit. Exact deadlines are stale. Any completion restores freshness, including a failure or panic. The budget cancels nothing and changes no schedule.
- **Supervision** uses `background.Group` for the HTTP server and module work that must be drained at shutdown. Moving periodic telemetry onto the runner does not remove the server's supervised lifecycle.
- **External tools** (`pg_dump`, `pg_basebackup`, `pg_receivewal`, `mysqldump` and others) run through the `lib/process` helper. The helper binds the process to the operation's context and kills the whole process group when the context ends. The one exception is the verification agent's daemon start. It uses `os/exec` with `Setsid` and `context.WithoutCancel` because the daemon must outlive the command that starts it. `lib/process` kills tools when their parent exits.
- **Goroutines** are started only through `lib` helpers that tie them to a context and recover a panic into an error. In production code a bare `go` statement exists only inside `lib`; a test may start one directly to drive concurrent calls.
- **Cancellation.** A running backup, restore or verification is registered under its ID, and a cancel request ends its context.
- **TimescaleDB restore cleanup.** After successful pre-restore, post-restore runs synchronously before the restore outcome is recorded and its target tunnel closes. Cleanup keeps context values but has its own 30-second limit, independent of user cancellation, the restore timeout and archive errors.

  The application owns a shared shutdown budget in `lib/shutdown` and publishes its absolute six-second drain deadline before canceling module work. Shutdown shortens both new and active cleanup attempts to one second before that deadline; an exhausted cutoff starts no new cleanup SQL. The remaining second is reserved for recording the outcome, whose context is bounded by the drain deadline. A restore stopped first by user cancellation stays canceled when shutdown shortens its cleanup.

  An archive failure is established before detached cleanup starts. The PostgreSQL adapter carries that classification into the restore pipeline so later cancellation, timeout or shutdown cannot replace it; errors reading the archive from storage retain their precedence over a tool error. Cleanup failure preserves an existing restore error and is logged with a warning that the target may remain in restoring mode; when the archive succeeded, cleanup failure fails the restore. If work does not stop by the drain deadline, the shutdown report identifies unfinished steps and startup recovery fails any restore without a recorded outcome. Startup recovery cannot repair the target because its credentials are never persisted. Draining still has at most six seconds, followed by at most two seconds for log flushing.

- **Shutdown.** SIGTERM or SIGINT cancels the root context, and then:
  - the HTTP server stops accepting connections and waits for started requests;
  - the runner starts no new passes;
  - backups and restores interrupted by shutdown end with the status `FAILED` and the reason that Databasus was stopping, and their files are handed to the deletion worker;
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
- Before any sink sees a record, the log handler masks the values of attributes whose names mark a secret (password, token, secret, key, authorization, cookie), secrets that match known patterns inside messages, such as a password in a connection string, and email addresses. Audit records identify the audit author by user ID, so masking emails does not cost the file its forensic value.

## Verification agent

The verification agent is a CLI worker that runs on a machine the user provides. It sends capacity heartbeats, claims verification jobs, restores each backup into a disposable PostgreSQL container through the Docker or Podman API and reports the result. It has no HTTP server and no database schema. The agent ships inside the Databasus image and is downloaded from `/api/v1/system/verification-agent`. It replaces itself with the version of the backend it is connected to. Because both are built from one commit, the agent and the backend always share the contract and the engine code.

## Automatic checks

| Rule                                                                                                                                                                          | Checked by                                                                                                  |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Layer direction, the module boundary through the root package, `module.go` as the only root file, the agent's allowed imports                                                 | `archlint`                                                                                                  |
| File roles, errors in `errors.go`, a bare `go` statement only in `lib`                                                                                                        | `archlint`                                                                                                  |
| No `testing`, testify or testcontainers in either production binary, except the two third-party rclone importers listed below; no GORM, Gin or goose in the agent binary      | `archlint` (dependency graphs of `cmd/databasus` and `cmd/verification-agent`)                              |
| A migration adds no `CASCADE` foreign key to another module's table                                                                                                           | `archlint` (table prefixes)                                                                                 |
| A migration runs in its transaction: no `-- +goose NO TRANSACTION`                                                                                                            | `archlint`                                                                                                  |
| `*gorm.DB` and `database.Conn` only in repositories; no `time.Sleep`, `Eventually`, `fmt.Print*`, `os.Exit`, `os.Getenv` or `context.Background()` outside the places allowed | `forbidigo`                                                                                                 |
| Log calls: attributes only, `...Context` methods, constant messages, `snake_case` keys                                                                                        | `sloglint`                                                                                                  |
| Imports of test libraries only in test code, and no Gin or GORM in the verification agent or the engines; fixed import aliases                                                | `depguard`, `importas`                                                                                      |
| Production code starts an external process only through `lib/process`, except the verification agent's detached daemon start, which must outlive its parent                   | `forbidigo`                                                                                                 |
| No package-level dependency variables, no `init()`; every `Deps` literal lists all fields                                                                                     | `gochecknoglobals`, `gochecknoinits`, `exhaustruct`                                                         |
| Exhaustive `switch` on enums                                                                                                                                                  | `exhaustive`                                                                                                |
| Frontend import direction and slice public APIs, with a finite migration baseline                                                                                             | `frontend:lint` through `lint:fsd`                                                                          |
| The contract follows the API rules; generated code is up to date                                                                                                              | `vacuum`; `backend:check-generated`, which `backend:lint` runs, and `check:api`, which `frontend:lint` runs |

The server retains two third-party test-library importers for rclone storage support: `github.com/Files-com/files-sdk-go/v3/lib` contains assertion helpers used by the Files.com backend, and `github.com/rclone/Proton-API-Bridge` imports `testing` for the Proton Drive backend. The exact accepted importers and their reasons live in `getAcceptedThirdPartyTestImporters` in [`production_dependencies.go`](../backend/tools/archlint/production_dependencies.go). Every other importer is rejected; repository-owned test support stays out of both production binaries.

`mise run lint` runs all module checks. `mise run backend:lint` runs the backend checks; `mise run frontend:lint` checks frontend formatting, lint, types, architecture and generated API freshness. The linters check test files too. The `archlint` rules on imports, the module root, file roles, errors and bare `go` check production files only: a module's API tests build the whole application through `apptest`, which sits above the modules, and tests start goroutines of their own to drive concurrent calls.

All backend packages and commands follow these checks. Immutable shared tables and process-level test fixture synchronization require a specific, explained lint suppression; application dependencies always belong to their owning instance.
