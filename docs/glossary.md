# Glossary

This is the project's common language. Each shared concept has one term, used in the documentation, the interface, the code and the API. Define a new shared concept in the change that introduces it. Internal helpers need no glossary entry.

Definitions explain meaning and distinctions. [Architecture](architecture.md) holds the rules, and [API contracts](../backend/api) hold exact field names and values.

## Product

| Term | Meaning |
| --- | --- |
| Database | A user's database that Databasus connects to and backs up. The term never means Databasus's own database. |
| Internal database | The embedded PostgreSQL inside the Databasus container that holds Databasus's own state. |
| Engine | A kind of database server Databasus supports: PostgreSQL, MySQL, MariaDB or MongoDB. The term also names the code in `internal/engines` that drives it. |
| Storage | A place where backups are kept: local disk, S3, NAS, Azure Blob, FTP or SFTP. |
| SFTP authentication method | The kind of credential selected for one SFTP storage: a password or a private key. A storage can select both. |
| Notifier | A channel Databasus sends notifications to: email, Telegram, Slack, Discord, Teams, Mattermost or webhook. |
| Notification | A message about an event of a database. It has a category, a heading and a text, and Databasus sends it to the notifiers of that database. |
| Notification category | What a notification is about, such as a failed backup. A webhook notifier can accept only some categories. |
| Delivery | One send of one notification to one notifier, limited by a timeout. Its outcome becomes the notifier's last send error. |
| Provider | One implementation of a database, storage or notifier type, with its own settings table. |
| Managed object storage service | An S3-compatible object storage that a provider runs for the user, such as Neon Object Storage. Databasus saves it as an ordinary S3 storage. The storage form merely offers it as a separate choice, with its own credential paste and guide. Such a service may accept fewer S3 requests and smaller objects than AWS S3. |
| Managed PostgreSQL service | A PostgreSQL server a provider runs for the user, such as Neon, Supabase, Amazon RDS or Azure. The user gets an owner role that is not a superuser and cannot change the server's files. Some settings and physical replication therefore depend on what the provider allows. |
| Neon host | The configured host of a database when that host is `neon.tech` or a subdomain of it. The SSH tunnel's address does not count. It matters for two reasons: Neon accepts no physical replication, and it suspends idle compute and bills compute only while it runs. |
| Transaction-mode pooler | A connection pooler, such as PgBouncer in transaction mode, that may hand a client to another server process between transactions. Settings made once per session can be lost there, and `pg_dump` and `pg_restore` rely on such settings. A session-mode pooler keeps one server process for the whole client session, like a direct connection. Neon's `-pooler` hosts and the Supabase pooler on port 6543 are transaction-mode poolers. |
| Limited user | A user that Databasus creates on the user's server for its backups. It is a read-only user, or a replication-only user for physical PostgreSQL. |
| Resource address | The settings of a saved storage, notifier or database that decide where its stored secrets go, such as a host and port or an endpoint. A stored secret is filled in only while the address stays the stored one. The `secret-handling` spec lists these settings for each type. |
| Workspace | A group of users and the databases, storages and notifiers they share. |
| Member | A user in a workspace, with a workspace role. |
| Member ID | The ID of one member of one workspace. It is not the user ID. The API uses it to change or remove that member and to name the new owner in an ownership transfer. |
| Workspace role | What a member may do in a workspace. `WORKSPACE_MEMBER` is the role that may edit the workspace content. The word "member" alone means any user in a workspace, whatever the role. |
| Workspace admin | A member who may also rename the workspace and manage the members who are neither admins nor the owner. In code and text, the word `admin` alone means the instance admin. `access.RoleAdmin` and `ActionManageAdmins` name workspace admins. |
| Owner | The member who may also manage workspace admins, transfer ownership and delete the workspace. |
| Ownership transfer | Handing a workspace to another of its members. At once, that member becomes the owner and the previous owner becomes a workspace admin. This is the only way the owner role changes hands. |
| Workspace content | The databases, storages and notifiers of a workspace and everything attached to them: backup configs, backups, restores, healthchecks and verification configs. |
| Workspace action | One thing a member can be allowed to do in a workspace, such as viewing it or editing its content. Each workspace role allows a fixed set of actions. |
| Instance admin | A user with the global admin role, who manages users and instance settings. |
| Root admin | The user persistently designated as the instance's bootstrap administrator. Only root admin manages Databasus backups, exports the master key and recovers the instance. Root admin is an active instance admin who also carries the designation. The global `ADMIN` role alone does not make an instance admin root admin. |
| Global role | A user's role on the whole instance. `ADMIN` makes the user an instance admin. `MEMBER` grants nothing beyond the user's workspace roles. The word "member" alone means a member of a workspace. |
| User | A person with an account on the instance: email, name, global role and status (`INVITED`, `ACTIVE` or `INACTIVE`). |
| Session | A signed-in user's right to call the API. It is carried as a token that the sign-in returns once. A session carries the user's password version. Every successful password replacement ends all earlier sessions, even those that have not expired. |
| Password version | A random UUID assigned to each user. It is rotated atomically with every password replacement. Sessions and pending sign-ins are usable only with the current version. |
| OAuth sign-in attempt | A single-use, time-limited browser record. It binds an external provider's callback to the sign-in started in that tab. Unlike a pending sign-in, which waits on the server for a second factor, it binds the OAuth state to the provider and the redirect address. It is consumed before the code exchange, where the provider's authorization code is traded for tokens. |
| Pending sign-in | A password sign-in that waits for its second factor. The factor is the code emailed to the user, an app code from the user's authenticator app, or the authenticator setup the instance requires. |
| Instance settings | The settings an instance admin sets for the whole instance: open registration, member invitations, workspace creation by members and the two-factor requirement. |
| Two-factor requirement | The instance setting that says whether every password sign-in needs a second factor, and which one: none, an emailed code or an authenticator app. It applies to administrators too. A user with an authenticator app is asked for an app code whatever the requirement says. |
| Authenticator app | The app on a user's phone or password manager that holds the user's TOTP (time-based one-time password) secret. It shows a new six-digit code every 30 seconds. A user has at most one. The secret is stored encrypted with the master key. |
| App code | The six-digit code an authenticator app shows. Each one is accepted once. Wrong app codes and wrong recovery codes count toward a per-account limit. |
| Authenticator setup | Connecting an authenticator app. The instance issues a new secret, and the setup completes when the user enters a code the app computes from it. A setup expires, and it ends after five wrong codes. |
| Recovery code | One of ten single-use codes, each of which replaces an app code once. They are issued when an app is connected, and again on request. They are shown only in the answer that issues them. |
| Sign-in options | What the sign-in page offers, such as the first registration on an instance without users. |
| Invitation | A user with the status `INVITED`, created when a workspace adds an address that has no user. The user finishes the sign-up with that address. |
| Password reset | Replacing a forgotten password with a code emailed to the user. |
| Actor | The user on whose behalf a request runs: user ID and global role. |
| Audit event | A record of who did what to which object. It is written in the transaction of the action, or right after an action that happens outside the database. |
| Audit author | The user an audit event names as the one who acted. The event keeps a snapshot of their user ID, email and name. |
| Backup config | The settings that say how a database is backed up. They cover whether backups run, the storage, the encryption and the notification triggers. They also hold the schedule and retention of its backup method. A database has one backup config, of the method its type implies. |
| Backup usage | The stored size of the retained completed backups and committed archived WAL of a database or storage, as the catalog records it. It is not the provider's actual consumption. |
| Backup method | How a database is backed up: a physical PostgreSQL database physically, every other database logically. |
| Notification trigger | A backup or verification outcome that makes its config send a notification through the database's notifiers. |
| Row-level security (RLS) | PostgreSQL policies that restrict which table rows a role can access. |
| Logical backup | A dump produced by the engine's dump tool (`pg_dump`, `mysqldump`, `mariadb-dump`, `mongodump`). |
| Physical backup | A copy of a PostgreSQL cluster's files taken over the replication protocol. It is either full or incremental. |
| Base backup | The `pg_basebackup` stream of one full or incremental physical backup. |
| Full backup | A physical backup that contains the whole cluster. |
| Incremental backup | A physical backup that contains only the blocks changed since its parent backup. |
| Chain | A full backup with the incremental backups built on it. |
| WAL segment | One file of PostgreSQL's write-ahead log (WAL), the log of every change to the cluster. It is archived to the storage after it is completed. |
| WAL range | Consecutive archived WAL segments shown as one row in the physical backup list. |
| Timeline history file | The `.history` file PostgreSQL writes when a timeline starts, archived beside the WAL segments. |
| Running backup | The one full or incremental backup row that a database may have in progress. |
| WAL stream sample | A periodic observation of WAL streaming used to compute reported health. |
| WAL stream | Continuous archiving of a database's WAL segments by `pg_receivewal`. |
| WAL stream manager | The process that starts, stops and monitors the WAL streamers of all databases. |
| WAL streamer | The process that runs the WAL stream of one database. |
| Forced WAL rotation | A WAL switch that Databasus asks the source for, so a segment completes even when the source writes rarely. |
| Archive staleness threshold | How long the archive progress point may stay still while the source writes. After that, archiving counts as fallen behind. |
| Local WAL queue | The directory where `pg_receivewal` writes segments before they are uploaded. |
| Replication slot | A slot on the source that keeps WAL until Databasus has received it. |
| Primary | The PostgreSQL node accepting writes and serving physical backups and WAL reception. |
| Standby | A PostgreSQL node replaying WAL from the primary until it is promoted. |
| Promotion | Making a standby writable on a new PostgreSQL timeline. |
| Switchover | A planned transfer of the primary role to a standby. |
| Failover | Electing and promoting a standby after the primary becomes unavailable. |
| Distributed configuration store (DCS) | The shared store Patroni uses to coordinate configuration and primary ownership, such as etcd. |
| CloudNativePG (CNPG) | The Kubernetes operator that runs a PostgreSQL cluster as pods and chooses its primary. "CNPG operator" means this program, while "operator" alone means the person who runs Databasus. CNPG calls a PostgreSQL node an instance. Databasus reserves "instance" for a Databasus instance. |
| Kubernetes Service | The Kubernetes object that routes connections to the pods its selector matches. One example is CNPG's `-rw` Kubernetes Service, which follows the primary. The term is always written in full, because "Service" alone means a module's service code. |
| Point-in-time recovery (PITR) | Replaying archived WAL after a physical backup up to a selected recoverable time. |
| Timeline | PostgreSQL's history branch number, which changes when a replica is promoted. |
| Fork point | The WAL position at which a promoted timeline leaves its parent timeline, recorded in the promoted timeline's history file. |
| Timeline line | The WAL a chain can replay. It is its full backup's timeline up to the fork point, followed by each later timeline whose history is published. Each later timeline counts up to the next fork point. |
| Restore set | The files needed to restore a database to a moment: a full backup, its incrementals and the WAL up to the target. |
| Recovery script | The script Databasus serves that rebuilds a cluster from a downloaded restore set. |
| Restore | Putting a backup's data into a target database. Recovering Databasus's own state from a Databasus backup is a Databasus recovery, not a restore. |
| Retention | The rule that decides which backups are kept, including the grandfather-father-son (GFS) policy. |
| GFS candidate | The newest successful eligible backup protected for the current week, month or year of an enabled GFS level. It is protected in addition to that level's completed-period slots. Only a newer success in that period replaces it. |
| GFS representative | The newest successful eligible backup selected for a calendar period. A completed weekly, monthly or yearly representative occupies a completed-period slot. The protected selection for a current long period is a GFS candidate. |
| GFS reference instant | The creation time of the newest successful eligible backup. It determines the current UTC calendar periods. Logical retention uses completed logical backups, and physical full-backup retention uses completed full backups. Failed, cancelled and running backups do not advance it. For full-backup retention, incremental backups do not advance it either. |
| Schedule | When something runs: every N hours, daily, weekly, monthly or a cron expression. |
| Verification | A check that a backup restores. The agent restores it into a disposable container and compares what it finds. One execution of this check is a verification run. |
| Failure diagnostic | The bounded, sanitized diagnostic of a verification's final failed attempt. It is separate from the short failure summary (`failMessage`) used in lists and notifications. |
| Verification config | The settings of a database that decide when its backups are verified automatically and which outcomes notify. |
| Verification trigger | What started a verification: a user, the interval schedule or a completed backup. |
| Verification agent | The program on the user's machine that runs verifications. |
| Verification image | The container image a verification restores a PostgreSQL backup into. It is a PostgreSQL server with the extensions backups commonly use. The agent pulls it from its verification image repository: the Databasus one or another one its owner configures. The agent cleans up only that repository. |
| Pinned PostgreSQL version | The exact PostgreSQL server version, such as 18.7, that one release of the verification agent restores into for a given major version. A verification image of any other version is superseded. |
| Assignment eligibility | Whether a verification agent has contacted the server recently enough to receive new work. Registration keeps the agent listed regardless of contact. Capacity determines whether a particular verification fits its resource budget. |
| Restore peak | The most disk a verification's restore is expected to occupy at once. It counts the backup file, the restored database and a reserve for WAL and the empty cluster. It also counts the largest indexes built at the same time, one per parallel restore job. |
| Disk budget | The disk a verification may use on its agent, set from its restore peak. It is reserved from the assignment until the verification stops running. The agent stops a restore that reaches it. |
| Instance health | Whether this instance’s local dependencies, verification-agent freshness and periodic work pass their monitoring checks. It is distinct from a healthcheck of a user’s database. |
| Healthcheck | A scheduled check that a user's database answers. |
| Healthcheck config | The settings of a database that decide whether and how often it is checked and how many failed checks make it unavailable. They also decide whether status changes notify and how long attempts are kept. |
| Healthcheck attempt | The recorded result of one availability check of a database. |
| Master key | The secret in the data volume that encrypts stored secrets, backup files and Databasus backups. Root admin saves this same key outside the server, because a Databasus backup and the user-database backup files are useless without it. Exporting the key hands out the existing key and creates no second recovery secret. |
| Databasus backup | An encrypted snapshot of the internal database: accounts, settings, encrypted credentials and the backup catalog. It differs from a backup of a user's database. Its file contains neither the master key nor the backup files of user databases. |
| Databasus backup destination | One existing workspace storage currently selected for Databasus backups. It owns no provider settings or credentials. A selected storage cannot be deleted. |
| Databasus backup copy | One recorded delivery of one Databasus backup to one storage. A historical copy does not imply that the storage is still a destination. |
| Databasus backup health | Whether the destinations have sufficiently fresh successful copies. It differs from instance health and from healthchecks of user databases. |
| Databasus backup incident | One unresolved protection problem, such as a failed, partially saved, overdue or stopped Databasus backup. It carries the state of its system-mail delivery and reminders. It differs from notifications about user databases, which go through workspace notifiers. |
| Databasus recovery | An operation that replaces the instance's internal state and master key with a Databasus backup and its separately saved key. The instance then runs its work at once. A recovery starts either from mounted files at startup or from an archive file uploaded by root admin. It restores no user database and copies no backup files. |
| Schema identity | The digest of a release's migration bundle, together with the migration versions applied in a database. A Databasus backup records it, and recovery compares it with the running release. |
| Download token | A single-use token that lets the browser download one logical backup. |
| Restore target | The database a restore writes into, given with its connection settings and restore options. |
| Connection draft | The connection settings being entered in the interface, before they are sent. They can belong to a new database, a saved database or a restore target. |
| Physical database backup type | The configured strategy: full backups, full plus incremental backups, or both with continuous WAL archiving. It determines schedules and retention. The type of an individual stored backup is a different thing: it distinguishes full from incremental. |
| User stream | The one heavy download a user may run at a time: a backup download or a physical restore stream. |
| Cancellation registry | The cancel functions of running work, keyed by its ID. |
| Restore token | A single-use token that lets a user download one physical restore set. |
| Restore token subject | What a restore token opens. It is either a database restored to a target time or to its latest restorable moment, or one full or incremental backup with its ancestors. |

## Code

| Term | Meaning |
| --- | --- |
| Release gate | The set of CI checks a commit on `main` must pass before a version is selected. |
| Severity gate | The release gate job that reads CodeQL alerts by severity and stops the release on an open critical one. |
| Browser end-to-end suite | The Python suite that tests the application image as a black box through its screens. It imports nothing from the backend or the frontend. |
| Critical path | The essential user journey whose failure prevents the main product outcome: configuring backups and recovering usable data. |
| Supervisor | The process that owns one run of the browser end-to-end suite. It starts the run's compose project and `pytest`, masks the output and removes what the run started. |
| Lifecycle probe | A whole run of the browser end-to-end suite, through its public command, that fails, gets interrupted or overlaps on purpose. It checks the run's exit status, its output and that nothing of the run remains. |
| Crashed-run probe | A lifecycle probe that kills a run or prevents its cleanup, so that a later run must remove what remains. Unlike other lifecycle probes, it runs only while its caller starts no other run. Any run that starts meanwhile would remove the remains first. |
| Image check | The smoke checks of a built application image plus its Trivy scans. |
| Smoke check | One start-and-probe check of a running image. |
| CodeQL alert | A result of the static analysis, which can be dismissed on GitHub. |
| Secret scan finding | A gitleaks result, which can only be allowlisted in `.gitleaksignore`. |
| Review client | The CLI that runs AI reviews and apply consultations: `codex` or `claude`. It is selected explicitly or by the active agent environment. |
| Review profile | The model selection, reasoning effort, review contract and execution policy of a review. Together they identify which full-review findings and acceptance can be reused. |
| Hook tool check | The first part of `mise run doctor`. It reports the review client and whether the review hooks can find Python, git, OpenSpec and that client's CLI. |
| Development environment check | The second part of `mise run doctor`. It only reads the machine's state, reports each prerequisite on its own line and names the section of the [VM setup guide](how-to-setup-vm.md) that fixes a missing one. A prerequisite is a property of the machine that the project's commands and tests rely on, such as Docker, an ext4 root or the installed commit hook; any missing one fails the check. |
| Module | A part of the backend that owns one domain area and its tables. Only its `module.go` is visible from outside. |
| Shared code | Packages in `internal/lib` without domain logic. |
| Composition root | `internal/app`, which builds the whole object graph and wires modules together. |
| Connection settings | The decrypted settings of one database, with its type's options. The databases module hands them to the code that connects. |
| Engine source | An engine's handle on one user database, built from plain connection settings with the secrets in clear text. |
| Ignorable error | A `pg_restore` error that a restore or verification still counts as success, under a rule the caller chooses. |
| Client tool bundle | The vendored client tools of one engine version: their directory and the commands they must hold. |
| API contract | The user API contract (`backend/api/openapi.yaml`) describes the SPA and script operations. The agent API contract (`backend/api/agent.yaml`) describes the verification agent operations. Each generates its servers and clients, and the user contract also generates frontend types. |
| Problem | The RFC 9457 error body that every operation of the contract answers with. |
| Error code | The machine-readable cause in a `Problem`, such as `not_found`. The frontend uses it to choose the text it shows. |
| Contract-checking test client | A test client that rejects HTTP responses whose status, headers or body differ from the declared user or agent contract. |
| Access mark | The mark of a contract operation that says who may call it: `public` or `admin`. When the mark is absent, a signed-in user may call it. The contract guard reads the mark, and the users module enforces it. |
| Handler | Code that decodes a request, calls a service and encodes the response. |
| Service | Code that holds the rules of a module's operations. |
| Repository | Code that holds the queries of a module's tables. |
| Event | A notice a module publishes so that higher modules act in the same transaction, for example before a workspace is deleted. |
| After-commit hook | Work outside the database, such as dropping a replication slot, that runs only after the transaction that registered it commits. |
| Job snapshot | A copied observation of one periodic job’s running state and its latest start and completion. It can be read without waiting for the job. |
| Permitted pass duration | The monitoring budget for a pass and its completion deadline. When it expires, instance health is affected, but execution is not cancelled. |
| Background job | Work that `background.Runner` runs on a schedule. Each run of a job is called a pass. |
| Pending backup request | A request timestamp in a physical database's backup config. A manual start, a WAL slot rebuild or a storage switch writes it. |
| Full scheduler pass | A pass of the physical backup scheduler that evaluates every enabled backup config. |
| Backup request job | The background job that evaluates only the backup configs with a pending backup request, between full scheduler passes. |
| File store | The storages module's only way to write, read and delete files in a storage. It records every write so that no file is left without an owner. |
| Write receipt | What the file store returns for a written file. The transaction that publishes the backup spends it to keep the file. |
| Stored file reference | The storage and the name of one stored file, which backup code keeps to read or delete the file later. |
| Storage reader | Opens the files of one storage for reading. A restore or a download can then read many files without looking the storage up for each one. |
| Cleanup scope | A database and the workspace that accepted its storage reassignment. Current cleanup counts include only outstanding files attributed to that pair. |
| Pending deletion | One durable queue entry per storage and file name. It is retried until the file is confirmed absent or the storage is explicitly removed. Reassignment attribution is optional, and protected uploads use the same queue. |
| Deletion worker | The background job of the storages module that deletes files with a pending deletion. |
| Console command | An operation of the Databasus binary that the operator runs on the host or in the container instead of starting the server, such as a password reset. The `healthcheck` subcommand is not a console command, because it only asks the running server. |
| Migration | A goose SQL file in `backend/migrations`, the only way to change the schema. |
| Test application | A real backend instance isolated for one test, with its own database, clock, lifecycle and permitted external substitutes. |
| Test plan | The `tests.md` artifact of an OpenSpec change. Its impact map says what the change touches and which tests that can break, its exclusions say what is left out and why, and its selection lists the selectors that `mise run test-change` runs. [Testing](testing.md#test-plans) defines the format. |
| Selector | One line of a test plan's selection. It names tests to run, such as a backend package with an optional `-run` filter, a frontend path or a named suite, and never a command or its flags. |

## Dependency maintenance

| Term | Meaning |
| --- | --- |
| Dependency verification | Checks of known local pins, related declarations and committed client hashes. Native package managers verify their own locked graphs and downloaded content. The updater reviews release age, upstream changes and compatibility. |
| Release hold | The normal waiting period used when selecting a dependency update. The upgrade guide defines its duration. It also defines how the updater reports missing age information or a departure from that period. |

The [dependency upgrade guide](how-to-upgrade-deps.md) describes the procedure, sources and verification limits.
