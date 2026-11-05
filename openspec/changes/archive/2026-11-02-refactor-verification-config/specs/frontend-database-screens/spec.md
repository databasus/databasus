## ADDED Requirements

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
