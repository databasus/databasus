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

On each refresh tick, a list SHALL re-request only its first page and merge it into the rows already shown. When that page has a continuation cursor and its last row occurs in the loaded rows with a tail below it, rows below that anchor SHALL stay loaded and the loaded tail's continuation cursor SHALL be preserved. Refreshed rows SHALL replace their loaded copies without duplication. The refresh interval of each list SHALL stay unchanged.

When the refreshed first page's last row cannot anchor a loaded tail, including when enough new rows eliminate usable overlap, the list SHALL replace its loaded extent with the refreshed page and use that page's continuation cursor. The same replacement SHALL apply when the refreshed page is empty or reports that it contains the complete list. These fallback cases SHALL NOT cause refresh to request additional pages; older rows that still exist SHALL remain reachable through subsequent scrolling from the refreshed cursor.

#### Scenario: New backup appears while the user is scrolled deep

- **WHEN** the user has loaded five pages, a new backup is created and the refreshed first page's last row still anchors the loaded tail
- **THEN** the next refresh shows the new backup at the top
- **AND** the rows from the second to the fifth page remain shown below the refreshed portion
- **AND** no row is shown twice

#### Scenario: A row in the first page changes

- **WHEN** a backup in the first page moves from in progress to completed
- **THEN** the next refresh shows it as completed in the same position

#### Scenario: Refresh while the user is scrolled deep

- **WHEN** the user has loaded ten pages and a refresh tick fires
- **THEN** the refresh requests a single page

#### Scenario: New backups eliminate usable overlap

- **WHEN** the user has loaded several pages and enough new backups appear that the refreshed first page's last row is absent from the loaded rows
- **THEN** refresh shows only the refreshed first page and uses its continuation cursor
- **AND** it does not request additional pages automatically
- **AND** subsequent scrolling can reach the older backups that still exist

#### Scenario: Refresh reports an empty list

- **WHEN** the refreshed first page is empty and carries no continuation cursor
- **THEN** the visible list is empty and no stale tail or cursor remains

#### Scenario: Refresh reports the complete list

- **WHEN** the refreshed first page contains rows but carries no continuation cursor
- **THEN** the visible list contains exactly those rows and no stale tail or cursor remains

#### Scenario: An overlap leaves no loaded tail

- **WHEN** the refreshed first page's last row is also the last loaded row and the response has a continuation cursor
- **THEN** the list adopts the refreshed page and its cursor without preserving an obsolete cursor

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
