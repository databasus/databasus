## MODIFIED Requirements

### Requirement: Data written before the upgrade is there after it

Before the upgrade, the test SHALL write through the API of the old release, the way a user does: an account, a workspace, one storage of every currently supported type with every setting filled in, one notifier of every type with every setting filled in, a PostgreSQL database with a password that uses all of those notifiers, with a schema list that names the schema of the seeded rows and table and table data exclusion lists that name only a table without rows, a MySQL, a MariaDB and a MongoDB database each with a password and an exclusion list that names only an object without rows, the PostgreSQL database's backup config with backups turned on, backup encryption turned on, the local storage, a weekly schedule on a day and at a time other than the default's, grandfather-father-son retention with every period set, retries of failed backups with a count other than the default's, and only the failure notification trigger, so that every value of the config differs from the logical default, and one completed, encrypted logical backup. It SHALL then stop the old instance and start the current code on the same data volume. Through the API of the current code, the test SHALL check that the account signs in with its password, that every item above is listed with the values written before the upgrade, apart from secrets, which the API does not return, and that the PostgreSQL database still uses every notifier. The notifier, storage and database secrets, which the API never returns, are covered by the requirements "An upgrade keeps every notifier's secrets" of the `notifiers` spec, "An upgrade keeps every storage's secrets" of the `storages` spec and "An upgrade keeps every database" of the `databases` spec instead.

#### Scenario: Upgrade keeps every item

- **WHEN** the current code starts on the volume of the old release
- **THEN** the account signs in, and the workspace, storages, notifiers, databases, backup config and backup are listed with their values
- **AND** the backup config still has encryption turned on

#### Scenario: Upgrade keeps every backup config setting

- **WHEN** the current code starts on the volume of an old release whose PostgreSQL database's backup config has backups turned on and runs weekly with grandfather-father-son retention, retries and only the failure notification trigger, each value different from the logical default
- **THEN** reading that config returns backups turned on, the same schedule, every retention period, the retry count, only the failure notification trigger, the encryption and the local storage

#### Scenario: Upgrade keeps every notifier

- **WHEN** the current code starts on the volume of an old release that has an email, Telegram, Slack, Discord, Teams, Mattermost and webhook notifier
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, the webhook header names included, and the database still uses all seven

#### Scenario: Upgrade keeps every storage

- **WHEN** the current code starts on the volume of an old release that has a local, S3, NAS, Azure Blob, FTP and SFTP storage
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, and the backup config still uses the local storage

#### Scenario: Upgrade keeps every logical database type

- **WHEN** the current code starts on the volume of an old release that has a PostgreSQL, a MySQL, a MariaDB and a MongoDB database
- **THEN** each of them is listed with its type, its name, its settings that are not secrets and every entry of its exclusion lists

#### Scenario: A migration loses a value

- **WHEN** a migration of the current code drops or empties a column that holds one of the written values that the API returns
- **THEN** the upgrade test fails and names the item and the value that differs

## ADDED Requirements

### Requirement: A 3.67.1 rclone installation has a tested refusal and recovery path

Upgrade coverage SHALL use the actual Databasus 3.67.1 image, with its resolved release identity pinned, to create an account, a workspace, an unused rclone storage with synthetic configuration and a supported storage through its public API. It SHALL attempt startup of the candidate on the same volume and observe a non-zero exit with rclone-only recovery guidance naming 3.67.1 and both recovery options. It SHALL compare persisted records, encrypted configuration and migration history before and after refusal and prove they are unchanged without disclosing credentials.

The regression SHALL restart 3.67.1 on that same volume, sign in, check the records, delete the unused rclone storage through its API and retry the candidate directly. Startup SHALL succeed, supported records and encrypted credentials SHALL remain available, and obsolete rclone settings schema SHALL be absent. The regression SHALL require no real cloud account, valid remote credentials or external storage access. It SHALL remain anchored to 3.67.1 when the newest release used for general upgrade coverage changes.

#### Scenario: Refusal leaves 3.67.1 usable

- **WHEN** the candidate attempts to upgrade the 3.67.1 volume containing rclone
- **THEN** it exits non-zero without advancing migrations or changing stored records
- **AND** 3.67.1 can restart, sign in and list the unchanged records

#### Scenario: Removal permits the direct retry

- **WHEN** the test removes the unused rclone storage through the restarted 3.67.1 API and starts the candidate directly
- **THEN** the upgrade succeeds, supported records and encrypted credentials survive, and the obsolete settings schema is absent

#### Scenario: The newest release advances

- **WHEN** the newest published release no longer accepts rclone
- **THEN** general upgrade coverage still seeds only supported storage types and this regression still starts from 3.67.1

### Requirement: Combined blockers have a released-image recovery regression

Upgrade coverage SHALL use the actual Databasus 3.67.0 image with pinned release identity to create an account, workspace, unused Google Drive and rclone storages with synthetic credentials, and a supported storage through its public API. Starting the candidate directly on that volume SHALL exit non-zero before any pending migration. One diagnostic SHALL name both blocking types and select 3.67.0 to remove both, or remain on 3.67.0. Persisted records, encrypted credentials and migration history SHALL remain unchanged, and output SHALL disclose none of those credentials.

The test SHALL restart 3.67.0 on the original volume, sign in, verify the records and remove both unused storages through its API. Starting the candidate directly SHALL then succeed and preserve supported records and encrypted credentials. Both obsolete providers' settings schemas SHALL be absent. The regression SHALL NOT start an intermediate 3.67.1 instance. It SHALL remain anchored to 3.67.0 independently of newest-release selection and SHALL retain the existing Google Drive-only regression as independent coverage.

#### Scenario: Direct jump from a release supporting both types

- **WHEN** the candidate starts on a 3.67.0 volume containing both blocking types
- **THEN** it reports both types together and identifies 3.67.0 as the common recovery version
- **AND** records and migration history are unchanged and 3.67.0 can reopen the volume

#### Scenario: Both types are removed on 3.67.0

- **WHEN** the regression deletes both unused storages through the restarted 3.67.0 API and retries the candidate
- **THEN** startup succeeds without running 3.67.1 and supported records and credentials survive
