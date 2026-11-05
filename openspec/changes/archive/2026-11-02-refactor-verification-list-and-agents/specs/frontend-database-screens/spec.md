## ADDED Requirements

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
