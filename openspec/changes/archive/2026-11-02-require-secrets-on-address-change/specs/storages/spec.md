## MODIFIED Requirements

### Requirement: Changing a storage keeps the secrets the request leaves out

Changing a storage SHALL replace its name and settings with those in the request, except a secret the request leaves out, which SHALL keep its stored value, unless the request changes the storage's resource address as the `secret-handling` spec describes. For SFTP, this preservation SHALL apply only to credentials included in the explicitly submitted authentication-method set; excluded credentials SHALL be removed. A selected SFTP credential missing from both the request and stored settings SHALL cause `400` with `code` `invalid_request` and SHALL change nothing. The S3 and Azure Blob prefixes SHALL be fixed when the storage is created, because every stored file is named under them: an update with another prefix SHALL get `400` with `code` `invalid_request` and SHALL change nothing.

#### Scenario: Renaming an S3 storage

- **WHEN** a member changes the name of an S3 storage and sends its bucket and region without the keys
- **THEN** the storage has the new name, and the next backup to it authenticates with the stored keys

#### Scenario: Changing the S3 prefix

- **WHEN** a member changes a saved S3 storage and sends another prefix
- **THEN** the response is `400` with `code` `invalid_request`, and the storage keeps its prefix

#### Scenario: Renaming a key-based SFTP storage without secrets

- **WHEN** a member renames an SFTP storage, retains private-key authentication and omits the private key
- **THEN** the name changes and the stored private key remains unchanged
- **AND** reading the storage reports private-key authentication without returning the key

#### Scenario: Switching SFTP authentication to a password

- **WHEN** a member selects password authentication, excludes private-key authentication and supplies a password or already has one stored
- **THEN** saving retains or replaces the selected password and removes the private key
- **AND** a subsequent read reports only password authentication

#### Scenario: Switching SFTP authentication to a private key

- **WHEN** a member selects private-key authentication, excludes password authentication and supplies a private key or already has one stored
- **THEN** saving retains or replaces the selected private key and removes the password
- **AND** a subsequent read reports only private-key authentication

#### Scenario: A newly selected credential is missing

- **WHEN** a member selects a method without supplying its credential and that credential is not already stored
- **THEN** the response is `400` with `code` `invalid_request`, and all stored settings and credentials remain unchanged

#### Scenario: An SFTP storage moved to another host without its credential

- **WHEN** a member changes the host of an SFTP storage, retains private-key authentication and omits the private key
- **THEN** the response is `400` with `code` `invalid_request`, and the storage keeps its host and its private key

### Requirement: Unsaved settings can be tested before saving

Testing settings that are not saved yet SHALL run the same connection check with the request's settings and SHALL answer with the outcome, like a test of a saved storage. When the request names a saved storage of the same workspace, a secret the request leaves out SHALL be taken from that storage, unless the request changes the storage's resource address as the `secret-handling` spec describes. For SFTP, this fallback SHALL apply only to credentials included in the explicitly submitted authentication-method set; excluded credentials SHALL NOT be taken from the named storage or tried during the test. A selected SFTP credential missing from both the request and stored settings SHALL cause `400` with `code` `invalid_request` before connecting. When it names a storage that does not exist or belongs to another workspace, the test SHALL get `404` with `code` `not_found`, and nothing SHALL leave the backend. When it names a storage of another type than the request's settings, or an S3 or Azure Blob storage with another prefix, the test SHALL get `400` with `code` `invalid_request`, and nothing SHALL leave the backend, so the stored secrets of one storage never reach settings that a save of it would refuse. Such a test SHALL change no saved storage. Settings that a save would refuse SHALL get `400` with `code` `invalid_request`, and nothing SHALL leave the backend. The existing requirement that only users allowed to change storages can test unsaved settings SHALL remain in force.

#### Scenario: New storage tested from the form

- **WHEN** a member fills in an FTP storage in the form and tests it before saving
- **THEN** the backend connects to the entered FTP server with the entered user name, and the workspace has no new storage

#### Scenario: Edited storage tested with its stored secret

- **WHEN** a member edits the path of a saved SFTP storage, retains password authentication and tests the form without entering the password
- **THEN** the backend signs in to the SFTP server with the stored password and creates the new path there, and the saved storage keeps its old path

#### Scenario: Saved storage tested at another address

- **WHEN** a member tests unsaved NAS settings that name a saved NAS storage of the workspace and another host, without a password
- **THEN** the response is `400` with `code` `invalid_request`, and no connection reaches that host

#### Scenario: Storage of another workspace named in the test

- **WHEN** a member of one workspace tests unsaved NAS settings without a password and names a NAS storage of another workspace
- **THEN** the response is `404` with `code` `not_found`, and no connection leaves the backend

#### Scenario: Storage of another type named in the test

- **WHEN** a member tests unsaved FTP settings without a password and names a saved SFTP storage of the same workspace
- **THEN** the response is `400` with `code` `invalid_request`, and no connection leaves the backend

#### Scenario: Excluded SFTP credential is already stored

- **WHEN** a member submits a private-key-only test against a storage with a saved password
- **THEN** the saved password is not used, and the test uses a supplied or retained private key or rejects a missing key before connecting
- **AND** the saved credentials remain unchanged

### Requirement: SFTP testing uses the exact credentials that saving would select

An unsaved SFTP test SHALL resolve each selected method from its supplied credential or, unless the request changes the storage's resource address as the `secret-handling` spec describes, the named storage's corresponding stored credential. It SHALL NOT try an excluded method or use one method's credential to satisfy another. An invalid selection SHALL be rejected before connecting. Testing SHALL NOT remove, replace or persist any credential, even when the connection succeeds. Saved-storage tests SHALL retain the exact persisted method set. Existing permissions and checks on the named storage's workspace and type SHALL continue to apply before any connection.

#### Scenario: Testing a switch before saving

- **WHEN** a member tests private-key-only settings against a password-based saved storage and supplies a private key
- **THEN** the connection uses only that key
- **AND** the saved password and saved method set remain unchanged

#### Scenario: An excluded password would make the test pass

- **WHEN** a storage has a valid saved password but a private-key-only test supplies a key that the server rejects
- **THEN** the test reports failure without trying the saved password

#### Scenario: Testing a retained credential

- **WHEN** a member tests a new path while retaining the saved method set and omitting its credentials
- **THEN** the test uses those selected saved credentials and changes no stored path or secret

#### Scenario: Testing at another host without a selected credential

- **WHEN** a member tests settings that name a saved SFTP storage and another host, retains the saved method set and omits its credentials
- **THEN** the response is `400` with `code` `invalid_request`, and no connection reaches that host

### Requirement: SFTP upgrades and unrelated edits preserve existing credentials

Upgrading SHALL derive authentication methods from the existing stored credentials without rewriting or deleting them. Changing the name, path or other settings outside the resource address and the credentials while retaining the reported method set SHALL preserve every encrypted credential value, including a storage with both credentials. Supplying one selected credential SHALL replace only that credential; another selected credential that is omitted SHALL remain unchanged, unless the request changes the storage's resource address as the `secret-handling` spec describes. Only a successful explicit method change SHALL remove an excluded credential.

#### Scenario: A dual-credential storage is renamed after upgrade

- **WHEN** an existing storage with both credentials is read and renamed with both methods selected and neither secret supplied
- **THEN** both encrypted values remain unchanged and a subsequent read reports both methods

#### Scenario: One credential is replaced

- **WHEN** a member retains both methods and supplies a replacement password without a private key
- **THEN** only the password is replaced and both methods remain selected

#### Scenario: One credential is replaced at another host

- **WHEN** a member retains both methods, changes the host and supplies a password without a private key
- **THEN** the response is `400` with `code` `invalid_request` and a field error for the private key, and the host and both credentials stay unchanged

### Requirement: Only a user who may change storages tests unsaved settings

Testing unsaved storage settings SHALL need the right to change the workspace's content. A test with the ID of a saved storage uses that storage's stored secrets with the other settings the request names, and a test without an ID makes the backend connect wherever the request points, so a user who may only view the workspace SHALL get `403` with `code` `permission_denied`, and no connection SHALL leave the backend.

#### Scenario: Viewer tests settings with a saved storage's ID

- **WHEN** a viewer of a workspace tests NAS settings that name a host of the viewer's choice and the ID of a saved NAS storage of the workspace, without a password
- **THEN** the response is `403` with `code` `permission_denied`, and no connection reaches that host

## ADDED Requirements

### Requirement: The storage form asks for a secret again after an address change

While the resource address of a saved storage in the edit form differs from the saved one, as the `secret-handling` spec defines it, the form SHALL require each secret that the settings in the form use before it offers the test and the save, and SHALL show under each such empty field a message that the address changed and the secret must be entered again. The SFTP form SHALL NOT show its saved-credential text for such a field. When the address returns to the saved value, the message SHALL disappear, the secret SHALL stop being required and the SFTP form SHALL show its saved-credential text again. A value the user typed SHALL stay in the field and be sent in the request. A secret that the settings in the form do not use SHALL stay optional.

#### Scenario: Endpoint changed in the form

- **WHEN** a member changes the endpoint of a saved S3 storage in the form and leaves the keys empty
- **THEN** the message appears under both key fields, and the test and save controls are unavailable

#### Scenario: Endpoint returned to the saved value

- **WHEN** a member changes the endpoint of a saved S3 storage in the form and then restores its saved value
- **THEN** the message disappears, and the test control is available

#### Scenario: Address returned after a secret was typed

- **WHEN** a member changes the host of a saved FTP storage, types a password and then restores the saved host
- **THEN** the password stays in the field, and the test request carries it

#### Scenario: A change outside the address in the form

- **WHEN** a member changes the path of a saved FTP storage in the form and leaves the password empty
- **THEN** the test control is available, and the request omits the password

#### Scenario: NAS host changed in the form

- **WHEN** a member changes the host of a saved NAS storage in the form and leaves the password empty
- **THEN** the message appears under the password field, and the test control is unavailable

#### Scenario: SFTP host changed with one method selected

- **WHEN** a member changes the host of a saved SFTP storage with both credentials, selects only private-key authentication and leaves the private key empty
- **THEN** the message appears under the private key field, no saved-credential text is shown, no password is required, and the test control is unavailable

#### Scenario: Azure address changed with account key authentication

- **WHEN** a member changes the endpoint or the account name of a saved Azure Blob storage with account key authentication and leaves the account key empty
- **THEN** the message appears under the account key field, and the test control is unavailable

#### Scenario: Azure address changed with a connection string

- **WHEN** a member changes the endpoint of a saved Azure Blob storage that uses a connection string and leaves the connection string empty
- **THEN** no secret is required, and the test control is available
