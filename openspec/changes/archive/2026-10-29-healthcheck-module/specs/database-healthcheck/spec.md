## Purpose

Defines the scheduled availability check of a user's database: when Databasus checks it, when the database counts as unavailable and available again, who is notified, how long the history of checks is kept and how that history is read.

## ADDED Requirements

### Requirement: Every database has a healthcheck config that starts switched on

Every database SHALL have one healthcheck config. A new database SHALL get the default config when it is created: checks switched on, notifications switched on, a check every minute, three failed checks in a row before the database counts as unavailable, and attempts kept for seven days. A database that existed before the upgrade without a config SHALL have the default config after the upgrade. Reading a config SHALL change nothing.

#### Scenario: A new database

- **WHEN** a member creates a database and reads its healthcheck config
- **THEN** the config has checks switched on, notifications switched on, an interval of 1 minute, 3 failed attempts before unavailable and 7 days of retention

#### Scenario: A database without a config before the upgrade

- **WHEN** Databasus is upgraded on an instance where a database has no healthcheck config
- **THEN** after the upgrade the config of that database is the default config

### Requirement: A saved config is validated and audited

Saving a healthcheck config SHALL replace every editable value: whether checks are switched on, whether notifications are sent, the interval in minutes, the number of failed attempts before the database counts as unavailable and the number of days attempts are kept. The interval, the number of attempts and the number of days SHALL each be at least 1; a request that breaks this SHALL get `400` with `code` `invalid_request` and SHALL change nothing. A successful save SHALL answer with the config as saved and SHALL add an event to the workspace's audit log that names the database, in the same step as the save. A request without a valid session SHALL get `401` with `code` `unauthenticated`.

#### Scenario: Saving new values

- **WHEN** a member saves a config with an interval of 5 minutes, 2 failed attempts, 30 days of retention and notifications switched off
- **THEN** the response is `200` with those values, a later read returns them, and the workspace's audit log has an event naming the database

#### Scenario: A value below 1

- **WHEN** a member saves a config with an interval of 0 minutes
- **THEN** the response is `400` with `code` `invalid_request`, and the stored config is unchanged

#### Scenario: No session

- **WHEN** a request without a session reads or saves a healthcheck config, or lists attempts
- **THEN** the response is `401` with `code` `unauthenticated`

### Requirement: A database is checked no more often than its interval

Databasus SHALL check each database whose checks are switched on, about once a minute, when no attempt is recorded yet or its interval, less one second of slack, has passed since its last attempt. A database whose checks are switched off SHALL NOT be checked. A check SHALL connect to the database with its stored settings and record one attempt with the time of the check and the result `AVAILABLE` or `UNAVAILABLE`. When many databases are due at once, their checks MAY take longer than a minute, so a database MAY be checked later than its interval, but never earlier than one second before its interval has passed.

#### Scenario: Before the interval has passed

- **WHEN** a database with an interval of 5 minutes was checked 4 minutes ago, and Databasus looks for due checks
- **THEN** no attempt is added for that database

#### Scenario: After the interval has passed

- **WHEN** a database with an interval of 5 minutes was checked 5 minutes ago, and Databasus looks for due checks
- **THEN** one attempt is added for that database

#### Scenario: Checks switched off

- **WHEN** a database has its checks switched off, and Databasus looks for due checks
- **THEN** no attempt is added for that database

### Requirement: A database counts as unavailable after the configured number of failed checks in a row

When the status of a database is available or not set, and a check fails, the database SHALL become unavailable when, and only when, its last attempts, as many as the config's number of failed attempts and including this one, all failed. With fewer attempts recorded than that number, the status SHALL stay as it is. With the number set to 1, the first failed check SHALL make the database unavailable. When a database becomes unavailable, Databasus SHALL notify that it is unavailable, as the notification requirement describes. A failed check of a database that is already unavailable SHALL change nothing and SHALL NOT notify again.

#### Scenario: Fewer failures than the threshold

- **WHEN** an available database with a threshold of 3 fails its first and second checks
- **THEN** both attempts are recorded as unavailable, the database is still available, and no notification is sent

#### Scenario: The threshold is reached

- **WHEN** an available database with a threshold of 3 fails its third check in a row
- **THEN** the database is unavailable, and its notifiers get one notification that it is unavailable

#### Scenario: A success breaks the row

- **WHEN** an available database with a threshold of 3 fails a check, succeeds, then fails twice more
- **THEN** the database is still available

#### Scenario: Still down

- **WHEN** an unavailable database fails another check
- **THEN** it stays unavailable, and no notification is sent

### Requirement: A database counts as available after one successful check

When the status of a database is unavailable or not set, and a check succeeds, the database SHALL become available at once, and Databasus SHALL notify that it is online. A database has no status after its checks were switched off, so the first successful check after they are switched on again SHALL notify that the database is online, which confirms that Databasus reached it. A new database starts as available, so its first successful check SHALL change nothing and SHALL NOT notify. A successful check of a database that is already available SHALL change nothing and SHALL NOT notify.

#### Scenario: Back from unavailable

- **WHEN** an unavailable database passes a check
- **THEN** the database is available, and its notifiers get one notification that it is online

#### Scenario: First check of a new database

- **WHEN** a new database with a notifier passes its first check
- **THEN** the database is available, and the notifier gets no notification

#### Scenario: First check after switching the checks on again

- **WHEN** a member switches the checks of a database with a notifier off and on again, and the database passes its next check
- **THEN** the database is available, and the notifier gets one notification that it is online

#### Scenario: Still up

- **WHEN** an available database passes another check
- **THEN** it stays available, and no notification is sent

### Requirement: Status notifications follow the config

A notification about a change of status SHALL go to the notifiers attached to the database, with the category of a failed healthcheck for unavailable and of a successful healthcheck for online, and with the database's name in its heading and message. When the config has notifications switched off, the status SHALL change as the other requirements describe, and no notification SHALL be sent.

#### Scenario: Notifications switched off

- **WHEN** an available database with a threshold of 1 and notifications switched off fails a check
- **THEN** the database is unavailable, and its notifiers get no notification

### Requirement: Switching the checks off clears the status

Saving a config with the checks switched off SHALL clear the database's health status, so the database shows no health status. While the checks are off, no attempt SHALL be added. The attempts recorded before SHALL stay until their retention ends.

#### Scenario: Switching off an unavailable database

- **WHEN** a member switches off the checks of an unavailable database
- **THEN** the database shows no health status, and no attempt is added while the checks stay off

### Requirement: Attempts are kept for the configured number of days

After each check of a database, Databasus SHALL delete the attempts of that database that are older than the config's number of days, counted from the time of the check.

#### Scenario: An old attempt

- **WHEN** a database with 7 days of retention has an attempt 8 days old and is checked
- **THEN** the old attempt is no longer listed, and the new attempt is

### Requirement: A stop of Databasus leaves no half-recorded check

When Databasus is stopped while checks are running, it SHALL wait for them to end before it exits. A check that the stop interrupts SHALL record no attempt, SHALL NOT change the health status and SHALL NOT send a notification, because the interruption says nothing about the database.

#### Scenario: Stop during a check

- **WHEN** Databasus is stopped while a check waits for a database that does not answer
- **THEN** the check ends, no attempt is added, the health status is unchanged, and no notification is sent

### Requirement: A check of a database deleted meanwhile leaves no trace

When a database is deleted while its check is running, the deletion SHALL succeed, and the check SHALL record no attempt and send no notification.

#### Scenario: Deleted during a check

- **WHEN** a member deletes a database while its check waits for an answer, and the check then fails
- **THEN** the deletion answers `204`, and no attempt or notification of that database appears

### Requirement: Attempts are listed newest first from a given time

Listing the attempts of a database SHALL require the database and a start time, and SHALL return the attempts recorded at or after that time, newest first, each with its result and time. The list SHALL page with a cursor: a page holds at most the requested number of attempts, from 1 to 1440, or 1440 when the request names none, and the last page carries no cursor. A page size outside that range, a cursor that the list did not issue and a missing start time SHALL each get `400` with `code` `invalid_request`.

#### Scenario: Attempts of the last hour

- **WHEN** a database has attempts 2 hours, 30 minutes and 10 minutes old, and a viewer lists its attempts from one hour ago
- **THEN** the list holds the attempts 10 and 30 minutes old, in that order

#### Scenario: Several pages

- **WHEN** a database has 5 attempts, and a viewer lists them with a page size of 2 and follows the cursor
- **THEN** the pages hold 2, 2 and 1 attempts, newest first without gaps or repeats, and the last page has no cursor

#### Scenario: A new database

- **WHEN** a viewer lists the attempts of a database that was never checked
- **THEN** the list is empty

#### Scenario: A foreign cursor

- **WHEN** a viewer lists the attempts of a database with a cursor that the list did not issue
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: No start time

- **WHEN** a viewer lists the attempts of a database without a start time
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: A page size above the maximum

- **WHEN** a viewer lists the attempts of a database with a page size of 1441
- **THEN** the response is `400` with `code` `invalid_request`
