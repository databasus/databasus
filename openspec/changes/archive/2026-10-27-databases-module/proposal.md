## Why

`databases` is the next module of `docs/architecture.md` to move into the target layout. It is the second half of milestone 8 of the backend rewrite plan, after `engines-layer` put the engine behavior into `internal/engines`. Today it is an old feature:
- one GORM model serves as the table row, the request body and the response;
- `switch` statements on the database type sit in the model, the service, the repository and the tunnel wrapper;
- controllers answer most errors as 400 with the internal error text;
- listeners registered through setters let each consumer commit its own transaction;
- `databases` imports the physical backup service to show the last backup time, a cycle that keeps both features from becoming modules.

The old feature also has these defects:
- **A database of another workspace can be taken over.** Creating a database keeps the `id` the request carries, and the repository saves a known ID as an update by primary key. A member of one workspace who knows the ID of a database in another workspace can rewrite that row and move it, with its backups, into their own workspace.
- **A viewer can leak stored secrets.** The read-only user check takes the ID of a saved database from a viewer, replaces the host with the one in the request and keeps the stored password, TLS key and SSH secrets for the fields the request leaves empty. A viewer can make the backend send them to a server of the viewer's choice. The storages and notifiers modules closed the same hole for their tests.
- **Testing a saved database rewrites it.** A viewer's connection test saves the whole row, settings and notifiers included, to clear the last backup error.
- **Deleting a database is not atomic.** Four listeners each commit their own work before the row is deleted. When the last delete fails, the logical backups are already gone while the database stays.
- **Attaching a notifier races its transfer.** A notifier's workspace is checked before the database is written, so a notifier transfer that commits in between leaves the database using a notifier of another workspace.

## What Changes

- **New module.** `internal/modules/databases` appears with `module.go`, `Deps`, `New` and its code under `internal/`. It owns:
  - databases and their settings per type;
  - connection tests and the read-only and replication users;
  - the notifiers attached to a database;
  - the last backup time, the last backup error and the health status.

  Its services open transactions through the `database.Transactor`, check access through `workspaces.Authorize` and write audit events through `lib/audit` in the same transaction.
- **A provider per database type.** Each of the five types (PostgreSQL logical, PostgreSQL physical, MySQL, MariaDB, MongoDB) becomes a provider package under `internal/providers`. It holds its settings, validation, secret merge, conversion to the engine's settings with decryption, and the repository of its own table. A registry keyed by type is the only place that knows every type. It calls the engine source of its own type from `engines-layer`, and the module reaches every type through one `ConnectedDatabase` interface. The `switch` statements, the five `Preload` calls and the GORM associations are deleted.
- **What other modules get.** `module.go` offers:
  - batch lookups of databases without secrets;
  - typed getters that hand a backup, a restore or a verification the decrypted settings of one database of a known type;
  - the trusted liveness probe for health checks;
  - single-column writes of the last backup success, the last backup failure and the health status;
  - the instance-wide summary for telemetry;
  - the handlers of `WorkspaceDeleting` and `NotifierTransferring`;
  - the events `DatabaseCreated`, `DatabaseCopied`, `DatabaseBackupTypeChanged` and `DatabaseDeleting`.

  `databases` no longer imports `backups`.
- **Every backup records the last backup time when it is published.** Physical backups start writing it through the module when a full or incremental backup completes and when a WAL segment is archived. Logical backups move their write, which today runs after the publication has committed, into the publishing transaction. The time never moves backwards. The migration fills it once for existing physical databases.
- **Deleting a database is one transaction.** The deletion publishes `DatabaseDeleting` inside its transaction. The backup handlers delete their rows and request the deletion of their files in it. The effects outside the database run only after the commit: cancelling a running backup, stopping the WAL streamer and dropping the replication slots. `lib/database` gains after-commit hooks on the transaction in the context for this. A database whose logical backup is still running is refused, as today, now with 409 `in_use`.
- **Attaching notifiers is checked inside the transaction.** The check reads each notifier's workspace after the attachment rows are inserted, so a concurrent notifier transfer either sees the attachment or waits for it.
- **10 operations enter `backend/api/openapi.yaml` under the tag `databases`:**
  - list with a cursor and an optional notifier filter;
  - create, read, update, delete and copy;
  - test a saved database;
  - test unsaved settings;
  - check whether a limited user is advisable;
  - create a limited user.

  The settings are a `oneOf` with the discriminator `type`. Secrets are `writeOnly`, and an omitted secret keeps its stored value on update. A database lists the IDs of its notifiers instead of embedding them.
- **Who may do what.** Testing unsaved settings, checking whether a limited user is advisable and creating a limited user need the right to edit the workspace's content, because each of them can send stored secrets to the host in the request. A viewer can still test a saved database with its stored settings. That test writes nothing.
- **Create takes no ID.** Creating a database never writes to an existing one.
- **The old code reaches the module through a bridge in `features/databases`:** the installed module, plus the test helpers behind the `testhelpers` tag.
  - Backups, restores, health checks, verification and telemetry switch to `module.go`.
  - The old restore request keeps its JSON shape with request types of its own.
  - The database transfer stays in the old backup configs until milestone 9, and calls the module to transfer the database and replace its notifiers.
- **Schema.**
  - The provider tables become `databases_postgresql_logical`, `databases_postgresql_physical`, `databases_mysql`, `databases_mariadb` and `databases_mongodb`, and `database_notifiers` becomes `databases_notifiers`. Each provider row takes its database's ID as its primary key and references it through `(database_id, type)`.
  - The comma-separated list columns become `text[]`.
  - The logical PostgreSQL table gets the foreign key it never had.
  - `databases` gets `CHECK` on its type and health status, `UNIQUE (id, type)` and `NOT NULL` on `workspace_id`.
  - Rows that would break the new constraints are repaired first.
- **Frontend.** The frontend calls the 10 operations through the generated client and types. The hand-written models of `entity/databases` are deleted, except the restore target types, which move to `entity/restores` while restores keep their old route.
- **Tests and the upgrade test.** The tests of `databases` become module API tests through `apptest` and the contract-checking client. The upgrade test writes a database of every logical type with a password and exclusion lists.
- **BREAKING** for users' scripts:
  - the database operations change shape: `POST /databases/create` and `POST /databases/update` become `POST /databases` and `PUT /databases/{databaseId}`;
  - the list takes `workspaceId` and returns a page;
  - a database carries `type` and `settings` instead of one field per engine, and `notifierIds` instead of notifier objects;
  - the direct connection test moves to `/database-tests`, and the limited-user routes move to `/limited-user-checks` and `/limited-users`;
  - the two notifier usage routes are replaced by the list's notifier filter;
  - errors are `Problem` bodies.
- **BREAKING** for viewers: they can no longer check whether a limited user is advisable.
- **BREAKING** for the schema: six tables are renamed, five lose their own ID column, list columns change type, and constraints are added.

## Capabilities

### New Capabilities

- `databases`: creating, changing, testing, copying and deleting a database, its secrets and fixed type, its notifiers, the limited users, the last backup time and health status, and who may use stored secrets.

### Modified Capabilities

- `workspace-access`: the database operations answer access refusals with 404 `not_found` and 403 `permission_denied`, like the other content of a workspace, and the role list names the database tests.
- `internal-postgresql-protection`: the direct connection test of unsaved settings answers through the new operation, with 404 for a database the caller may not see and `Problem` bodies for internal failures.
- `upgrade-test`: the test writes a database of every logical type instead of one PostgreSQL database.

## Out of scope

- Moving `backups`, `restores`, `healthcheck`, `verification` or `telemetry` into modules. They call `databases` through the bridge.
- Moving the database transfer into the `databases` module or running it in one transaction. It belongs to the old backup configs, which carry the storage and notifier choices, and moves in milestone 9.
- Switching the references from backup, health check and verification tables to `databases` from `CASCADE` to `RESTRICT`. Each moves with its module.
- Moving dump and restore into engines, and moving their launches onto `lib/process`.
- New engines, new settings, and any change to what a connection test, a version detection or a limited user does.
- Copying the physical backup config with a copied physical database. The handler that would do it was never registered; this change deletes it and keeps today's behavior.

## Impact

- **Backend, new and changed:**
  - new `backend/internal/modules/databases`;
  - `backend/api/openapi.yaml`;
  - `backend/internal/lib/database` (after-commit hooks);
  - `backend/internal/app` (construction, routes, event wiring) and `backend/internal/app/apptest`.
- **Backend, reduced:** `backend/internal/features/databases`, down to the bridge and its test helpers, and what `engines-layer` left in `backend/internal/features/sshtunnel`.
- **Old callers that switch to the bridge and the aliases of `module.go`:**
  - `features/backups` (configs, backuping, services, use cases, core);
  - `features/restores`, `features/healthcheck`, `features/verification` and `features/telemetry`;
  - `features/tests`;
  - the test files that use `databases.CreateTestDatabase` and `RemoveTestDatabase`;
  - `backend/e2e/upgrade` and `backend/cmd/seed_dev`.
- **Frontend:**
  - `frontend/src/entity/databases`, which moves to the generated client;
  - `entity/restores`, which receives the restore target types;
  - the screens of `features/databases`;
  - the database parts of `features/backups`, `features/healthcheck`, `features/restores`, `features/verification`, `features/notifiers` and `features/workspaces`;
  - the six dictionaries.
- **Migrations:** a first migration renames six tables, repairs rows, converts the lists, fills the physical last backup time and adds the constraints. A second one, after the old models are gone, makes each provider row's database ID its primary key.
- **Running instances:** databases keep their settings, secrets, notifiers, backups and backup configs. The migration repairs these rows:
  - a provider row whose database has another type, or that duplicates another row of the same database, is deleted;
  - a logical PostgreSQL row without its database is deleted;
  - a database of an unknown type, or without its type's provider row, is deleted when nothing references it; a referenced one without its provider row stays, and a referenced one of an unknown type stops the upgrade with its ID, because no release writes such a type;
  - a database without a workspace takes the oldest workspace when backups or a backup config reference it, and is deleted otherwise.
- **Documents:**
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the checklist and the table of temporary relaxations;
  - `docs/architecture.md`: after-commit hooks in "Data", and capability interfaces declared by the consumer in "Engines and providers";
  - `docs/glossary.md`: limited user, connection settings, after-commit hook, and provider extended to database types.
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
