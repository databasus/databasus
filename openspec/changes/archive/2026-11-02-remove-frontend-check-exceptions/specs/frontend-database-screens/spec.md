## ADDED Requirements

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

An action's answer that arrives after another database was selected SHALL show no feedback and SHALL keep the selection, while the list still follows the server: a late rename shows the new name in the list, a late copy adds the copy without selecting it, and a late delete removes the deleted database. An answer that arrives after another tab of the same database was opened SHALL show no feedback, and the list and selection SHALL follow the server as if the tab had stayed. An answer that arrives after the user left the database screen or the session ended SHALL report nothing and change nothing.

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
- **AND** a copy becomes selected, a rename shows the new name, and a delete moves the selection to another database

#### Scenario: Answers come after another database was selected

- **WHEN** a copy, rename or delete of database A answers after the user selected database B
- **THEN** database B stays selected and no feedback appears
- **AND** the list adds the copy, shows the new name or removes database A, as the action requires

#### Scenario: The answer comes after the screen is gone

- **WHEN** a copy, test, rename or delete answers after the user left the database screen or the session ended
- **THEN** no feedback appears and nothing changes
