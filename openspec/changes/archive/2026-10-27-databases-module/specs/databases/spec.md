## Purpose

Defines the databases of a workspace that Databasus backs up: how a database is created, changed, tested, copied and deleted, how its secrets are kept, which notifiers it uses, how its last backup and health are shown, and who may make the backend use its stored secrets.

## ADDED Requirements

### Requirement: A saved database is returned without its secrets

Creating a database SHALL answer `201` with the database as it was saved: its ID, workspace, name, type, settings, notifier IDs, last backup time, last backup error and health status. Reading a database, listing databases, creating, changing and copying one SHALL never return a secret, neither in clear text nor encrypted. The secrets are the connection password, the TLS client key, and the SSH tunnel's password, private key and private key passphrase. Every other setting SHALL be returned as it was saved, the TLS client certificate and root certificate in clear text.

#### Scenario: Creating a PostgreSQL database

- **WHEN** a member of a workspace creates a PostgreSQL database with a host, a port, a user name, a password and a TLS client key
- **THEN** the response is `201` with the database's ID, name, type, host, port, user name and detected version, and the body contains neither the password nor the key in any form

#### Scenario: Listing databases with an SSH tunnel

- **WHEN** a member lists the databases of a workspace that has a MySQL database behind an SSH tunnel with a private key
- **THEN** the list shows the database with its host, port, user name and tunnel host, and without the password and the private key

### Requirement: Changing a database keeps the secrets the request leaves out

Changing a database SHALL replace its name, settings and notifiers with those in the request, except a secret the request leaves out, which SHALL keep its stored value. A change SHALL first connect with the resulting settings. When the connection fails, the database SHALL keep its stored settings, and the response SHALL be `400` with `code` `database_connection_failed` and the server's reason in `detail`.

#### Scenario: Renaming a database without its password

- **WHEN** a member changes the name of a PostgreSQL database and sends its connection settings without the password
- **THEN** the database has the new name, and its next backup connects with the stored password

#### Scenario: A change that cannot connect

- **WHEN** a member changes the host of a database to one where no server answers
- **THEN** the response is `400` with `code` `database_connection_failed`, and the database keeps its stored host

### Requirement: A saved database keeps its type

A database's type SHALL be fixed when it is created. Changing a database with settings of another type SHALL get `400` with `code` `invalid_request` and SHALL change nothing. A user who needs another type SHALL create a new database.

#### Scenario: MySQL database sent with MongoDB settings

- **WHEN** a member changes a saved MySQL database and sends MongoDB settings
- **THEN** the response is `400` with `code` `invalid_request`, and the database is still the same MySQL database

### Requirement: Invalid settings are refused without a change

Creating or changing a database with an empty name, a missing required setting or a setting its type rejects SHALL get `400` with `code` `invalid_request`, name the setting, and change nothing. Creating a database whose server cannot be reached or refuses the connection SHALL get `400` with `code` `database_connection_failed` and SHALL create nothing. The rules of each type that exist today SHALL keep applying, the refusal of the embedded internal database included.

#### Scenario: Database without a host

- **WHEN** a member creates a PostgreSQL database without a host
- **THEN** the response is `400` with `code` `invalid_request` naming the host, and the workspace has no new database

#### Scenario: The internal database as a target

- **WHEN** a member creates a PostgreSQL database that points at the embedded internal database of Databasus
- **THEN** the response is `400` with `code` `invalid_request`, no connection is attempted, and the workspace has no new database

### Requirement: Creating a database never changes another database

Creating a database SHALL always create a new database with a new ID in the workspace of the request. An ID, a workspace or a status in the request body SHALL NOT select, change or move any existing database, whatever workspace that database belongs to.

#### Scenario: Create with the ID of another team's database

- **WHEN** a member of one workspace creates a database and sends in the body the ID of a database in a workspace they do not belong to
- **THEN** the response is `201` with a database that has a new ID in the member's workspace, and the other database keeps its workspace, name, settings and backups

### Requirement: Databases are listed per workspace in name order

Listing databases SHALL return the databases of one workspace ordered by name, a page at a time, with a cursor to the next page. A list filtered by a notifier SHALL return only the databases of that workspace that use the notifier.

#### Scenario: Listing across pages

- **WHEN** a member lists a workspace with the databases `c`, `a` and `b` with a page size of two
- **THEN** the first page holds `a` and `b` and points to a next page, and the next page holds `c` and points to none

#### Scenario: Databases that use a notifier

- **WHEN** a member lists the databases of a workspace filtered by a notifier that one of three databases uses
- **THEN** the list holds that one database

### Requirement: A connection test reports its outcome and changes nothing

Testing a saved database SHALL connect with its stored settings and answer `200` with whether the connection succeeded. A failed test SHALL report the server's reason and, where the type classifies it, a machine-readable reason code that the form explains, such as a missing replication privilege or a missing `pg_hba.conf` entry. Testing SHALL NOT change the database: its settings, notifiers, last backup error and health status SHALL stay as they were.

#### Scenario: A viewer tests a saved database

- **WHEN** a viewer of a workspace tests a saved database whose server answers
- **THEN** the response is `200` and reports a successful connection

#### Scenario: A failed test changes nothing

- **WHEN** a member tests a database that shows a backup error and whose server refuses the stored password
- **THEN** the response is `200` and reports the failure with its reason, and the database still shows the backup error

#### Scenario: A physical source without the replication privilege

- **WHEN** a member tests a physical PostgreSQL database whose user lacks the replication privilege
- **THEN** the response reports the failure with the reason code for a missing replication privilege

### Requirement: Unsaved settings can be tested before saving

Testing unsaved settings SHALL connect with the settings in the request and answer `200` with the outcome, as a test of a saved database does. When the request names a saved database of the same workspace, the stored secrets of that database SHALL fill the secrets the request leaves out. A named database of another workspace, or one that does not exist, SHALL get `404` with `code` `not_found`. Settings of another type than the named database SHALL get `400` with `code` `invalid_request`, and no stored secret SHALL be used. In both cases no connection SHALL be attempted.

#### Scenario: Testing a new database

- **WHEN** a member tests the settings of a database that is not saved yet
- **THEN** the response is `200` and reports the outcome of the connection

#### Scenario: Testing changed settings of a saved database

- **WHEN** a member tests the settings of a saved database with a new port and without its password
- **THEN** the test connects to the new port with the stored password

#### Scenario: A database of another workspace

- **WHEN** a member of one workspace tests settings and names a database of another workspace
- **THEN** the response is `404` with `code` `not_found`, and no connection is attempted

#### Scenario: Settings of another type than the named database

- **WHEN** a member tests PostgreSQL settings and names a saved MySQL database of the workspace
- **THEN** the response is `400` with `code` `invalid_request`, and no connection is attempted

### Requirement: Only a user who may change databases sends stored secrets elsewhere

Testing unsaved settings, checking whether a limited user is advisable and creating a limited user SHALL require the right to change the workspace's content. Each of them connects to the host in the request and, with a saved database named, uses that database's stored secrets. A viewer SHALL get `403` with `code` `permission_denied` from each of them, and no connection SHALL be attempted.

#### Scenario: A viewer points a saved database at another host

- **WHEN** a viewer of a workspace checks whether a limited user is advisable, names a saved database, and sends a host of their own without the password
- **THEN** the response is `403` with `code` `permission_denied`, and no connection is made to that host

#### Scenario: A viewer tests unsaved settings

- **WHEN** a viewer of a workspace tests unsaved settings
- **THEN** the response is `403` with `code` `permission_denied`

### Requirement: A limited user can be created for a database

For a logical PostgreSQL, MySQL, MariaDB or MongoDB database, checking SHALL report whether the connecting user has more privileges than a backup needs, and which ones. Creating SHALL add a read-only user with a generated name and password and answer `201` with both. For a physical PostgreSQL database, both SHALL do the same for a replication-only user, and the answer SHALL say whether that user may force a WAL segment switch. The generated password SHALL be returned only in that answer.

#### Scenario: Creating a read-only user

- **WHEN** a member creates a limited user with the settings of a MySQL database whose user may write
- **THEN** the response is `201` with a new user name and password, and that user can read the database and cannot write to it

#### Scenario: A name already taken on the server

- **WHEN** the first generated user name already exists on the server
- **THEN** the backend generates another name, and the response is `201` with the new user

### Requirement: A database is deleted completely or not at all

Deleting a database while one of its logical backups is in progress SHALL get `409` with `code` `in_use` and SHALL delete nothing, as today. Otherwise deleting a database SHALL remove, in one step, the database, its settings, its notifier attachments, its backup configs, and the records of its logical and physical backups and WAL. It SHALL ask for the deletion of every stored file those records describe. When any part fails, nothing SHALL be removed. Only after the deletion has succeeded SHALL a running backup of the database be cancelled, its WAL streaming stopped and its replication slots dropped on the source server.

#### Scenario: Deleting a database with backups

- **WHEN** a member deletes a database that has logical backups in a storage
- **THEN** the response is `204`, the database and its backups are no longer listed, and the backup files are removed from the storage

#### Scenario: A logical backup is running

- **WHEN** a member deletes a database while its logical backup is in progress
- **THEN** the response is `409` with `code` `in_use`, and the database and its backups are still listed

#### Scenario: A failure during the deletion

- **WHEN** the deletion of a database's backup records fails
- **THEN** the database and all its backups are still listed, and no backup file is removed

#### Scenario: Deleting a database that streams WAL

- **WHEN** a member deletes a physical PostgreSQL database with WAL streaming
- **THEN** the streaming stops, and the replication slots Databasus created on the source server are dropped after the database is gone

### Requirement: A database uses only notifiers of its own workspace

Creating or changing a database with notifiers SHALL check that each notifier exists and belongs to the database's workspace, and SHALL get `400` with `code` `invalid_request` otherwise. The check SHALL hold against a notifier transfer that runs at the same time: either the database is saved with the notifier and the transfer is refused with `code` `in_use`, or the transfer completes and the database is refused.

#### Scenario: A notifier of another workspace

- **WHEN** a member creates a database with a notifier of another workspace
- **THEN** the response is `400` with `code` `invalid_request`, and no database is created

#### Scenario: Attaching while the notifier moves

- **WHEN** a member saves a database with a notifier while another member transfers that notifier to another workspace
- **THEN** afterwards the database does not use a notifier of another workspace

### Requirement: A copy starts with no backup history

Copying a database SHALL create a database in the same workspace named after the original with ` (Copy)` appended, with the same type, settings, secrets and notifiers. The copy SHALL get a copy of the original's verification config when the original has one. The copy of a logical database SHALL also get a copy of the original's backup config when the original has one, and the copy of a physical PostgreSQL database SHALL get no backup config. The copy SHALL show no last backup time, no last backup error and the original's health status. A copied physical PostgreSQL database SHALL use its own replication slot.

#### Scenario: Copying a logical database

- **WHEN** a member copies a PostgreSQL database that has backups
- **THEN** a new database named `<name> (Copy)` is listed with the same settings and notifiers, no last backup time, and a backup config with the original's settings

#### Scenario: Copying a physical database

- **WHEN** a member copies a physical PostgreSQL database that has a backup config
- **THEN** a new database named `<name> (Copy)` is listed with the same settings, its own replication slot and no backup config

### Requirement: The last backup time shows the newest backup of any kind

A database SHALL show the time of its newest successful backup. Each kind of backup counts with one time: a logical backup with the time it completed, a physical full or incremental backup with the time it started, and a WAL segment with the time it was received. For a physical database, the newest of its completed full and incremental backups and its archived WAL segments counts. Recording a backup SHALL never replace a later time with an earlier one, and recording a success SHALL clear the last backup error. Every kind is recorded in the same step that publishes the backup, so a backup whose publication is rolled back never shows.

#### Scenario: A physical database that streams WAL

- **WHEN** a WAL segment of a physical database is archived an hour after its last full backup
- **THEN** the database's last backup time is the time of that segment

#### Scenario: An older backup finishes later

- **WHEN** a full backup that started before a WAL segment was received completes after that segment was archived
- **THEN** the database's last backup time stays the time the segment was received

#### Scenario: A logical backup completes

- **WHEN** a logical backup of a database that showed a backup error completes
- **THEN** the database shows the backup's completion time and no backup error

### Requirement: An upgrade keeps every database

An upgrade SHALL keep every database that has a known type and its type's settings, with its type, name, workspace, settings, secrets, notifiers, last backup time, last backup error and health status. The schema and exclusion lists SHALL keep their entries. A physical database that had backups before the upgrade SHALL show their newest time right after it. A database without a workspace SHALL get the oldest workspace when backups or a backup config reference it, and SHALL be removed otherwise. A database without its type's settings that nothing references SHALL be removed, and a second settings row of one database SHALL be removed. A database of an unknown type SHALL be removed when nothing references it; when backups or a backup config reference it, the upgrade SHALL stop before the instance starts and name the database, because no release writes such a type and its backup files cannot be cleaned up without it.

#### Scenario: A MySQL database with excluded tables

- **WHEN** an instance with a MySQL database that excludes two tables and uses a password is upgraded
- **THEN** the database is listed with both tables excluded, and its next backup connects with the stored password

#### Scenario: A physical database with WAL streaming

- **WHEN** an instance with a physical PostgreSQL database that archived WAL yesterday is upgraded
- **THEN** the database shows the time of its newest archived segment as its last backup time

#### Scenario: A database of an unknown type that backups reference

- **WHEN** an instance with a database of an unknown type that has backups is upgraded
- **THEN** the upgrade stops before the instance starts, and the error names that database

#### Scenario: A database without a workspace that has backups

- **WHEN** an instance with a database that has no workspace and has backups is upgraded
- **THEN** the database is listed in the oldest workspace with its backups

#### Scenario: A database without a workspace and without backups

- **WHEN** an instance with a database that has no workspace, no backups and no backup config is upgraded
- **THEN** the database is gone after the upgrade
