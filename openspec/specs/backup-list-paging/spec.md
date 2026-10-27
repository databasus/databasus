# Backup List Paging Specification

## Purpose

Defines how the physical backup, logical backup and restore check lists load further rows as the user scrolls and how they stay current while open.

## Requirements

### Requirement: Scrolling appends the next page

The physical backup, logical backup and restore check lists SHALL load the next page of rows when the user scrolls to the bottom and append it below the rows already shown. Every row in the catalog SHALL be reachable by scrolling, regardless of how many rows precede it.

#### Scenario: Physical list longer than 1000 rows

- **WHEN** a database has 1500 physical backup list rows and the user keeps scrolling to the bottom
- **THEN** the list keeps growing until all 1500 rows are shown

#### Scenario: Earlier pages are not requested again

- **WHEN** the user scrolls to the bottom of a list that already shows three pages
- **THEN** the list requests only the fourth page

### Requirement: Loading stops at the end of the list

A list SHALL stop requesting further pages and stop showing the loading indicator once the server reports that no rows remain. It SHALL then show that every row is loaded, together with the number of rows shown.

#### Scenario: Last page reached

- **WHEN** the server returns a page and reports that no further rows exist
- **THEN** scrolling to the bottom sends no further page request and shows no loading indicator
- **AND** the list shows that all rows are loaded

### Requirement: Refresh updates the newest rows without reloading the rest

On each refresh tick, a list SHALL re-request only its first page and merge it into the rows already shown. Rows below the first page SHALL stay as loaded. The refresh interval of each list SHALL stay unchanged.

#### Scenario: New backup appears while the user is scrolled deep

- **WHEN** the user has loaded five pages and a new backup is created
- **THEN** the next refresh shows the new backup at the top
- **AND** the rows from the second to the fifth page are still shown
- **AND** no row is shown twice

#### Scenario: A row in the first page changes

- **WHEN** a backup in the first page moves from in progress to completed
- **THEN** the next refresh shows it as completed in the same position

#### Scenario: Refresh while the user is scrolled deep

- **WHEN** the user has loaded ten pages and a refresh tick fires
- **THEN** the refresh requests a single page

### Requirement: Rows are not duplicated across pages

A list SHALL show each row at most once, even when rows are added at the top between two page requests.

#### Scenario: Backup added between page loads

- **WHEN** the user has loaded one page and a new backup is created before the user scrolls to the bottom
- **THEN** after the second page loads, no row appears twice in the list

### Requirement: Changing context reloads from the first page

A list SHALL discard loaded rows and load its first page again when the user switches to another database, changes a filter, triggers a backup or deletes a backup.

#### Scenario: Filter changed after scrolling

- **WHEN** the user has loaded several pages and then changes the type filter
- **THEN** the list shows only the first page of rows that match the new filter

### Requirement: Physical backup list pages with a continuation cursor

The physical backup list API SHALL return, with each page, a continuation cursor while further rows exist and no cursor on the last page. A request that passes the cursor SHALL return the rows that follow the previous page, with no row repeated or skipped, even when backups or WAL segments were added at the top between the two requests. The API SHALL no longer accept an offset and SHALL no longer report a total row count.

#### Scenario: Following the cursor

- **WHEN** a client requests the first page with a limit of 50 and then requests again with the returned cursor
- **THEN** the second response contains the next rows in the same newest-first order, starting right after the last row of the first page

#### Scenario: New rows between requests

- **WHEN** a new FULL backup is created between the first and the second page request
- **THEN** the second page neither repeats a row from the first page nor skips a row that existed when the first page was served

#### Scenario: Last page

- **WHEN** a client requests the page that contains the oldest row
- **THEN** the response carries no continuation cursor

#### Scenario: Malformed cursor

- **WHEN** a client passes a cursor the server did not issue or cannot decode
- **THEN** the server rejects the request as invalid and returns no rows

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

### Requirement: A page cursor grants no access

The continuation cursor SHALL carry only a position in the list. Access to a database's backup list SHALL be decided by the same permission check as without a cursor, and a cursor SHALL never cause rows of a database other than the requested one to be returned.

#### Scenario: Cursor from another database

- **WHEN** a user passes a cursor obtained from database A in a list request for database B
- **THEN** the response contains only database B's rows

#### Scenario: Cursor without access

- **WHEN** a user who cannot view database B requests its backup list with any cursor
- **THEN** the request is rejected exactly as it would be without a cursor
