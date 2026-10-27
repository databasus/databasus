## Purpose

Defines how logical backups of PostgreSQL, MySQL, MariaDB and MongoDB databases are started, scheduled, retried, listed, cancelled, kept or removed, and downloaded.

## ADDED Requirements

### Requirement: A manual backup answers with the backup it started

Starting a backup of a database SHALL answer `202` with the new backup in the status `IN_PROGRESS`, and the backup SHALL appear in the database's backup list at once. When a backup of the database is already running, the request SHALL get `409` with `code` `already_running` and SHALL start nothing. When the database's backup config names no storage, or the database is a physical PostgreSQL database, which has no logical backups, the request SHALL get `400` with `code` `invalid_request` and SHALL start nothing.

#### Scenario: Starting a backup

- **WHEN** a member starts a backup of a database whose config names a storage
- **THEN** the response is `202` with a backup in `IN_PROGRESS`, and reading that backup later shows `COMPLETED` with its size

#### Scenario: A backup is already running

- **WHEN** a member starts a backup of a database while another backup of it is running
- **THEN** the response is `409` with `code` `already_running`, and the database has still one backup in progress

#### Scenario: No storage

- **WHEN** a member starts a backup of a database whose config names no storage
- **THEN** the response is `400` with `code` `invalid_request`, and the backup list is unchanged

### Requirement: The scheduler starts due backups and retries failed ones

For every database whose config has backups turned on and names a storage, Databasus SHALL start a backup when the config's schedule says one is due since the last backup. When the last backup failed and the config retries failed backups, Databasus SHALL start another backup at the next check, until the number of failed backups among the most recent ones reaches the configured number of tries. Databasus SHALL NOT start a backup of a database while one of its backups is running. Every failed backup SHALL send the failure notification, as the `logical-backup-outcome` spec states.

#### Scenario: A daily backup is due

- **WHEN** a database's config runs daily at 04:00, its last backup completed yesterday at 04:00, and the time passes 04:00 today
- **THEN** a backup of the database starts at the next check

#### Scenario: Retries stop after the configured tries

- **WHEN** a config retries failed backups up to three times, and the last three backups of the database failed
- **THEN** no further backup starts until the schedule says the next one is due

### Requirement: Retention removes the backups the policy no longer keeps

Databasus SHALL remove, on a regular check, the logical backups of each database that its config's retention no longer keeps, and SHALL request the deletion of their files and metadata files. A backup that the policy drops but that is still running, or that a running restore reads, SHALL stay until a check after that work has ended. A backup younger than one hour SHALL always be kept. The time period policy SHALL remove every backup older than the period. The count policy SHALL keep the newest completed backups up to the count. The grandfather-father-son policy SHALL keep the newest completed backup of each hour, day, week, month and year for as many periods as it names. The count and grandfather-father-son policies SHALL keep failed and cancelled backups.

#### Scenario: Keeping the last five

- **WHEN** a database has seven completed backups older than one hour, and its config keeps the last five
- **THEN** after the next check its two oldest backups are no longer listed, and the deletion of their files is requested

#### Scenario: A restore reads an expired backup

- **WHEN** a backup is older than its config's time period, and a restore of it is running at the next check
- **THEN** the backup is still listed after that check
- **AND** after the first check once the restore has ended, it is no longer listed

### Requirement: Deleting a backup removes it and its file

Deleting a logical backup SHALL remove it from the backup list and SHALL request the deletion of its file and its metadata file, in one step. A backup that is running SHALL get `409` with `code` `already_running`, and a backup that a running restore reads SHALL get `409` with `code` `in_use`. Neither refusal SHALL change anything.

#### Scenario: A restore reads the backup

- **WHEN** a member deletes a backup while a restore of it is running
- **THEN** the response is `409` with `code` `in_use`, and the backup is still listed

### Requirement: A running backup can be cancelled

Cancelling a running logical backup SHALL answer `202`, stop its dump tool together with every process the tool started, record the backup as `CANCELED` and request the deletion of the file it was writing. Cancelling a backup that is no longer running SHALL get `409` with `code` `not_running` and SHALL change nothing.

#### Scenario: Cancelling a running backup

- **WHEN** a member cancels a running backup
- **THEN** the response is `202`, and reading the backup later shows `CANCELED`

#### Scenario: Cancelling a finished backup

- **WHEN** a member cancels a backup that has completed
- **THEN** the response is `409` with `code` `not_running`, and the backup is still `COMPLETED`

### Requirement: Backups are listed newest first

The backup list of a database SHALL return its logical backups newest first, filtered by status and by a creation time before which they were made when the request names them. Each backup SHALL show its status, its fail message, its size, its duration, its encryption and the time it was made.

#### Scenario: Only failed backups

- **WHEN** a member lists a database's backups with the status filter `FAILED`
- **THEN** the response holds only its failed backups, newest first

### Requirement: Deleting a database deletes its logical backups in one step

Deleting a database SHALL remove all its logical backups, their restores and their download tokens in the deletion's step, and SHALL request the deletion of every file of those backups and of their metadata files. When the deletion is refused or fails, every backup SHALL stay listed and no file SHALL be requested for deletion. When a deletion is refused is stated in the `databases` spec.

#### Scenario: A database with restored backups

- **WHEN** a member deletes a database with two logical backups, one of which was restored and has a download token
- **THEN** the response is `204`, and reading the backups, the restore and a download with the token each answers as for a backup, a restore or a token that does not exist

### Requirement: A backup is downloaded with a single-use token

Downloading a backup SHALL take two requests. A signed-in user who may view the backup SHALL first get a download token for it, which expires after five minutes. Then the token SHALL be passed in the download request, which needs no session and returns the decrypted backup file with its file name. A token SHALL be usable once and only for the backup it was issued for. An expired, used, unknown or mismatched token SHALL get `401` with `code` `unauthenticated` and no file. This is a security rule: the token is the only credential of a request that a browser link can carry.

A user SHALL stream at most one backup file or restore set at a time. While one of their streams runs, a request for a new download token SHALL get `409` with `code` `already_running` and SHALL issue no token. A download request with a valid token SHALL get the same answer while another stream of the same user runs, and its token SHALL stay unused, so the user can download with it once the first stream ends.

#### Scenario: A token is used twice

- **WHEN** a user downloads a backup with a token and then requests the same download with the same token
- **THEN** the first response returns the file, and the second is `401` with `code` `unauthenticated`

#### Scenario: Two requests race with one token

- **WHEN** two download requests with the same new token arrive at the same time
- **THEN** exactly one of them returns the file, and the other gets `409` with `code` `already_running` or `401` with `code` `unauthenticated` and no file
- **AND** the token is used afterwards: a third request with it gets `401` with `code` `unauthenticated`

#### Scenario: A token past five minutes

- **WHEN** a user requests a download with a token issued more than five minutes earlier
- **THEN** the response is `401` with `code` `unauthenticated`, and it holds no file

#### Scenario: A token for another backup

- **WHEN** a user requests the download of backup B with a token issued for backup A
- **THEN** the response is `401` with `code` `unauthenticated`, and the token can still download backup A

#### Scenario: An encrypted backup

- **WHEN** a user downloads an encrypted backup
- **THEN** the file is the dump the tool produced, decrypted

#### Scenario: A token while another download streams

- **WHEN** a user asks for a download token while another download of theirs is streaming
- **THEN** the response is `409` with `code` `already_running`, and no token is issued

#### Scenario: A second token used while the first download streams

- **WHEN** a user holds two tokens issued before either download started, and uses the second while the first download is streaming
- **THEN** the second request is `409` with `code` `already_running`, and after the first download ends the second token downloads the file
