# frontend-database-screens Specification

## Purpose

Keep the database screen's settings, creation wizard, copy, transfer and loads predictable, so a user never loses their place or their input to a reload or a late response.

## Requirements

### Requirement: One database settings section is edited at a time

The database settings SHALL let the user edit one section at a time: the name, the connection, the backup configuration, the healthcheck, the restore verification or the notifiers. While a section is open, including during its pending save, the other sections SHALL offer no edit control. Canceling a section SHALL close it, discard its unsaved input and show the last acknowledged values, without a loading state for the rest of the screen and without a new request for the database.

#### Scenario: A user tries to open a second section

- **WHEN** the connection section is open and the user looks at the notifier section
- **THEN** the notifier section offers no edit control until the connection section is closed

#### Scenario: A user cancels a section

- **WHEN** the user changes values in a section and cancels it
- **THEN** the section shows the last acknowledged values, the rest of the screen stays as it was and the database is not requested again

### Requirement: Saving a database settings section keeps the user's place

A successful save SHALL close only the saved section and show the acknowledged values both in that section and in the database list, except where the limited user suggestion below keeps the connection section open. The selected database, the open tab and the state of other sections SHALL remain unchanged, and the screen SHALL not be replaced by a loading state. A section that stores its settings on its own, such as the healthcheck or the restore verification, SHALL show its newly stored values after it closes. A refused save SHALL keep the section open with the entered values.

Saving the connection section SHALL show the acknowledged values on the screen as soon as the save succeeds, whatever follows. The section SHALL then check whether a limited user is advisable. The connection section SHALL close when no limited user is advisable, when the user skips the suggestion, or when the check right after the save fails, in which case the screen SHALL show the translated reason, because the settings are already saved. A failed check inside the open suggestion SHALL show its reason and keep the suggestion with its choices. Closing the suggestion or going back from it SHALL return to the open form with the saved values. Creating a limited user SHALL return to the open form with the generated credentials, and the user SHALL test and save them. An answer to a pending save or check SHALL change nothing on the screen if the section has left the screen, the selected database has changed or the session has changed.

#### Scenario: A user saves the connection settings

- **WHEN** an editor saves the connection section on the settings tab and no limited user is advisable
- **THEN** the section closes, the settings tab stays open and the screen shows no full-screen loading state

#### Scenario: A user renames a database

- **WHEN** a rename succeeds
- **THEN** the new name appears in the settings and in the database list, and the selection stays on that database

#### Scenario: A user saves the healthcheck settings

- **WHEN** the healthcheck section saves and closes
- **THEN** it shows the stored interval and threshold rather than the values from before the save

#### Scenario: A user closes the limited user suggestion after a save

- **WHEN** a save of the connection section succeeds, the limited user suggestion opens and the user closes it
- **THEN** the form stays open with the saved values, the database list shows the saved values, and cancelling the section then shows the saved values rather than the values from before the save

#### Scenario: A user goes back from the limited user suggestion

- **WHEN** a save of the connection section succeeds, the limited user suggestion opens and the user goes back from it
- **THEN** the form stays open with the saved values and asks for a new connection test before another save

#### Scenario: A user skips the limited user suggestion

- **WHEN** a save of the connection section succeeds, the limited user suggestion opens and the user confirms skipping it
- **THEN** the section closes and shows the saved values

#### Scenario: A user creates a limited user after a save

- **WHEN** a save of the connection section succeeds and the user creates a limited user from the suggestion
- **THEN** the form stays open with the generated user name and password and requires a new test and save

#### Scenario: The check after the save fails

- **WHEN** a save of the connection section succeeds and the check right after it fails
- **THEN** the screen shows the translated reason, the section closes and it shows the saved values

#### Scenario: The check inside the suggestion fails

- **WHEN** the limited user suggestion has opened and its own check fails
- **THEN** the suggestion stays open with its reason and its choices

#### Scenario: The selected database changes during a save

- **WHEN** the user selects another database while a save of the connection section is pending and the save then succeeds
- **THEN** the screen keeps showing the newly selected database, and no limited user suggestion opens

#### Scenario: The session changes during a save

- **WHEN** the session changes while a save of the connection section is pending and the save then answers
- **THEN** no feedback appears and no limited user suggestion opens

### Requirement: Copy and transfer update the database screen in place

A successful database copy or transfer SHALL update the list and selection, as the workspace selection rule defines, without reloading the application. When the selection changes, the address SHALL change to the selected database and offered tab, or the workspace database list when none remains. A refused copy or transfer SHALL keep the list, selection and address unchanged, and a refused transfer SHALL keep the transfer dialog open with its input. A late completion after another database, workspace or main section is selected SHALL NOT navigate away from that selection. Switching only tabs of the same database SHALL preserve the accepted completion behavior defined by Database actions report their outcome through themed feedback; it SHALL NOT count as abandoning that database selection.

#### Scenario: A copy succeeds

- **WHEN** an editor confirms copying the selected database and the server creates the copy while the source remains continuously selected, including after a tab change within that database
- **THEN** the copy appears in the list and is selected at the currently displayed tab when offered, otherwise Backups, with its matching address and without reloading the application
- **AND** a completion after a tab change shows no feedback

#### Scenario: A transfer succeeds

- **WHEN** an editor transfers the selected database to another workspace while it remains selected
- **THEN** the database leaves the current list and an available remaining database or the empty list becomes the selected destination with its matching address
- **AND** the application is not reloaded

#### Scenario: A copy finishes after another database is selected

- **WHEN** a copy finishes after the user selects another database
- **THEN** the list accepts the copy while the selected database and address remain unchanged

### Requirement: The database creation wizard keeps the input of every step

Until the database exists, going back from a later wizard step SHALL show each earlier step with the values the user entered there. This holds for the name and engine, the connection settings, the backup configuration of a logical or a physical database and the notifier selection. Going back from the connection step and returning to it without changing the type SHALL keep the connection input, including input that has not passed a test. The database type SHALL be compared when the user continues from the engine step or the PostgreSQL backup type step, so choosing another engine and returning to the original one before continuing, or clicking the already highlighted PostgreSQL engine for a physical database, SHALL keep the input. A different type at that moment SHALL discard the connection settings and backup configuration entered for the previous type, because they do not apply to the new one. Completing the wizard SHALL create the database with the latest values of every step entered before the creation, and a pending completion SHALL not be submitted twice.

#### Scenario: A user goes back from the notifier step

- **WHEN** the user configures logical backups, continues to the notifier step and goes back
- **THEN** the backup configuration step shows the configuration the user entered rather than the defaults

#### Scenario: A user completes the wizard after going back

- **WHEN** the user changes the backup configuration after going back and completes the wizard
- **THEN** the database is created with the changed configuration

#### Scenario: A user goes back from the connection step

- **WHEN** the user enters a host and password, goes back to the previous step without changing the type and returns
- **THEN** the connection step shows the entered host and password and asks for a connection test

#### Scenario: A user changes the engine after entering a connection

- **WHEN** the user enters a host and password, goes back to the engine step, chooses another engine and continues
- **THEN** the connection step shows the new engine's defaults with an empty host and password

#### Scenario: A user changes the PostgreSQL backup type after entering a connection

- **WHEN** the user enters a host and password for a logical database, goes back, chooses a physical database and continues
- **THEN** the connection step shows the physical defaults with an empty host and password

#### Scenario: A user returns to the original engine before continuing

- **WHEN** the user enters a host and password, goes back to the engine step, chooses another engine, chooses the original engine again and continues
- **THEN** the connection step shows the entered host and password

#### Scenario: A user clicks the highlighted PostgreSQL engine

- **WHEN** the user enters a host and password for a physical database, goes back to the engine step, clicks the highlighted PostgreSQL engine and continues
- **THEN** the backup type step still shows physical, and continuing shows the entered host and password

### Requirement: The last backup error stays until the server clears it

The database settings SHALL show the last backup error the server reports, together with the rule that clears it: the next successful backup. A successful connection test SHALL not hide the error and SHALL not be presented as a way to clear it, because a connection test changes nothing on the server.

#### Scenario: A connection test succeeds for a database with a backup error

- **WHEN** a user successfully tests the connection of a database that shows a last backup error
- **THEN** the success is reported and the last backup error remains shown with the next successful backup as the way to clear it

### Requirement: Database screen loads end in content or a recoverable failure

Loading the database list, the selected database, its backup configuration, the storage of the logical or physical configuration view or the first page of its logical backup list SHALL end either in content or in translated feedback with an explicit retry that reloads the same database and workspace, except when a current read establishes resource or workspace unavailability. A current HTTP 403 or 404 for the selected database itself SHALL clear its details, reconcile the workspace collection and access, and automatically resolve to available content with its matching address under frontend-navigation. Network failures and server failures SHALL retain the requested database address and offer retry rather than trigger resource fallback. Failure of a subordinate configuration, storage or backup read alone SHALL NOT establish that the database is unavailable; those views SHALL retain their recovery behavior.

A failed load SHALL never leave a permanent loading indicator. A retryable failure SHALL NOT appear as an empty list or read as a statement about the data, such as scheduled backups being disabled. The logical and physical backup configuration sections SHALL offer that retry together with a way to close the section. While the first page of the logical backup list has failed, the screen SHALL offer no backup start. A failed next page of the logical backup list SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A response for a database or workspace no longer shown SHALL not change the screen, and leaving the screen SHALL end its pending loads without later feedback. A failed background refresh of a list SHALL keep the current list without feedback unless the workspace database list load establishes that the workspace is unavailable. A current HTTP 403 or 404 during that load SHALL instead clear the workspace database content and refresh accessible workspace selection, on both initial load and background refresh.

#### Scenario: Loading the selected database fails

- **WHEN** the request for the selected database fails because of a network or server failure without establishing resource unavailability
- **THEN** the loading indicator ends and a translated error with a retry is shown in its place
- **AND** the requested database address stays unchanged and retry reloads that database

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

#### Scenario: The selected database read establishes resource unavailability

- **WHEN** the current request for the selected database returns HTTP 403 or 404
- **THEN** its details are cleared and the workspace collection and access are reconciled
- **AND** an available replacement database or list is shown at its matching address instead of offering repeated retries of the unavailable database

#### Scenario: Reconciliation fails after the selected database becomes unavailable

- **WHEN** the selected database read establishes unavailability but refreshing the workspace collection or access fails temporarily
- **THEN** the unavailable details remain absent and a translated error offers retry of reconciliation
- **AND** no replacement is claimed available until that reconciliation succeeds

### Requirement: The database details region has one scroll owner

The database details SHALL scroll in one vertical scroll region on desktop and at 320 CSS pixels, with no competing scroll region around it. Scrolling a long backup or verification list in that region SHALL keep loading its further pages.

#### Scenario: A user scrolls a long backup list

- **WHEN** the backups tab holds more backups than fit on the screen and the user scrolls to its end
- **THEN** the details region scrolls on its own and the next page of backups loads

### Requirement: Database screen controls keep their workspace permissions

Moving and recomposing the database screen SHALL keep every existing permission. A workspace viewer SHALL see the databases, their settings, backups and saved connection test, but no control to add, edit, copy, transfer or delete a database. The interface hiding a control SHALL not replace the server's authorization of the request.

#### Scenario: A viewer opens a database

- **WHEN** a workspace viewer opens a database's settings
- **THEN** the sections offer no edit control, and the copy, transfer and delete actions are absent

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

When the selected database does not offer the open tab, the screen SHALL show that database's Backups tab instead of an empty area, for the Verifications and Monitoring tabs. The address SHALL become that database's Backups address. This fallback SHALL replace the unavailable tab selection; selecting another database afterward SHALL NOT revive a hidden earlier tab preference. Back and Forward SHALL still restore explicitly recorded tab addresses when those destinations are available. A verification start answered after the user has left the backup list SHALL NOT switch the tab or address and SHALL show no feedback.

#### Scenario: Selecting a database without verifications

- **WHEN** the Verifications tab of a PostgreSQL logical database is open and the user selects a MySQL database
- **THEN** the MySQL database's Backups tab is shown and marked as open, and the address names its Backups tab

#### Scenario: Selecting a database without monitoring

- **WHEN** the Monitoring tab of a physical database is open and the user selects a database that has no monitoring
- **THEN** that database's Backups tab is shown and marked as open, and the address names its Backups tab

#### Scenario: Returning to a database that offers the chosen tab

- **WHEN** MySQL Backups is shown after fallback from Verifications and the user selects a PostgreSQL logical database
- **THEN** the PostgreSQL database's Backups tab opens at its matching address

#### Scenario: A verification start answers after a switch

- **WHEN** the user starts a backup verification, selects another database while the start is pending and the start then succeeds
- **THEN** the open tab and address stay as they were and no feedback appears

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

### Requirement: Completing the database creation wizard creates one database

The wizard SHALL create the database at most once, however many times the user completes it. When a request after the creation fails, the wizard SHALL stay on its last step and keep the entered values. It SHALL show a translated message that the database was created, together with the reason for the failure, and the message SHALL stay on the notifier and backup configuration steps until the next completion attempt. Completing the wizard again SHALL perform only the work that has not finished: it SHALL send the notifier selection only if it differs from the created database, save the backup configuration only if the server has not acknowledged it, and then request the first logical backup when logical backups are enabled. While this work runs, the wizard SHALL say that it is finishing the setup and not that it is creating a database.

For a physical database with backups enabled, saving its configuration SHALL leave the first full backup to the physical scheduler. The wizard SHALL send no separate physical backup request after that save, including when the scheduler starts or completes the first full before the save acknowledgement reaches the browser. The wizard SHALL close after the configuration is acknowledged. Explicit manual physical backup requests outside creation SHALL retain their existing queueing behavior.

Once the database exists, the wizard SHALL offer no way back to the connection step, because connection changes would no longer be applied. Once the server has acknowledged the backup configuration, the wizard SHALL offer no way back to that step, because another storage would start a storage switch. The notifier selection SHALL stay editable.

When the wizard closes, whether it finished or not, the database created by it SHALL join the list and become selected, and the list and the selected screen SHALL show what the server holds at that moment. Closing the wizard while a request runs SHALL start no later step, and a late answer SHALL show no feedback and change nothing in a wizard opened afterwards. A database whose create answer arrives after its wizard closed SHALL reach the list through a later refresh without becoming selected, then or when a later wizard closes.

#### Scenario: Completing the wizard

- **WHEN** a logical database, its backup configuration and the first backup start all succeed
- **THEN** the wizard closes, the created database is selected, its list card shows its storage, and its screen does not say that scheduled backups are disabled

#### Scenario: The create answer arrives after the wizard closed

- **WHEN** the user closes the wizard while the create request is pending, the database is created, and the user opens a new wizard and cancels it
- **THEN** the selection is unchanged, and a later refresh lists the created database

#### Scenario: Completing the wizard with backups disabled

- **WHEN** backups are disabled and the database and its backup configuration are saved, on the first attempt or on a retry
- **THEN** the wizard closes and no backup start is requested

#### Scenario: The wizard stays after a failure

- **WHEN** the database is created and the server refuses its backup configuration
- **THEN** the wizard stays on the notifier step with the entered values and shows that the database was created, with the reason
- **AND** the message is still shown after the user goes to the backup configuration step and returns

#### Scenario: Saving the backup configuration fails

- **WHEN** the database is created, the server refuses its backup configuration and the user completes the wizard again
- **THEN** no second database is created, the configuration is sent again, and the wizard closes after it is acknowledged and, for a logical database with backups enabled, the first backup start succeeds

#### Scenario: Starting the first backup fails

- **WHEN** a logical database and its backup configuration are saved, the first backup start fails and the user completes the wizard again
- **THEN** only the backup start is requested again, and the notifier step offers no way back to the backup configuration step

#### Scenario: The user corrects a refused configuration

- **WHEN** the server refuses the backup configuration, the user goes back to that step, changes it and completes the wizard
- **THEN** the changed configuration is saved for the database already created

#### Scenario: The user changes notifiers before retrying

- **WHEN** a request after the creation fails and the user changes the notifier selection before completing the wizard again
- **THEN** the created database receives the changed selection and no second database is created

#### Scenario: A retry is running

- **WHEN** the user completes the wizard again after a failure and the requests are pending
- **THEN** the wizard says that it is finishing the setup, and it does not say that it is creating a database

#### Scenario: The user closes the wizard after a failure

- **WHEN** a request after the creation fails and the user closes the wizard
- **THEN** the created database joins the list and is selected

#### Scenario: The user closes the wizard during completion

- **WHEN** the user closes the wizard while a request after the creation is pending, opens the wizard again, and the request then answers
- **THEN** no later request is sent, no feedback appears, and the newly opened wizard stays on its first step

#### Scenario: The user looks for the connection step after a failure

- **WHEN** a request after the creation fails and the user goes back to the backup configuration step
- **THEN** that step offers no way back to the connection step

#### Scenario: The creation itself fails

- **WHEN** the server refuses to create the database
- **THEN** the wizard shows the reason, says nothing about a created database, and leaves every earlier step reachable

#### Scenario: Physical configuration acknowledgement follows the first scheduled full

- **WHEN** the scheduler starts or completes the first physical full backup before the wizard receives acknowledgement of the enabled configuration
- **THEN** the wizard closes without queuing another physical backup
- **AND** no additional full or incremental backup is requested by completing the wizard

### Requirement: A changed physical backup type starts its backup configuration over

The physical backup type SHALL be compared when the user continues from the connection step. A backup type that differs from the one the stored backup configuration was entered for SHALL discard that configuration, because its schedules and retention do not apply to the new type, and the backup configuration step SHALL show the defaults of the new type. Returning to the original backup type before continuing SHALL keep the configuration.

#### Scenario: The user chooses another backup type after configuring backups

- **WHEN** the user continues from the physical backup configuration step, goes back to the connection step, chooses FULL and continues
- **THEN** the backup configuration step shows the FULL defaults, and completing the wizard sends policy `FULL_BACKUPS`, a chain count of 0, no incremental schedule and a WAL lag threshold of 0

#### Scenario: The user returns to the original backup type

- **WHEN** the user continues from the physical backup configuration step, goes back to the connection step, chooses another backup type, chooses the original one again and continues
- **THEN** the backup configuration step shows the configuration the user entered

### Requirement: The verification section ends its loads and saves recoverably

Loading the verification config in the settings tab or in its editor SHALL end in content or in translated feedback with an explicit retry, and the editor SHALL also offer a way to close the section. A refused save SHALL keep the section open with the entered values and show the translated reason. While a save is pending, the section SHALL offer no way to cancel. Save SHALL follow the values the form shows against the last acknowledged config, so returning a value to the acknowledged one disables it again, and a value the form hides SHALL not count as a difference. A save SHALL send the last acknowledged values for everything the form hides. An answer that arrives after the section has left the screen, the selected database has changed or the session has changed SHALL change nothing and show no feedback. None of these failures SHALL use a blocking alert.

#### Scenario: The editor cannot load

- **WHEN** the user opens the verification section and the request for its config fails
- **THEN** the section shows the translated reason with a retry and a way to close it, and no blocking alert appears
- **AND** a successful retry shows the form with the loaded values, and closing returns to the view

#### Scenario: The view cannot load

- **WHEN** the settings tab cannot load the verification config
- **THEN** the view shows the translated reason with a retry and no blocking alert appears, and a successful retry shows the config

#### Scenario: The server refuses a save

- **WHEN** the server refuses a verification config save
- **THEN** the section stays open with the entered values and shows the reason, and no blocking alert appears

#### Scenario: A save is pending

- **WHEN** a verification config save is pending
- **THEN** the cancel action is disabled until the save answers

#### Scenario: An edit is reverted

- **WHEN** the user changes a value and returns it to the acknowledged one
- **THEN** Save is disabled again

#### Scenario: A hidden schedule edit

- **WHEN** an acknowledged config verifies after each backup, and the user chooses a weekly schedule, changes its time and returns to verifying after each backup
- **THEN** Save is disabled

#### Scenario: A hidden notification edit

- **WHEN** automatic verification is off in the acknowledged config, and the user turns it on, changes a notification trigger and turns it off
- **THEN** Save is disabled

#### Scenario: A save after hiding edits

- **WHEN** verification is on in the acknowledged config, and the user changes a notification trigger and the schedule choice, turns verification off and saves
- **THEN** the request carries the acknowledged notification triggers and the acknowledged schedule choice

#### Scenario: The user leaves during a save

- **WHEN** the user opens another tab while a verification config save is pending and the save then answers, successfully or with a failure
- **THEN** the open tab stays as it is and no feedback about the save appears

#### Scenario: The selected database changes during a save

- **WHEN** the user selects another database while a verification config save is pending and the save then answers
- **THEN** the screen keeps showing the newly selected database and shows no feedback about the save

#### Scenario: The session changes during a save

- **WHEN** the session changes while a verification config save is pending and the save then answers
- **THEN** no feedback appears

#### Scenario: A load answers after the section has left

- **WHEN** the user leaves the settings tab while the verification config is loading and the request then answers, successfully or with a failure
- **THEN** the open tab stays as it is and no feedback appears

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

### Requirement: The physical backup list ends its loads recoverably

Loading the first page of the physical backup list or the backup configuration on its tab SHALL end in content or in translated feedback with an explicit retry that reloads the same database. A failed load SHALL never leave a permanent loading indicator, appear as an empty list, show a total usage or read as scheduled backups being disabled. While the first page has failed, the backup start SHALL stay disabled. A failed next page SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A refused backup start, cancel or delete SHALL show its reason once, keep the rows and leave the control usable. An answer that arrives after the list has left the screen or the session has changed SHALL change nothing and show no feedback. No failure on this screen SHALL use a blocking alert.

#### Scenario: The configuration cannot load

- **WHEN** the backups tab of a physical database cannot load the backup configuration
- **THEN** it shows the translated reason with a retry, no permanent loading indicator remains, and it does not say that scheduled backups are disabled

#### Scenario: The first page cannot load

- **WHEN** the first page of the physical backup list fails
- **THEN** the list shows the translated reason with a retry instead of the empty list text, shows no total usage, and keeps the backup start disabled

#### Scenario: The next page cannot load

- **WHEN** the user scrolls the physical backup list and the next page fails
- **THEN** the rows stay, a translated message appears once, and scrolling again requests that page

#### Scenario: A cancel or delete is refused

- **WHEN** the server refuses to cancel or delete a physical backup
- **THEN** a translated message appears once, the rows stay, no blocking alert appears, and the action is available again

#### Scenario: A backup start is refused

- **WHEN** the server refuses a physical backup start
- **THEN** a translated message appears once, the rows stay, no blocking alert appears, and the backup start is available again

#### Scenario: The user leaves during a list action

- **WHEN** the user opens another tab while the configuration load or a backup start, cancel or delete is pending and the request then answers, successfully or with a failure
- **THEN** the open tab stays as it is and no feedback appears

### Requirement: A manual physical backup keeps the loaded list

After the server accepts a manual physical backup, the loaded rows and the scroll position SHALL stay and the list SHALL show no loading state. The new backup SHALL appear on a later refresh when the current filters include it.

#### Scenario: A manual backup with a scrolled list

- **WHEN** the user has loaded several pages and starts a backup, and the server accepts it
- **THEN** the loaded rows and the scroll position stay with no loading state, and a later refresh that lists the backup shows it above them

### Requirement: The physical backup start follows the user's permission

The physical backup list SHALL offer the backup start only to users who may manage the database, because the server refuses it for the others. The restore action SHALL stay available to a user who can only view the database. Hiding a control SHALL not replace the server's authorization of the request. While the loaded backup config has backups turned off or names no storage, the backup start SHALL stay visible and disabled, because the server refuses such a start, and the screen SHALL say in the user's language that a backup can start once backups are turned on and a storage is chosen in the backup configuration. When the backup config failed to load, the screen SHALL NOT disable the start for this reason and SHALL NOT show that explanation, and the server's answer decides.

#### Scenario: A viewer opens the list

- **WHEN** a workspace viewer opens the backups tab of a physical database
- **THEN** the screen offers no backup start, cancel or delete, and it offers restore

#### Scenario: A member opens the list

- **WHEN** a workspace member opens the backups tab of a physical database
- **THEN** the screen offers the backup start

#### Scenario: A member opens the list of a database with backups turned off

- **WHEN** a workspace member opens the backups tab of a physical database whose loaded backup config has backups turned off
- **THEN** the backup start is disabled, no start request can be sent, and the screen shows the notice that scheduled backups are off together with the explanation of what a start needs

#### Scenario: A member opens the list of a database without a storage

- **WHEN** a workspace member opens the backups tab of a physical database whose loaded backup config has backups turned on and names no storage
- **THEN** the backup start is disabled, no start request can be sent, and the screen shows the explanation of what a start needs

#### Scenario: The backup config fails to load

- **WHEN** a workspace member opens the backups tab of a physical database and its backup config fails to load
- **THEN** the backup start stays enabled once the list has loaded, the screen shows the reason with a retry, and it does not show the explanation of what a start needs

### Requirement: The physical restore screen shows commands for the token of its shown target time

The physical restore screen SHALL show commands only with a token issued for the target time it shows. An empty target time SHALL mean the latest restorable moment and SHALL need its own token. A restore of one backup has no target time and SHALL have one token request. When the user changes the target time while a token request is pending, a later answer or failure for the earlier target time SHALL change nothing, and the screen SHALL keep showing that it is preparing until the newest request answers. When the newest request is refused, the screen SHALL show the translated reason with its hint and no commands. After the user closes the screen, an answer SHALL change nothing and show no feedback.

#### Scenario: An earlier token answers last

- **WHEN** the user chooses target time A, then target time B, B's answer arrives and A's answer arrives afterwards
- **THEN** the commands still use B's token and B's time

#### Scenario: An earlier request fails last

- **WHEN** the token request for target time B has answered and the request for the earlier target time A then fails
- **THEN** the commands for B stay and no error appears

#### Scenario: Waiting for the newest target time

- **WHEN** A's answer arrives while the request for the newer target time B is pending
- **THEN** the screen shows that it is preparing and shows no commands

#### Scenario: The newest request is refused

- **WHEN** the request for the shown target time is refused after an earlier target time had answered
- **THEN** the screen shows the translated reason with its hint and shows no commands

#### Scenario: Clearing a chosen time

- **WHEN** the user clears a chosen time while its request is pending and the earlier answer arrives last
- **THEN** the commands use the token issued without a target time and carry no recovery target time

#### Scenario: The user closes the screen during a request

- **WHEN** the user closes the restore screen while a token request is pending and the answer or failure then arrives
- **THEN** no message appears, and opening the screen again starts its own request, unaffected by the earlier answer

### Requirement: The verification list and its details end their loads recoverably

Loading the first page of a database's verifications or the details of one verification SHALL end in content or in translated feedback with an explicit retry that reloads the same database or verification. A failed first page SHALL never read as an empty list. A failed next page SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A failed cancel SHALL show its reason once and keep the rows. A failed background refresh SHALL keep the rows and show nothing. An answer that arrives after the list has left the screen or the session has changed SHALL change nothing and show no feedback. No failure on these screens SHALL use a blocking alert. The desktop list and the details SHALL show the same restore duration for the same verification. The mobile cards show no duration and are outside this rule. Closing the details SHALL return focus to the control that opened them.

#### Scenario: The first page cannot load

- **WHEN** the first page of the verification list fails
- **THEN** the list shows the translated reason with a retry instead of the empty list text
- **AND** a successful retry requests the same database's first page and shows its rows in place of the failure

#### Scenario: The next page cannot load

- **WHEN** the user scrolls the verification list and the next page fails
- **THEN** the rows stay, a translated message appears once, and scrolling again requests that page

#### Scenario: A cancel is refused

- **WHEN** the user cancels a pending or running verification and the request fails
- **THEN** a translated message appears once, the rows stay, no blocking alert appears, and the cancel action is available again

#### Scenario: The server is unreachable during refreshes

- **WHEN** the verification list is shown and its background refreshes fail
- **THEN** the rows stay and no alert or message appears

#### Scenario: The user leaves during a cancel

- **WHEN** the user opens another tab while a cancel is pending and the request then answers, successfully or with a failure
- **THEN** the open tab stays as it is and no feedback appears

#### Scenario: The details cannot load

- **WHEN** the details of a verification fail to load
- **THEN** the drawer shows the translated reason with a retry and no blocking alert, and a successful retry shows the details

#### Scenario: A short restore in the list and the details

- **WHEN** the restore of a verification took five seconds
- **THEN** the desktop list's restore line and the details' restore row both show `0m 5s`

#### Scenario: The same verification in the list and the details

- **WHEN** the restore of a verification took one hour and two minutes
- **THEN** the desktop list's restore line and the details' restore row both show `1h 2m 0s`

### Requirement: Healthcheck and notifier views end their loads recoverably

Loading the healthcheck configuration, the healthcheck attempts or the workspace notifiers on the database screen SHALL end in content or in translated feedback on the screen with an explicit retry that reloads the same database. The healthcheck editor SHALL also offer a way to close its section. A failed background refresh of shown attempts SHALL keep them and SHALL show no dialog or message. A refused healthcheck save SHALL keep the form and its values and show the translated reason. An answer that arrives after the section left the screen, the selected database changed or the session changed SHALL change nothing on the screen.

#### Scenario: Loading the healthcheck configuration in settings fails

- **WHEN** the settings view cannot load the healthcheck configuration of the selected database
- **THEN** the view shows the translated reason with a retry and no blocking alert appears
- **AND** a successful retry shows the configuration

#### Scenario: Loading the healthcheck configuration on the backups tab fails

- **WHEN** the backups tab cannot load the healthcheck configuration that decides whether attempts appear
- **THEN** the view shows the translated reason with a retry
- **AND** a successful retry shows the attempts when healthcheck is enabled and nothing when it is disabled

#### Scenario: Loading the configuration in the editor fails

- **WHEN** the healthcheck editor cannot load the configuration
- **THEN** the section shows the translated reason with a retry and a way to close it

#### Scenario: A refresh of the attempts fails

- **WHEN** a background refresh of healthcheck attempts fails while attempts are shown
- **THEN** the attempts stay on the screen and no alert or message appears

#### Scenario: The first load of attempts fails

- **WHEN** the first load of healthcheck attempts fails
- **THEN** the view shows the translated reason with a retry instead of an empty list

#### Scenario: The load after a period change fails

- **WHEN** the user changes the attempts period and that load fails
- **THEN** the view shows the translated reason with a retry and not the attempts of the old period
- **AND** a later polling tick does not replace the failure; only the retry or another period starts a new load

#### Scenario: The server refuses a healthcheck save

- **WHEN** the server refuses to save the healthcheck configuration
- **THEN** the form stays open with the entered values and shows the translated reason

#### Scenario: Loading the workspace notifiers fails

- **WHEN** the notifier view cannot load the workspace notifiers
- **THEN** the view shows the translated reason with a retry instead of a permanent loading indicator

#### Scenario: A healthcheck answer comes after the database changed

- **WHEN** a healthcheck load, retry or save answers after the user selected another database
- **THEN** the view of the selected database does not change and the section is not closed by that answer

### Requirement: Database actions report their outcome through themed feedback

A successful database copy and a successful connection test SHALL show a notice that follows the active theme and language and disappears on its own. A successful rename SHALL show the new name, and a successful delete SHALL remove the database from the list. The reason of a failed copy, rename, connection test or delete SHALL stay on the screen, follow the selected language, leave the entered name in place, and last until the next action of that database, until the user closes it, or until the user leaves the settings tab or selects another database. No database screen SHALL use a browser-native dialog.

An action's answer that arrives after another database was selected SHALL show no feedback and SHALL keep the selection, while the list still follows the server: a late rename shows the new name in the list, a late copy adds the copy without selecting it, and a late delete removes the deleted database. Returning to the source database after selecting another one SHALL NOT restore the earlier copy's right to change selection. An answer that arrives after only another tab of the same database was opened SHALL show no feedback, and the list and selection SHALL follow the server as if the tab had stayed: a copy becomes selected at the currently displayed tab when offered, otherwise Backups; a rename retains the current database and tab address; and a delete resolves to an available remaining database or the empty list with its matching address. This same-database tab change SHALL NOT invalidate accepted copy selection or deletion fallback. An answer that arrives after the user left the database screen or the session ended SHALL report nothing and change nothing. Any accepted list update that removes the currently selected resource SHALL still apply ordinary availability reconciliation rather than leave an unavailable selection or address.

#### Scenario: A connection test reports no connection

- **WHEN** testing the connection of a saved database reports that it cannot connect
- **THEN** the screen shows themed feedback with the translated reason and no browser-native dialog appears

#### Scenario: A connection test request fails

- **WHEN** the connection test request itself fails
- **THEN** the screen shows the translated reason and the test can start again

#### Scenario: A database is copied in the dark theme

- **WHEN** a copy succeeds while the dark theme is active
- **THEN** the success notice follows the dark theme and disappears on its own

#### Scenario: A copy is refused

- **WHEN** the server refuses to copy the database
- **THEN** the screen shows the translated reason, the selection stays and the copy action is available again

#### Scenario: A rename is refused

- **WHEN** the server refuses a new database name
- **THEN** the name field stays open with the entered name and the screen shows the translated reason

#### Scenario: A delete is refused

- **WHEN** the server refuses to delete the database
- **THEN** the screen shows the translated reason, the removing indicator ends, the database stays selected and the delete action is available again

#### Scenario: The language changes while a failure is shown

- **WHEN** the language changes while the reason of a failed action is on the screen
- **THEN** the reason appears in the selected language

#### Scenario: The user leaves while a failure is shown

- **WHEN** the reason of a failed action is on the screen and the user switches to another tab or database and returns
- **THEN** the reason is gone

#### Scenario: Answers come after another tab was opened

- **WHEN** a copy, rename or delete answers after the user opened another tab of the same database
- **THEN** no feedback appears and the list follows the server
- **AND** a copy becomes selected at the currently displayed offered tab address, a rename shows the new name without changing the address, and a delete replaces the address with that of an available remaining database or the empty list

#### Scenario: Answers come after another database was selected

- **WHEN** a copy, rename or delete of database A answers after the user selected database B
- **THEN** database B stays selected and no feedback appears
- **AND** the list adds the copy, shows the new name or removes database A, as the action requires

#### Scenario: The answer comes after the screen is gone

- **WHEN** a copy, test, rename or delete answers after the user left the database screen or the session ended
- **THEN** no feedback appears and nothing changes

#### Scenario: A copy answers after leaving and returning to its source database

- **WHEN** a copy of database A is pending, the user selects database B and then A again, and the copy succeeds
- **THEN** the copy joins the list without changing the current database, tab or address and no feedback appears

### Requirement: The verify action of a logical backup starts one check per confirmation

The logical backup list SHALL keep one verify confirmation open at a time, so repeated use of a row's verify action starts one restore check. Once that confirmation is declined or its request settles, the action SHALL work again.

#### Scenario: The verify action is used twice before the confirmation is answered

- **WHEN** the user activates a row's verify action twice and confirms what appears
- **THEN** one check request is sent

#### Scenario: A refused check can be tried again

- **WHEN** the server refuses a confirmed check and the user activates the row's verify action again
- **THEN** the translated reason appears, a new confirmation opens, and confirming it sends another check request

#### Scenario: A declined confirmation does not block the action

- **WHEN** the user declines the confirmation and activates the row's verify action again
- **THEN** a new confirmation opens

### Requirement: A language change keeps database drafts and restore commands

Changing the interface language SHALL keep the unsaved input of the physical backup config and the verification config, and SHALL keep the restore commands shown on the physical restore screen. It SHALL NOT start a new restore token request. A repeated request would replace the command on screen after the user may have copied it, would leave an extra live token unused until it expires, and would be refused while the user's restore stream runs, which would replace the commands with an error. Labels around the kept values SHALL follow the selected language.

#### Scenario: Language changes with a physical backup config draft

- **WHEN** a user changes the language while the physical backup config holds an unsaved edit
- **THEN** the edit remains, no save request is sent, and the form labels use the selected language

#### Scenario: Language changes with a verification config draft

- **WHEN** a user changes the language while the verification config holds an unsaved edit
- **THEN** the edit remains, no save request is sent, and the form labels use the selected language

#### Scenario: Language changes with restore commands on screen

- **WHEN** a user changes the language while the physical restore screen shows restore commands
- **THEN** the commands and their token stay the same, no new token is requested and the titles around them use the selected language
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

### Requirement: GFS help explains completed periods and current candidates

The logical and physical backup configuration screens SHALL explain GFS through their existing help controls in every supported interface language. Help SHALL distinguish hourly and daily counts, which include the current calendar period, from weekly, monthly and yearly counts, which reserve completed calendar periods and also protect the current period's newest successful candidate. It SHALL explain that periods use UTC, weeks start on Monday, and the reference is the newest successful eligible backup rather than the user's clock. Physical help SHALL describe full backups as the eligible backups. Help SHALL explain that one backup can satisfy several levels, so the configured counts do not promise a fixed total number of files. Short descriptions elsewhere in these screens SHALL agree with the help. Opening help SHALL NOT change draft values or mark the configuration as changed.

#### Scenario: Logical GFS help with two daily periods

- **WHEN** a user opens GFS help while editing logical retention with two daily periods, one weekly period and one monthly period
- **THEN** the help explains that the weekly and monthly counts preserve completed periods in addition to the current candidates
- **AND** the entered retention values and the draft's changed state stay the same

#### Scenario: Physical GFS help

- **WHEN** a user opens the physical configuration's GFS help
- **THEN** the help describes the same calendar rules for successful full backups
- **AND** it does not imply that selecting a full backup by GFS preserves its incremental backups or WAL

#### Scenario: A different interface language

- **WHEN** the user reads either GFS help in any supported interface language
- **THEN** the explanation conveys the same period, candidate and overlap rules
- **AND** UTC retention boundaries do not change with the interface language or browser region

### Requirement: Database creation and deletion reconcile the current address

Completing the creation wizard SHALL retain the existing rule that selects the created database only when the wizard closes, and that selection SHALL open its Backups address. A successful deletion of the still-selected database SHALL replace its address with an available remaining database's offered tab address or the workspace database list when none remains, including when the user has switched tabs within that database. Refused mutations and completions belonging to an abandoned database selection, workspace, main section or session SHALL NOT select another resource. Accepted removal of a currently selected resource SHALL still trigger ordinary availability reconciliation and canonical address replacement.

#### Scenario: The wizard closes after creating a database

- **WHEN** the wizard has created a database and closes while its workspace is still active
- **THEN** the database joins the list and its Backups tab becomes the selected destination with its matching address

#### Scenario: The selected database is deleted

- **WHEN** the selected database is successfully deleted while it remains selected
- **THEN** an available remaining database or the empty list becomes the destination and replaces the deleted database's address
