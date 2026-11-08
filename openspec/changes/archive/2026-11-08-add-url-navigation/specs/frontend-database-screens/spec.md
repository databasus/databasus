## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Database creation and deletion reconcile the current address

Completing the creation wizard SHALL retain the existing rule that selects the created database only when the wizard closes, and that selection SHALL open its Backups address. A successful deletion of the still-selected database SHALL replace its address with an available remaining database's offered tab address or the workspace database list when none remains, including when the user has switched tabs within that database. Refused mutations and completions belonging to an abandoned database selection, workspace, main section or session SHALL NOT select another resource. Accepted removal of a currently selected resource SHALL still trigger ordinary availability reconciliation and canonical address replacement.

#### Scenario: The wizard closes after creating a database

- **WHEN** the wizard has created a database and closes while its workspace is still active
- **THEN** the database joins the list and its Backups tab becomes the selected destination with its matching address

#### Scenario: The selected database is deleted

- **WHEN** the selected database is successfully deleted while it remains selected
- **THEN** an available remaining database or the empty list becomes the destination and replaces the deleted database's address
