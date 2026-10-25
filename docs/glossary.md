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
| Workspace role | workspace role, `workspaces.Role`, `WORKSPACE_OWNER`, `WORKSPACE_ADMIN`, `WORKSPACE_MEMBER`, `WORKSPACE_VIEWER` | What a member may do in a workspace. `WORKSPACE_MEMBER` is the role that may edit the workspace content; the word "member" alone means any user in a workspace, whatever the role. |
| Workspace admin | workspace admin, `WORKSPACE_ADMIN`, `workspaces.RoleAdmin` | A member who may also rename the workspace and manage the members who are not admins or the owner. The word `admin` alone in code and text means the instance admin; `workspaces.RoleAdmin` and `ActionManageAdmins` name workspace admins. |
| Owner | owner, `WORKSPACE_OWNER`, `workspaces.RoleOwner` | The member who may also manage workspace admins, transfer ownership and delete the workspace. |
| Workspace content | workspace content | The databases, storages and notifiers of a workspace and everything attached to them: backup configs, backups, restores, healthchecks and verification configs. |
| Workspace action | workspace action, `workspaces.Action` | One thing a member can be allowed to do in a workspace, such as viewing it or editing its content. Each workspace role allows a fixed set of actions. |
| Instance admin | admin | A user with the global admin role, who manages users and instance settings. |
| Global role | global role, `ADMIN`, `MEMBER` | A user's role on the whole instance: `ADMIN` makes the user an instance admin, and `MEMBER` grants nothing beyond the user's workspace memberships. The word "member" alone means a member of a workspace. |
| User | user | A person with an account on the instance: email, name, global role and status (`INVITED`, `ACTIVE` or `INACTIVE`). |
| Session | session | A signed-in user's right to call the API, carried as a token the sign-in returns once. A password change ends the sessions issued before it. |
| Pending sign-in | pending sign-in, `pendingSignInId` | A password sign-in that waits for the code emailed to the user, when the instance requires two-factor sign-in. |
| Instance settings | instance settings | The settings an instance admin sets for the whole instance: open registration, member invitations, workspace creation by members and two-factor sign-in. |
| Sign-in options | sign-in options | What the sign-in page offers, such as the first registration on an instance without users. |
| Invitation | invitation | A user with the status `INVITED`, created when a workspace adds an address that has no user. The user finishes the sign-up with that address. |
| Password reset | password reset | Replacing a forgotten password with a code emailed to the user. |
| Actor | actor | The user on whose behalf a request runs: user ID and global role. |
| Audit event | audit event | A record of who did what to which object, written in the transaction of the action. |
| Audit author | audit author | The user an audit event names as the one who acted, kept in the event as a snapshot of user ID, email and name. |
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
| Problem | Problem | The RFC 9457 error body that every operation of the contract answers with. |
| Error code | `code` | The machine-readable cause in a `Problem`, such as `not_found`. The frontend chooses the text it shows by it. |
| Contract-checking test client | `apiclient` | The client backend tests call the API through. It fails the test on any response that differs from the contract. |
| Access mark | `x-access` | The mark of a contract operation that says who may call it: `public`, `admin`, or a signed-in user when the mark is absent. The contract guard reads it, and the users module enforces it. |
| Handler | handler | Code that decodes a request, calls a service and encodes the response. |
| Service | service | Code that holds the rules of a module's operations. |
| Repository | repository | Code that holds the queries of a module's tables. |
| Event | event | A notice a module publishes so that higher modules act in the same transaction, for example before a workspace is deleted. |
| Background job | job | Work that `background.Runner` runs on a schedule. Each run of a job is called a pass. |
| File store | file store | The storages module's only way to write and delete files in a storage. It records every write so that no file is left without an owner. |
| Write receipt | write receipt | What the file store returns for a written file. The transaction that publishes the backup spends it to keep the file. |
| Pending deletion | pending deletion | A recorded obligation to delete a file from a storage, carried out by the deletion worker. |
| Deletion worker | deletion worker | The background job of the storages module that deletes files with a pending deletion. |
| Console command | console command | An operation of the Databasus binary that the operator runs on the host or in the container instead of starting the server, such as a password reset. The `healthcheck` subcommand is not one: it only asks the running server. |
| Migration | migration | A goose SQL file in `backend/migrations`, the only way to change the schema. |
| Test application | test app | A backend instance built for one test by `apptest.New`, with its own database, clock and fakes. |
