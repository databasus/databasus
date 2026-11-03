## ADDED Requirements

### Requirement: A released 3.64.0 verification agent recovers through background self-update

Upgrade coverage SHALL run the actual verification agent distributed with release 3.64.0 against the current server. It SHALL establish that the old heartbeat is rejected before allowing the background self-update to finish. After the agent downloads and restarts into the current server's agent version, the server SHALL accept its heartbeat and the agent SHALL complete verification of a real backup, including reporting statistics that match the backup's known contents. The agent SHALL retain its registration, token and resource settings across the update.

This coverage SHALL remain anchored to release 3.64.0 when the newest release used by the general server upgrade test changes. The existing checks that verification records survive server upgrade SHALL complete before an agent starts consuming their pending work.

#### Scenario: Background update after rejection of the old heartbeat

- **WHEN** the released 3.64.0 agent runs against the current server and its heartbeat is rejected because it does not report memory in megabytes
- **THEN** its independent background update downloads the current agent and restarts it
- **AND** the current server accepts a heartbeat from the restarted agent using the original registration and resource settings
- **AND** a verification of a real backup completes and reports the known tables and row counts

#### Scenario: The newest server release changes

- **WHEN** the general upgrade test starts using a release newer than 3.64.0
- **THEN** the released-agent regression still starts with the actual 3.64.0 binary

### Requirement: Failed agent replacement has a tested manual recovery path

Upgrade coverage SHALL exercise a released 3.64.0 agent that can run and use its working files but cannot write to the directory holding its executable. It SHALL observe a failed background update and a rejected heartbeat while the executable remains version 3.64.0. It SHALL then stop the agent, replace its executable with the current binary using an identity permitted to do so, and restart it using the existing configuration without resupplying initial credentials or resource flags. A missing saved configuration SHALL cause restart to fail rather than recreating settings from the initial setup. After that recovery, a heartbeat SHALL be accepted and verification of a real backup SHALL complete. This scenario SHALL NOT require assignments to be issued to an agent whose heartbeat is rejected.

The repository documentation SHALL explain the permissions needed for self-update, how to recognize failure and how to perform this manual recovery. Instructions SHALL cover both the agent's own daemon and an agent managed by a service or container, preserve its credentials and settings, and direct the operator to check server contact and a completed verification after restart.

#### Scenario: The executable directory is not writable

- **WHEN** a running released 3.64.0 agent encounters a current server but cannot write its replacement executable
- **THEN** the background update reports a failure, the executable remains version 3.64.0 and its old heartbeat remains rejected
- **AND** replacing the executable and restarting with the saved configuration restores accepted heartbeats and successful backup verification

#### Scenario: An operator follows the recovery instructions

- **WHEN** an operator reads the repository recovery instructions after an agent fails to update
- **THEN** the instructions explain how to stop the applicable process or service, download the correct architecture from the connected server to a temporary file, install the replacement, and restart using the existing settings
- **AND** the instructions explain how to confirm server contact and a completed backup verification without using or exposing a new token
