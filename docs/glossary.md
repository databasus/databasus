# Glossary

This is the project's common language. Shared concepts have one term in documentation, the interface, code and the API. Define a new shared concept in the change that introduces it; internal helpers need no glossary entry. Definitions explain meaning and distinctions, while [architecture](architecture.md) holds the rules and [API contracts](../backend/api) hold exact field names and values.

## Product

| Term | Meaning |
| --- | --- |
| Database | A user's database that Databasus connects to and backs up. The term never means Databasus's own database. |
| Internal database | The embedded PostgreSQL inside the Databasus container that holds Databasus's own state. |
| Engine | A kind of database server Databasus supports: PostgreSQL, MySQL, MariaDB or MongoDB. The term also names the code in `internal/engines` that drives it. |
| Storage | A place where backups are kept: local disk, S3, NAS, Azure Blob, FTP, SFTP. |
| SFTP authentication method | A credential kind selected for one SFTP storage: password or private key. A storage can select both. |
| Notifier | A channel Databasus sends notifications to: email, Telegram, Slack, Discord, Teams, Mattermost, webhook. |
| Notification | A message about an event of a database, with a category, a heading and a text, which Databasus sends to the notifiers of that database. |
| Notification category | What a notification is about, such as a failed backup. A webhook notifier can accept only some categories. |
| Delivery | One send of one notification to one notifier, ended by a timeout. Its outcome becomes the notifier's last send error. |
| Provider | One implementation of a database, storage or notifier type, with its own settings table. |
| Limited user | A user that Databasus creates on the user's server for its backups: a read-only user, or for physical PostgreSQL a replication-only user. |
| Resource address | The settings of a saved storage, notifier or database that decide where its stored secrets go, such as a host and port or an endpoint. A stored secret fills in only while the address stays the stored one. The `secret-handling` spec lists them per type. |
| Workspace | A group of users and the databases, storages and notifiers they share. |
| Member | A user in a workspace, with a workspace role. |
| Member ID | The ID of one member of one workspace. The API uses it to change or remove that member and to name the new owner in an ownership transfer; it is not the user ID. |
| Workspace role | What a member may do in a workspace. `WORKSPACE_MEMBER` is the role that may edit the workspace content; the word "member" alone means any user in a workspace, whatever the role. |
| Workspace admin | A member who may also rename the workspace and manage the members who are not admins or the owner. The word `admin` alone in code and text means the instance admin; `access.RoleAdmin` and `ActionManageAdmins` name workspace admins. |
| Owner | The member who may also manage workspace admins, transfer ownership and delete the workspace. |
| Ownership transfer | Handing a workspace to another of its members: that member becomes the owner and the previous owner becomes a workspace admin, at once. The owner role changes hands only this way. |
| Workspace content | The databases, storages and notifiers of a workspace and everything attached to them: backup configs, backups, restores, healthchecks and verification configs. |
| Workspace action | One thing a member can be allowed to do in a workspace, such as viewing it or editing its content. Each workspace role allows a fixed set of actions. |
| Instance admin | A user with the global admin role, who manages users and instance settings. |
| Global role | A user's role on the whole instance: `ADMIN` makes the user an instance admin, and `MEMBER` grants nothing beyond the user's workspace roles. The word "member" alone means a member of a workspace. |
| User | A person with an account on the instance: email, name, global role and status (`INVITED`, `ACTIVE` or `INACTIVE`). |
| Session | A signed-in user's right to call the API, carried as a token the sign-in returns once. A session carries the user's password version; every successful password replacement ends all earlier sessions, independently of time. |
| Password version | A random UUID assigned to each user and rotated atomically with every password replacement. Sessions and pending sign-ins are usable only with the current version. |
| OAuth sign-in attempt | A single-use, time-limited browser record binding an external-provider callback to sign-in started in that tab. Unlike a server-side pending sign-in awaiting an emailed code, it binds OAuth state to the provider and redirect address and is consumed before code exchange. |
| Pending sign-in | A password sign-in that waits for the code emailed to the user, when the instance requires two-factor sign-in. |
| Instance settings | The settings an instance admin sets for the whole instance: open registration, member invitations, workspace creation by members and two-factor sign-in. |
| Sign-in options | What the sign-in page offers, such as the first registration on an instance without users. |
| Invitation | A user with the status `INVITED`, created when a workspace adds an address that has no user. The user finishes the sign-up with that address. |
| Password reset | Replacing a forgotten password with a code emailed to the user. |
| Actor | The user on whose behalf a request runs: user ID and global role. |
| Audit event | A record of who did what to which object, written in the transaction of the action, or right after an action outside the database. |
| Audit author | The user an audit event names as the one who acted, kept in the event as a snapshot of user ID, email and name. |
| Backup config | The settings that say how a database is backed up: whether backups run, the storage, the encryption, the notification triggers and the schedule and retention of its backup method. A database has one backup config, of the method its type implies. |
| Backup usage | The catalog-recorded stored size of retained completed backups and committed archived WAL for a database or storage, rather than actual provider consumption. |
| Backup method | How a database is backed up: a physical PostgreSQL database physically, every other database logically. |
| Notification trigger | A backup or verification outcome for which its config sends a notification through the database's notifiers. |
| Row-level security (RLS) | PostgreSQL policies that restrict which table rows a role can access. |
| Logical backup | A dump produced by the engine's dump tool (`pg_dump`, `mysqldump`, `mariadb-dump`, `mongodump`). |
| Physical backup | A copy of a PostgreSQL cluster's files taken over the replication protocol. Either full or incremental. |
| Base backup | The `pg_basebackup` stream of one full or incremental physical backup. |
| Full backup | A physical backup that contains the whole cluster. |
| Incremental backup | A physical backup that contains only the blocks changed since its parent backup. |
| Chain | A full backup with the incremental backups built on it. |
| WAL segment | One file of PostgreSQL's write-ahead log, archived to the storage after it is completed. |
| WAL range | Consecutive archived WAL segments shown as one row in the physical backup list. |
| Timeline history file | The `.history` file PostgreSQL writes when a timeline starts, archived beside the WAL segments. |
| Running backup | The one full or incremental backup row that a database may have in progress. |
| WAL stream sample | A periodic observation of WAL streaming used to compute reported health. |
| WAL stream | Continuous archiving of a database's WAL segments by `pg_receivewal`. |
| WAL stream manager | The process that starts, stops and monitors the WAL streamers of all databases. |
| WAL streamer | The process that runs the WAL stream of one database. |
| Forced WAL rotation | A WAL switch Databasus asks the source for, so a segment completes even when the source writes rarely. |
| Archive staleness threshold | How long the archive progress point may stay still while the source writes before archiving counts as fallen behind. |
| Local WAL queue | The directory where `pg_receivewal` writes segments before they are uploaded. |
| Replication slot | A slot on the source that keeps WAL until Databasus has received it. |
| Primary | The PostgreSQL node accepting writes and serving physical backups and WAL reception. |
| Standby | A PostgreSQL node replaying WAL from the primary until it is promoted. |
| Promotion | Making a standby writable on a new PostgreSQL timeline. |
| Switchover | A planned transfer of the primary role to a standby. |
| Failover | Electing and promoting a standby after the primary becomes unavailable. |
| Distributed configuration store (DCS) | The shared store Patroni uses to coordinate configuration and primary ownership, such as etcd. |
| CloudNativePG (CNPG) | The Kubernetes operator that runs a PostgreSQL cluster as pods and chooses its primary. "CNPG operator" means this program; "operator" alone means the person who runs Databasus. CNPG calls a PostgreSQL node an instance; Databasus reserves instance for a Databasus instance. |
| Kubernetes Service | The Kubernetes object that routes connections to the pods its selector matches, such as CNPG's `-rw` Kubernetes Service, which follows the primary. It is always written in full, because "Service" alone means a module's service code. |
| Point-in-time recovery (PITR) | Replaying archived WAL after a physical backup up to a selected recoverable time. |
| Timeline | PostgreSQL's history branch number, which changes when a replica is promoted. |
| Restore set | The files needed to restore a database to a moment: a full backup, its incrementals and the WAL up to the target. |
| Recovery script | The script Databasus serves that rebuilds a cluster from a downloaded restore set. |
| Restore | Putting a backup's data into a target database. |
| Retention | The rule that decides which backups are kept, including the grandfather-father-son (GFS) policy. |
| GFS candidate | The newest successful eligible backup protected for the current week, month or year of an enabled GFS level, in addition to its completed-period slots. Only a newer success in that period replaces it. |
| GFS representative | The newest successful eligible backup selected for a calendar period. A completed weekly, monthly or yearly representative occupies a completed-period slot; the protected selection for a current long period is a GFS candidate. |
| GFS reference instant | The creation time of the newest successful eligible backup, which determines current UTC calendar periods. Logical retention uses completed logical backups; physical full-backup retention uses completed full backups. Failed, cancelled, running and, for full-backup retention, incremental backups do not advance it. |
| Schedule | When something runs: every N hours, daily, weekly, monthly or a cron expression. |
| Verification | A check that a backup restores: the agent restores it into a disposable container and compares what it finds. One execution of this check is a verification run. |
| Failure diagnostic | The bounded, sanitized diagnostic of a verification's final failed attempt. It is separate from the short failure summary (`failMessage`) used in lists and notifications. |
| Verification config | The settings of a database that decide when its backups are verified automatically and which outcomes notify. |
| Verification trigger | What started a verification: a user, the interval schedule or a completed backup. |
| Verification agent | The program on the user's machine that runs verifications. |
| Assignment eligibility | Whether a verification agent has contacted the server recently enough to receive new work. Registration keeps the agent listed regardless of contact; capacity determines whether a particular verification fits its resource budget. |
| Instance health | Whether this instance’s local dependencies, verification-agent freshness and periodic work pass their monitoring checks; distinct from a healthcheck of a user’s database. |
| Healthcheck | A scheduled check that a user's database answers. |
| Healthcheck config | The settings of a database that decide whether and how often it is checked, how many failed checks make it unavailable, whether status changes notify and how long attempts are kept. |
| Healthcheck attempt | The recorded result of one availability check of a database. |
| Master key | The secret in the data volume that encrypts stored secrets and backup files. |
| Download token | A single-use token that lets the browser download one logical backup. |
| Restore target | The database a restore writes into, given with its connection settings and restore options. |
| Connection draft | The connection settings being entered in the interface for a new database, a saved database or a restore target, before any of them is sent. |
| Physical database backup type | The configured strategy: full backups, full plus incremental backups, or both with continuous WAL archiving. It determines schedules and retention; the type of an individual stored backup instead distinguishes full from incremental. |
| User stream | The one heavy download a user may run at a time: a backup download or a physical restore stream. |
| Cancellation registry | The cancel functions of running work, keyed by its ID. |
| Restore token | A single-use token that lets a user download one physical restore set. |
| Restore token subject | What a restore token opens: a database restored to a target time or its latest restorable moment, or one full or incremental backup with its ancestors. |

## Code

| Term | Meaning |
| --- | --- |
| Release gate | The set of CI checks a commit on `main` must pass before a version is selected. |
| Severity gate | The release gate job that reads CodeQL alerts by severity and stops the release on an open critical one. |
| Browser end-to-end suite | The Python suite that tests the application image as a black box through its screens. It imports nothing from the backend or the frontend. |
| Critical path | The essential user journey whose failure prevents the main product outcome: configuring backups and recovering usable data. |
| Supervisor | The process that owns one run of the browser end-to-end suite: it starts the run's compose project and `pytest`, masks the output and removes what the run started. |
| Lifecycle probe | A whole run of the browser end-to-end suite through its public command that fails, gets interrupted or overlaps on purpose, to check its exit status, its output and that nothing of it remains. |
| Crashed-run probe | A lifecycle probe that kills a run or prevents its cleanup, so that a later run must remove what remains. Unlike other lifecycle probes, it runs only while its caller starts no other run, because any run starting meanwhile would remove the remains first. |
| Image check | The smoke checks of a built application image plus its Trivy scans. |
| Smoke check | One start-and-probe check of a running image. |
| CodeQL alert | A result of the static analysis, which can be dismissed on GitHub. |
| Secret scan finding | A gitleaks result, which can only be allowlisted in `.gitleaksignore`. |
| Review client | The CLI that runs AI review and apply consultations: `codex` or `claude`, selected explicitly or by the active agent environment. |
| Review profile | The model selection, reasoning effort, review contract and execution policy that identify which full-review findings and acceptance can be reused. |
| Module | A part of the backend that owns one domain area and its tables. Only its `module.go` is visible from outside. |
| Shared code | Packages in `internal/lib` without domain logic. |
| Composition root | `internal/app`, which builds the whole object graph and wires modules together. |
| Connection settings | The decrypted settings of one database with its type's options, which the databases module hands to the code that connects. |
| Engine source | An engine's handle on one user database, built from plain connection settings with the secrets in clear text. |
| Ignorable error | A `pg_restore` error that a restore or verification still counts as success, under a rule the caller chooses. |
| Client tool bundle | The vendored client tools of one engine version: their directory and the commands they must hold. |
| API contract | The user API contract (`backend/api/openapi.yaml`) describes the SPA and script operations. The agent API contract (`backend/api/agent.yaml`) describes the verification agent operations. Each generates its servers and clients; the user contract also generates frontend types. |
| Problem | The RFC 9457 error body that every operation of the contract answers with. |
| Error code | The machine-readable cause in a `Problem`, such as `not_found`. The frontend chooses the text it shows by it. |
| Contract-checking test client | A test client that rejects HTTP responses whose status, headers or body differ from the declared user or agent contract. |
| Access mark | The mark of a contract operation that says who may call it: `public`, `admin`, or a signed-in user when the mark is absent. The contract guard reads it, and the users module enforces it. |
| Handler | Code that decodes a request, calls a service and encodes the response. |
| Service | Code that holds the rules of a module's operations. |
| Repository | Code that holds the queries of a module's tables. |
| Event | A notice a module publishes so that higher modules act in the same transaction, for example before a workspace is deleted. |
| After-commit hook | Work outside the database, such as dropping a replication slot, that runs only after the transaction that registered it commits. |
| Job snapshot | A copied observation of one periodic job’s running state and latest start and completion, readable without waiting for the job. |
| Permitted pass duration | The monitoring budget for a pass and its completion deadline; expiration affects instance health and does not cancel execution. |
| Background job | Work that `background.Runner` runs on a schedule. Each run of a job is called a pass. |
| Pending backup request | A request timestamp in a physical database's backup config, written by a manual start, a WAL slot rebuild or a storage switch. |
| Full scheduler pass | A pass of the physical backup scheduler that evaluates every enabled backup config. |
| Backup request job | The background job that evaluates only the backup configs with a pending backup request, between full scheduler passes. |
| File store | The storages module's only way to write, read and delete files in a storage. It records every write so that no file is left without an owner. |
| Write receipt | What the file store returns for a written file. The transaction that publishes the backup spends it to keep the file. |
| Stored file reference | The storage and the name of one stored file, which backup code keeps to read or delete the file later. |
| Storage reader | Opens the files of one storage for reading, so a restore or a download reads many files without looking the storage up for each. |
| Cleanup scope | A database and the workspace that accepted its storage reassignment. Current cleanup counts include only outstanding files attributed to that pair. |
| Pending deletion | One durable queue entry per storage and file name, retried until confirmed absence or explicit storage removal. Reassignment attribution is optional; protected uploads use the same queue. |
| Deletion worker | The background job of the storages module that deletes files with a pending deletion. |
| Console command | An operation of the Databasus binary that the operator runs on the host or in the container instead of starting the server, such as a password reset. The `healthcheck` subcommand is not one: it only asks the running server. |
| Migration | A goose SQL file in `backend/migrations`, the only way to change the schema. |
| Test application | A real backend instance isolated for one test, with its own database, clock, lifecycle and permitted external substitutes. |

## Dependency maintenance

| Term | Meaning |
| --- | --- |
| Dependency verification | Checks of known local pins, related declarations and committed client hashes. Native package managers verify their own locked graphs and downloaded content; the updater reviews release age, upstream changes and compatibility. |
| Release hold | The normal waiting period used when selecting a dependency update. The upgrade guide defines its duration and how the updater reports missing age information or a departure from that period. |

The [dependency upgrade guide](how-to-upgrade-deps.md) describes the procedure, sources and verification limits.
