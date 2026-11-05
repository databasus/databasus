# Backup Verification Specification

## Purpose

Defines the relationship between on-demand backup verification and the configuration that controls automatic verification runs.

## Requirements

### Requirement: Manual verification is independent of automatic scheduling

The system SHALL allow an authorized user to enqueue and run a manual backup verification while automatic verification is disabled.

#### Scenario: Enqueue manual verification while automatic verification is disabled

- **WHEN** an authorized user requests manual verification of an eligible backup whose automatic verification is disabled and no other manual verification for that database is pending or running
- **THEN** the system accepts the request and keeps the manual verification pending for an agent

#### Scenario: Disable automatic verification while a manual verification is pending

- **WHEN** automatic verification is disabled for a database with a pending manual verification
- **THEN** the manual verification remains pending

#### Scenario: Disable automatic verification while a manual verification is running

- **WHEN** automatic verification is disabled for a database with a running manual verification
- **THEN** the manual verification remains running and can reach its normal terminal status

### Requirement: Disabling automatic verification cancels only automatic runs

The system SHALL cancel non-terminal automatic verification runs when automatic verification is disabled without canceling manual verification runs for the same database.

#### Scenario: Disable automatic verification with a time-based run

- **WHEN** automatic verification is disabled for a database with a pending or running time-based verification
- **THEN** the system cancels that automatic verification

#### Scenario: Disable automatic verification with an after-backup run

- **WHEN** automatic verification is disabled for a database with a pending or running after-backup verification
- **THEN** the system cancels that automatic verification

#### Scenario: Manual and automatic runs coexist when scheduling is disabled

- **WHEN** automatic verification is disabled for a database with both manual and automatic non-terminal verifications
- **THEN** the system cancels the automatic verifications and leaves the manual verifications unchanged

### Requirement: A verification accepts only ignorable errors that say nothing about the backup

An ignorable error is an error that the PostgreSQL restore tool reported and skipped, and that still lets the restore count as successful. When the restore tool of a verification exits with an error, the verification agent SHALL count the restore as successful only if two conditions hold. First, the tool reports how many errors it skipped, and it listed exactly that many errors. Second, every listed error is an ignorable error of a verification: the `public` schema that already exists in the restore target, or an extension that the verification environment cannot create, together with the comment on that extension. The agent SHALL judge by the tool's whole error output, however long it is.

When the restore tool fails because the restore ran out of the job's disk budget, the agent SHALL report the failure without the tool's exit code, so the backend retries the verification instead of rejecting the backup, as before this change. Any other failure that is not made only of ignorable errors, a count that does not match the listed errors, or a missing count SHALL fail the verification with the tool's exit code.

A missing extension is an ignorable error of a verification because the verification environment does not carry the extensions of the source. A restore into the user's database does not treat it as ignorable, as the `database-tool-failure-output` spec states.

#### Scenario: Only extensions the environment lacks

- **WHEN** the restore tool of a verification exits with an error, reports 4 skipped errors and lists two extensions that are not available, each with the failed comment on it
- **THEN** the verification completes and collects the statistics of the restored database

#### Scenario: Only the existing public schema

- **WHEN** the restore tool of a verification exits with an error, reports 1 skipped error and lists only that the `public` schema already exists
- **THEN** the verification completes

#### Scenario: A missing extension that the data depends on

- **WHEN** the restore tool reports that an extension is not available, and then that a type from it and a table using that type do not exist
- **THEN** the verification fails with the tool's exit code

#### Scenario: The count does not match the listed errors

- **WHEN** the restore tool reports 3 skipped errors but lists only 2, both about an extension that is not available
- **THEN** the verification fails with the tool's exit code

#### Scenario: Ignorable errors that make the output long

- **WHEN** the restore tool lists more than 8 KiB of errors about extensions that are not available and reports their exact count
- **THEN** the verification completes

#### Scenario: A non-zero exit without a count

- **WHEN** the restore tool exits with an error and does not report how many errors it skipped
- **THEN** the verification fails with the tool's exit code

#### Scenario: The restore runs out of the job's disk budget

- **WHEN** the restore tool exits with an error that reports no space left on the device of the restore container
- **THEN** the agent reports the failure without the tool's exit code, and the backend retries the verification instead of rejecting the backup

### Requirement: Removing a backup removes its verifications

Whenever a logical backup is removed, whether a member deletes it, retention drops it, its database's backup config moves to another storage or its database is deleted, its verifications SHALL be removed in the same step, whatever their status, a running one included. When that removal is refused, the backup and all its verifications SHALL stay. Deleting a database SHALL also remove its verification config.

#### Scenario: A member deletes a verified backup

- **WHEN** a member deletes a logical backup that has a completed verification, a running one and a pending one
- **THEN** the response is `204`, and none of the three verifications is listed for the database any more

#### Scenario: Retention drops a verified backup

- **WHEN** a database's retention no longer keeps a backup that has a verification
- **THEN** after the next retention check the backup is not listed, and its verification is no longer listed for the database

#### Scenario: The storage of a verified backup changes

- **WHEN** a member saves a database's backup config with another storage while one of its backups has a verification
- **THEN** the save succeeds, the backup is not listed, and its verification is no longer listed for the database

#### Scenario: A database with verifications is deleted

- **WHEN** a member deletes a database that has a verification config and verified backups
- **THEN** the response is `204`, and no verification or verification config of the database remains

#### Scenario: A refused deletion keeps the verifications

- **WHEN** a member deletes a database while one of its logical backups is in progress, and another backup of it has a verification
- **THEN** the response is `409` with `code` `in_use`, the backup is still listed, the verification is still listed for the database, and the database's verification config is still returned

### Requirement: Only an instance admin manages verification agents

Creating, listing, rotating the token of and deleting verification agents SHALL be allowed only to an instance admin. Any other signed-in user SHALL get `403` with `code` `permission_denied`, a request without a session SHALL get `401` with `code` `unauthenticated`, and neither SHALL change anything. Creating an agent and rotating its token SHALL return the agent's token once, in that response only. No other response SHALL carry a token or its hash. After a rotation the previous token SHALL stop authenticating the agent. A deleted agent SHALL no longer be listed, its token SHALL stop authenticating it, and a verification it was running SHALL be handed back for another agent while attempts remain, as when an agent goes silent. Agents SHALL be listed newest first, one page at a time, and a cursor the list did not issue SHALL get `400` with `code` `invalid_request`.

#### Scenario: An admin creates an agent

- **WHEN** an instance admin creates an agent with a name
- **THEN** the response is `201` with the agent and its token, and the agent is listed without a token

#### Scenario: Paging the agents

- **WHEN** an admin lists three agents two per page
- **THEN** the first page holds the two newest and a cursor, and the second page holds the oldest and no cursor

#### Scenario: A request without a session

- **WHEN** a request without a session creates or lists agents
- **THEN** each response is `401` with `code` `unauthenticated`, and no agent is added

#### Scenario: A workspace owner manages agents

- **WHEN** a workspace owner who is not an instance admin creates, lists, rotates the token of or deletes an agent
- **THEN** each response is `403` with `code` `permission_denied`, the list an admin reads is unchanged, and the agent's heartbeat with its current token still succeeds

#### Scenario: A rotated token

- **WHEN** an admin rotates an agent's token and the agent then sends a heartbeat with the previous token and with the new one
- **THEN** the heartbeat with the previous token is refused with `401`, and the one with the new token succeeds

#### Scenario: A deleted agent

- **WHEN** an admin deletes an agent that is running a verification
- **THEN** the agent is no longer listed, its next heartbeat is refused with `401`, and after the scheduler's next pass the verification is pending again with one more attempt

#### Scenario: An unknown agent

- **WHEN** an admin rotates the token of or deletes an agent that does not exist or was deleted
- **THEN** the response is `404` with `code` `not_found`

### Requirement: Any signed-in user sees whether agents exist

Any signed-in user SHALL be able to read how many verification agents exist, so the interface can say whether a verification can run. The answer SHALL carry only the number, and nothing that identifies an agent.

#### Scenario: A member reads the number of agents

- **WHEN** a member who is not an instance admin reads the number of agents while two agents exist and a third was deleted
- **THEN** the answer is `200` with the number 2 and no agent's ID, name or token

### Requirement: A database has one verification config

Reading the verification config of a database SHALL return the saved config, or the default when none was saved: automatic verification off, run after each backup, a weekly interval schedule on Sunday at 04:00, and only the failure notification. Saving SHALL replace every editable field: whether automatic verification is on, whether it runs after each backup or on an interval schedule, the schedule, and which outcomes send a notification. A save with automatic verification on and the interval type SHALL need a valid schedule, and a save with an invalid schedule, schedule type or notification SHALL get `400` with `code` `invalid_request` and the field in `errors`, and SHALL change nothing.

#### Scenario: A database without a saved config

- **WHEN** a member reads the verification config of a database whose config was never saved
- **THEN** the answer is the default config

#### Scenario: Saving a config

- **WHEN** a member saves a config with automatic verification on, a weekly interval schedule and both notifications, and reads it again
- **THEN** the save answers `200` with the saved config, and the read returns the same values

#### Scenario: An invalid schedule

- **WHEN** a member saves a config with automatic verification on, the interval type and a cron schedule whose expression does not parse
- **THEN** the response is `400` with `code` `invalid_request` and the schedule in `errors`, and the config is unchanged

### Requirement: A manual verification starts only for a completed backup

Starting a manual verification of a logical backup SHALL answer `202` with the verification pending, without waiting for an agent. A backup that is not completed SHALL get `409` with `code` `backup_not_completed`. A second manual verification of a database while one is pending or running SHALL get `409` with `code` `already_running`. Neither refusal SHALL create a verification.

#### Scenario: Starting a verification

- **WHEN** a member starts a verification of a completed backup
- **THEN** the response is `202` with a pending manual verification of that backup, and reading it returns the same

#### Scenario: A backup that is not completed

- **WHEN** a member starts a verification of a failed backup
- **THEN** the response is `409` with `code` `backup_not_completed`, and the database's list of verifications is unchanged

#### Scenario: A manual verification is already pending

- **WHEN** a member starts a manual verification of a database that already has a pending manual one
- **THEN** the response is `409` with `code` `already_running`, and only the first verification is listed

### Requirement: A pending or running verification can be cancelled

Cancelling a pending or running verification SHALL answer `200` with the verification cancelled. When an agent runs it, the agent's next heartbeat that reports it as current SHALL list it among the verifications to abort. Cancelling a verification that is completed, failed or cancelled SHALL get `409` with `code` `not_running` and SHALL change nothing.

#### Scenario: Cancelling a running verification

- **WHEN** a member cancels a verification that an agent is running
- **THEN** the response is `200` with the verification cancelled, and the agent's next heartbeat that reports the verification as current lists it among the verifications to abort

#### Scenario: Cancelling a finished verification

- **WHEN** a member cancels a completed verification
- **THEN** the response is `409` with `code` `not_running`, and the verification is still completed

### Requirement: The verifications of a database are listed newest first

The verifications of a database SHALL be listed newest first, one page at a time, with a cursor that leads to the next page and is absent on the last one. A list entry SHALL carry the verification's status, trigger, attempts, times, durations and results, without the row counts of its tables. Reading one verification SHALL also return the row count of each table the agent counted. A cursor the list did not issue SHALL get `400` with `code` `invalid_request`.

#### Scenario: Paging the verifications

- **WHEN** a member lists the verifications of a database that has three, two per page
- **THEN** the first page holds the two newest and a cursor, and the second page holds the oldest and no cursor

#### Scenario: Table counts of one verification

- **WHEN** a member reads a completed verification after its agent reported two tables
- **THEN** the answer holds both tables with their row counts, and the list entry of the same verification holds none

#### Scenario: A foreign cursor

- **WHEN** a member lists the verifications of a database with a cursor the list did not issue
- **THEN** the response is `400` with `code` `invalid_request`

### Requirement: A verification agent authenticates with its own token

Every agent operation SHALL name the agent in its path and SHALL carry that agent's current token as a bearer token. The operation SHALL run only when the token belongs to the named agent and the agent is not deleted. A missing token, a token of another agent, a user's session token, a token replaced by a rotation, a deleted agent, an unknown agent and an agent ID that is not a UUID SHALL each get `401` with `code` `unauthenticated` and the same `detail`, so the response does not reveal which check failed, and SHALL change nothing. The server SHALL log each refusal with its reason and the client's address, and SHALL NOT log the token.

Agent requests SHALL be limited per agent: within one second the eleventh and later requests for the same agent ID SHALL get `429` with `code` `rate_limited`.

#### Scenario: Credentials that do not belong to the agent

- **WHEN** a heartbeat for an agent is sent without a token, with an invalid token, with an instance admin's session token and with the token of another agent
- **THEN** each response is `401` with `code` `unauthenticated` and the same `detail`, and the agent's last seen time does not change

#### Scenario: An unknown agent

- **WHEN** a heartbeat is sent with a token to the path of an agent ID that no agent has
- **THEN** the response is `401` with `code` `unauthenticated` and the same `detail`

#### Scenario: An agent ID that is not a UUID

- **WHEN** a heartbeat is sent to an agent path whose ID is not a UUID
- **THEN** the response is `401` with `code` `unauthenticated`

#### Scenario: Too many requests for one agent

- **WHEN** eleven heartbeats with valid credentials go to one agent within one second
- **THEN** the first ten succeed, and the eleventh gets `429` with `code` `rate_limited`

### Requirement: A verification agent reports its memory in megabytes

A heartbeat SHALL include the required nonnegative integer `maxRamMb` with the memory the agent may use in megabytes, as given to the agent. The server SHALL store that value and SHALL return it in megabytes in the list of verification agents. An explicit zero SHALL remain valid. The agents list in the interface SHALL show the memory of any agent that reported more than zero, with the unit that fits the value, such as "512 MB" or "4 GB". Stored memory SHALL survive upgrades without being reset or converted again.

A heartbeat without `maxRamMb`, including one that supplies memory only in whole gigabytes, SHALL get 400 with code `invalid_request`. The server SHALL neither infer memory from another unit nor replace a missing value with zero. Rejection SHALL leave the stored capacity and last-seen timestamp unchanged.

#### Scenario: An agent with less than one gigabyte of memory

- **WHEN** an agent started with 512 MB of memory sends a heartbeat
- **THEN** the list of verification agents returns that agent with `512` as its memory in megabytes, and the agents list shows "512 MB RAM" for it

#### Scenario: An agent with whole gigabytes of memory

- **WHEN** an agent started with 4096 MB of memory sends a heartbeat
- **THEN** the list of verification agents returns `4096` as its memory in megabytes, and the agents list shows "4 GB RAM" for it

#### Scenario: Memory is explicitly zero

- **WHEN** a valid agent sends a heartbeat with `maxRamMb` equal to zero
- **THEN** the heartbeat succeeds and stores zero megabytes

#### Scenario: Memory is omitted or reported only in gigabytes

- **WHEN** a valid agent sends a heartbeat without `maxRamMb`, with or without a gigabyte memory value
- **THEN** the response is 400 with code `invalid_request`
- **AND** the stored capacity and last-seen timestamp stay unchanged

#### Scenario: Persisted memory survives the upgrade

- **WHEN** the instance upgrades with an agent whose stored memory is 2048 megabytes
- **THEN** the list of verification agents still returns 2048 megabytes before the next heartbeat

### Requirement: New assignments require a fresh accepted heartbeat

For an authenticated, non-deleted agent with an otherwise valid claim request, the server SHALL assign new verification work only if it has previously accepted a heartbeat from that agent and that heartbeat is no more than five minutes old. A heartbeat exactly five minutes old SHALL remain eligible; one older than five minutes SHALL be expired. An agent without an accepted heartbeat or with an expired heartbeat SHALL receive an empty `204` response, even when pending work fits the capacity in its claim.

Withholding an assignment SHALL leave pending verification status, ownership and attempt count unchanged and SHALL NOT refresh the agent's last-seen time or capacity. Another eligible agent SHALL be able to claim that pending work. An agent SHALL regain eligibility when the server accepts a new heartbeat, using the same registration and current token.

This rule SHALL govern new assignments only. An agent SHALL remain listed and included in the registered-agent count while it exists, regardless of assignment eligibility. A user SHALL still be able to enqueue a verification while no agent is eligible. Existing ownership, cancellation and lifecycle rules SHALL continue to govern streams and reports for previously assigned work.

#### Scenario: A registered agent has never sent an accepted heartbeat

- **WHEN** an authenticated agent with no accepted heartbeat claims work and a pending verification fits its claimed capacity
- **THEN** the response is an empty `204`
- **AND** the verification stays pending with no assigned agent and the same attempt count
- **AND** the agent's last-seen time and stored capacity do not change

#### Scenario: A fresh accepted heartbeat allows an assignment

- **WHEN** an agent claims fitting pending work less than five minutes after an accepted heartbeat
- **THEN** it receives the assignment through the normal claim response

#### Scenario: The accepted heartbeat is exactly five minutes old

- **WHEN** an agent claims fitting pending work exactly five minutes after its last accepted heartbeat
- **THEN** it remains eligible and receives the assignment

#### Scenario: The accepted heartbeat has expired

- **WHEN** an agent claims fitting pending work more than five minutes after its last accepted heartbeat
- **THEN** the response is an empty `204`
- **AND** the verification stays pending with no assigned agent and the same attempt count
- **AND** repeated claims do not refresh the accepted heartbeat or consume attempts

#### Scenario: A healthy agent takes work withheld from an expired agent

- **WHEN** an expired agent receives no assignment and another agent with a fresh accepted heartbeat claims the same fitting pending verification
- **THEN** the healthy agent receives the assignment with no attempt consumed by the refused claim

#### Scenario: A recovered agent sends a valid heartbeat

- **WHEN** an agent without an accepted heartbeat or with an expired heartbeat sends a valid heartbeat and then claims fitting pending work
- **THEN** the heartbeat is accepted and the claim receives an assignment without new registration or token rotation

### Requirement: Rejected heartbeats do not establish or extend assignment eligibility

A rejected heartbeat SHALL NOT establish or refresh the last accepted heartbeat used for assignment eligibility. Its rejection SHALL preserve the previously accepted timestamp and stored capacity. If there is no previously accepted heartbeat, the agent SHALL remain ineligible. If one exists, its original five-minute window SHALL continue to determine eligibility: a rejected request SHALL neither extend that window nor revoke it early.

#### Scenario: Only an incompatible heartbeat has been sent

- **WHEN** the server rejects an agent's heartbeat that supplies memory only in gigabytes and the agent has no previously accepted heartbeat
- **THEN** a subsequent otherwise valid claim receives an empty `204` and fitting pending work remains unassigned

#### Scenario: Rejected heartbeats do not renew an earlier accepted heartbeat

- **WHEN** an agent sends an accepted heartbeat, later sends rejected heartbeats, and claims fitting pending work more than five minutes after the accepted heartbeat
- **THEN** the response is an empty `204`, even if a rejected heartbeat was sent recently
- **AND** the stored capacity and last-seen timestamp remain those of the accepted heartbeat

#### Scenario: A rejected request does not revoke a fresh accepted heartbeat

- **WHEN** an agent sends an accepted heartbeat, later sends a rejected heartbeat and claims fitting pending work within the original five-minute window
- **THEN** the agent remains eligible based on the accepted heartbeat
- **AND** the rejected heartbeat changes neither stored capacity nor the accepted timestamp

### Requirement: An agent's capacity and report are checked before they are used

A heartbeat or a claim with a negative capacity value SHALL get `400` with `code` `invalid_request`. A refused heartbeat SHALL leave the stored capacity unchanged, and a refused claim SHALL assign no verification. A report with a status other than completed or failed, or with a table statistic that lacks its schema name or its table name, SHALL get `400` with `code` `invalid_request` and SHALL change nothing.

#### Scenario: A heartbeat with a negative capacity

- **WHEN** an agent sends a heartbeat with a negative memory value
- **THEN** the response is `400` with `code` `invalid_request`, and the agent's stored capacity does not change

#### Scenario: A claim with a negative capacity

- **WHEN** an agent claims a verification with a negative disk capacity while a pending verification would fit a valid capacity
- **THEN** the response is `400` with `code` `invalid_request`, and the verification is still pending

#### Scenario: A report with an unknown status

- **WHEN** an agent reports a verification it runs with a status that is neither completed nor failed
- **THEN** the response is `400` with `code` `invalid_request`, and the verification is still running

#### Scenario: A report with an unnamed table

- **WHEN** an agent reports completion of a verification it runs with a table statistic whose table name is empty
- **THEN** the response is `400` with `code` `invalid_request`, and the verification is still running

### Requirement: An agent is told a verification is gone only when it no longer owns it

The backup stream and the report of a verification SHALL answer `410` with `code` `verification_gone` when the verification does not exist, is no longer running, is assigned to another agent, or its backup has been deleted. The agent SHALL then drop the verification without retrying. When the server fails for a reason the agent did not cause, such as an unreadable stored backup file, the operation SHALL answer `500` with `code` `internal`, and the agent SHALL retry it as it retries any other server failure. An agent operation SHALL answer no error code other than `invalid_request`, `unauthenticated`, `rate_limited`, `verification_gone` and `internal`.

#### Scenario: A report after the verification was cancelled

- **WHEN** an agent reports completion of a verification that a user cancelled after the agent claimed it
- **THEN** the response is `410` with `code` `verification_gone`, and the verification stays cancelled

#### Scenario: A report for an unknown verification

- **WHEN** an agent reports completion of a verification ID that does not exist
- **THEN** the response is `410` with `code` `verification_gone`

#### Scenario: The backup stream after the verification was cancelled

- **WHEN** an agent requests the backup stream of a verification that a user cancelled after the agent claimed it
- **THEN** the response is `410` with `code` `verification_gone`, and the verification stays cancelled

#### Scenario: A verification reclaimed by another agent

- **WHEN** an agent claims a verification, goes silent until the verification is handed to a second agent that claims it, and then requests its backup stream and reports it
- **THEN** both responses are `410` with `code` `verification_gone`, and the verification is still running and assigned to the second agent

#### Scenario: The stored backup cannot be read

- **WHEN** an agent requests the backup stream of a verification it claimed, and the backup's stored file is missing
- **THEN** the response is `500` with `code` `internal`, and the verification is still running and assigned to the agent

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
