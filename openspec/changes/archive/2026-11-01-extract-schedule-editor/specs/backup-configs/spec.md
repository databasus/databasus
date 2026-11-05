## MODIFIED Requirements

### Requirement: A config follows its database's physical backup type

When the physical backup type of a database changes, its saved physical config SHALL be adjusted in the same step to fit the new type, keeping every value the new type allows. When the adjusted config still fails validation, such as a stored full schedule with a cron expression that has no date to run on, the change SHALL get `400` with `code` `invalid_request` and SHALL change neither the database nor its config. When WAL streaming is no longer expected, the WAL streamer of the database SHALL be stopped and removed after the change is saved.

An earlier release could leave a saved physical config that does not fit its database's type, because its type switch did not adjust the config. Reading such a config, alone or in a list, SHALL return it adjusted to the type and SHALL store nothing; the next save stores the adjusted values.

#### Scenario: Switching to full backups only

- **WHEN** a physical database with incremental backups and a weekly incremental schedule is switched to full backups only
- **THEN** its config has no incremental schedule and keeps its full schedule, storage and notifications

#### Scenario: Switching off WAL streaming

- **WHEN** a physical database whose WAL is streamed is switched to full and incremental backups without WAL streaming
- **THEN** after the switch is saved, the database has no WAL streamer

#### Scenario: A config left unfit by an earlier release

- **WHEN** an earlier release left a saved physical config with an incremental schedule while its database takes full backups only, and a member reads the config and lists the workspace's configs
- **THEN** both responses hold the config without an incremental schedule
- **AND** the stored config keeps its incremental schedule until the next save, which stores it without one

#### Scenario: Changing the backup type with a stored full expression that has no date

- **WHEN** an earlier release stored the cron expression `0 0 31 2 *` as the full schedule of a physical config and a member changes the database's physical backup type
- **THEN** the response is `400` with `code` `invalid_request`, and the database and its stored config are unchanged
