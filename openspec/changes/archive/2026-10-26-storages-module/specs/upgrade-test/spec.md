## MODIFIED Requirements

### Requirement: Data written before the upgrade is there after it

Before the upgrade, the test SHALL write through the API of the old release, the way a user does: an account, a workspace, one storage of every type with every setting filled in, one notifier of every type with every setting filled in, a PostgreSQL database with a password that uses all of those notifiers, its backup config with backup encryption turned on and the local storage, and one completed, encrypted logical backup. It SHALL then stop the old instance and start the current code on the same data volume. Through the API of the current code, the test SHALL check that the account signs in with its password, that every item above is listed with the values written before the upgrade, apart from secrets, which the API does not return, and that the database still uses every notifier. The notifier and storage secrets, which the API never returns, are covered by the requirements "An upgrade keeps every notifier's secrets" of the `notifiers` spec and "An upgrade keeps every storage's secrets" of the `storages` spec instead.

#### Scenario: Upgrade keeps every item

- **WHEN** the current code starts on the volume of the old release
- **THEN** the account signs in, and the workspace, storages, notifiers, database, backup config and backup are listed with their values
- **AND** the backup config still has encryption turned on

#### Scenario: Upgrade keeps every notifier

- **WHEN** the current code starts on the volume of an old release that has an email, Telegram, Slack, Discord, Teams, Mattermost and webhook notifier
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, the webhook header names included, and the database still uses all seven

#### Scenario: Upgrade keeps every storage

- **WHEN** the current code starts on the volume of an old release that has a local, S3, Google Drive, NAS, Azure Blob, FTP, SFTP and rclone storage
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, and the backup config still uses the local storage

#### Scenario: A migration loses a value

- **WHEN** a migration of the current code drops or empties a column that holds one of the written values that the API returns
- **THEN** the upgrade test fails and names the item and the value that differs
