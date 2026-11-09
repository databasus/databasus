## MODIFIED Requirements

### Requirement: A saved storage is returned without its secrets

Creating a storage SHALL answer `201` with the storage as it was saved: its ID, workspace, name, type and settings. Reading a storage, listing storages, creating one and changing one SHALL never return a secret, neither in clear text nor encrypted. The secrets are the S3 access key and secret key, the NAS password, the Azure Blob connection string and account key, the FTP password, the SFTP password and private key. Every other setting SHALL be returned as it was saved.

#### Scenario: Creating an S3 storage

- **WHEN** a member of a workspace creates an S3 storage with a bucket, a region, an access key and a secret key
- **THEN** the response is `201` with the storage's ID, name, type, bucket and region, and the body contains neither key in any form

#### Scenario: Listing storages with secrets

- **WHEN** a member lists the storages of a workspace that has an SFTP storage with a password and a private key
- **THEN** the list shows the SFTP storage with its host, port, user name and path, and without the password and the private key

### Requirement: Invalid settings are refused without a change

Creating or changing a storage with an empty name, a missing required setting or a setting its provider rejects SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The provider rules that exist today SHALL keep applying: an S3 storage needs a bucket and both keys; NAS, FTP and SFTP storages need a host, a port between 1 and 65535 and a user name, NAS also a share, NAS and FTP a password, SFTP a password or a private key; an Azure Blob storage needs a container and, by its authentication method, a connection string or an account name with an account key.

#### Scenario: S3 storage without a bucket

- **WHEN** a member creates an S3 storage without a bucket
- **THEN** the response is `400` with `code` `invalid_request`, and the workspace has no new storage

#### Scenario: Invalid change of a saved storage

- **WHEN** a member changes an FTP storage and sends the port `70000`
- **THEN** the response is `400` with `code` `invalid_request`, and the storage keeps its previous port

### Requirement: A connection test reports its outcome and changes nothing

Testing a saved storage SHALL run its provider's connection check with the stored settings, and SHALL answer `200` with the outcome: connected, or not connected together with the error the storage gave. Each provider keeps the check it runs today: it connects with the settings and reaches the configured location, and the S3 and Azure Blob providers also write a probe object there and remove it. A viewer of the workspace SHALL be allowed to test a saved storage. A test SHALL change nothing in the saved storage, its secrets included.

#### Scenario: Test of a working storage

- **WHEN** a viewer tests a saved local storage whose folders are writable
- **THEN** the response is `200` and reports the storage as connected

#### Scenario: Test of a storage that refuses the write

- **WHEN** a member tests a saved S3 storage whose bucket does not exist
- **THEN** the response is `200` and reports the storage as not connected with an error that names the missing bucket

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

## REMOVED Requirements

### Requirement: Only an instance admin manages rclone settings

**Reason**: rclone settings are not a supported storage variant for any role.

**Migration**: Remove existing rclone storages through the supporting release identified by the pre-upgrade diagnostic, then retry the upgrade. Other storage types keep their workspace permissions.

## ADDED Requirements

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
