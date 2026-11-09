## MODIFIED Requirements

### Requirement: A saved storage is returned without its secrets

Creating a storage SHALL answer `201` with the storage as it was saved: its ID, workspace, name, type and settings. Reading a storage, listing storages, creating one and changing one SHALL never return a secret, neither in clear text nor encrypted. The secrets are the S3 access key and secret key, the NAS password, the Azure Blob connection string and account key, the FTP password, the SFTP password and private key, and the rclone configuration. Every other setting SHALL be returned as it was saved.

#### Scenario: Creating an S3 storage

- **WHEN** a member of a workspace creates an S3 storage with a bucket, a region, an access key and a secret key
- **THEN** the response is `201` with the storage's ID, name, type, bucket and region, and the body contains neither key in any form

#### Scenario: Listing storages with secrets

- **WHEN** a member lists the storages of a workspace that has an SFTP storage with a password and a private key
- **THEN** the list shows the SFTP storage with its host, port, user name and path, and without the password and the private key

### Requirement: Invalid settings are refused without a change

Creating or changing a storage with an empty name, a missing required setting or a setting its provider rejects SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The provider rules that exist today SHALL keep applying: an S3 storage needs a bucket and both keys; NAS, FTP and SFTP storages need a host, a port between 1 and 65535 and a user name, NAS also a share, NAS and FTP a password, SFTP a password or a private key; an Azure Blob storage needs a container and, by its authentication method, a connection string or an account name with an account key; an rclone configuration needs exactly one remote section.

#### Scenario: S3 storage without a bucket

- **WHEN** a member creates an S3 storage without a bucket
- **THEN** the response is `400` with `code` `invalid_request`, and the workspace has no new storage

#### Scenario: Invalid change of a saved storage

- **WHEN** a member changes an FTP storage and sends the port `70000`
- **THEN** the response is `400` with `code` `invalid_request`, and the storage keeps its previous port

### Requirement: An upgrade keeps every storage's secrets

Upgrading Databasus SHALL keep every stored secret of every supported storage as it was, encrypted values included, so each storage reads and writes with the secrets saved before the upgrade.

#### Scenario: S3 keys and SFTP key after the upgrade

- **WHEN** an instance has an S3 storage with encrypted keys and an SFTP storage with an encrypted private key, and it is upgraded to this release
- **THEN** all stored values are unchanged, and a connection test of the S3 storage authenticates with the keys saved before the upgrade

### Requirement: A connection test reports its outcome and changes nothing

Testing a saved storage SHALL run its provider's connection check with the stored settings, and SHALL answer `200` with the outcome: connected, or not connected together with the error the storage gave. Each provider keeps the check it runs today: it connects with the settings and reaches the configured location, and the S3, Azure Blob and rclone providers also write a probe object there and remove it. A viewer of the workspace SHALL be allowed to test a saved storage. A test SHALL change nothing in the saved storage, its secrets included.

#### Scenario: Test of a working storage

- **WHEN** a viewer tests a saved local storage whose folders are writable
- **THEN** the response is `200` and reports the storage as connected

#### Scenario: Test of a storage that refuses the write

- **WHEN** a member tests a saved S3 storage whose bucket does not exist
- **THEN** the response is `200` and reports the storage as not connected with an error that names the missing bucket

### Requirement: Stored secrets stay encrypted

Every secret of a storage SHALL be stored encrypted with the instance's master key, whether it was entered by the user or produced by the backend. No operation SHALL write a storage's secret to the database in clear text.

#### Scenario: Connection test writes nothing

- **WHEN** a member tests a saved S3 storage and then a saved local storage
- **THEN** no column of either storage or of its provider settings changes, so no test can store a credential that the backend changed in memory

## ADDED Requirements

### Requirement: Built-in Google Drive storage is unavailable

The supported built-in storage types SHALL be local, S3-compatible, NAS, Azure Blob, FTP, SFTP and rclone. The public API SHALL NOT advertise or accept `GOOGLE_DRIVE` settings. Creating, updating or testing unsaved settings with that type SHALL return 400 with code `invalid_request`, SHALL change nothing and SHALL make no request to Google. The interface SHALL offer no Google Drive storage option, editor, details or storage authorization flow. Visiting the retired storage callback SHALL follow ordinary unknown-route behavior without exchanging an authorization code or creating or changing a storage. Google account sign-in and the generic rclone integration SHALL retain their existing behavior.

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
