## Purpose

Defines the storages of a workspace: how a storage is created, changed, tested, moved to another workspace and deleted, how its secrets are kept, and who may test settings that are not saved yet.

## ADDED Requirements

### Requirement: A saved storage is returned without its secrets

Creating a storage SHALL answer `201` with the storage as it was saved: its ID, workspace, name, type and settings. Reading a storage, listing storages, creating one and changing one SHALL never return a secret, neither in clear text nor encrypted. The secrets are the S3 access key and secret key, the Google Drive client secret and token, the NAS password, the Azure Blob connection string and account key, the FTP password, the SFTP password and private key, and the rclone configuration. Every other setting SHALL be returned as it was saved.

#### Scenario: Creating an S3 storage

- **WHEN** a member of a workspace creates an S3 storage with a bucket, a region, an access key and a secret key
- **THEN** the response is `201` with the storage's ID, name, type, bucket and region, and the body contains neither key in any form

#### Scenario: Listing storages with secrets

- **WHEN** a member lists the storages of a workspace that has an SFTP storage with a password and a private key
- **THEN** the list shows the SFTP storage with its host, port, user name and path, and without the password and the private key

### Requirement: Changing a storage keeps the secrets the request leaves out

Changing a storage SHALL replace its name and settings with those in the request, except a secret the request leaves out, which SHALL keep its stored value. The S3 and Azure Blob prefixes SHALL be fixed when the storage is created, because every stored file is named under them: an update with another prefix SHALL get `400` with `code` `invalid_request` and SHALL change nothing.

#### Scenario: Renaming an S3 storage

- **WHEN** a member changes the name of an S3 storage and sends its bucket and region without the keys
- **THEN** the storage has the new name, and the next backup to it authenticates with the stored keys

#### Scenario: Changing the S3 prefix

- **WHEN** a member changes a saved S3 storage and sends another prefix
- **THEN** the response is `400` with `code` `invalid_request`, and the storage keeps its prefix

### Requirement: A saved storage keeps its type

A storage's type SHALL be fixed when it is created. Changing a storage with settings of another type SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The edit form SHALL show the type of a saved storage without letting the user change it. A user who needs another kind of storage SHALL create a new storage and point the backup configs at it.

#### Scenario: Local storage sent with S3 settings

- **WHEN** a member changes a saved local storage and sends S3 settings
- **THEN** the response is `400` with `code` `invalid_request`, and the storage is still the same local storage

#### Scenario: Editing a saved storage in the form

- **WHEN** a member opens the edit form of a saved S3 storage
- **THEN** the form shows the type S3 and offers no way to choose another type

### Requirement: Invalid settings are refused without a change

Creating or changing a storage with an empty name, a missing required setting or a setting its provider rejects SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The provider rules that exist today SHALL keep applying: an S3 storage needs a bucket and both keys; a Google Drive storage needs a client ID, a client secret and a token with a refresh token; NAS, FTP and SFTP storages need a host, a port between 1 and 65535 and a user name, NAS also a share, NAS and FTP a password, SFTP a password or a private key; an Azure Blob storage needs a container and, by its authentication method, a connection string or an account name with an account key; an rclone configuration needs exactly one remote section.

#### Scenario: S3 storage without a bucket

- **WHEN** a member creates an S3 storage without a bucket
- **THEN** the response is `400` with `code` `invalid_request`, and the workspace has no new storage

#### Scenario: Invalid change of a saved storage

- **WHEN** a member changes an FTP storage and sends the port `70000`
- **THEN** the response is `400` with `code` `invalid_request`, and the storage keeps its previous port

### Requirement: Storages are listed per workspace in name order

The storages of a workspace SHALL be listed in pages, ordered by name and then by ID. A page SHALL hold at most the requested number of storages and SHALL point to the next page while more storages remain. Listing SHALL need the workspace's ID.

#### Scenario: Workspace with more storages than one page

- **WHEN** a workspace has three storages named `b`, `a` and `c`, and a member lists them two per page
- **THEN** the first page holds `a` and `b` and points to a next page, and the next page holds `c` and points to none

### Requirement: Only an instance admin manages rclone settings

Creating an rclone storage, updating an rclone storage in any way, a rename with unchanged settings included, and testing unsaved rclone settings SHALL be allowed only to an instance admin, because an rclone configuration can read local files of the host and open connections to any address. Any other user SHALL get `403` with `code` `permission_denied`, and nothing SHALL change or leave the backend. Reading, testing, transferring and deleting a saved rclone storage SHALL follow the rules for any saved storage.

#### Scenario: Member creates an rclone storage

- **WHEN** a member of a workspace who is not an instance admin creates an rclone storage
- **THEN** the response is `403` with `code` `permission_denied`, and the workspace has no new storage

#### Scenario: Instance admin creates an rclone storage

- **WHEN** an instance admin creates an rclone storage in a workspace
- **THEN** the response is `201`, and the storage is listed without its configuration

#### Scenario: Member renames an rclone storage

- **WHEN** a member of the workspace who is not an instance admin renames an rclone storage that an instance admin created, sending its remote path without the configuration
- **THEN** the response is `403` with `code` `permission_denied`, and the storage keeps its name

### Requirement: A connection test reports its outcome and changes nothing

Testing a saved storage SHALL run its provider's connection check with the stored settings, and SHALL answer `200` with the outcome: connected, or not connected together with the error the storage gave. Each provider keeps the check it runs today: it connects with the settings and reaches the configured location, and the S3, Azure Blob, Google Drive and rclone providers also write a probe object there and remove it. A viewer of the workspace SHALL be allowed to test a saved storage. A test SHALL change nothing in the saved storage, its secrets included.

#### Scenario: Test of a working storage

- **WHEN** a viewer tests a saved local storage whose folders are writable
- **THEN** the response is `200` and reports the storage as connected

#### Scenario: Test of a storage that refuses the write

- **WHEN** a member tests a saved S3 storage whose bucket does not exist
- **THEN** the response is `200` and reports the storage as not connected with an error that names the missing bucket

### Requirement: Unsaved settings can be tested before saving

Testing settings that are not saved yet SHALL run the same connection check with the request's settings and SHALL answer with the outcome, like a test of a saved storage. When the request names a saved storage of the same workspace, a secret the request leaves out SHALL be taken from that storage. When it names a storage that does not exist or belongs to another workspace, the test SHALL get `404` with `code` `not_found`, and nothing SHALL leave the backend. When it names a storage of another type than the request's settings, or an S3 or Azure Blob storage with another prefix, the test SHALL get `400` with `code` `invalid_request`, and nothing SHALL leave the backend, so the stored secrets of one storage never reach settings that a save of it would refuse. Such a test SHALL change no saved storage. Settings that a save would refuse SHALL get `400` with `code` `invalid_request`, and nothing SHALL leave the backend.

#### Scenario: New storage tested from the form

- **WHEN** a member fills in an FTP storage in the form and tests it before saving
- **THEN** the backend connects to the entered FTP server with the entered user name, and the workspace has no new storage

#### Scenario: Edited storage tested with its stored secret

- **WHEN** a member edits the path of a saved SFTP storage and tests the form without entering the password
- **THEN** the backend signs in to the SFTP server with the stored password and creates the new path there, and the saved storage keeps its old path

#### Scenario: Storage of another workspace named in the test

- **WHEN** a member of one workspace tests unsaved NAS settings without a password and names a NAS storage of another workspace
- **THEN** the response is `404` with `code` `not_found`, and no connection leaves the backend

#### Scenario: Storage of another type named in the test

- **WHEN** a member tests unsaved FTP settings without a password and names a saved SFTP storage of the same workspace
- **THEN** the response is `400` with `code` `invalid_request`, and no connection leaves the backend

### Requirement: Only a user who may change storages tests unsaved settings

Testing unsaved storage settings SHALL need the right to change the workspace's content. A test with the ID of a saved storage sends that storage's stored secrets to the host, endpoint or remote named in the request, so a user who may only view the workspace SHALL get `403` with `code` `permission_denied`, and no connection SHALL leave the backend.

#### Scenario: Viewer tests settings with a saved storage's ID

- **WHEN** a viewer of a workspace tests NAS settings that name a host of the viewer's choice and the ID of a saved NAS storage of the workspace, without a password
- **THEN** the response is `403` with `code` `permission_denied`, and no connection reaches that host

### Requirement: A storage that backups use cannot be deleted

Deleting a storage that a backup config uses or that still holds backups SHALL get `409` with `code` `in_use` and SHALL change nothing, as the `storage-file-cleanup` spec describes. Deleting a storage nothing references SHALL answer `204`, and the storage SHALL no longer be listed.

#### Scenario: Storage used by a backup config

- **WHEN** a member deletes a storage that the backup config of a database of the workspace uses
- **THEN** the response is `409` with `code` `in_use`, and the storage is still listed

#### Scenario: Unused storage

- **WHEN** a member deletes a storage that nothing references
- **THEN** the response is `204`, and the storage is no longer listed

### Requirement: A storage moves to another workspace only with its databases

Transferring a storage SHALL require the right to change content in both the storage's workspace and the target workspace. A transfer SHALL get `409` with `code` `in_use` and SHALL change nothing while the backup config of a database outside the target workspace uses the storage. Transferring a database together with its storage SHALL move the database and the storage, and SHALL leave the database's backup config on that storage.

#### Scenario: Unused storage to another workspace

- **WHEN** a user who is a member of two workspaces transfers an unused storage from the first to the second
- **THEN** the storage is listed in the second workspace and no longer in the first

#### Scenario: Storage still used in its workspace

- **WHEN** a member transfers a storage that the backup config of a database of its current workspace uses
- **THEN** the response is `409` with `code` `in_use`, and the storage stays in its workspace

#### Scenario: Target workspace where the user is a viewer

- **WHEN** a member of the storage's workspace who is only a viewer of the target workspace transfers the storage
- **THEN** the response is `403` with `code` `permission_denied`, and the storage stays in its workspace

#### Scenario: Database transferred with its storage

- **WHEN** a member transfers a database to another workspace and asks to take its storage along
- **THEN** the database and its storage are both in the target workspace, and the database's backup config still uses the storage

### Requirement: An upgrade keeps every storage's secrets

Upgrading Databasus SHALL keep every stored secret of every storage as it was, encrypted values included, so each storage reads and writes with the secrets saved before the upgrade.

#### Scenario: S3 keys and SFTP key after the upgrade

- **WHEN** an instance has an S3 storage with encrypted keys and an SFTP storage with an encrypted private key, and it is upgraded to this release
- **THEN** all stored values are unchanged, and a connection test of the S3 storage authenticates with the keys saved before the upgrade

### Requirement: Stored secrets stay encrypted

Every secret of a storage SHALL be stored encrypted with the instance's master key, whether it was entered by the user or produced by the backend, such as a refreshed Google Drive token. No operation SHALL write a storage's secret to the database in clear text.

#### Scenario: Connection test writes nothing

- **WHEN** a member tests a saved S3 storage and then a saved local storage
- **THEN** no column of either storage or of its provider settings changes, so no test can store a credential that the backend changed in memory, such as a refreshed Google Drive token
