## MODIFIED Requirements

### Requirement: The physical backup start follows the user's permission

The physical backup list SHALL offer the backup start only to users who may manage the database, because the server refuses it for the others. The restore action SHALL stay available to a user who can only view the database. Hiding a control SHALL not replace the server's authorization of the request. While the loaded backup config has backups turned off or names no storage, the backup start SHALL stay visible and disabled, because the server refuses such a start, and the screen SHALL say in the user's language that a backup can start once backups are turned on and a storage is chosen in the backup configuration. When the backup config failed to load, the screen SHALL NOT disable the start for this reason and SHALL NOT show that explanation, and the server's answer decides.

#### Scenario: A viewer opens the list

- **WHEN** a workspace viewer opens the backups tab of a physical database
- **THEN** the screen offers no backup start, cancel or delete, and it offers restore

#### Scenario: A member opens the list

- **WHEN** a workspace member opens the backups tab of a physical database
- **THEN** the screen offers the backup start

#### Scenario: A member opens the list of a database with backups turned off

- **WHEN** a workspace member opens the backups tab of a physical database whose loaded backup config has backups turned off
- **THEN** the backup start is disabled, no start request can be sent, and the screen shows the notice that scheduled backups are off together with the explanation of what a start needs

#### Scenario: A member opens the list of a database without a storage

- **WHEN** a workspace member opens the backups tab of a physical database whose loaded backup config has backups turned on and names no storage
- **THEN** the backup start is disabled, no start request can be sent, and the screen shows the explanation of what a start needs

#### Scenario: The backup config fails to load

- **WHEN** a workspace member opens the backups tab of a physical database and its backup config fails to load
- **THEN** the backup start stays enabled once the list has loaded, the screen shows the reason with a retry, and it does not show the explanation of what a start needs
