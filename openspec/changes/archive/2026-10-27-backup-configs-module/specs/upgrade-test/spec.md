## MODIFIED Requirements

### Requirement: Data written before the upgrade is there after it

Before the upgrade, the test SHALL write through the API of the old release, the way a user does: an account, a workspace, one storage of every type with every setting filled in, one notifier of every type with every setting filled in, a PostgreSQL database with a password that uses all of those notifiers, with a schema list that names the schema of the seeded rows and table and table data exclusion lists that name only a table without rows, a MySQL, a MariaDB and a MongoDB database each with a password and an exclusion list that names only an object without rows, the PostgreSQL database's backup config with backups turned on, backup encryption turned on, the local storage, a weekly schedule on a day and at a time other than the default's, grandfather-father-son retention with every period set, retries of failed backups with a count other than the default's, and only the failure notification trigger, so that every value of the config differs from the logical default, and one completed, encrypted logical backup. It SHALL then stop the old instance and start the current code on the same data volume. Through the API of the current code, the test SHALL check that the account signs in with its password, that every item above is listed with the values written before the upgrade, apart from secrets, which the API does not return, and that the PostgreSQL database still uses every notifier. The notifier, storage and database secrets, which the API never returns, are covered by the requirements "An upgrade keeps every notifier's secrets" of the `notifiers` spec, "An upgrade keeps every storage's secrets" of the `storages` spec and "An upgrade keeps every database" of the `databases` spec instead.

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

- **WHEN** the current code starts on the volume of an old release that has a local, S3, Google Drive, NAS, Azure Blob, FTP, SFTP and rclone storage
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, and the backup config still uses the local storage

#### Scenario: Upgrade keeps every logical database type

- **WHEN** the current code starts on the volume of an old release that has a PostgreSQL, a MySQL, a MariaDB and a MongoDB database
- **THEN** each of them is listed with its type, its name, its settings that are not secrets and every entry of its exclusion lists

#### Scenario: A migration loses a value

- **WHEN** a migration of the current code drops or empties a column that holds one of the written values that the API returns
- **THEN** the upgrade test fails and names the item and the value that differs
