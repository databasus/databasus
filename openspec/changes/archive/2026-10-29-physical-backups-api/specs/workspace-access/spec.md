## ADDED Requirements

### Requirement: Physical backups follow the workspace roles

The operations on the physical backups of a workspace's databases SHALL apply the workspace roles:

- a viewer MAY list the physical backups, read the WAL stream statistics and the current values of the WAL stream, and issue restore tokens;
- a member MAY also start, cancel and delete physical backups;
- an instance admin MAY do all of these in every workspace.

A viewer who starts, cancels or deletes a physical backup SHALL get `403` with `code` `permission_denied`, and nothing SHALL change. A user who is neither a member of the database's workspace nor an instance admin SHALL get `404` with `code` `not_found` for every one of these operations. That answer SHALL be the same as for a database or a backup that does not exist, and SHALL reveal nothing of the database or its backups. The restore set download carries no session and is outside this rule: its token decides, as the `physical-backups/operations` spec states.

#### Scenario: Viewer reads and issues a restore token

- **WHEN** a viewer of a workspace lists the physical backups of one of its databases, reads its WAL stream statistics and current values, and issues a restore token for it
- **THEN** each request succeeds

#### Scenario: Viewer starts, cancels and deletes

- **WHEN** a viewer of a workspace starts a physical backup of one of its databases, cancels a running one and deletes a completed one
- **THEN** each response is `403` with `code` `permission_denied`, no new backup of the database appears in the list after the scheduler's next pass, the running backup is still `IN_PROGRESS` and the completed backup is still listed

#### Scenario: Member starts, cancels and deletes

- **WHEN** a user with the role `WORKSPACE_MEMBER` starts a physical backup of a database of the workspace, cancels a running one and deletes a completed one
- **THEN** the start and the cancel answer `202` and the delete answers `204`

#### Scenario: Another team's physical backups

- **WHEN** a user who is not a member of a workspace lists the physical backups of one of its databases, reads its WAL stream statistics and current values, issues a restore token, starts a backup, cancels a running backup and deletes a completed one
- **THEN** each response is `404` with `code` `not_found`, no new backup of the database appears in the list after the scheduler's next pass, the running backup is still `IN_PROGRESS` and the completed backup is still listed

#### Scenario: Same answer as a missing physical backup

- **WHEN** a user who is not a member of a workspace deletes one of its physical backups and then a backup ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body

#### Scenario: Instance admin outside the workspace

- **WHEN** an instance admin who is not a member of a workspace lists the physical backups of one of its databases and starts a full backup
- **THEN** the list answers `200` and the start answers `202`
