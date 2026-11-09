# storages Specification

## Purpose
Defines the storages of a workspace: how a storage is created, changed, tested, moved to another workspace and deleted, how its secrets are kept, and who may test settings that are not saved yet.

## Requirements

### Requirement: A saved storage is returned without its secrets

Creating a storage SHALL answer `201` with the storage as it was saved: its ID, workspace, name, type and settings. Reading a storage, listing storages, creating one and changing one SHALL never return a secret, neither in clear text nor encrypted. The secrets are the S3 access key and secret key, the NAS password, the Azure Blob connection string and account key, the FTP password, the SFTP password and private key. Every other setting SHALL be returned as it was saved.

#### Scenario: Creating an S3 storage

- **WHEN** a member of a workspace creates an S3 storage with a bucket, a region, an access key and a secret key
- **THEN** the response is `201` with the storage's ID, name, type, bucket and region, and the body contains neither key in any form

#### Scenario: Listing storages with secrets

- **WHEN** a member lists the storages of a workspace that has an SFTP storage with a password and a private key
- **THEN** the list shows the SFTP storage with its host, port, user name and path, and without the password and the private key

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

### Requirement: A saved storage keeps its type

A storage's type SHALL be fixed when it is created. Changing a storage with settings of another type SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The edit form SHALL show the type of a saved storage without letting the user change it. A user who needs another kind of storage SHALL create a new storage and point the backup configs at it.

#### Scenario: Local storage sent with S3 settings

- **WHEN** a member changes a saved local storage and sends S3 settings
- **THEN** the response is `400` with `code` `invalid_request`, and the storage is still the same local storage

#### Scenario: Editing a saved storage in the form

- **WHEN** a member opens the edit form of a saved S3 storage
- **THEN** the form shows the type S3 and offers no way to choose another type

### Requirement: Invalid settings are refused without a change

Creating or changing a storage with an empty name, a missing required setting or a setting its provider rejects SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The provider rules that exist today SHALL keep applying: an S3 storage needs a bucket and both keys; NAS, FTP and SFTP storages need a host, a port between 1 and 65535 and a user name, NAS also a share, NAS and FTP a password, SFTP a password or a private key; an Azure Blob storage needs a container and, by its authentication method, a connection string or an account name with an account key.

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

### Requirement: A connection test reports its outcome and changes nothing

Testing a saved storage SHALL run its provider's connection check with the stored settings, and SHALL answer `200` with the outcome: connected, or not connected together with the error the storage gave. Each provider keeps the check it runs today: it connects with the settings and reaches the configured location, and the S3 and Azure Blob providers also write a probe object there and remove it. A viewer of the workspace SHALL be allowed to test a saved storage. A test SHALL change nothing in the saved storage, its secrets included.

#### Scenario: Test of a working storage

- **WHEN** a viewer tests a saved local storage whose folders are writable
- **THEN** the response is `200` and reports the storage as connected

#### Scenario: Test of a storage that refuses the write

- **WHEN** a member tests a saved S3 storage whose bucket does not exist
- **THEN** the response is `200` and reports the storage as not connected with an error that names the missing bucket

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

### Requirement: Only a user who may change storages tests unsaved settings

Testing unsaved storage settings SHALL need the right to change the workspace's content. A test with the ID of a saved storage uses that storage's stored secrets with the other settings the request names, and a test without an ID makes the backend connect wherever the request points, so a user who may only view the workspace SHALL get `403` with `code` `permission_denied`, and no connection SHALL leave the backend.

#### Scenario: Viewer tests settings with a saved storage's ID

- **WHEN** a viewer of a workspace tests NAS settings that name a host of the viewer's choice and the ID of a saved NAS storage of the workspace, without a password
- **THEN** the response is `403` with `code` `permission_denied`, and no connection reaches that host

### Requirement: A storage that backups use cannot be deleted

Deleting a storage that a backup config uses or that still holds backups SHALL get `409` with `code` `in_use` and SHALL change nothing, as the `storage-file-cleanup` spec describes. Deleting an otherwise unused storage SHALL follow the pending-cleanup acknowledgement rules of `storage-file-cleanup`: queued files require explicit abandonment, and successful deletion answers 204. Abandonment SHALL NOT override retained references.

#### Scenario: Storage used by a backup config

- **WHEN** a member deletes a storage that the backup config of a database of the workspace uses
- **THEN** the response is `409` with `code` `in_use`, and the storage is still listed

#### Scenario: Unused storage

- **WHEN** a member deletes a storage that nothing references and that has no pending file deletion
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

Upgrading Databasus SHALL keep every stored secret of every supported storage as it was, encrypted values included, so each storage reads and writes with the secrets saved before the upgrade.

#### Scenario: S3 keys and SFTP key after the upgrade

- **WHEN** an instance has an S3 storage with encrypted keys and an SFTP storage with an encrypted private key, and it is upgraded to this release
- **THEN** all stored values are unchanged, and a connection test of the S3 storage authenticates with the keys saved before the upgrade

### Requirement: Stored secrets stay encrypted

Every secret of a storage SHALL be stored encrypted with the instance's master key, whether it was entered by the user or produced by the backend. No operation SHALL write a storage's secret to the database in clear text.

#### Scenario: Connection test writes nothing

- **WHEN** a member tests a saved S3 storage and then a saved local storage
- **THEN** no column of either storage or of its provider settings changes, so no test can store a credential that the backend changed in memory

### Requirement: S3 storage classes are provider-defined settings

Creating, reading, changing and testing an S3-compatible storage SHALL accept a storage class supplied by the user without restricting it to the application's suggested classes. The application SHALL preserve and pass that value to the storage service exactly. An omitted or empty class SHALL select the bucket's default. The edit form SHALL allow a custom class and SHALL preserve an existing custom class when another field changes.

A remote service's rejection of an unsupported class SHALL be reported through the normal connection-test or backup failure outcome; it SHALL NOT be silently retried with a different class.

#### Scenario: Saving a custom class

- **WHEN** a member creates an otherwise valid S3 storage with class COLD
- **THEN** creation succeeds and reading the storage returns COLD

#### Scenario: Editing another field

- **WHEN** a member renames a storage whose class is DEEP_ARCHIVE without changing that class
- **THEN** the saved storage still uses DEEP_ARCHIVE

#### Scenario: Default class

- **WHEN** a member saves an S3 storage without a class or with an empty class
- **THEN** writes leave the class choice to the bucket

#### Scenario: Provider rejects the class

- **WHEN** the storage service rejects the configured class during a connection test
- **THEN** the test reports the provider failure and the saved class stays unchanged

### Requirement: SFTP settings declare their authentication methods without revealing credentials

SFTP create, update and unsaved-test requests SHALL contain `authMethods`, a nonempty set drawn from `PASSWORD` and `PRIVATE_KEY`. Duplicate, unsupported, empty or absent method sets, empty supplied credentials and credentials supplied for an excluded method SHALL get `400` with `code` `invalid_request`. Reading, listing, creating and changing a storage SHALL report the methods represented by its persisted credentials without returning either credential. The interface SHALL display and edit password-only, private-key-only and dual-credential settings without inferring the method from secret fields. Existing role and workspace access rules SHALL remain in force.

#### Scenario: Reading a password-only storage

- **WHEN** a member reads an SFTP storage with a stored password and no private key
- **THEN** its settings report only `PASSWORD`, and neither secret is present

#### Scenario: Reading a key-only storage

- **WHEN** a member reads an SFTP storage with a stored private key and no password
- **THEN** its settings and interface report only `PRIVATE_KEY`, and neither secret is present

#### Scenario: Reading a dual-credential storage

- **WHEN** a member reads an SFTP storage that already has both credentials
- **THEN** both methods are reported and displayed, without exposing or removing either credential

#### Scenario: A caller omits the method set

- **WHEN** a create, update or unsaved-test request omits SFTP `authMethods`
- **THEN** it receives `400` with `code` `invalid_request`, no credential is inferred and no connection leaves the backend

#### Scenario: A credential contradicts the selected methods

- **WHEN** a request selects only password authentication but supplies a private key
- **THEN** it receives `400` with `code` `invalid_request`, changes nothing and connects nowhere

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

### Requirement: Skipping SFTP host key verification has a visible security warning

Whenever the SFTP host key verification option is visible in the storage editor, the interface SHALL show adjacent explanatory text warning that skipping verification reduces security. The warning SHALL be visible without hovering, focusing or enabling the option, SHALL be available as the checkbox's accessible description and SHALL use the selected interface language in each of the six supported languages.

#### Scenario: Reviewing the option before enabling it

- **WHEN** a user opens the SFTP editor with host key verification enabled and expands the advanced settings
- **THEN** the unchecked option to skip verification has a visible warning that skipping verification reduces security
- **AND** assistive technology can read that warning as the checkbox's description

#### Scenario: Editing settings that already skip verification

- **WHEN** a user opens the SFTP editor with the option to skip host key verification already enabled
- **THEN** the advanced settings show the checked option and its security warning immediately

#### Scenario: Displaying the warning in each supported language

- **WHEN** a user views the SFTP host key verification option in any of the six supported interface languages
- **THEN** the visible warning and the checkbox's accessible description use that language

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

### Requirement: Built-in Google Drive storage is unavailable

The supported built-in storage types SHALL be local, S3-compatible, NAS, Azure Blob, FTP and SFTP. The public API SHALL NOT advertise or accept `GOOGLE_DRIVE` settings. Creating, updating or testing unsaved settings with that type SHALL return 400 with code `invalid_request`, SHALL change nothing and SHALL make no request to Google. The interface SHALL offer no Google Drive storage option, editor, details or storage authorization flow. Visiting the retired storage callback SHALL follow ordinary unknown-route behavior without exchanging an authorization code or creating or changing a storage. Google account sign-in SHALL retain its existing behavior.

#### Scenario: A caller submits retired settings

- **WHEN** a caller submits `GOOGLE_DRIVE` settings to storage creation, update or the unsaved connection test
- **THEN** the response is 400 with code `invalid_request`, nothing changes and no request reaches Google

#### Scenario: A user chooses storage

- **WHEN** a user chooses a storage provider in the storage list or the editor opened from backup settings
- **THEN** Google Drive is absent and the remaining supported providers are available under their existing permissions

#### Scenario: An old storage callback arrives

- **WHEN** a browser visits the retired Google Drive storage callback with an authorization code
- **THEN** the application follows its ordinary unknown-route behavior without exchanging the code or mutating storage

#### Scenario: Google account sign-in remains available

- **WHEN** a user signs in through a configured Google identity provider
- **THEN** the existing account sign-in flow continues to work independently of storage support

### Requirement: Public materials no longer offer Google Drive storage

The website and README in every supported language SHALL contain no Google Drive storage mentions, setup instructions, provider links or images. Retired setup addresses SHALL have no dedicated page or redirect and SHALL return the website's ordinary not-found response. Navigation, search-engine metadata, structured data, machine-readable page listings and language alternatives SHALL contain no links to those retired pages. Public materials SHALL continue to describe supported storage providers accurately.

Current project documentation SHALL contain no explanations about historical Google Drive support, its removal or upgrade recovery. Recovery guidance SHALL remain in startup and migration diagnostics, with its behavior and regression coverage defined by the internal-database-migrations and upgrade-test specifications.

#### Scenario: Reading public materials in any language

- **WHEN** a reader opens the home page, storage documentation, installation instructions, comparison pages or README in any supported language
- **THEN** no content offers or refers to Google Drive storage and the remaining provider descriptions and links are valid

#### Scenario: Reading current project documentation

- **WHEN** a reader opens the architecture, testing or other current project documentation
- **THEN** it contains no explanation of historical Google Drive support, its removal or upgrade recovery
- **AND** an upgrade blocked by a Google Drive storage still gives the recovery diagnostic required by the internal-database-migrations specification

#### Scenario: Opening retired setup addresses

- **WHEN** a reader requests either former Google Drive setup address or a former localized setup page
- **THEN** the website returns its ordinary not-found response without a Google Drive guide or redirect

#### Scenario: Discovering documentation pages

- **WHEN** a consumer reads the website's navigation, sitemap, machine-readable page listing or language alternatives
- **THEN** none points to a retired Google Drive page

### Requirement: Rclone storage is unavailable

The supported storage types SHALL be local, S3-compatible, NAS, Azure Blob, FTP and SFTP. The public API SHALL NOT advertise or accept `RCLONE` settings. An authenticated caller with permission to edit the workspace submitting that type to storage creation, update or the unsaved connection test SHALL receive `400` with code `invalid_request`, even if the caller is an instance admin. The request SHALL change nothing and SHALL make no storage connection or read any local file named by the supplied configuration. The interface SHALL offer no rclone storage option, editor, details or configuration documentation link, including when storage creation is opened from backup settings. Supported types SHALL retain their validation, secret handling, connection tests and permissions.

#### Scenario: An instance admin submits rclone settings

- **WHEN** an instance admin submits `RCLONE` settings to storage creation, update or the unsaved connection test
- **THEN** the response is `400` with code `invalid_request`, nothing changes and no configured destination or local file is accessed

#### Scenario: A workspace member submits rclone settings

- **WHEN** a member allowed to edit workspace content submits `RCLONE` settings
- **THEN** the response rejects the unsupported type with `400 invalid_request` rather than treating it as an admin-only provider

#### Scenario: A user chooses a storage provider

- **WHEN** a user opens storage creation from the storage list or backup settings
- **THEN** local, S3-compatible, NAS, Azure Blob, FTP and SFTP are available and rclone is absent

### Requirement: Public storage descriptions offer only supported destinations

The website and README in every supported language SHALL contain no rclone or Dropbox storage offers, configuration instructions, provider images, provider links or claims of access to more than 70 providers. This SHALL include visible text, provider comparisons, search metadata, structured data and machine-readable page listings. Prominent home-page and README storage summaries SHALL replace unsupported destinations with supported ones such as SFTP, FTP or Azure Blob Storage, keeping approximately their existing text length and preserving surrounding markup. The result SHALL remain readable at narrow and wide viewport sizes without clipped or overflowing text.

Current project documentation and code comments SHALL describe current behavior without historical rclone support, removal notices or upgrade recovery explanations. Recovery guidance SHALL remain in startup and migration diagnostics and its behavioral specifications. Historical migrations and focused refusal and recovery tests SHALL remain available; OpenSpec history SHALL NOT be rewritten to remove historical references.

#### Scenario: Reading public materials in any language

- **WHEN** a reader opens a home page, database-backup page, storage documentation, security page, comparison page or README in any supported language
- **THEN** storage descriptions offer supported destinations without rclone, Dropbox or the broad provider-count claim

#### Scenario: A machine reads website metadata

- **WHEN** a consumer reads page metadata, structured data or the machine-readable documentation listing
- **THEN** none advertises rclone or Dropbox support

#### Scenario: A prominent summary is rendered at different widths

- **WHEN** a home-page or README summary is rendered after replacing unsupported storage names
- **THEN** the surrounding structure is preserved and its text remains readable without clipping or overflow

#### Scenario: Reading current project documentation and comments

- **WHEN** a reader consults current architecture, glossary or code comments
- **THEN** they describe current behavior without explaining that rclone once existed or how its removal was implemented
- **AND** an operator blocked by existing storages still receives actionable runtime recovery guidance
