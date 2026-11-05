# logical-backups Specification

## Purpose
Defines how logical backups of PostgreSQL, MySQL, MariaDB and MongoDB databases are started, scheduled, retried, listed, cancelled, kept or removed, and downloaded.

## Requirements

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

### Requirement: A logical backup download announces its exact file length

A successful logical backup file download SHALL include `Content-Length` equal to the byte count of the complete downloaded file. The length SHALL describe the decrypted file in its existing compression format, including when the backup is encrypted in storage. The server SHALL begin streaming without first reading the complete file into memory or staging another copy. The announced length SHALL be exact rather than an estimate rounded to megabytes.

#### Scenario: Unencrypted backup download

- **WHEN** a client downloads a completed logical backup stored without encryption
- **THEN** `Content-Length` equals the number of file bytes the response delivers
- **AND** the file retains its existing compression format and download name

#### Scenario: Encrypted backup download

- **WHEN** a client downloads a completed logical backup encrypted in storage
- **THEN** the delivered file is decrypted and `Content-Length` equals its byte count
- **AND** encryption overhead in storage is not included in the announced length

#### Scenario: Download size is not a whole megabyte

- **WHEN** a client downloads a completed logical backup whose file size is not an exact multiple of a megabyte
- **THEN** `Content-Length` equals the exact delivered byte count without rounding to a whole megabyte

#### Scenario: Client accepts gzip

- **WHEN** a client downloads a completed logical backup with `Accept-Encoding: gzip`
- **THEN** the response has no `Content-Encoding` header and its `Content-Length` equals the delivered file byte count

#### Scenario: Stored file is truncated

- **WHEN** a completed logical backup's stored file has been truncated and a client downloads it
- **THEN** the announced length remains the complete backup's recorded byte count, allowing the client to detect the incomplete response
- **AND** the download does not change the backup's stored completed status

### Requirement: One failed removal does not stop retention for the database

When retention cannot remove one expired logical backup, it SHALL report that failure in the log and SHALL still remove the database's other expired backups in the same check. The backup that could not be removed SHALL stay listed and SHALL be tried again on a later check.

#### Scenario: One of several expired backups cannot be removed

- **WHEN** a database has three expired backups and removing one of them fails
- **THEN** the other two are removed in that check and the third stays listed
- **AND** the log names the backup that could not be removed

#### Scenario: The failure has passed

- **WHEN** the removal that failed can succeed on the next check
- **THEN** that check removes the remaining expired backup

### Requirement: A backup refreshes its size and duration every 5 seconds while its dump streams

While the dump of a logical backup streams to the storage, Databasus SHALL refresh the size and the duration that reading the backup returns every 5 seconds, and SHALL NOT refresh them more often, apart from one refresh when the stream ends successfully. A refresh that falls due while the previous one is still being saved SHALL be skipped. The duration SHALL advance at every refresh, also when the dump produced no output since the previous one. Before the stream starts, and between the last refresh and the backup's outcome, the size and the duration SHALL NOT refresh. Until the first refresh, reading the backup SHALL return size 0 and duration 0. Copying the dump to the storage SHALL NOT wait for these refreshes to be saved. A periodic refresh that fails during the stream, because its save returns an error or because it panics, SHALL be logged and SHALL NOT fail the backup. A refresh SHALL NOT change the status or the size of a backup that is no longer running.

#### Scenario: A fast dump

- **WHEN** a dump produces many megabytes after a refresh and within the same 5-second interval
- **THEN** reading the backup during that interval returns the size and the duration of that refresh

#### Scenario: Before the first refresh

- **WHEN** a backup has been running for less than 5 seconds of streaming
- **THEN** reading the backup returns `IN_PROGRESS` with size 0 and duration 0

#### Scenario: A backup that finishes within one interval

- **WHEN** a backup completes before its first periodic refresh
- **THEN** reading the backup shows no intermediate value, only size 0 before the stream ends and the size the dump reached afterwards

#### Scenario: A dump that produces no output

- **WHEN** a streaming dump produces no output for longer than 5 seconds
- **THEN** the backup keeps its size, and its duration advances at every refresh

#### Scenario: A backup that completes between refreshes

- **WHEN** a backup completes after its last periodic refresh
- **THEN** reading the backup shows `COMPLETED` with the size the dump reached when it completed, not the size of the last periodic refresh

#### Scenario: A slow internal database

- **WHEN** saving a refresh takes longer than the rest of the dump
- **THEN** the dump still reaches the storage while the refresh is being saved

#### Scenario: A periodic refresh that fails

- **WHEN** a periodic refresh panics or its save returns an error during the dump
- **THEN** the failure is logged, and the stream of the backup ends without an error because of it

#### Scenario: A backup cancelled during its dump

- **WHEN** a user cancels a backup while its dump streams
- **THEN** the backup stays `CANCELED` with size 0, and no refresh changes either
