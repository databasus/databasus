## MODIFIED Requirements

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

### Requirement: The wizard's backup configuration step keeps its input

A background refresh of the database list, or the page becoming visible again, SHALL NOT reset the backup configuration step of the creation wizard or close a storage creation opened from it, for a logical or a physical database. After going back from the notifier step, the user SHALL be able to continue from the backup configuration step without changing a field. Until the database exists, going back from the logical backup configuration step to the connection step and returning SHALL show the values entered there, even if the user never continued from that step.

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

## ADDED Requirements

### Requirement: Completing the database creation wizard creates one database

The wizard SHALL create the database at most once, however many times the user completes it. When a request after the creation fails, the wizard SHALL stay on its last step and keep the entered values. It SHALL show a translated message that the database was created, together with the reason for the failure, and the message SHALL stay on the notifier and backup configuration steps until the next completion attempt. Completing the wizard again SHALL perform only the work that has not finished: it SHALL send the notifier selection only if it differs from the created database, save the backup configuration only if the server has not acknowledged it, and then start the first backup when backups are enabled. While this work runs, the wizard SHALL say that it is finishing the setup and not that it is creating a database.

Once the database exists, the wizard SHALL offer no way back to the connection step, because connection changes would no longer be applied. Once the server has acknowledged the backup configuration, the wizard SHALL offer no way back to that step, because another storage would start a storage switch. The notifier selection SHALL stay editable.

When the wizard closes, whether it finished or not, the database created by it SHALL join the list and become selected, and the list and the selected screen SHALL show what the server holds at that moment. Closing the wizard while a request runs SHALL start no later step, and a late answer SHALL show no feedback and change nothing in a wizard opened afterwards. A database whose create answer arrives after its wizard closed SHALL reach the list through a later refresh without becoming selected, then or when a later wizard closes.

#### Scenario: Completing the wizard

- **WHEN** the database, its backup configuration and the first backup start all succeed
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
- **THEN** no second database is created, the configuration is sent again, and the wizard closes after it and the first backup start succeed

#### Scenario: Starting the first backup fails

- **WHEN** the database and its backup configuration are saved, the first backup start fails and the user completes the wizard again
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

### Requirement: A changed physical backup type starts its backup configuration over

The physical backup type SHALL be compared when the user continues from the connection step. A backup type that differs from the one the stored backup configuration was entered for SHALL discard that configuration, because its schedules and retention do not apply to the new type, and the backup configuration step SHALL show the defaults of the new type. Returning to the original backup type before continuing SHALL keep the configuration.

#### Scenario: The user chooses another backup type after configuring backups

- **WHEN** the user continues from the physical backup configuration step, goes back to the connection step, chooses FULL and continues
- **THEN** the backup configuration step shows the FULL defaults, and completing the wizard sends policy `FULL_BACKUPS`, a chain count of 0, no incremental schedule and a WAL lag threshold of 0

#### Scenario: The user returns to the original backup type

- **WHEN** the user continues from the physical backup configuration step, goes back to the connection step, chooses another backup type, chooses the original one again and continues
- **THEN** the backup configuration step shows the configuration the user entered
