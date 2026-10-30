## MODIFIED Requirements

### Requirement: A workspace whose storage holds another workspace's backups cannot be deleted

Deleting a workspace without databases SHALL get 409 with code in_use and SHALL delete nothing when one of its storages is still referenced by backup records of a database in another workspace. A workspace that still has databases SHALL get workspace_not_empty as before. The workspace, members, notifiers, storages and referenced backups SHALL remain, and the response SHALL NOT reveal the internal cause of refusal.

Old files whose backup records were removed by a completed storage switch SHALL be cleanup obligations, not retained backups. Those obligations SHALL NOT permanently block deletion of the previous storage or workspace; deletion SHALL report leftover files according to storage-file-cleanup and settle any affected switch cleanup.

#### Scenario: Storage with backups of a database in another workspace

- **WHEN** the owner deletes a workspace without databases while its storage is still referenced by a backup of a database in another workspace
- **THEN** the response is 409 with code in_use and the workspace, members, notifiers and storages remain
- **AND** a member of the other workspace still sees that backup in the database's backup list

#### Scenario: Only old switch cleanup remains

- **WHEN** the owner deletes a workspace with no databases or referenced backups but with unresolved switch cleanup files
- **THEN** cleanup alone does not permanently prevent deletion and leftover files are reported
- **AND** affected switch operations reach a complete or complete-with-remainders outcome
