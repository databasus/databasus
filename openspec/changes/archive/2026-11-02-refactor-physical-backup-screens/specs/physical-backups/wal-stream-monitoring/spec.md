## MODIFIED Requirements

### Requirement: The Monitoring tab offers to start a full backup when that is the fix

When a full backup is what repairs the situation, the verdict banner SHALL offer a button that starts one. That is the case while all of these hold:

- the verdict is healthy, needs attention or broken, and the current state is confirmed;
- the WAL chain broken reason or the slot missing reason is listed, or no full backup has completed;
- no full backup is requested or running.

The not streaming verdict SHALL never offer the button, because Databasus takes no backups while streaming is off. The collecting data verdict SHALL not offer it either: it lasts at most two minutes, and the restore sentence already says whether a first full backup exists.

The button SHALL request a full backup at once, the same way the Backups tab does. While the request runs, the button SHALL show that it is working and SHALL not accept another click. After the request succeeds, the tab SHALL refresh at once, so the banner says that a full backup is on its way without waiting for the next scheduled refresh. While the tab is open, a failed request SHALL show the error and leave the button available.

#### Scenario: Broken chain fixed from the tab

- **WHEN** a user who may manage the database sees the WAL chain broken reason with no full backup pending, and presses the button
- **THEN** a full backup is requested
- **AND** the banner says that a full backup is on its way

#### Scenario: Full backup already pending

- **WHEN** the WAL chain broken reason is listed and a full backup is requested or running
- **THEN** the banner offers no button
- **AND** says that a new full backup is on its way

#### Scenario: Request refused

- **WHEN** the full backup request fails while the tab is open
- **THEN** the tab shows the error
- **AND** the button can be pressed again

#### Scenario: Missing slot

- **WHEN** a user who may manage the database sees the slot missing reason with no full backup pending
- **THEN** the banner offers the full backup button

#### Scenario: Healthy streaming before the first full backup

- **WHEN** the verdict is healthy, no full backup has completed and none is requested or running
- **THEN** the banner offers the full backup button

#### Scenario: Streaming off

- **WHEN** the verdict is not streaming because backups are disabled, and no full backup has completed
- **THEN** the banner offers no full backup button

## ADDED Requirements

### Requirement: The full backup request keeps one owner while the Monitoring tab is open

While a full backup request runs, switching the interface language or receiving a scheduled refresh SHALL keep the button busy, and the button SHALL accept no other click. Messages about the request SHALL use the language current when they appear. Leaving the tab means opening another tab of the database page or selecting another database. When the user leaves, the request SHALL continue on the server, but its answer, including a failure, SHALL show no feedback, start no refresh and leave no refresh running. Hiding the browser tab SHALL not count as leaving.

#### Scenario: The language changes during a request

- **WHEN** the user presses the full backup button, switches the interface language while the request runs and a scheduled refresh arrives
- **THEN** the button stays busy until the request has answered and the refresh after it has landed, and only one request was sent

#### Scenario: The request fails in the new language

- **WHEN** the language changes while the request runs and the request then fails
- **THEN** the error appears in the new language, and the button can be pressed again

#### Scenario: The request succeeds in the new language

- **WHEN** the language changes while the request runs and the request then succeeds
- **THEN** the success message appears in the new language

#### Scenario: The user leaves during a request

- **WHEN** the user opens another tab of the database, such as Backups, while a full backup request runs and the request then succeeds
- **THEN** no message appears and no statistics request follows

#### Scenario: The request fails after the user leaves

- **WHEN** the user opens another tab of the database while a full backup request runs and the request then fails
- **THEN** no error appears, and on return the button is offered again because no full backup is pending

#### Scenario: The browser tab is hidden during a request

- **WHEN** the user hides the browser tab while a full backup request runs and the request then succeeds
- **THEN** the message appears and the button stays busy until the refresh after it has landed
