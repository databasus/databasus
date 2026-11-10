## MODIFIED Requirements

### Requirement: A restore token opens one restore set

Issuing a restore token SHALL answer `201` with the token and the time it expires, 15 minutes after it is issued. A token SHALL name either a database with an optional target time, which restores to that time or to the latest restorable moment, or one full or incremental backup, which restores that backup and the backups it is built on without replaying WAL. The token SHALL be refused at issue, and no token issued, when:

- the database has no completed full backup, or the chosen backup is not completed or misses a backup it is built on: `409` with `code` `no_restorable_backup`;
- the target time is earlier than the earliest full backup, a WAL gap lies between the full backup and the target, or reaching the target requires archived WAL on a timeline above 1 whose verified history is not published: `409` with `code` `target_time_unreachable`;
- another download or restore stream of the same user is running: `409` with `code` `already_running`;
- the chosen backup does not exist: `404` with `code` `not_found`;
- the database is not a physical PostgreSQL database: `400` with `code` `invalid_request`.

#### Scenario: Restoring to the latest moment

- **WHEN** a viewer asks for a restore token of a database with a completed full backup and archived WAL after it, without a target time and with the required timeline history published
- **THEN** the response is `201` with a token and its expiry time

#### Scenario: A target time behind a WAL gap

- **WHEN** a viewer asks for a restore token to a time that lies after a gap in the archived WAL
- **THEN** the response is `409` with `code` `target_time_unreachable`, and its detail names the latest restorable position

#### Scenario: No full backup yet

- **WHEN** a viewer asks for a restore token of a database that has no completed full backup
- **THEN** the response is `409` with `code` `no_restorable_backup`

#### Scenario: A running download

- **WHEN** a user asks for a restore token while a backup download of the same user runs
- **THEN** the response is `409` with `code` `already_running`

For archive-based restores on a timeline above 1, Databasus SHALL include only WAL whose timeline has verified history published under the current backup storage selection. Without that history, a latest restore SHALL stop at the newest completed backup in the selected chain and SHALL include neither later WAL nor unrelated history files. An explicit target beyond that backup SHALL get `409` with `code` `target_time_unreachable`. Restoring an explicitly selected completed backup SHALL remain independent of separately archived history. Timeline 1 SHALL require no history file.

#### Scenario: Latest restore while promoted timeline history is pending

- **WHEN** a completed backup on timeline 2 has subsequent archived WAL but its verified timeline history has not been published
- **THEN** a latest restore token is issued for recovery through that backup only
- **AND** the downloaded set excludes the subsequent WAL

#### Scenario: Target requires unpublished history

- **WHEN** an explicit restore target requires WAL after the newest included backup on timeline 2 and that timeline's verified history is not published
- **THEN** token issuance returns `409` with `code` `target_time_unreachable`
- **AND** no token is issued

#### Scenario: Explicit backup remains restorable

- **WHEN** a completed backup is selected by its identifier and separately archived history is unavailable
- **THEN** token issuance succeeds if that backup and its required backup ancestors remain available
- **AND** the restore set contains no additional archived WAL or history

#### Scenario: History becomes available

- **WHEN** the verified history for timeline 2 is published while its completed backups and contiguous archived WAL remain available
- **THEN** latest and targeted restore selection can include that WAL without creating another FULL

#### Scenario: First timeline needs no history

- **WHEN** a restore on timeline 1 has a completed backup and sufficient contiguous archived WAL
- **THEN** absence of a timeline history file does not restrict that restore

### Requirement: The restore set downloads once with its token

Downloading a restore set SHALL need no session. The restore token SHALL be its credential and SHALL be passed as a query parameter. The download SHALL stream the restore set as a tar archive. A token that is unknown, expired or already used SHALL get `401` with `code` `unauthenticated`. A token SHALL be spent on its first use, even when that use is refused because another download or restore stream of the same user runs, which gets `409` with `code` `already_running`. When the restore set the token names can no longer be built, because a backup it needs was deleted, a WAL gap appeared, or required timeline history became unavailable after the token was issued, the download SHALL get the answer that issuing the token would get now.

#### Scenario: Downloading with a fresh token

- **WHEN** a client downloads the restore set with a token issued a minute ago
- **THEN** the response is `200` with a tar archive that holds the full backup and the WAL up to the target

#### Scenario: Using a token twice

- **WHEN** a client downloads a restore set with a token and then downloads again with the same token
- **THEN** the second response is `401` with `code` `unauthenticated`

#### Scenario: A backup deleted after the token was issued

- **WHEN** the only full backup of a database is deleted after a restore token of the database was issued, and the token is then used
- **THEN** the response is `409` with `code` `no_restorable_backup`

#### Scenario: The chosen backup deleted after the token was issued

- **WHEN** a full or incremental backup is deleted after a restore token naming that backup was issued, and the token is then used
- **THEN** the response is `404` with `code` `not_found`, as a new token request for that backup would receive

#### Scenario: Targeted restore loses its required history before download

- **WHEN** a token for a target requiring archived WAL is issued while the promoted timeline's history is published, that history is then removed from the catalog, and the token is used
- **THEN** the download returns `409` with `code` `target_time_unreachable` before streaming the archive
