## MODIFIED Requirements

### Requirement: Deleting a workspace removes its notifiers, storages and members

Deleting a workspace without databases SHALL remove the workspace, its notifiers, its storages and its members, unless retained backup references, selected Databasus backup destinations or outstanding storage cleanup refuse deletion. After authorization, refusal precedence SHALL be: databases in the workspace cause `workspace_not_empty`; otherwise any selected Databasus backup destination in that workspace causes `databasus_backup_in_use`; otherwise retained database-backup references cause `in_use`; otherwise outstanding cleanup causes `cleanup_pending`. Each refusal SHALL answer 409 and leave the entire workspace unchanged. The Databasus destination check SHALL cover all its storages before other storage blockers, including blockers in a different storage of that workspace. Its interface SHALL explain that root admin must first remove the selection from the Databasus backups section. Workspace deletion SHALL NOT implicitly remove that selection or acknowledge abandonment of its storages' cleanup. The users who were its members SHALL keep their accounts and stay members of their other workspaces.

#### Scenario: Workspace with a notifier and a storage

- **WHEN** the owner of a workspace with no databases, one notifier and one otherwise unused storage deletes the workspace
- **THEN** the workspace is no longer listed, and its notifier and storage no longer exist

#### Scenario: Workspace whose storage has pending cleanup

- **WHEN** the owner of a workspace with no databases deletes it while its storage holds a file with a pending deletion and none of its storages has a selected Databasus backup destination or retained database-backup references
- **THEN** the response is 409 with code `cleanup_pending`, the whole deletion is refused, and no file is removed by that request

#### Scenario: Members of a deleted workspace

- **WHEN** a workspace whose members also belong to another workspace is deleted
- **THEN** each of those users can still sign in and still sees the other workspace

#### Scenario: Workspace contains a selected Databasus backup destination

- **WHEN** an authorized owner deletes an otherwise empty workspace containing a selected Databasus backup storage
- **THEN** deletion gets 409 with code `databasus_backup_in_use`, the interface explains how root admin can deselect it, and the workspace, members, notifiers and all its storages remain

#### Scenario: Workspace has Databasus and cross-workspace database backup blockers

- **WHEN** a workspace without databases contains a selected Databasus backup destination and storage referenced by another workspace's database backups
- **THEN** deletion gets 409 with code `databasus_backup_in_use` and the entire workspace remains unchanged
- **AND** after deselecting all Databasus backup destinations there, deletion still gets `in_use` while the cross-workspace database-backup references remain

### Requirement: A workspace whose storage holds another workspace's backups cannot be deleted

Deleting a workspace without databases SHALL get 409 with code `in_use` and SHALL delete nothing when one of its storages is still referenced by backup records of a database in another workspace, unless a selected Databasus backup destination takes precedence with `databasus_backup_in_use`. A workspace that still has databases SHALL get `workspace_not_empty` before either storage guard. The workspace, members, notifiers, storages and referenced backups SHALL remain, and the response SHALL NOT reveal the internal cause beyond the defined safe refusal code and Databasus deselection guidance.

Old files whose backup records were removed by a completed storage switch SHALL be cleanup obligations, not retained backups. Those obligations SHALL return `cleanup_pending` for workspace deletion only when neither a selected Databasus backup destination nor retained database-backup references take precedence, until cleanup completes or an authorized user individually deletes the affected storage with explicit abandonment. This provides an exit without silently discarding work through workspace deletion.

#### Scenario: Storage with backups of a database in another workspace

- **WHEN** the owner deletes a workspace without databases or selected Databasus backup destinations while its storage is still referenced by a backup of a database in another workspace
- **THEN** the response is 409 with code `in_use` and the workspace, members, notifiers and storages remain
- **AND** a member of the other workspace still sees that backup in the database's backup list

#### Scenario: Only old switch cleanup remains

- **WHEN** the owner deletes a workspace with no databases, selected Databasus backup destinations or referenced database backups but with unresolved switch cleanup files
- **THEN** the response is 409 with code `cleanup_pending`, and all workspace resources and obligations remain
- **AND** the interface directs the user to wait or explicitly delete the affected storages individually

#### Scenario: Both storage guards apply

- **WHEN** the owner deletes a workspace without databases that has both a selected Databasus backup destination and cross-workspace database-backup references
- **THEN** the response is 409 with code `databasus_backup_in_use` and no part of the workspace is removed

### Requirement: Resolving storage cleanup permits workspace deletion

After its storages finish their file cleanup or are individually deleted with explicit abandonment, an otherwise deletable workspace SHALL be removable through the ordinary deletion request. Refusal precedence SHALL remain `workspace_not_empty`, then `databasus_backup_in_use`, then `in_use`, then `cleanup_pending`. Cleanup completion or abandonment SHALL NOT override databases, selected Databasus backup destinations or retained database-backup references. An obligation against storage outside the workspace SHALL NOT be removed just because its cleanup was attributed to a database or former acceptance scope inside the deleted workspace.

#### Scenario: A user removes an unreachable storage explicitly

- **WHEN** an owner explicitly deletes the last pending storage and then deletes its empty workspace with no selected Databasus backup destinations or retained database-backup references
- **THEN** workspace deletion succeeds without a bulk abandonment parameter

#### Scenario: A workspace has databases and pending cleanup

- **WHEN** a workspace deletion encounters both databases and pending file cleanup
- **THEN** it returns `workspace_not_empty` without changing either

#### Scenario: Cleanup also encounters retained backup references

- **WHEN** an empty workspace has a storage with retained database-backup references and outstanding cleanup, and none of its storages is a selected Databasus backup destination
- **THEN** its deletion returns `in_use` and preserves all resources

#### Scenario: Cleanup encounters a selected destination and retained references

- **WHEN** an empty workspace has a selected Databasus backup destination, retained database-backup references and outstanding cleanup, including when these belong to different storages
- **THEN** its deletion returns `databasus_backup_in_use` and preserves all resources
- **AND** after all Databasus backup destinations there are deselected, deletion returns `in_use` while the retained references remain
- **AND** after those references are removed, deletion returns `cleanup_pending` until cleanup finishes or the affected storages are individually deleted with explicit abandonment

#### Scenario: Attribution names a removed workspace

- **WHEN** a workspace is deleted but a file obligation attributed to it addresses storage in another surviving workspace
- **THEN** that obligation survives and remains retryable
