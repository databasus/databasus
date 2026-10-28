## MODIFIED Requirements

### Requirement: Physical backup list pages with a continuation cursor

The physical backup list API SHALL return, with each page, a continuation cursor while further rows exist and no cursor on the last page. A request that passes the cursor SHALL return the rows that follow the previous page, with no row repeated or skipped, even when backups or WAL segments were added at the top between the two requests. The type filter, the status filter and the created-before filter SHALL apply to every page. The API SHALL NOT accept an offset and SHALL NOT report a total row count. A cursor the server did not issue or cannot decode SHALL be rejected with `400` and `code` `invalid_request`.

#### Scenario: Following the cursor

- **WHEN** a client requests the first page with a limit of 50 and then requests again with the returned cursor
- **THEN** the second response contains the next rows in the same newest-first order, starting right after the last row of the first page

#### Scenario: New rows between requests

- **WHEN** a new FULL backup is created between the first and the second page request
- **THEN** the second page neither repeats a row from the first page nor skips a row that existed when the first page was served

#### Scenario: Following the cursor with a filter

- **WHEN** a client requests the physical backups of a database with three full backups, incremental backups and WAL with the type filter `FULL` and a limit of 2, and then requests again with the returned cursor and the same filter
- **THEN** the first page holds the two newest full backups, and the second page holds only the oldest full backup and carries no continuation cursor

#### Scenario: Following the cursor with a time filter

- **WHEN** a client requests the physical backups of a database created before a given time with a limit of 2, and then requests again with the returned cursor and the same time
- **THEN** both pages hold only rows created before that time, and the second page starts right after the last row of the first page and repeats none of them

#### Scenario: Following the cursor with a status filter

- **WHEN** a client requests the physical backups of a database with two failed and two completed full backups with the status filter `ERROR` and a limit of 1, and then requests again with the returned cursor and the same filter
- **THEN** each page holds one failed backup, the second page holds the other one, and neither holds a completed backup

#### Scenario: Last page

- **WHEN** a client requests the page that contains the oldest row
- **THEN** the response carries no continuation cursor

#### Scenario: Malformed cursor

- **WHEN** a client passes a cursor the server did not issue or cannot decode
- **THEN** the response is `400` with `code` `invalid_request`, and it holds no rows
