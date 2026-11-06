## MODIFIED Requirements

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

