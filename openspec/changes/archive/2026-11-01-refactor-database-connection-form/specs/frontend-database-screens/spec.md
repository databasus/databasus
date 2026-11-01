## MODIFIED Requirements

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

### Requirement: The database creation wizard keeps the input of every step

Going back from a later wizard step SHALL show each earlier step with the values the user entered there. This holds for the name and engine, the connection settings, the backup configuration of a logical or a physical database and the notifier selection. Going back from the connection step and returning to it without changing the type SHALL keep the connection input, including input that has not passed a test. The database type SHALL be compared when the user continues from the engine step or the PostgreSQL backup type step, so choosing another engine and returning to the original one before continuing, or clicking the already highlighted PostgreSQL engine for a physical database, SHALL keep the input. A different type at that moment SHALL discard the connection settings and backup configuration entered for the previous type, because they do not apply to the new one. Completing the wizard SHALL create the database with the latest values of every step, and a pending completion SHALL not be submitted twice.

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
