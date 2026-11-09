# backup-usage Specification

## Purpose

Show compact, current totals of backup data recorded by Databasus for each database and saved storage, without measuring the storage provider's actual consumption.

## Requirements

### Requirement: Backup usage is calculated from the retained catalog

Databasus SHALL calculate backup usage from the recorded stored sizes of retained completed logical backups, retained completed full and incremental physical backups, and committed archived WAL segments. Each WAL segment SHALL contribute once, independently of how many backups depend on it. A database total SHALL cover that database's eligible records; a storage total SHALL cover eligible records whose recorded storage is that storage, across all databases. Neither total SHALL depend on list filters, pagination, or the storage selected in a database's current configuration.

Running, failed and canceled backups, unpublished WAL uploads, sidecar metadata files and pending file deletions SHALL NOT contribute. Removing an eligible catalog record SHALL remove its contribution even if physical file cleanup has not finished. An empty eligible set SHALL yield zero. Reads SHALL NOT connect to the source database or storage provider to measure usage.

#### Scenario: All supported database kinds

- **WHEN** usage is read for physical PostgreSQL or logical PostgreSQL, MySQL, MariaDB or MongoDB with completed backups
- **THEN** the database total includes the recorded stored size of every eligible backup of that database
- **AND** physical PostgreSQL also includes its committed archived WAL

#### Scenario: Shared WAL

- **WHEN** two completed physical backups depend on the same committed WAL segment
- **THEN** the database and storage totals each include that segment's recorded stored size exactly once

#### Scenario: A storage contains backups from several databases

- **WHEN** one storage holds a completed logical backup of 100 MB, a completed physical backup of 200 MB and a committed WAL segment of 4 MB
- **THEN** its total usage is 304 MB
- **AND** records belonging to another storage do not contribute

#### Scenario: A filtered page

- **WHEN** a database has 150 MB of eligible records and a filtered page contains only a 10 MB backup
- **THEN** the page reports a database total of 150 MB
- **AND** following its next-page cursor preserves the full database scope of the total

#### Scenario: Uploads and failed attempts

- **WHEN** a database has a completed 100 MB backup and running, failed or canceled attempts with recorded progress sizes
- **THEN** its total remains 100 MB until another backup completes
- **AND** a WAL upload contributes only after it is committed as archived

#### Scenario: Catalog removal precedes file cleanup

- **WHEN** a completed backup's catalog record is removed and its file is still awaiting deletion
- **THEN** subsequent totals exclude that backup

#### Scenario: Stored attribution differs from the current configuration

- **WHEN** a retained backup names storage A and its database's current configuration names storage B
- **THEN** that backup contributes to A and not B

### Requirement: Backup usage is available through the documented API

Logical backup pages SHALL include a numeric `totalUsageMb` for the requested database alongside their existing items and cursor. Physical backup pages SHALL continue to include that field. A read operation for one saved storage SHALL return its numeric `totalUsageMb`. These values SHALL follow the catalog calculation above, use the existing megabyte convention, and be available even when scheduled backups are disabled or the storage provider is unavailable. Existing list request shapes, ordering, filters and cursors SHALL remain valid.

#### Scenario: A logical page reports its total

- **WHEN** an authorized reader requests a logical backup page
- **THEN** the response includes its items, its continuation cursor when another page exists, and `totalUsageMb` for the whole database

#### Scenario: An unreachable storage

- **WHEN** an authorized reader requests backup usage for a saved storage whose provider cannot be reached
- **THEN** the read succeeds with the total recorded in Databasus without contacting the provider

#### Scenario: No recorded backups

- **WHEN** an authorized reader requests a logical page or saved-storage usage with no eligible records
- **THEN** `totalUsageMb` is zero

### Requirement: Usage reads enforce workspace access

A database usage read SHALL require the same permission as viewing that database's backups. A storage usage read SHALL require permission to view the storage's owning workspace, including for a total spanning several databases. An unauthenticated request SHALL receive 401 with code `unauthenticated`; a missing resource or one hidden by workspace membership SHALL receive 404 with code `not_found`; a known resource whose viewing permission is insufficient SHALL receive 403 with code `permission_denied`. Refusals SHALL disclose no usage or backup details. Existing administrator privileges SHALL continue to apply.

#### Scenario: A viewer reads usage

- **WHEN** a workspace viewer reads the usage of a database or storage in that workspace
- **THEN** the read succeeds without requiring permission to manage backups or storages

#### Scenario: No session

- **WHEN** a request without a session reads storage usage
- **THEN** it receives 401 with code `unauthenticated` and no usage

#### Scenario: Another workspace

- **WHEN** a non-admin without membership in a storage's workspace reads its usage
- **THEN** it receives 404 with code `not_found` and no usage

#### Scenario: Unknown storage

- **WHEN** an authenticated user requests usage for a storage that does not exist
- **THEN** it receives 404 with code `not_found` and no usage

### Requirement: Database totals appear on the same line as Size

Every desktop database backup table SHALL display its compact total immediately after the translated Size label on the same line. The total SHALL use smaller muted text with regular font weight, while Size retains its header emphasis. Logical backup size help SHALL remain available. The physical backup screen SHALL no longer show a separate Total usage line above its actions. The same presentation SHALL apply to physical PostgreSQL and logical PostgreSQL, MySQL, MariaDB and MongoDB.

On layouts that replace the table with cards, the screen SHALL show one compact, localized Total usage line above the cards. Totals SHALL remain readable in both themes and at narrow widths or enlarged text. Initial loading or a failed first page SHALL show no numeric total; a successfully loaded empty list SHALL show zero.

#### Scenario: Desktop header

- **WHEN** the English desktop backup table has 10.5 GB of eligible backups
- **THEN** the header reads `Size 10.5 GB` on one line with the value visually secondary
- **AND** no standalone physical-backup Total usage line appears above the actions

#### Scenario: Cards on mobile

- **WHEN** a database's backups are displayed as cards
- **THEN** their total remains available once above the cards without relying on a hidden table header

#### Scenario: The first page fails

- **WHEN** the current database's first backup page fails to load
- **THEN** the screen offers its existing translated retry and shows no usage value or invented zero

### Requirement: Saved-storage details show backup usage beneath the name

Every saved storage's detail page SHALL display a localized Total usage line directly beneath its name, before Storage settings. Its value SHALL use the compact total format and visually secondary text. This SHALL apply to local, S3-compatible, NAS, Azure Blob, FTP and SFTP storages, including for users who can only view them. Usage loading and refresh SHALL NOT reset a storage name or settings draft or its connection-test state.

An initial usage failure SHALL show localized feedback with an explicit retry without claiming zero or preventing the storage details from being used. A failed background usage refresh SHALL retain the last value with a visible translated indication that it could not be refreshed and an explicit retry. A current authorization failure SHALL hide that value. A successful empty usage read SHALL show `0 MB`.

#### Scenario: A saved storage is opened

- **WHEN** a user opens a saved storage with 104 MB of eligible backups in English
- **THEN** `Total usage: 104 MB` appears directly under the name and above Storage settings

#### Scenario: A refresh during editing

- **WHEN** usage changes while a user is editing the storage's name or settings
- **THEN** its total updates without changing the draft or resetting a successful connection test

#### Scenario: Usage cannot initially load

- **WHEN** a saved storage loads successfully but its usage read fails
- **THEN** the storage details remain usable and localized usage feedback offers retry instead of a numeric value

### Requirement: Aggregate sizes use compact localized formatting

Totals SHALL use MB below 1024 MB and GB at or above 1024 MB, with 1024 MB per GB. MB values SHALL round to the nearest whole number; GB values SHALL round to at most one decimal place. Unnecessary decimal zeroes SHALL be omitted. Unit selection SHALL use the unrounded total. Zero SHALL display as `0 MB`. Unit symbols SHALL remain `MB` and `GB`, and digits and decimal separators SHALL follow the selected interface language and update when it changes. This formatting SHALL apply to totals without changing individual backup size formatting.

#### Scenario: Compact values

- **WHEN** English totals are 104.4 MB, 10752 MB and 10240 MB
- **THEN** they display as `104 MB`, `10.5 GB` and `10 GB`, respectively

#### Scenario: The unit threshold

- **WHEN** English totals are 1023.4 MB and 1024 MB
- **THEN** they display as `1,023 MB` and `1 GB`, respectively

#### Scenario: Live language switch

- **WHEN** the interface language changes from English to Russian while a 10.5 GB total is visible
- **THEN** its value changes from `10.5 GB` to `10,5 GB` and surrounding labels change language without reloading the page

### Requirement: Visible totals refresh without reloading the page

While the relevant page is visible, database totals SHALL refresh with the backup list's one-second polling once its current first page has succeeded, and storage totals SHALL be polled once per second once their current usage session is ready. Polling SHALL remain suspended for a failed database initial load or reload and for a storage usage session awaiting explicit retry. A successful current response SHALL update its total even if the listed rows, their order and continuation cursor are unchanged. A newly completed backup, newly committed WAL segment or catalog deletion SHALL be reflected by the next successful current refresh, without a page reload. Running uploads SHALL follow the eligibility rule above rather than reporting uncommitted progress.

Hidden pages SHALL pause periodic reads. Returning to visibility SHALL trigger a fresh read only for a ready database list or a ready storage usage session that is not awaiting explicit retry. A visibility change SHALL NOT retry a failed database initial load or reload, or a storage usage session awaiting explicit retry. Leaving the page or changing database, storage, workspace or session SHALL invalidate earlier usage requests and end their polling. Late successes or failures SHALL NOT replace the current resource's value or show feedback. Database background failures SHALL preserve the accepted total under the existing list-refresh policy. Storage usage failures SHALL stop automatic usage polling and expose the retry described above; retry SHALL resume polling after a successful read.

#### Scenario: A new completed logical backup

- **WHEN** a logical backup completes after its database tab has loaded
- **THEN** the next successful refresh includes its stored size in the header without navigation or reload

#### Scenario: Only metadata changes

- **WHEN** a successful backup-list refresh changes only `totalUsageMb`
- **THEN** the visible total changes even though the list items and continuation cursor are unchanged

#### Scenario: WAL continues to arrive

- **WHEN** another WAL segment is committed while the physical-backup tab or its storage details are visible
- **THEN** the next successful refresh increases the relevant total by that segment's recorded stored size

#### Scenario: An earlier storage answers late

- **WHEN** usage for storage A answers after the user has selected storage B
- **THEN** A's answer changes neither B's value nor its feedback

#### Scenario: The tab becomes visible again

- **WHEN** a hidden page with a ready database list or ready storage usage session that is not awaiting explicit retry becomes visible after eligible backups were added
- **THEN** a fresh read updates its total without a page reload

#### Scenario: Visibility does not retry a failed database first page

- **WHEN** the current database's initial backup page or explicit reload fails and the page is hidden and then made visible
- **THEN** no new backup-list or usage read starts because of the visibility change or a polling tick
- **AND** the translated failure and explicit Retry remain until the user retries

#### Scenario: Visibility does not retry failed storage usage

- **WHEN** a storage usage session awaits explicit retry after an initial or background read failure and the page is hidden and then made visible
- **THEN** neither the visibility change nor a polling tick starts another usage read
- **AND** explicit Retry can start a new read and successful completion resumes polling
