## ADDED Requirements

### Requirement: The agent screen ends its loads recoverably

Loading the verification agents for the first time SHALL end in the list or in translated feedback with an explicit retry, and a failed first load SHALL never read as an empty list. While that failure is shown, the periodic refresh SHALL send no request, only the retry SHALL reload, and creating an agent SHALL be disabled. Creating an agent SHALL also be disabled while a deletion or any load of the list runs, including the reload after a deletion, so no create dialog can be open when the failure appears. A failed background refresh SHALL keep the shown agents and show no feedback, and the next refresh SHALL try again. A refresh while a dialog is open SHALL leave the dialog and its entered values as they are. A confirmed deletion SHALL reload the list, so the deleted agent leaves at once and a refresh already running cannot bring it back. A failed reload SHALL show its reason with a retry, as a failed first load does. A refused creation SHALL keep the dialog open with the entered name. A refused rotation SHALL keep its dialog open and reveal no token. A refused deletion SHALL keep the agent in the list. Each refusal SHALL show its reason once, and none of these failures SHALL use a blocking alert.

#### Scenario: The first load fails

- **WHEN** an admin opens the agent screen and the request for the agents fails
- **THEN** the screen shows the translated reason with a retry instead of an empty list
- **AND** a refresh tick while the reason is shown sends no request for the agents, and creating an agent is disabled
- **AND** a successful retry shows the agents and enables creating, and a failed retry keeps the reason and the retry

#### Scenario: The first load is running

- **WHEN** the first load, a retry or the reload after a deletion of the agents is running
- **THEN** creating an agent is disabled

#### Scenario: The server becomes unreachable

- **WHEN** the agents are shown and the periodic refreshes keep failing
- **THEN** the agents stay on the screen and no message appears
- **AND** a later refresh that succeeds shows the current agents

#### Scenario: A dialog is open

- **WHEN** a refresh tick occurs while a dialog is open
- **THEN** the list refreshes, and the dialog and its entered values stay as they are

#### Scenario: A refused creation

- **WHEN** the server refuses to create an agent
- **THEN** the dialog stays open with the entered name, and the reason appears once

#### Scenario: A refused rotation

- **WHEN** the server refuses to rotate an agent's token
- **THEN** the rotate dialog stays open, the reason appears once, and no token is revealed

#### Scenario: A confirmed deletion

- **WHEN** the server confirms an agent's deletion, including while a refresh is running or refreshes are failing
- **THEN** the agent leaves the list at once, and a refresh that was already running does not bring it back
- **AND** creating an agent is disabled from the deletion request until the reload succeeds
- **AND** a successful reload shows the remaining agents, and a failed one shows the reason with a retry

#### Scenario: A refused deletion

- **WHEN** the server refuses to delete an agent
- **THEN** the agent stays in the list and the reason appears once

### Requirement: The agent token appears once in the interface

The interface SHALL show an agent's token only in the dialog that follows its creation or rotation. Closing that dialog SHALL remove the token from the screen, and the interface SHALL not store it in the browser. Install commands opened later SHALL use a placeholder in place of the token.

#### Scenario: The token dialog closes

- **WHEN** an admin closes the dialog that shows a token from a creation or a rotation
- **THEN** the token text is absent from the page and from `localStorage` and `sessionStorage`

#### Scenario: Install commands opened later

- **WHEN** an admin opens the install commands of an existing agent
- **THEN** the commands show a placeholder in place of the token
