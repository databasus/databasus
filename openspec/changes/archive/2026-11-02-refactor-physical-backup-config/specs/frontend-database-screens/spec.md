## MODIFIED Requirements

### Requirement: Database screen loads end in content or a recoverable failure

Loading the database list, the selected database, its backup configuration, the storage of the logical or physical configuration view or the first page of its logical backup list SHALL end either in content or in translated feedback with an explicit retry that reloads the same database and workspace. A failed load SHALL never leave a permanent loading indicator, appear as an empty list or read as a statement about the data, such as scheduled backups being disabled. The logical and physical backup configuration sections SHALL offer that retry together with a way to close the section. While the first page of the logical backup list has failed, the screen SHALL offer no backup start. A failed next page of the logical backup list SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A response for a database or workspace no longer shown SHALL not change the screen, and leaving the screen SHALL end its pending loads without later feedback. A failed background refresh of a list SHALL keep the current list without feedback.

#### Scenario: Loading the selected database fails

- **WHEN** the request for the selected database fails
- **THEN** the loading indicator ends and a translated error with a retry is shown in its place

#### Scenario: An earlier database responds late

- **WHEN** the user selects database B while the request for database A is pending, and A's response arrives last
- **THEN** the screen keeps showing database B

#### Scenario: The background refresh fails

- **WHEN** the periodic list refresh fails while the list is shown
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

### Requirement: The wizard's backup configuration step keeps its input

A background refresh of the database list, or the page becoming visible again, SHALL NOT reset the backup configuration step of the creation wizard or close a storage creation opened from it, for a logical or a physical database. After going back from the notifier step, the user SHALL be able to continue from the backup configuration step without changing a field. Until the database exists, going back from the logical or physical backup configuration step to the connection step and returning SHALL show the values entered there, even if the user never continued from that step, unless a changed physical backup type starts the configuration over.

#### Scenario: The database list refreshes during the wizard

- **WHEN** the user changes the backup configuration in the wizard and the database list refreshes in the background
- **THEN** the step keeps the entered values and shows no loading state

#### Scenario: The page becomes visible again during the wizard

- **WHEN** the user changes the backup configuration in the wizard, leaves the browser tab and returns to it
- **THEN** the step keeps the entered values and shows no loading state

#### Scenario: A storage is being created during a refresh

- **WHEN** the user has opened storage creation from the backup configuration step and the database list refreshes in the background
- **THEN** the storage creation stays open with its input

#### Scenario: Continuing after going back from the notifier step

- **WHEN** the user goes back from the notifier step to the backup configuration step
- **THEN** the continue action is available without changing a field

#### Scenario: Returning from the connection step

- **WHEN** the user changes the logical or physical backup configuration, goes back to the connection step and returns without changing the database type or the backup type
- **THEN** the backup configuration step shows the changed values

#### Scenario: Going back without continuing and changing the backup type

- **WHEN** the user changes the physical backup configuration, goes back to the connection step without continuing, chooses another backup type and continues
- **THEN** the backup configuration step shows the defaults of the new backup type

## ADDED Requirements

### Requirement: The physical retention form keeps entered values

Switching the physical retention policy or the full-backup retention policy SHALL keep the values entered for the other policy, so switching back shows them. Saving SHALL send only the retention the chosen policies allow: chains only send no full-backup retention, full backups only send a chain count of zero, a last-N full-backup retention sends its count and zero GFS periods, and a GFS full-backup retention sends its periods and a count of zero. A setting the form does not edit SHALL keep its stored value through a save.

A physical configuration SHALL be complete when its retention is valid, and, while backups are on, its encryption is chosen and its schedules are complete by the rules of `recurring-schedules`. A valid retention has a positive chain count when the policy keeps chains and, when the policy keeps full backups, a positive count for a last-N full-backup retention or at least one GFS period for a GFS full-backup retention. The retention rules SHALL apply whether backups are on or off, because the server validates retention in both cases. The wizard's step SHALL also require a storage.

#### Scenario: Switching the retention policy and back

- **WHEN** the user enters a chain count and full-backup retention, chooses chains only and returns to chains and full backups
- **THEN** the form shows the entered values

#### Scenario: Switching the full-backup policy and back

- **WHEN** the user enters GFS periods, chooses last-N, enters a count and returns to GFS
- **THEN** the form shows the entered periods, and returning to last-N shows the entered count

#### Scenario: Saving chains only

- **WHEN** the user saves a configuration with chains only after entering full-backup retention
- **THEN** the request carries no full-backup retention

#### Scenario: Saving a last-N retention

- **WHEN** the user chooses last-N over the default GFS periods, enters a count and saves
- **THEN** the request carries the count and zero GFS periods, and the server accepts it

#### Scenario: Saving a GFS retention

- **WHEN** the user chooses GFS after entering a last-N count and saves
- **THEN** the request carries the GFS periods and a count of zero

#### Scenario: A setting the form does not edit

- **WHEN** the user saves a physical configuration of a WAL streaming database
- **THEN** the stored WAL lag threshold is sent unchanged

#### Scenario: Invalid retention with backups off

- **WHEN** the user turns backups off and sets every GFS period to zero
- **THEN** Save is disabled until the retention is valid

### Requirement: Physical configuration save follows the acknowledged values

Save in the physical backup configuration section SHALL be available only when the values the form shows differ from the last acknowledged configuration and the configuration is complete. Returning a value to the acknowledged one SHALL disable Save again. A value of an unchosen policy SHALL not count as a difference.

#### Scenario: An edit is reverted

- **WHEN** the user changes a value and returns it to the acknowledged one
- **THEN** Save is disabled again

#### Scenario: A user tries a policy and returns

- **WHEN** a saved chains-only configuration holds no full-backup retention, and the user chooses chains and full backups and returns to chains only
- **THEN** Save is disabled
