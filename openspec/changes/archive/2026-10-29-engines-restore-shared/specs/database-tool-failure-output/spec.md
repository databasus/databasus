## MODIFIED Requirements

### Requirement: A failed restore reports the restore tool's error output

When the database tool that loads a backup into a PostgreSQL, MySQL, MariaDB or MongoDB database exits with an error, the restore SHALL be recorded as failed, and its failure message SHALL contain the error output the tool wrote before it exited. This SHALL hold however quickly the tool exits after writing that output, and whether or not the tool read the whole backup before it exited. One exception applies: a PostgreSQL restore whose only error is that the `public` schema already exists SHALL count as successful. A PostgreSQL restore that could not create an extension SHALL fail, because for the user's database a missing extension means missing data; only a verification treats it as an ignorable error, as the `backup-verification` spec states. When the backup stream from storage failed first, the failure message SHALL report that stream failure instead of the tool's output, because the tool's error is then only a consequence of the incomplete input.

#### Scenario: Tool rejects the backup and exits immediately

- **WHEN** a restore runs, the restore tool reads the whole backup, writes `ERROR 1064 (42000) at line 1: You have an error in your SQL syntax` to its error output and exits with a non-zero status at once
- **THEN** the restore is recorded as failed, and its failure message contains `You have an error in your SQL syntax`

#### Scenario: Tool exits before reading the whole backup

- **WHEN** a restore runs, and the restore tool writes an error and exits with a non-zero status before it has read the whole backup
- **THEN** the restore is recorded as failed, and its failure message contains the error the tool wrote, not only a failure to pass the backup data to the tool

#### Scenario: Every engine keeps the tool's output

- **WHEN** the restore tool of a PostgreSQL, a MySQL, a MariaDB or a MongoDB restore writes an error and exits at once
- **THEN** each restore's failure message contains the error that tool wrote

#### Scenario: Only the existing public schema fails

- **WHEN** a PostgreSQL restore tool exits with an error whose only complaint is that the `public` schema already exists
- **THEN** the restore is recorded as successful

#### Scenario: An extension the target cannot create

- **WHEN** a PostgreSQL restore tool exits with an error whose only complaints are that an extension is not available and that the comment on it failed
- **THEN** the restore is recorded as failed, and its failure message contains the tool's complaint about the extension

#### Scenario: Backup stream fails before the tool does

- **WHEN** the backup stream from storage ends early, and the restore tool then fails on the incomplete input
- **THEN** the failure message reports that the backup stream from storage failed mid-restore, and does not contain the tool's complaint about the incomplete input
