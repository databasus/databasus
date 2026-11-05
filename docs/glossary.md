# Glossary

This is the project's common language. Each concept has one term in the documentation and the interface, and one name in code and the API. Synonyms are not used. A new term is added here in the same change that introduces it into the contract, a `module.go` or the documentation. Definitions stay short. The rules that apply to a concept live in [architecture.md](architecture.md).

## Product

| Term | In code and API | Meaning |
|---|---|---|
| Database | database | A user's database that Databasus connects to and backs up. The term never means Databasus's own database. |
| Internal database | internal database | The embedded PostgreSQL inside the Databasus container that holds Databasus's own state. |
| Engine | engine | A kind of database server Databasus supports: PostgreSQL, MySQL, MariaDB or MongoDB. The term also names the code in `internal/engines` that drives it. |
| Storage | storage | A place where backups are kept: local disk, S3, Google Drive, NAS, Azure Blob, FTP, SFTP, rclone. |
| SFTP authentication method | `sftp.AuthMethod`, generated `SftpAuthMethod`, API `authMethods`; `PASSWORD`, `PRIVATE_KEY` | A credential kind selected for one SFTP storage: password or private key. A storage can select both. |
| Notifier | notifier | A channel Databasus sends notifications to: email, Telegram, Slack, Discord, Teams, Mattermost, webhook. |
| Notification | notification, `notifiers.Notification` | A message about an event of a database, with a category, a heading and a text, which Databasus sends to the notifiers of that database. |
| Notification category | notification category, `notifiers.Category`; `NotificationCategory` in the API | What a notification is about, such as a failed backup. A webhook notifier can accept only some categories. |
| Delivery | delivery | One send of one notification to one notifier, ended by a timeout. Its outcome becomes the notifier's last send error. |
| Provider | provider | One implementation of a database, storage or notifier type, with its own settings table. |
| Limited user | limited user, `limiteduser`, `checkLimitedUser`, `createLimitedUser` | A user that Databasus creates on the user's server for its backups: a read-only user, or for physical PostgreSQL a replication-only user. |
| Resource address | resource address | The settings of a saved storage, notifier or database that decide where its stored secrets go, such as a host and port or an endpoint. A stored secret fills in only while the address stays the stored one. The `secret-handling` spec lists them per type. |
| Workspace | workspace | A group of users and the databases, storages and notifiers they share. |
| Member | member | A user in a workspace, with a workspace role. |
| Member ID | member ID, `memberId` | The ID of one member of one workspace. The API uses it to change or remove that member and to name the new owner in an ownership transfer; it is not the user ID. |
| Workspace role | workspace role, `workspaces.Role`, `WORKSPACE_OWNER`, `WORKSPACE_ADMIN`, `WORKSPACE_MEMBER`, `WORKSPACE_VIEWER` | What a member may do in a workspace. `WORKSPACE_MEMBER` is the role that may edit the workspace content; the word "member" alone means any user in a workspace, whatever the role. |
| Workspace admin | workspace admin, `WORKSPACE_ADMIN`, `workspaces.RoleAdmin` | A member who may also rename the workspace and manage the members who are not admins or the owner. The word `admin` alone in code and text means the instance admin; `workspaces.RoleAdmin` and `ActionManageAdmins` name workspace admins. |
| Owner | owner, `WORKSPACE_OWNER`, `workspaces.RoleOwner` | The member who may also manage workspace admins, transfer ownership and delete the workspace. |
| Ownership transfer | ownership transfer, `transferWorkspaceOwnership` | Handing a workspace to another of its members: that member becomes the owner and the previous owner becomes a workspace admin, at once. The owner role changes hands only this way. |
| Workspace content | workspace content | The databases, storages and notifiers of a workspace and everything attached to them: backup configs, backups, restores, healthchecks and verification configs. |
| Workspace action | workspace action, `workspaces.Action` | One thing a member can be allowed to do in a workspace, such as viewing it or editing its content. Each workspace role allows a fixed set of actions. |
| Instance admin | admin | A user with the global admin role, who manages users and instance settings. |
| Global role | global role, `ADMIN`, `MEMBER` | A user's role on the whole instance: `ADMIN` makes the user an instance admin, and `MEMBER` grants nothing beyond the user's workspace roles. The word "member" alone means a member of a workspace. |
| User | user | A person with an account on the instance: email, name, global role and status (`INVITED`, `ACTIVE` or `INACTIVE`). |
| Session | session | A signed-in user's right to call the API, carried as a token the sign-in returns once. A session carries the user's password version; every successful password replacement ends all earlier sessions, independently of time. |
| Password version | `PasswordVersion`, password version | A random UUID assigned to each user and rotated atomically with every password replacement. Sessions and pending sign-ins are usable only with the current version. |
| OAuth sign-in attempt | `OAuthSignInAttempt` | A single-use, time-limited browser record binding an external-provider callback to sign-in started in that tab. Unlike a server-side pending sign-in awaiting an emailed code, it binds OAuth state to the provider and redirect address and is consumed before code exchange. |
| Storage authorization attempt | `StorageAuthorizationAttempt` | A single-use, time-limited in-memory record binding a Google Drive response to the editor, account session, workspace, client settings and redirect address that initiated storage authorization. Unlike `OAuthSignInAttempt`, it authorizes storage access rather than account sign-in and introduces no API resource. |
| Pending sign-in | pending sign-in, `pendingSignInId` | A password sign-in that waits for the code emailed to the user, when the instance requires two-factor sign-in. |
| Instance settings | instance settings | The settings an instance admin sets for the whole instance: open registration, member invitations, workspace creation by members and two-factor sign-in. |
| Sign-in options | sign-in options | What the sign-in page offers, such as the first registration on an instance without users. |
| Invitation | invitation | A user with the status `INVITED`, created when a workspace adds an address that has no user. The user finishes the sign-up with that address. |
| Password reset | password reset | Replacing a forgotten password with a code emailed to the user. |
| Actor | actor | The user on whose behalf a request runs: user ID and global role. |
| Audit event | audit event | A record of who did what to which object, written in the transaction of the action, or right after an action outside the database. |
| Audit author | audit author | The user an audit event names as the one who acted, kept in the event as a snapshot of user ID, email and name. |
| Backup config | backup config, `backups.BackupConfig` | The settings that say how a database is backed up: whether backups run, the storage, the encryption, the notification triggers and the schedule and retention of its backup method. A database has one backup config, of the method its type implies. |
| Backup method | backup method, `backups.Method`, `LOGICAL`, `PHYSICAL` | How a database is backed up: a physical PostgreSQL database physically, every other database logically. |
| Notification trigger | notification trigger, `backups.NotificationTrigger`; `NotificationTrigger` and `VerificationNotificationTrigger` in the API | A backup or verification outcome for which its config sends a notification through the database's notifiers. |
| Logical backup | logical backup | A dump produced by the engine's dump tool (`pg_dump`, `mysqldump`, `mariadb-dump`, `mongodump`). |
| Physical backup | physical backup | A copy of a PostgreSQL cluster's files taken over the replication protocol. Either full or incremental. |
| Base backup | base backup, `StreamBaseBackup` | The `pg_basebackup` stream of one full or incremental physical backup. |
| Full backup | full backup, `FULL` | A physical backup that contains the whole cluster. |
| Incremental backup | incremental backup, `INCREMENTAL` | A physical backup that contains only the blocks changed since its parent backup. |
| Chain | chain | A full backup with the incremental backups built on it. |
| WAL segment | WAL segment | One file of PostgreSQL's write-ahead log, archived to the storage after it is completed. |
| WAL range | `WAL` row | Consecutive archived WAL segments shown as one row in the physical backup list. |
| Timeline history file | timeline history file | The `.history` file PostgreSQL writes when a timeline starts, archived beside the WAL segments. |
| Running backup | `IN_PROGRESS`, `uq_backups_physical_database_in_progress` | The one full or incremental backup row that a database may have in progress. |
| WAL stream sample | `backups_physical_wal_stream_samples` | The state of a WAL stream recorded every 30 seconds, from which its health is computed. |
| WAL stream | WAL stream | Continuous archiving of a database's WAL segments by `pg_receivewal`. |
| WAL stream manager | `walstreaming.Manager` | The process that starts, stops and monitors the WAL streamers of all databases. |
| WAL streamer | WAL streamer, `walstreaming.Streamer` | The process that runs the WAL stream of one database. |
| Forced WAL rotation | `ForcedRotationInterval` | A WAL switch Databasus asks the source for, so a segment completes even when the source writes rarely. |
| Archive staleness threshold | `ArchiveStalenessThreshold` | How long the archive progress point may stay still while the source writes before archiving counts as fallen behind. |
| Local WAL queue | local WAL queue, `WalQueueDir`, `LocalQueuePauseBytes` | The directory where `pg_receivewal` writes segments before they are uploaded. |
| Replication slot | slot | A slot on the source that keeps WAL until Databasus has received it. |
| Timeline | timeline | PostgreSQL's history branch number, which changes when a replica is promoted. |
| Restore set | restore set | The files needed to restore a database to a moment: a full backup, its incrementals and the WAL up to the target. |
| Recovery script | recovery script | The script Databasus serves that rebuilds a cluster from a downloaded restore set. |
| Restore | restore | Putting a backup's data into a target database. |
| Retention | retention | The rule that decides which backups are kept, including the grandfather-father-son (GFS) policy. |
| Schedule | schedule | When something runs: every N hours, daily, weekly, monthly or a cron expression. |
| Verification | verification, `verification_runs` | A check that a backup restores: the agent restores it into a disposable container and compares what it finds. Table names call one verification a run because the module prefix already says "verification". |
| Verification config | verification config, `VerificationConfig` | The settings of a database that decide when its backups are verified automatically and which outcomes notify. |
| Verification trigger | verification trigger, `VerificationTrigger` | What started a verification: a user, the interval schedule or a completed backup. |
| Verification agent | verification agent, `VerificationAgent` in the API | The program on the user's machine that runs verifications. |
| Assignment eligibility | assignment eligibility | Whether a verification agent has accepted contact recent enough to be considered for new work, judged from `lastSeenAt` when it calls `claimVerification`. Registration differs: it keeps the agent listed and counted whatever its contact. Capacity differs too: it decides whether one particular verification fits the agent's resource budget. No field stores eligibility. |
| Instance health | `HealthReport`, `HealthProblem`, `getInstanceHealth` | Whether this instance’s local dependencies, verification-agent freshness and periodic work pass their monitoring checks; distinct from a healthcheck of a user’s database. |
| Healthcheck | healthcheck | A scheduled check that a user's database answers. |
| Healthcheck config | `HealthcheckConfig` | The settings of a database that decide whether and how often it is checked, how many failed checks make it unavailable, whether status changes notify and how long attempts are kept. |
| Healthcheck attempt | `HealthcheckAttempt`, `healthcheck_attempts` | The recorded result of one availability check of a database. |
| Master key | master key | The secret in the data volume that encrypts stored secrets and backup files. |
| Download token | download token, `createDownloadToken` | A single-use token that lets the browser download one logical backup. |
| Restore target | restore target, `RestoreTarget` | The database a restore writes into, given with its connection settings and restore options. |
| Connection draft | `DatabaseConnectionDraft`: `DatabaseCreateDraft`, `DatabaseEditDraft`, `RestoreTargetDraft` | The connection settings being entered in the interface for a new database, a saved database or a restore target, before any of them is sent. |
| Physical backup config draft | `PhysicalBackupConfigDraft` | The physical backup config being entered in the interface. Unlike `PhysicalBackupConfigInput`, it always holds a full-backup retention and keeps the values of the policies not chosen, so switching back shows them. Only the input is sent. |
| Physical database backup type | `PhysicalDatabaseBackupType`: `FULL`, `FULL_INCREMENTAL`, `FULL_INCREMENTAL_WAL_STREAM` | The backup strategy chosen for a physical database, which decides the schedules and retention its config may use. A list item's `PhysicalBackupType` instead names what one stored backup is. |
| User stream | user stream, `backups.UserStream` | The one heavy download a user may run at a time: a backup download or a physical restore stream. |
| Cancellation registry | cancellation registry, `background.CancellationRegistry` | The cancel functions of running work, keyed by its ID. |
| Restore token | restore token, `createPhysicalRestoreToken` | A single-use token that lets a user download one physical restore set. |
| Restore token subject | restore token subject, `RestoreTokenSubject` | What a restore token opens: a database restored to a target time or its latest restorable moment, or one full or incremental backup with its ancestors. |

## Code

| Term | In code | Meaning |
|---|---|---|
| Release gate | `is-release-gate` | The set of CI checks a commit on `main` must pass before a version is selected. |
| Release analysis | `.github/workflows/ci-release.yml:analyze` | The CodeQL analysis that runs as part of the release gate. |
| Severity gate | `codeql-severity-gate` | The release gate job that reads CodeQL alerts by severity and stops the release on an open critical one. |
| Browser end-to-end suite | `e2e/browser` | The Python suite that tests the application image as a black box through its screens. It imports nothing from the backend or the frontend. |
| Critical path | `test_critical_path.py` | The path a new user depends on, tested through the screens of the application image: first instance admin, sign-in, a PostgreSQL database added through the wizard, its first logical backup, a restore into an empty database and a comparison of the rows. |
| Supervisor | `e2e/browser/supervisor.py` | The process that owns one run of the browser end-to-end suite: it starts the run's compose project and `pytest`, masks the output and removes what the run started. |
| Lifecycle probe | `e2e/browser/lifecycle` | A whole run of the browser end-to-end suite through its public command that fails, gets interrupted or overlaps on purpose, to check its exit status, its output and that nothing of it remains. |
| Image check | `scripts/check_image.sh` | The smoke checks of a built application image plus its Trivy scans. |
| Smoke check | smoke check | One start-and-probe check of a running image. |
| CodeQL alert | code scanning alert | A result of the static analysis, which can be dismissed on GitHub. |
| Secret scan finding | gitleaks finding | A gitleaks result, which can only be allowlisted in `.gitleaksignore`. |
| Review client | review client | The CLI that runs AI review and apply consultations: `codex` or `claude`, selected explicitly or by the active agent environment. |
| Review profile | `ReviewProfile` | The model selection, reasoning effort, review contract and execution policy that identify which full-review findings and acceptance can be reused. |
| Module | module | A part of the backend that owns one domain area and its tables. Only its `module.go` is visible from outside. |
| Shared code | lib | Packages in `internal/lib` without domain logic. |
| Composition root | app | `internal/app`, which builds the whole object graph and wires modules together. |
| Connection settings | `*ConnectionSettings` of `databases` | The decrypted settings of one database with its type's options, which the databases module hands to the code that connects. |
| Engine source | `Source` of an engine package | An engine's handle on one user database, built from plain connection settings with the secrets in clear text. |
| Ignorable error | `pgrestore.IgnorableErrorRule`, `HasOnlyIgnorableErrors` | A `pg_restore` error that a restore or verification still counts as success, under a rule the caller chooses. |
| Client tool bundle | `clienttools.Bundle` | The vendored client tools of one engine version: their directory and the commands they must hold. |
| API contract | contract | The user API contract (`backend/api/openapi.yaml`) describes the SPA and script operations. The agent API contract (`backend/api/agent.yaml`) describes the verification agent operations. Each generates its servers and clients; the user contract also generates frontend types. |
| Problem | Problem | The RFC 9457 error body that every operation of the contract answers with. |
| Error code | `code` | The machine-readable cause in a `Problem`, such as `not_found`. The frontend chooses the text it shows by it. |
| Contract-checking test client | `apiclient`, `agentclienttest` | The clients backend tests use for the user and agent APIs. Each fails the test on any response that differs from its contract. |
| Contract check | `contracttest` | The response check both contract-checking test clients share. |
| Access mark | `x-access` | The mark of a contract operation that says who may call it: `public`, `admin`, or a signed-in user when the mark is absent. The contract guard reads it, and the users module enforces it. |
| Handler | handler | Code that decodes a request, calls a service and encodes the response. |
| Service | service | Code that holds the rules of a module's operations. |
| Repository | repository | Code that holds the queries of a module's tables. |
| Event | event | A notice a module publishes so that higher modules act in the same transaction, for example before a workspace is deleted. |
| After-commit hook | `database.AfterCommit` | Work outside the database, such as dropping a replication slot, that runs only after the transaction that registered it commits. |
| Job snapshot | `background.JobSnapshot`, `Runner.GetSnapshots` | A copied observation of one periodic job’s running state and latest start and completion, readable without waiting for the job. |
| Permitted pass duration | `MaxPassDuration`, `maxPassDurationSeconds` | The monitoring budget for a pass and its completion deadline; expiration affects instance health and does not cancel execution. |
| Background job | job | Work that `background.Runner` runs on a schedule. Each run of a job is called a pass. |
| Pending backup request | `ForceFullRequestedAt`, `ForceIncrementalRequestedAt` | A request timestamp in a physical database's backup config, written by a manual start, a WAL slot rebuild or a storage switch. |
| Full scheduler pass | `backups_physical_scheduler` | A pass of the physical backup scheduler that evaluates every enabled backup config. |
| Backup request job | `backups_physical_request_scheduler` | The background job that evaluates only the backup configs with a pending backup request, between full scheduler passes. |
| File store | file store | The storages module's only way to write, read and delete files in a storage. It records every write so that no file is left without an owner. |
| Write receipt | write receipt | What the file store returns for a written file. The transaction that publishes the backup spends it to keep the file. |
| Stored file reference | stored file reference, `storages.StoredFileReference` | The storage and the name of one stored file, which backup code keeps to read or delete the file later. |
| Storage reader | storage reader, `storages.StorageReader` | Opens the files of one storage for reading, so a restore or a download reads many files without looking the storage up for each. |
| Background write | background write, `storages.BackgroundWrite` | A write of a file that runs beside the producer feeding it through a pipe, as a dump does; it ends with a write receipt or an error. |
| Cleanup scope | `CleanupScope` | A database and the workspace that accepted its storage reassignment. Current cleanup counts include only outstanding files attributed to that pair. |
| Pending deletion | pending deletion, `PendingDeletion` | One durable queue entry per storage and file name, retried until confirmed absence or explicit storage removal. Reassignment attribution is optional; protected uploads use the same queue. |
| Deletion worker | deletion worker | The background job of the storages module that deletes files with a pending deletion. |
| Console command | console command | An operation of the Databasus binary that the operator runs on the host or in the container instead of starting the server, such as a password reset. The `healthcheck` subcommand is not one: it only asks the running server. |
| Migration | migration | A goose SQL file in `backend/migrations`, the only way to change the schema. |
| Test application | test app | A backend instance built for one test by `apptest.New`, with its own database, clock and fakes. |
