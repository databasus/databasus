## MODIFIED Requirements

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
