## ADDED Requirements

### Requirement: Live file cleanup appears only beside Storage

Logical and physical backup settings SHALL show current storage name and provider icon with a compact contextual cleanup notice directly in that Storage section only while the visible pending count is nonzero. The notice SHALL state that old files are being deleted, show how many remain, and identify files awaiting retry without claiming that old backups are being copied. It SHALL use a progress activity indicator without inventing a percentage or fixed total.

The Storage section and its active cleanup SHALL remain available with scheduled backups disabled or no current storage selected. Empty or completed history SHALL occupy no space. The workspace database list, backup editor and transfer dialog SHALL contain no storage-switch history. A committed transfer SHALL close its dialog and update the source list in place; opening the transferred database in its target workspace SHALL read its current cleanup normally.

The notice SHALL fit the existing page scroll region in light and dark themes, support keyboard access to its retry action and wrap at narrow widths and enlarged text. Its status SHALL remain understandable without color alone.

#### Scenario: No cleanup is pending

- **WHEN** a database's current pending count is zero
- **THEN** the settings show ordinary Storage information without a history heading, empty disclosure or cleanup notice

#### Scenario: Cleanup is active

- **WHEN** a selected logical or physical database has visible pending file deletions
- **THEN** a single notice appears directly beside Storage and no banner appears above the database list

#### Scenario: Scheduled backups are disabled

- **WHEN** a database has pending cleanup and scheduled backups are disabled
- **THEN** Storage and the cleanup notice remain visible without enabling backups

#### Scenario: The configured storage is cleared

- **WHEN** no current storage is selected but earlier visible obligations remain
- **THEN** the Storage section still shows their cleanup notice

#### Scenario: A storage-changing transfer succeeds

- **WHEN** a transfer commits and returns its ordinary acknowledgement
- **THEN** the dialog closes and the database leaves the source list without a history screen or application reload

#### Scenario: The screen is narrow or uses a dark theme

- **WHEN** the active notice is viewed at 320 CSS pixels, enlarged text or in either supported theme
- **THEN** its text and actions remain readable inside the settings region without a new page-wide overflow or scroll owner

### Requirement: Background cleanup reads preserve the settings session

Opening database settings or acknowledging a storage change SHALL load current cleanup. While a known pending count is nonzero, the interface SHALL periodically read current state. It SHALL stop those reads when the count reaches zero, a read fails, the page unmounts, or the database, workspace or session changes. A failed read SHALL preserve the last known count and offer Retry status without resubmitting a configuration change.

A cleanup response SHALL update only cleanup presentation, never replace an open configuration draft, change selection or reset scroll. Late success or failure for an obsolete scope SHALL not change the current screen. A language change SHALL translate active notices and errors without resetting drafts or restarting the operation. Re-entry and reload SHALL recover work from the current summary rather than a remembered switch identifier.

#### Scenario: Cleanup completes during polling

- **WHEN** a background read reports zero outstanding files
- **THEN** the notice disappears and its polling stops

#### Scenario: A read fails during cleanup

- **WHEN** a background read fails while files are known to remain
- **THEN** the last known notice remains with a read-retry action and no mutation is replayed

#### Scenario: A user edits while cleanup progresses

- **WHEN** a user has an unsaved settings draft while a cleanup response arrives
- **THEN** only cleanup presentation changes and the entered fields, selection and scroll remain

#### Scenario: A pending response belongs to a previous selection

- **WHEN** a user changes database, workspace or session before a cleanup read returns
- **THEN** that response changes no current state or feedback and old polling is stopped

#### Scenario: Language changes while cleanup is shown

- **WHEN** a user switches interface language during cleanup or its status-read error
- **THEN** the notice and error use the new language while the current draft and operation remain
