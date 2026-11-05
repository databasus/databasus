## MODIFIED Requirements

### Requirement: A storage that backups use cannot be deleted

Deleting a storage that a backup config uses or that still holds backups SHALL get `409` with `code` `in_use` and SHALL change nothing, as the `storage-file-cleanup` spec describes. Deleting an otherwise unused storage SHALL follow the pending-cleanup acknowledgement rules of `storage-file-cleanup`: queued files require explicit abandonment, and successful deletion answers 204. Abandonment SHALL NOT override retained references.

#### Scenario: Storage used by a backup config

- **WHEN** a member deletes a storage that the backup config of a database of the workspace uses
- **THEN** the response is `409` with `code` `in_use`, and the storage is still listed

#### Scenario: Unused storage

- **WHEN** a member deletes a storage that nothing references and that has no pending file deletion
- **THEN** the response is `204`, and the storage is no longer listed

## ADDED Requirements

### Requirement: Only a storage editor can abandon its cleanup

Abandoning cleanup SHALL require an authenticated actor with permission to edit the storage's workspace. The acknowledgement SHALL NOT bypass authentication, workspace isolation or storage-in-use restrictions. Refusals SHALL reveal no queue contents, paths or provider information. An instance administrator SHALL retain the existing workspace-access privileges.

#### Scenario: No session acknowledges abandonment

- **WHEN** an unauthenticated request deletes a storage with explicit abandonment
- **THEN** the response is 401 with code `unauthenticated`, and nothing changes

#### Scenario: Another workspace acknowledges abandonment

- **WHEN** a non-admin without membership in the storage's workspace requests explicit deletion
- **THEN** the response is 404 with code `not_found`, revealing no cleanup information

#### Scenario: A viewer acknowledges abandonment

- **WHEN** a workspace viewer without editing permission requests explicit deletion
- **THEN** the response is 403 with code `permission_denied`, and the storage and its work remain

#### Scenario: An administrator deletes an unused storage

- **WHEN** an instance administrator without workspace membership explicitly deletes an unused storage with pending work
- **THEN** deletion is authorized under the existing administrator privileges and follows the same abandonment audit rules
