## ADDED Requirements

### Requirement: The physical backup list ends its loads recoverably

Loading the first page of the physical backup list or the backup configuration on its tab SHALL end in content or in translated feedback with an explicit retry that reloads the same database. A failed load SHALL never leave a permanent loading indicator, appear as an empty list, show a total usage or read as scheduled backups being disabled. While the first page has failed, the backup start SHALL stay disabled. A failed next page SHALL keep the loaded rows and show its translated reason once, and a later scroll SHALL request that page again. A refused backup start, cancel or delete SHALL show its reason once, keep the rows and leave the control usable. An answer that arrives after the list has left the screen or the session has changed SHALL change nothing and show no feedback. No failure on this screen SHALL use a blocking alert.

#### Scenario: The configuration cannot load

- **WHEN** the backups tab of a physical database cannot load the backup configuration
- **THEN** it shows the translated reason with a retry, no permanent loading indicator remains, and it does not say that scheduled backups are disabled

#### Scenario: The first page cannot load

- **WHEN** the first page of the physical backup list fails
- **THEN** the list shows the translated reason with a retry instead of the empty list text, shows no total usage, and keeps the backup start disabled

#### Scenario: The next page cannot load

- **WHEN** the user scrolls the physical backup list and the next page fails
- **THEN** the rows stay, a translated message appears once, and scrolling again requests that page

#### Scenario: A cancel or delete is refused

- **WHEN** the server refuses to cancel or delete a physical backup
- **THEN** a translated message appears once, the rows stay, no blocking alert appears, and the action is available again

#### Scenario: A backup start is refused

- **WHEN** the server refuses a physical backup start
- **THEN** a translated message appears once, the rows stay, no blocking alert appears, and the backup start is available again

#### Scenario: The user leaves during a list action

- **WHEN** the user opens another tab while the configuration load or a backup start, cancel or delete is pending and the request then answers, successfully or with a failure
- **THEN** the open tab stays as it is and no feedback appears

### Requirement: A manual physical backup keeps the loaded list

After the server accepts a manual physical backup, the loaded rows and the scroll position SHALL stay and the list SHALL show no loading state. The new backup SHALL appear on a later refresh when the current filters include it.

#### Scenario: A manual backup with a scrolled list

- **WHEN** the user has loaded several pages and starts a backup, and the server accepts it
- **THEN** the loaded rows and the scroll position stay with no loading state, and a later refresh that lists the backup shows it above them

### Requirement: The physical backup start follows the user's permission

The physical backup list SHALL offer the backup start only to users who may manage the database, because the server refuses it for the others. The restore action SHALL stay available to a user who can only view the database. Hiding a control SHALL not replace the server's authorization of the request.

#### Scenario: A viewer opens the list

- **WHEN** a workspace viewer opens the backups tab of a physical database
- **THEN** the screen offers no backup start, cancel or delete, and it offers restore

#### Scenario: A member opens the list

- **WHEN** a workspace member opens the backups tab of a physical database
- **THEN** the screen offers the backup start

### Requirement: The physical restore screen shows commands for the token of its shown target time

The physical restore screen SHALL show commands only with a token issued for the target time it shows. An empty target time SHALL mean the latest restorable moment and SHALL need its own token. A restore of one backup has no target time and SHALL have one token request. When the user changes the target time while a token request is pending, a later answer or failure for the earlier target time SHALL change nothing, and the screen SHALL keep showing that it is preparing until the newest request answers. When the newest request is refused, the screen SHALL show the translated reason with its hint and no commands. After the user closes the screen, an answer SHALL change nothing and show no feedback.

#### Scenario: An earlier token answers last

- **WHEN** the user chooses target time A, then target time B, B's answer arrives and A's answer arrives afterwards
- **THEN** the commands still use B's token and B's time

#### Scenario: An earlier request fails last

- **WHEN** the token request for target time B has answered and the request for the earlier target time A then fails
- **THEN** the commands for B stay and no error appears

#### Scenario: Waiting for the newest target time

- **WHEN** A's answer arrives while the request for the newer target time B is pending
- **THEN** the screen shows that it is preparing and shows no commands

#### Scenario: The newest request is refused

- **WHEN** the request for the shown target time is refused after an earlier target time had answered
- **THEN** the screen shows the translated reason with its hint and shows no commands

#### Scenario: Clearing a chosen time

- **WHEN** the user clears a chosen time while its request is pending and the earlier answer arrives last
- **THEN** the commands use the token issued without a target time and carry no recovery target time

#### Scenario: The user closes the screen during a request

- **WHEN** the user closes the restore screen while a token request is pending and the answer or failure then arrives
- **THEN** no message appears, and opening the screen again starts its own request, unaffected by the earlier answer
