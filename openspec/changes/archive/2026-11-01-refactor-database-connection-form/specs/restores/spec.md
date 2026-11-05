## MODIFIED Requirements

### Requirement: The interface never fills a restore target from the backed-up database

The restore target form in the interface SHALL open with the default settings of a new database of the backup's engine. It SHALL NOT carry any connection setting of the backed-up database, including its host, port, user, database name, TLS certificates and SSH bastion, and it SHALL NOT present any secret as already stored. The target SHALL require the same fields as adding a database of that engine, including the password. No request the interface makes for a restore SHALL carry the backed-up database's ID as the source of omitted settings, and the flow SHALL send no database settings test, so no stored secret of that database is used to reach the target. While the dialog of a backup stays open, opening the form again after a start SHALL show the values and restore options entered for the previous target, with the advanced settings open when any of them holds a non-default value, so a failed restore can be retried. Closing that dialog SHALL discard them. The form SHALL show only settings a restore uses, so it SHALL NOT offer exclusions, schema selections or other settings that only affect a backup. A start answer SHALL change nothing on the screen if the dialog has closed or the session has changed. This is a security rule: a restore target is usually another server, and a secret of the source must not reach it unless the user types it.

#### Scenario: The source database uses an SSH tunnel and a client certificate

- **WHEN** a member opens the restore target form for a backup of a PostgreSQL database that connects through an SSH bastion with a client certificate on a non-default port
- **THEN** the form shows the engine's default port, an empty host, user, password and database name, a disabled SSH tunnel and no stored certificate or SSH credentials

#### Scenario: The password is left empty

- **WHEN** a member fills the host, user and database name of the target and leaves its password empty
- **THEN** the start button is disabled and no restore start request is sent

#### Scenario: The restore request holds only entered values

- **WHEN** a member enters a target and starts the restore
- **THEN** the start request holds only the values entered in the form and the engine defaults, and no request of the flow carries the backed-up database's ID

#### Scenario: A retry after a failed restore

- **WHEN** a restore of a PostgreSQL backup started with the extension exclusion option fails and the member opens the target form again without closing the dialog
- **THEN** the form shows the entered target with the advanced settings open and the extension exclusion option on, and any typed certificate appears as typed rather than as stored

#### Scenario: Closing the dialog discards the target

- **WHEN** a member enters a target, closes the backup's restore dialog and opens it again
- **THEN** the target form shows the engine's defaults with an empty password

#### Scenario: The dialog closes during a start

- **WHEN** a member closes the backup's restore dialog while a start is pending and the start then answers
- **THEN** no feedback appears and the dialog stays closed

#### Scenario: Backup-only settings of MySQL, MariaDB and MongoDB targets

- **WHEN** a member opens the advanced settings of the restore target form for a MySQL, MariaDB or MongoDB backup
- **THEN** the form offers no table, event or collection exclusions, and a MariaDB target still offers its Galera option
