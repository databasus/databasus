# Glossary

This is the project's common language. Each concept has one term in the documentation and the interface, and one name in code and the API. Synonyms are not used. A new term is added here in the same change that introduces it into the contract, a `module.go` or the documentation. Definitions stay short. The rules that apply to a concept live in [architecture.md](architecture.md).

## Product

| Term | In code and API | Meaning |
|---|---|---|
| Database | database | A user's database that Databasus connects to and backs up. The term never means Databasus's own database. |
| Internal database | internal database | The embedded PostgreSQL inside the Databasus container that holds Databasus's own state. |
| Engine | engine | A kind of database server Databasus supports: PostgreSQL, MySQL, MariaDB or MongoDB. The term also names the code in `internal/engines` that drives it. |
| Storage | storage | A place where backups are kept: local disk, S3, Google Drive, NAS, Azure Blob, FTP, SFTP, rclone. |
| Notifier | notifier | A channel Databasus sends notifications to: email, Telegram, Slack, Discord, Teams, Mattermost, webhook. |
| Provider | provider | One implementation of a storage or notifier type, with its own settings table. |
| Workspace | workspace | A group of users and the databases, storages and notifiers they share. |
| Member | member | A user in a workspace, with a workspace role. |
| Instance admin | admin | A user with the global admin role, who manages users and instance settings. |
| Actor | actor | The user on whose behalf a request runs: user ID and global role. |
| Audit event | audit event | A record of who did what to which object, written in the transaction of the action. |
| Backup config | backup config | The settings that say how a database is backed up: type, schedule, storage, retention, encryption. A database has at most one logical and one physical backup config. |
| Logical backup | logical backup | A dump produced by the engine's dump tool (`pg_dump`, `mysqldump`, `mariadb-dump`, `mongodump`). |
| Physical backup | physical backup | A copy of a PostgreSQL cluster's files taken over the replication protocol. Either full or incremental. |
| Full backup | full backup, `FULL` | A physical backup that contains the whole cluster. |
| Incremental backup | incremental backup, `INCREMENTAL` | A physical backup that contains only the blocks changed since its parent backup. |
| Chain | chain | A full backup with the incremental backups built on it. |
| WAL segment | WAL segment | One file of PostgreSQL's write-ahead log, archived to the storage after it is completed. |
| WAL stream | WAL stream | Continuous archiving of a database's WAL segments by `pg_receivewal`. |
| WAL streamer | WAL streamer | The process that runs the WAL stream of one database. |
| Replication slot | slot | A slot on the source that keeps WAL until Databasus has received it. |
| Timeline | timeline | PostgreSQL's history branch number, which changes when a replica is promoted. |
| Restore set | restore set | The files needed to restore a database to a moment: a full backup, its incrementals and the WAL up to the target. |
| Recovery script | recovery script | The script Databasus serves that rebuilds a cluster from a downloaded restore set. |
| Restore | restore | Putting a backup's data into a target database. |
| Retention | retention | The rule that decides which backups are kept, including the grandfather-father-son (GFS) policy. |
| Schedule | schedule | When something runs: every N hours, daily, weekly, monthly or a cron expression. |
| Verification | verification | A check that a backup restores: the agent restores it into a disposable container and compares what it finds. |
| Verification agent | verification agent | The program on the user's machine that runs verifications. |
| Healthcheck | healthcheck | A scheduled check that a user's database answers. |
| Master key | master key | The secret in the data volume that encrypts stored secrets and backup files. |
| Download token | download token | A single-use token that lets the browser download one logical backup. |
| Restore token | restore token | A single-use token that lets a user download one physical restore set. |

## Code

| Term | In code | Meaning |
|---|---|---|
| Module | module | A part of the backend that owns one domain area and its tables. Only its `module.go` is visible from outside. |
| Shared code | lib | Packages in `internal/lib` without domain logic. |
| Composition root | app | `internal/app`, which builds the whole object graph and wires modules together. |
| API contract | contract | `backend/api/openapi.yaml` and `backend/api/agent.yaml`, from which the servers, the test clients and the frontend types are generated. |
| Handler | handler | Code that decodes a request, calls a service and encodes the response. |
| Service | service | Code that holds the rules of a module's operations. |
| Repository | repository | Code that holds the queries of a module's tables. |
| Event | event | A notice a module publishes so that higher modules act in the same transaction, for example before a workspace is deleted. |
| Background job | job | Work that `background.Runner` runs on a schedule. Each run of a job is called a pass. |
| File store | file store | The storages module's only way to write and delete files in a storage. It records every write so that no file is left without an owner. |
| Write receipt | write receipt | What the file store returns for a written file. The transaction that publishes the backup spends it to keep the file. |
| Pending deletion | pending deletion | A recorded obligation to delete a file from a storage, carried out by the deletion worker. |
| Deletion worker | deletion worker | The background job of the storages module that deletes files with a pending deletion. |
| Migration | migration | A goose SQL file in `backend/migrations`, the only way to change the schema. |
| Test application | test app | A backend instance built for one test by `apptest.New`, with its own database, clock and fakes. |
