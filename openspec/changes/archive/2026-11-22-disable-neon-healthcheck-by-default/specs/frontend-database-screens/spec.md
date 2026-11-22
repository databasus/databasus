## ADDED Requirements

### Requirement: Switching the healthcheck on for Neon asks for a delayed confirmation

When the healthcheck editor shows a database whose config says that switching the checks on needs a confirmation, turning the checks switch from off to on SHALL NOT turn it on at once. The editor SHALL show a confirmation dialog that explains, in the interface language, that Neon stops billing compute while the database sleeps, that a check every minute keeps it awake, and that compute is then billed around the clock. The dialog's confirm button SHALL be disabled for 10 seconds after the dialog opens and SHALL show the seconds left while it waits. Confirming SHALL turn the switch on and leave the change unsaved until the user saves the section, as for any other edit. Cancelling or closing the dialog SHALL leave the switch off and leave the form's unsaved state as it was before the dialog opened. Turning the switch off SHALL need no confirmation, and a database whose config needs no confirmation SHALL get no dialog.

#### Scenario: The confirm button waits 10 seconds

- **WHEN** the user turns on the checks of a Neon database in the healthcheck editor
- **THEN** a dialog explains the extra compute cost, the switch stays off, and the confirm button stays disabled with a countdown until 10 seconds have passed

#### Scenario: Confirming after the wait

- **WHEN** 10 seconds have passed in the dialog and the user confirms
- **THEN** the dialog closes, the switch is on, the check fields appear and the save button is offered

#### Scenario: Cancelling the dialog

- **WHEN** the user cancels or closes the dialog
- **THEN** the switch stays off, and the save button is shown only if it was already shown before the dialog opened

#### Scenario: Turning the checks off

- **WHEN** the user turns off the checks of a Neon database whose checks are on
- **THEN** the switch turns off without a dialog

#### Scenario: A database outside Neon

- **WHEN** the user turns on the checks of a database whose config needs no confirmation
- **THEN** the switch turns on at once without a dialog
