## ADDED Requirements

### Requirement: Verifications follow the workspace roles

The operations on the verification configs and the verifications of a workspace's databases SHALL apply the workspace roles:

- a viewer MAY read the verification config of a database, list its verifications and read one of them;
- a member MAY also save the verification config, start a manual verification and cancel a verification;
- an instance admin MAY do all of these in every workspace.

A viewer who saves a config, starts or cancels a verification SHALL get `403` with `code` `permission_denied`, and nothing SHALL change. A user who is neither a member of the database's workspace nor an instance admin SHALL get `404` with `code` `not_found` for every one of these operations. That answer SHALL be the same as for a database, a backup or a verification that does not exist, and SHALL reveal nothing of them.

#### Scenario: Viewer reads verifications

- **WHEN** a viewer of a workspace reads the verification config of one of its databases, lists its verifications and reads one of them
- **THEN** each request succeeds

#### Scenario: Viewer changes verifications

- **WHEN** a viewer of a workspace saves the verification config of one of its databases, starts a verification of one of its completed backups and cancels a pending verification
- **THEN** each response is `403` with `code` `permission_denied`, the config is unchanged, no verification is added, and the pending verification is still pending

#### Scenario: Another team's verifications

- **WHEN** a user who is not a member of a workspace reads and saves the verification config of one of its databases, lists and reads its verifications, starts a verification of one of its completed backups and cancels one of its pending verifications
- **THEN** each response is `404` with `code` `not_found`, the config is unchanged, no verification of the database is added, and the pending verification is still pending

#### Scenario: Same answer as a missing object

- **WHEN** a user who is not a member of a workspace reads the verification config of one of its databases and of a database ID that does not exist, starts a verification of one of its completed backups and of a backup ID that does not exist, and reads one of its verifications and a verification ID that does not exist
- **THEN** every response is `404` with `code` `not_found`, and the two responses of each pair have the same body

#### Scenario: Instance admin outside the workspace

- **WHEN** an instance admin who is not a member of a workspace saves the verification config of one of its databases and starts a verification of one of its completed backups
- **THEN** the save answers `200` and the start answers `202`
