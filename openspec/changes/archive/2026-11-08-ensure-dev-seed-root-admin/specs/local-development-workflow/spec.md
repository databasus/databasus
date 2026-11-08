## ADDED Requirements

### Requirement: Development seeding transfers bootstrap privileges only after authentication

The local development seed SHALL assign bootstrap privileges only after the dev account successfully signs in or completes registration. A failed health check, failed authentication, incomplete two-factor sign-in or refusal to register SHALL leave existing privileges unchanged. An invocation outside development mode SHALL fail without changing data.

The local privilege operation SHALL verify that the authenticated account's identity and email match the active account in the selected internal database before changing privileges. A missing or mismatched account, unavailable database or failed transfer SHALL make the seed exit with a non-zero status and SHALL prevent subsequent workspace, storage and database seeding. Diagnostics SHALL identify the failed step without disclosing credentials, session tokens or the database connection string.

The transfer SHALL be atomic and SHALL preserve exactly one bootstrap administrator when replacing an existing holder. It SHALL leave the previous holder's administrator role unchanged and SHALL NOT reset passwords or change account settings. Repeating a successful assignment SHALL leave privileges unchanged and SHALL NOT add a duplicate transfer audit entry. A failed transfer SHALL leave the previous assignment intact; account registration that already succeeded is not rolled back.

Every completed privilege change SHALL be recorded in the audit log, identifying the dev account and the previous bootstrap administrator when one existed. A failure to save that audit event SHALL roll back the privilege change.

#### Scenario: An existing member receives bootstrap privileges

- **WHEN** the dev account is an active member, another account is the bootstrap administrator, and the seed authenticates the dev account successfully
- **THEN** the dev account becomes the sole bootstrap administrator with the administrator role
- **AND** the previous holder remains an administrator with its password and settings unchanged
- **AND** the audit log records the transfer

#### Scenario: An existing ordinary administrator receives bootstrap privileges

- **WHEN** the dev account is an administrator without the bootstrap designation and the seed authenticates it successfully
- **THEN** it receives the sole bootstrap designation and retains its administrator role
- **AND** the audit log records the transfer

#### Scenario: A new dev account replaces another bootstrap administrator

- **WHEN** another account already administers the instance and the seed successfully registers the dev account
- **THEN** the dev account receives the administrator role and the sole bootstrap designation before its workspace is created
- **AND** the previous holder remains an administrator

#### Scenario: The dev account already holds the required privileges

- **WHEN** the dev account is already the bootstrap administrator with the administrator role and the seed runs again
- **THEN** the privilege assignment and existing transfer audit entries remain unchanged

#### Scenario: Assignment fails after the previous designation is removed

- **WHEN** the seed cannot complete the new assignment or its audit event after starting a transfer
- **THEN** the previous bootstrap assignment and both accounts' roles remain as they were before the transfer
- **AND** the seed exits with a non-zero status before seeding other items

#### Scenario: The selected database does not match the authenticated account

- **WHEN** the selected internal database has no active account matching both the authenticated account identity and email
- **THEN** no privileges in that database change and the seed exits with a non-zero status before seeding other items

#### Scenario: The internal database cannot be reached

- **WHEN** authentication succeeds but the seed cannot connect to the selected internal database
- **THEN** the seed exits with a non-zero status before seeding other items and reports the failed step without exposing its connection string

#### Scenario: Authentication fails or needs a second factor

- **WHEN** the dev account cannot authenticate or requires completion of two-factor sign-in
- **THEN** neither the dev account's role nor the existing bootstrap assignment changes

#### Scenario: Seed is invoked outside development mode

- **WHEN** the seed is invoked with a mode other than development
- **THEN** it exits with a non-zero status without creating accounts or transferring privileges

## MODIFIED Requirements

### Requirement: The local app is seeded with a dev account and databases

When the backend started by `mise run run` begins answering requests, the seed SHALL make sure the local app contains:

- the dev account `test@email.com` with password `Q1234567`, holding the administrator role and the sole bootstrap-administrator designation;
- a workspace owned by that account;
- a local storage in that workspace;
- a PostgreSQL database with logical backups of the shared logical test fixture;
- a PostgreSQL database with WAL-streaming physical backups of the live WAL backup source.

Account sign-in, registration and workspace, storage and database seeding SHALL go through the same public API and validation as the web interface. After successful dev-account authentication or registration, and before seeding its workspace, the seed SHALL ensure that account holds the administrator role and is the sole bootstrap administrator. This privilege assignment is the sole exception to preserving existing items: the seed SHALL find each item by its name, SHALL create only missing items and SHALL NOT change an existing workspace, storage, database or backup configuration. When the seed creates a database, it SHALL switch its backups on with the local storage and SHALL start the first backup. The logical database SHALL keep its 3 newest backups, and the WAL-streaming database SHALL keep its 2 newest backup chains.

The seed command, `go run ./cmd/seed_dev` in the backend directory, SHALL also run on its own against the app already answering on the backend port, with the same behavior. The repository SHALL NOT provide a separate task for it. When nothing answers on the backend port, the seed command SHALL exit with a non-zero status and say that the app has to be started with `mise run run`. When the backend answers its health check with an error status, for example because disk usage is too high, the seed SHALL exit with a non-zero status and show the reason the backend gave, including failed dependency names, failure codes and stale background job names, and SHALL create nothing. The sole exception is an `instance_unhealthy` response whose complete diagnostics report passed disk, internal database and client tool checks, only stale verification agents and no stale background jobs: the seed SHALL log a warning with the stale agent count and continue. It SHALL NOT change those agents or the health endpoint's behavior.

When the seed cannot sign in as the dev account and cannot create it, for example because the account exists with a different password or registration is closed, it SHALL exit with a non-zero status and a message that names the account and the reason the backend gave, and SHALL create nothing. When the instance requires two-factor sign-in, the seed SHALL exit with a non-zero status and a message that names the account and says that two-factor sign-in is required, and SHALL create nothing. When the backend refuses to create a seeded item after privilege assignment, for example because a storage cannot be created, the seed SHALL exit with a non-zero status, show the name of that item and the reason the backend gave, and SHALL NOT change an existing workspace, storage, database or backup configuration. The completed privilege assignment SHALL remain in effect. A seed that fails SHALL NOT stop the other processes of `mise run run`.

#### Scenario: First run on an empty app

- **WHEN** a developer runs `mise run run` against a development database that has no users
- **THEN** after the backend starts, signing in as `test@email.com` with `Q1234567` succeeds
- **AND** the account's workspace lists the local storage and both databases, each with backups switched on and a first backup started
- **AND** the logical database's backup settings keep 3 backups, and the WAL-streaming database's keep 2 chains

#### Scenario: Second run

- **WHEN** a developer runs `mise run run` again with the seeded items already present
- **THEN** no item is created a second time
- **AND** no existing item changes, including one the developer edited after the first run, except restoring the dev account's administrator role and bootstrap designation when necessary

#### Scenario: One seeded database was deleted

- **WHEN** a developer deleted the WAL-streaming database and runs the seed command on its own
- **THEN** the seed creates that database again with backups switched on
- **AND** the other items stay unchanged

#### Scenario: Dev account has another password

- **WHEN** `test@email.com` exists with a password other than `Q1234567` and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the account and the failed sign-in
- **AND** no workspace, storage or database was created
- **AND** the account's password and role are unchanged
- **AND** the backend and the frontend keep running

#### Scenario: Members may not create workspaces

- **WHEN** the dev account is a member, members may not create workspaces, and the seed runs successfully
- **THEN** the account becomes the bootstrap administrator before workspace creation
- **AND** the missing dev workspace, storage and databases are created without changing the instance setting

#### Scenario: Backend refuses to create the workspace

- **WHEN** the bootstrap assignment succeeds but the backend refuses to create the dev workspace
- **THEN** the seed exits with a non-zero status and names the workspace and the backend's reason
- **AND** the dev account retains its assigned privileges and existing workspace, storage, database and backup settings stay unchanged
- **AND** the backend and the frontend keep running

#### Scenario: Two-factor sign-in is required

- **WHEN** the instance requires two-factor sign-in and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the account and saying that two-factor sign-in is required
- **AND** no workspace, storage or database was created

#### Scenario: Backend reports itself unhealthy

- **WHEN** the backend answers its health check with an error status and the reason "disk usage is above 95%", and the seed command runs on its own
- **THEN** the seed exits with a non-zero status and shows that reason
- **AND** nothing is created

#### Scenario: Only verification agents are stale

- **WHEN** the backend reports stale verification agents while disk, internal database, client tools and background jobs pass their health checks, and the seed runs
- **THEN** the seed logs a warning with the stale agent count and creates any missing dev items
- **AND** existing agent records stay unchanged and the health endpoint still reports the stale agents

#### Scenario: Stale agents accompany another health failure

- **WHEN** the backend reports stale verification agents together with a failed disk, internal database or client tool check, or a stale background job, and the seed runs
- **THEN** the seed exits with a non-zero status, identifies the failed checks or stale jobs and creates nothing

#### Scenario: App is not running

- **WHEN** a developer runs the seed command on its own while no backend answers on the backend port
- **THEN** the command exits with a non-zero status and tells the developer to start the app with `mise run run`

### Requirement: Seed credentials exist only in local development

The dev account, its password and the seeded databases SHALL be created only by `mise run run` and by the seed command run on its own in a development checkout. The production image and the production backend SHALL neither contain the seed nor run it, whatever the environment variables, and SHALL create no account with known credentials. The seed SHALL NOT change any existing account's password, profile, active state or settings. Its sole privilege exception SHALL be assigning the dev account the administrator role and bootstrap designation and removing that designation from its previous holder, whose administrator role SHALL remain unchanged. This exception SHALL be available only to the local development seed, never through a public operation or a production console command.

#### Scenario: Production image starts

- **WHEN** the production image starts with an empty data directory
- **THEN** no account exists until someone signs up, and `test@email.com` cannot sign in

#### Scenario: Another account exists

- **WHEN** the development database holds an account other than the dev account and the seed runs
- **THEN** that account's password, role and settings are unchanged afterwards
- **AND** if it was the bootstrap administrator, only that designation is removed when the dev account receives it

