## Purpose

Keep the database screen's settings, creation wizard, copy, transfer and loads predictable, so a user never loses their place or their input to a reload or a late response.

## ADDED Requirements

### Requirement: One database settings section is edited at a time

The database settings SHALL let the user edit one section at a time: the name, the connection, the backup configuration, the healthcheck, the restore verification or the notifiers. While a section is open, including during its pending save, the other sections SHALL offer no edit control. Canceling a section SHALL close it, discard its unsaved input and show the last acknowledged values, without a loading state for the rest of the screen and without a new request for the database.

#### Scenario: A user tries to open a second section

- **WHEN** the connection section is open and the user looks at the notifier section
- **THEN** the notifier section offers no edit control until the connection section is closed

#### Scenario: A user cancels a section

- **WHEN** the user changes values in a section and cancels it
- **THEN** the section shows the last acknowledged values, the rest of the screen stays as it was and the database is not requested again

### Requirement: Saving a database settings section keeps the user's place

A successful save SHALL close only the saved section and show the acknowledged values both in that section and in the database list. The selected database, the open tab and the state of other sections SHALL remain unchanged, and the screen SHALL not be replaced by a loading state. A section that stores its settings on its own, such as the healthcheck or the restore verification, SHALL show its newly stored values after it closes. A refused save SHALL keep the section open with the entered values.

#### Scenario: A user saves the connection settings

- **WHEN** an editor saves the connection section on the settings tab
- **THEN** the section closes, the settings tab stays open and the screen shows no full-screen loading state

#### Scenario: A user renames a database

- **WHEN** a rename succeeds
- **THEN** the new name appears in the settings and in the database list, and the selection stays on that database

#### Scenario: A user saves the healthcheck settings

- **WHEN** the healthcheck section saves and closes
- **THEN** it shows the stored interval and threshold rather than the values from before the save

### Requirement: Copy and transfer update the database screen in place

A successful database copy or transfer SHALL update the list and selection, as the workspace selection rule defines, without reloading the application. A refused copy or transfer SHALL keep the list and the selection unchanged, and a refused transfer SHALL keep the transfer dialog open with its input.

#### Scenario: A copy succeeds

- **WHEN** an editor confirms copying the selected database and the server creates the copy
- **THEN** the copy appears in the list and is selected, and the application is not reloaded

#### Scenario: A transfer succeeds

- **WHEN** an editor transfers the selected database to another workspace
- **THEN** the database leaves the current list and the application is not reloaded

### Requirement: The database creation wizard keeps the input of every step

Going back from a later wizard step SHALL show each earlier step with the values the user entered there. This holds for the name and engine, the connection settings, the backup configuration of a logical or a physical database and the notifier selection. Changing the engine or the PostgreSQL backup type SHALL discard the connection settings and backup configuration entered for the previous type, because they do not apply to the new one. Completing the wizard SHALL create the database with the latest values of every step, and a pending completion SHALL not be submitted twice.

#### Scenario: A user goes back from the notifier step

- **WHEN** the user configures logical backups, continues to the notifier step and goes back
- **THEN** the backup configuration step shows the configuration the user entered rather than the defaults

#### Scenario: A user completes the wizard after going back

- **WHEN** the user changes the backup configuration after going back and completes the wizard
- **THEN** the database is created with the changed configuration

### Requirement: The last backup error stays until the server clears it

The database settings SHALL show the last backup error the server reports, together with the rule that clears it: the next successful backup. A successful connection test SHALL not hide the error and SHALL not be presented as a way to clear it, because a connection test changes nothing on the server.

#### Scenario: A connection test succeeds for a database with a backup error

- **WHEN** a user successfully tests the connection of a database that shows a last backup error
- **THEN** the success is reported and the last backup error remains shown with the next successful backup as the way to clear it

### Requirement: Database screen loads end in content or a recoverable failure

Loading the database list, the selected database or its backup configuration SHALL end either in content or in translated feedback with an explicit retry that reloads the same database and workspace. A failed load SHALL never leave a permanent loading indicator or appear as an empty list. A response for a database or workspace no longer shown SHALL not change the screen, and leaving the screen SHALL end its pending loads without later feedback. A failed background refresh of the list SHALL keep the current list without feedback.

#### Scenario: Loading the selected database fails

- **WHEN** the request for the selected database fails
- **THEN** the loading indicator ends and a translated error with a retry is shown in its place

#### Scenario: An earlier database responds late

- **WHEN** the user selects database B while the request for database A is pending, and A's response arrives last
- **THEN** the screen keeps showing database B

#### Scenario: The background refresh fails

- **WHEN** the periodic list refresh fails while the list is shown
- **THEN** the list and selection stay unchanged and no error is shown

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
