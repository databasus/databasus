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

- **WHEN** the current code starts on the volume of an old release that has a local, S3, NAS, Azure Blob, FTP, SFTP and rclone storage
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, and the backup config still uses the local storage

#### Scenario: Upgrade keeps every logical database type

- **WHEN** the current code starts on the volume of an old release that has a PostgreSQL, a MySQL, a MariaDB and a MongoDB database
- **THEN** each of them is listed with its type, its name, its settings that are not secrets and every entry of its exclusion lists

#### Scenario: A migration loses a value

- **WHEN** a migration of the current code drops or empties a column that holds one of the written values that the API returns
- **THEN** the upgrade test fails and names the item and the value that differs

## ADDED Requirements

### Requirement: A 3.67.0 Google Drive installation has a tested refusal and recovery path

Upgrade coverage SHALL use the actual Databasus 3.67.0 image to create an account, a workspace, an unused Google Drive storage and a supported storage through its public API. It SHALL attempt startup of the current image on the same volume and observe a non-zero exit with the retirement diagnostic and both recovery options. It SHALL verify that no pending migration was recorded and that the stored application records and encrypted credentials remain unchanged.

The test SHALL then start 3.67.0 again on that volume, sign in and verify that the records remain available. It SHALL delete the unused Google Drive storage through that release's API and retry the current image. The upgrade SHALL succeed, preserve the supported records and expose no Google Drive storage option. An unused storage SHALL suffice; the regression SHALL require no Google account, real token exchange or external Drive access. This regression SHALL remain anchored to 3.67.0 when the newest release used by general upgrade coverage changes.

#### Scenario: Refusal and rollback on the original volume

- **WHEN** the current image attempts to upgrade a 3.67.0 installation containing a Google Drive storage
- **THEN** it exits non-zero before recording a pending migration and its diagnostic identifies 3.67.0 as the last supporting release
- **AND** restarting 3.67.0 on the same volume permits sign-in and lists the unchanged records

#### Scenario: Removal followed by a successful retry

- **WHEN** the regression removes the unused Google Drive storage through the restarted 3.67.0 API and retries the current image
- **THEN** startup succeeds and the supported records remain available

#### Scenario: The newest release no longer supports Google Drive

- **WHEN** the newest release used by the general upgrade test advances beyond 3.67.0
- **THEN** general upgrade coverage creates only currently supported storage types and the retirement regression still starts from 3.67.0
