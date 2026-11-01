## ADDED Requirements

### Requirement: The wizard's backup configuration step keeps its input

A background refresh of the database list, or the page becoming visible again, SHALL NOT reset the backup configuration step of the creation wizard or close a storage creation opened from it, for a logical or a physical database. After going back from the notifier step, the user SHALL be able to continue from the backup configuration step without changing a field. Going back from the logical backup configuration step to the connection step and returning SHALL show the values entered there, even if the user never continued from that step.

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

- **WHEN** the user changes the logical backup configuration, goes back to the connection step and returns without changing the type
- **THEN** the backup configuration step shows the changed values

### Requirement: A pending backup configuration save keeps its section

While a logical or physical backup configuration save is pending, the section SHALL offer no way to cancel, and the other sections SHALL offer no edit control. A refused save SHALL keep the section open with the entered values, SHALL show the translated reason and SHALL report no success. An answer that arrives after the section has left the screen, the selected database has changed or the session has changed SHALL change nothing on the screen and SHALL show no feedback.

#### Scenario: A user tries to cancel during a save

- **WHEN** a backup configuration save is pending
- **THEN** the cancel action is disabled until the save answers

#### Scenario: The server refuses a save

- **WHEN** the server refuses a logical or physical backup configuration save
- **THEN** the section stays open with the entered values and shows the reason, and the section does not report a saved configuration

#### Scenario: The selected database changes during a save

- **WHEN** the user selects another database while a backup configuration save is pending and the save then answers
- **THEN** the screen keeps showing the newly selected database and shows no feedback about the save

#### Scenario: The user leaves the settings tab during a save

- **WHEN** the user opens another tab while a backup configuration save is pending and the save then answers
- **THEN** the open tab stays as it is and no feedback about the save appears

#### Scenario: The session changes during a save

- **WHEN** the session changes while a backup configuration save is pending and the save then answers
- **THEN** no feedback appears

### Requirement: An answer for another database never replaces the shown one

A save answer for any database settings section, including a rename, SHALL NOT replace the database shown on the screen when the user has selected another database in the meantime.

#### Scenario: A rename answers after another database is selected

- **WHEN** the user renames database A, selects database B while the rename is pending and the rename then succeeds
- **THEN** the screen keeps showing database B and the database list shows A's new name

### Requirement: A selected database that lacks the open tab shows its backups

When the selected database does not offer the open tab, the screen SHALL show that database's backups tab instead of an empty area, for the verifications and the monitoring tabs. The user's choice of tab SHALL stay, so selecting a database that offers it SHALL open it again. A verification start answered after the user has left the backup list SHALL NOT switch the tab and SHALL show no feedback.

#### Scenario: Selecting a database without verifications

- **WHEN** the verifications tab of a PostgreSQL logical database is open and the user selects a MySQL database
- **THEN** the backups tab of the MySQL database is shown and marked as the open tab

#### Scenario: Selecting a database without monitoring

- **WHEN** the monitoring tab of a physical database is open and the user selects a database that has no monitoring
- **THEN** the backups tab of that database is shown and marked as the open tab

#### Scenario: Returning to a database that offers the chosen tab

- **WHEN** the backups tab of the MySQL database is shown because it has no verifications and the user selects a PostgreSQL logical database
- **THEN** the verifications tab of that database opens

#### Scenario: A verification start answers after a switch

- **WHEN** the user starts a backup verification, selects another database while the start is pending and the start then succeeds
- **THEN** the open tab stays as it was and no feedback appears

### Requirement: The logical retention form keeps entered values

Switching the retention policy of a logical backup configuration SHALL keep the values entered for every policy, so returning to a policy shows them again. When the chosen count policy has no positive count, or the chosen GFS policy has no period set among the ones the form shows, the form SHALL fill in that policy's default values. Saving a GFS policy SHALL require at least one period set among the shown ones, and the hourly period SHALL count when the form shows it. Saving a count policy SHALL require a positive count. These retention rules SHALL hold while backups are turned off too, because the form still shows retention and the server refuses such values.

#### Scenario: A user returns to an edited GFS policy

- **WHEN** the user sets GFS periods, switches to another policy and switches back to GFS
- **THEN** the form shows the periods the user set

#### Scenario: A saved configuration never used GFS

- **WHEN** a saved configuration holds no GFS period and the user switches to GFS
- **THEN** the form shows the default GFS periods and can be saved

#### Scenario: Only hourly backups are kept

- **WHEN** the schedule is hourly and the only GFS period set is the hourly one
- **THEN** the configuration can be saved

#### Scenario: A GFS policy has no period with backups off

- **WHEN** backups are turned off and the user sets every shown GFS period to zero
- **THEN** Save is disabled

### Requirement: Logical advanced settings follow their values

The advanced settings of a logical backup configuration SHALL start open when their values differ from the defaults, and closed otherwise, for both a loaded configuration and a wizard draft. When backups are turned off, the form SHALL show neither the advanced settings nor their toggle, because none of them apply.

#### Scenario: A configuration with retry turned off

- **WHEN** a user opens a saved configuration with retry turned off
- **THEN** the advanced settings are open

#### Scenario: A configuration with default retry

- **WHEN** a user opens a saved configuration with retry on and the default number of tries
- **THEN** the advanced settings are closed and the toggle opens them

#### Scenario: Backups are turned off

- **WHEN** the user turns backups off
- **THEN** the advanced settings and their toggle are not shown

### Requirement: Logical configuration save follows the acknowledged values

The settings section of a logical backup configuration SHALL enable Save only while the values the form shows differ from the acknowledged configuration. Reverting an edit SHALL disable Save again. Default values the form filled in for a retention policy that is no longer chosen SHALL not count as a difference.

#### Scenario: A user reverts an edit

- **WHEN** the user changes the retention period and changes it back
- **THEN** Save is disabled

#### Scenario: A user tries a policy and returns

- **WHEN** a saved configuration uses the time period policy and holds no GFS period, and the user switches to GFS and back to the time period
- **THEN** Save is disabled

## MODIFIED Requirements

### Requirement: Database screen loads end in content or a recoverable failure

Loading the database list, the selected database, its backup configuration, the storage of the logical configuration view or the first page of its logical backup list SHALL end either in content or in translated feedback with an explicit retry that reloads the same database and workspace. A failed load SHALL never leave a permanent loading indicator, appear as an empty list or read as a statement about the data, such as scheduled backups being disabled. The logical backup configuration section SHALL offer that retry together with a way to close the section. The loads of the physical configuration editor and show view are outside this requirement, and their failures stay an `alert`. While the first page of the logical backup list has failed, the screen SHALL offer no backup start. A failed next page of the logical backup list SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A response for a database or workspace no longer shown SHALL not change the screen, and leaving the screen SHALL end its pending loads without later feedback. A failed background refresh of a list SHALL keep the current list without feedback.

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

- **WHEN** the logical configuration is shown and the request for its storage fails
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
