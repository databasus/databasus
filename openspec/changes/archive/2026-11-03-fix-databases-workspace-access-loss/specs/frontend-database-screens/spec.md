## MODIFIED Requirements

### Requirement: Database screen loads end in content or a recoverable failure

Loading the database list, the selected database, its backup configuration, the storage of the logical or physical configuration view or the first page of its logical backup list SHALL end either in content or in translated feedback with an explicit retry that reloads the same database and workspace. A failed load SHALL never leave a permanent loading indicator, appear as an empty list or read as a statement about the data, such as scheduled backups being disabled. The logical and physical backup configuration sections SHALL offer that retry together with a way to close the section. While the first page of the logical backup list has failed, the screen SHALL offer no backup start. A failed next page of the logical backup list SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A response for a database or workspace no longer shown SHALL not change the screen, and leaving the screen SHALL end its pending loads without later feedback. A failed background refresh of a list SHALL keep the current list without feedback unless the workspace database list load establishes that the workspace is unavailable. A current HTTP 403 or 404 during that load SHALL instead clear the workspace database content and refresh accessible workspace selection, on both initial load and background refresh.

#### Scenario: Loading the selected database fails

- **WHEN** the request for the selected database fails
- **THEN** the loading indicator ends and a translated error with a retry is shown in its place

#### Scenario: An earlier database responds late

- **WHEN** the user selects database B while the request for database A is pending, and A's response arrives last
- **THEN** the screen keeps showing database B

#### Scenario: The background refresh fails

- **WHEN** the periodic list refresh fails temporarily while the list is shown, without establishing workspace unavailability
- **THEN** the list and selection stay unchanged and no error is shown

#### Scenario: The backup configuration editor cannot load

- **WHEN** the user opens the logical backup configuration section and the request for the configuration fails
- **THEN** the section shows the translated reason with a retry and a way to close the section

#### Scenario: The settings tab cannot load a logical configuration

- **WHEN** the settings tab of a logical database opens and the request for its backup configuration fails
- **THEN** the screen shows exactly one translated error with one retry and no blocking alert, and a successful retry shows the configuration

#### Scenario: The configuration view cannot load its storage

- **WHEN** a logical or physical configuration is shown and the request for its storage fails
- **THEN** the view shows the other values and the translated reason with a retry, and no blocking alert appears

#### Scenario: The backup list cannot load its first page

- **WHEN** the first page of the logical backup list fails
- **THEN** the list shows the translated reason with a retry instead of the empty list text

#### Scenario: The backup list cannot load its next page

- **WHEN** the user scrolls the logical backup list and the next page fails
- **THEN** the rows stay, a translated message appears once, and scrolling again requests that page

#### Scenario: The backups tab cannot load the configuration

- **WHEN** the backups tab cannot load the backup configuration
- **THEN** it shows the translated reason with a retry and does not say that scheduled backups are disabled

#### Scenario: The server is unreachable during polling

- **WHEN** the logical backup list is shown and its periodic refreshes keep failing
- **THEN** the rows stay on the screen and no blocking alert or message appears

#### Scenario: The physical configuration editor cannot load

- **WHEN** the user opens the physical backup configuration section and the request for the configuration fails
- **THEN** the section shows the translated reason with a retry and a way to close the section, and no blocking alert appears

#### Scenario: The settings tab cannot load a physical configuration

- **WHEN** the settings tab of a physical database opens and the request for its backup configuration fails
- **THEN** the screen shows exactly one translated error with one retry and no blocking alert, and a successful retry shows the configuration

#### Scenario: A network failure does not revoke workspace access

- **WHEN** the database list is shown and its background refresh fails because the network is unreachable
- **THEN** the list, selection, and workspace navigation stay unchanged without error feedback or a Retry action
- **AND** a later successful background refresh updates the list and validates the selection

#### Scenario: The initial database list load establishes workspace unavailability

- **WHEN** the initial workspace database list load fails with HTTP 403 or 404
- **THEN** the workspace content is cleared and accessible workspace selection is refreshed
- **AND** recovery does not repeatedly load the unavailable workspace through a local Retry action

#### Scenario: A background database list load establishes workspace unavailability

- **WHEN** a background workspace database list load fails with HTTP 403 or 404 while databases are shown
- **THEN** the previous database list and details are removed and accessible workspace selection is refreshed
- **AND** the silent-refresh policy does not retain the unavailable workspace content
