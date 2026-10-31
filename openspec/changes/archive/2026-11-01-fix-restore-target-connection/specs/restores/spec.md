## ADDED Requirements

### Requirement: The interface never fills a restore target from the backed-up database

The restore target form in the interface SHALL open with the default settings of a new database of the backup's engine. It SHALL NOT carry any connection setting of the backed-up database, including its host, port, user, database name, TLS certificates and SSH bastion, and it SHALL NOT present any secret as already stored. The target SHALL require the same fields as adding a database of that engine, including the password. No request the interface makes for a restore SHALL carry the backed-up database's ID as the source of omitted settings, and the flow SHALL send no database settings test, so no stored secret of that database is used to reach the target. This is a security rule: a restore target is usually another server, and a secret of the source must not reach it unless the user types it.

#### Scenario: The source database uses an SSH tunnel and a client certificate

- **WHEN** a member opens the restore target form for a backup of a PostgreSQL database that connects through an SSH bastion with a client certificate on a non-default port
- **THEN** the form shows the engine's default port, an empty host, user, password and database name, a disabled SSH tunnel and no stored certificate or SSH credentials

#### Scenario: The password is left empty

- **WHEN** a member fills the host, user and database name of the target and leaves its password empty
- **THEN** the start button is disabled and no restore start request is sent

#### Scenario: The restore request holds only entered values

- **WHEN** a member enters a target and starts the restore
- **THEN** the start request holds only the values entered in the form and the engine defaults, and no request of the flow carries the backed-up database's ID

### Requirement: The interface checks a restore target by starting the restore

The restore target form SHALL start the restore without a separate connection test, relying on the restore start to connect to the target and check its version before it answers. While a start is pending, the form SHALL prevent another submission. A refused start SHALL show the refusal with the reason the server gave and SHALL keep the form open with every entered value, so the user can correct a field and start again. Among submissions, only an accepted start SHALL close the form. A workspace viewer SHALL be able to start a restore from the interface, as the `workspace-access` spec allows.

#### Scenario: A wrong password

- **WHEN** a member starts a restore into a target that refuses the entered password
- **THEN** the form stays open with every entered value and shows the connection refusal with the server's reason, and the backup lists no new restore

#### Scenario: Starting again after a correction

- **WHEN** after a refused start the member corrects the password and starts the restore again
- **THEN** the restore starts with the corrected values and the form closes

#### Scenario: A repeated click during a pending start

- **WHEN** a member clicks the start button twice while the first start is still pending
- **THEN** exactly one restore start request is sent

#### Scenario: A viewer restores a backup

- **WHEN** a workspace viewer enters a reachable target and starts the restore
- **THEN** no database settings test request is sent and the restore start answers `202`
