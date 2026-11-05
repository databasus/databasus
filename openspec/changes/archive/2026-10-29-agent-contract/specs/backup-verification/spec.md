## ADDED Requirements

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

A heartbeat SHALL report the memory the agent may use in megabytes, as given to the agent. The server SHALL store that value and SHALL return it in megabytes in the list of verification agents. The agents list in the interface SHALL show the memory of any agent that reported more than zero, with the unit that fits the value, such as "512 MB" or "4 GB".

#### Scenario: An agent with less than one gigabyte of memory

- **WHEN** an agent started with 512 MB of memory sends a heartbeat
- **THEN** the list of verification agents returns that agent with `512` as its memory in megabytes, and the agents list shows "512 MB RAM" for it

#### Scenario: An agent with whole gigabytes of memory

- **WHEN** an agent started with 4096 MB of memory sends a heartbeat
- **THEN** the list of verification agents returns `4096` as its memory in megabytes, and the agents list shows "4 GB RAM" for it

### Requirement: An agent of the previous release keeps working until it updates

The agent operations SHALL keep the paths, request bodies and success responses that the agent of the previous release uses. A heartbeat that reports memory only in whole gigabytes, as the previous release does, SHALL be accepted and stored as that number multiplied by 1024 megabytes. A heartbeat that reports memory both in megabytes and in gigabytes SHALL be stored with the value in megabytes. A heartbeat that reports neither SHALL store zero, which reads as not reported.

Stored memory SHALL survive the upgrade: an agent whose memory was stored in gigabytes before the upgrade SHALL read after it as the same amount in megabytes.

#### Scenario: A heartbeat of the previous release

- **WHEN** an agent sends a heartbeat with 8 GB of memory and no value in megabytes
- **THEN** the heartbeat succeeds and the list of verification agents returns `8192` as its memory in megabytes

#### Scenario: A heartbeat with both units

- **WHEN** an agent sends a heartbeat with 512 MB and 0 GB of memory
- **THEN** the list of verification agents returns `512` as its memory in megabytes

#### Scenario: A heartbeat without memory

- **WHEN** an agent that reported 512 MB sends a heartbeat with no memory value in either unit
- **THEN** the heartbeat succeeds and the list of verification agents returns `0` as its memory in megabytes

#### Scenario: Memory stored before the upgrade

- **WHEN** the instance upgrades with an agent whose memory was stored as 2 GB
- **THEN** the list of verification agents returns `2048` as that agent's memory in megabytes

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
