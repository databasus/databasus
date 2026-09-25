## Purpose

Defines the dashboard that summarizes, for one workspace, the health and stored backups of every database, and for global admins, the stored backups of the whole installation.

## ADDED Requirements

### Requirement: The dashboard is the page shown after sign-in

The system SHALL show the dashboard of the selected workspace after a user signs in and after a user creates a workspace. The databases list SHALL remain reachable from the sidebar.

#### Scenario: A user signs in

- **WHEN** a user with at least one workspace signs in
- **THEN** the dashboard of the selected workspace is shown

#### Scenario: A user without workspaces signs in

- **WHEN** a user who belongs to no workspace signs in
- **THEN** the system offers to create a workspace instead of showing a dashboard

### Requirement: The dashboard lists every database of the workspace with its backup statistics

The system SHALL list every database of the workspace with its health status, its last backup time, its storage, the number of backups it currently stores, how many of those succeeded and failed, and the mean and total size of its successful backups.

The backup count SHALL include every stored backup regardless of status. For physical databases it SHALL count full and incremental backups but not WAL segments. Sizes SHALL count successful backups only, and for physical databases the total size SHALL also include the WAL segments. The mean size SHALL be the total size of successful full, incremental or logical backups divided by their number, and SHALL be absent when no backup succeeded.

#### Scenario: A database with mixed backup outcomes

- **WHEN** a logical database stores two successful backups of 10.5 MB, one failed backup and one canceled backup
- **THEN** its row shows 4 backups, 2 successful, 1 failed, a mean size of 10.5 MB and a total size of 21 MB

#### Scenario: A physical database with WAL segments

- **WHEN** a physical database stores one successful 100 MB full backup, one failed full backup and a 16 MB WAL segment
- **THEN** its row shows 2 backups, 1 successful, 1 failed, a mean size of 100 MB and a total size of 116 MB

#### Scenario: A database without backups

- **WHEN** a database has no stored backups
- **THEN** its row shows 0 backups, no mean size, a total size of 0 and no last backup time

### Requirement: The dashboard shows the recent healthcheck attempts of monitored databases

The system SHALL show, for every database with healthcheck enabled, its last ten healthcheck attempts, newest first. A database with healthcheck disabled SHALL show no attempts, even if attempts from an earlier period remain stored.

#### Scenario: More than ten attempts are stored

- **WHEN** a monitored database has twelve stored attempts
- **THEN** its row shows the ten newest, newest first

#### Scenario: Healthcheck is disabled

- **WHEN** a database has healthcheck disabled
- **THEN** its row shows no attempts

### Requirement: The dashboard shows the workspace totals

The system SHALL show the number of databases in the workspace, the sum of their backup counts and the sum of their total backup sizes.

#### Scenario: An empty workspace

- **WHEN** a workspace has no databases
- **THEN** the totals show 0 databases, 0 backups and a size of 0, and the list says the workspace has no databases yet

### Requirement: Only workspace members can read a workspace dashboard

The system SHALL let every member of a workspace, viewers included, and every global admin read that workspace's dashboard. Any other user SHALL be refused with the same response the databases list gives, so the response does not reveal whether the workspace exists.

#### Scenario: A viewer opens the dashboard

- **WHEN** a workspace viewer requests the workspace dashboard
- **THEN** the dashboard is returned

#### Scenario: A non-member requests the dashboard

- **WHEN** a user who is not a member of the workspace requests its dashboard
- **THEN** the request is refused with "insufficient permissions to access this workspace"

### Requirement: Only global admins can read the installation totals

The system SHALL return the number of databases, the number of backups and the total backup size across every workspace, including databases created by restores that belong to no workspace, only to global admins. Any other user SHALL be refused, and the dashboard SHALL NOT show the installation tile to them.

#### Scenario: An admin opens the dashboard

- **WHEN** a global admin opens the dashboard
- **THEN** an extra tile shows the totals across every workspace

#### Scenario: A member requests the installation totals

- **WHEN** a user who is not a global admin requests the installation totals
- **THEN** the request is refused as forbidden
