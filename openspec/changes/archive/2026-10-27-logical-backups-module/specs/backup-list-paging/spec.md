## ADDED Requirements

### Requirement: The logical backup list pages with a continuation cursor

The logical backup list API SHALL return, with each page, a continuation cursor while further rows exist and no cursor on the last page. A request that passes the cursor SHALL return the backups that follow the previous page in the same newest-first order, with no backup repeated or skipped, even when backups were added at the top between the two requests. The status filter and the `createdBefore` filter SHALL apply to every page. A cursor the server did not issue or cannot decode SHALL be rejected as an invalid request.

#### Scenario: Following the cursor

- **WHEN** a client requests the first page of a database's logical backups with a limit of 20 and then requests again with the returned cursor
- **THEN** the second response holds the next backups, starting right after the last backup of the first page

#### Scenario: A backup starts between the requests

- **WHEN** a backup of the database starts between the first and the second page request
- **THEN** the second page neither repeats a backup from the first page nor skips one that existed when the first page was served

#### Scenario: A filter applies to every page

- **WHEN** a client requests a database's logical backups with the status filter `FAILED` and a limit of 2, and then requests again with the returned cursor and the same filter
- **THEN** the second page holds only failed backups, starting right after the last failed backup of the first page

#### Scenario: A time filter applies to every page

- **WHEN** a client requests a database's logical backups made before a given time with a limit of 2, and then requests again with the returned cursor and the same time
- **THEN** the second page holds only backups made before that time, starting right after the last backup of the first page

#### Scenario: Last page

- **WHEN** a client requests the page that holds the oldest backup
- **THEN** the response carries no continuation cursor

#### Scenario: Malformed cursor

- **WHEN** a client passes a cursor the server did not issue or cannot decode
- **THEN** the response is `400` with `code` `invalid_request`, and it holds no rows
