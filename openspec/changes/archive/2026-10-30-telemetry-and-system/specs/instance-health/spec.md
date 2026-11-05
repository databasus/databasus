## Purpose

Defines the public monitoring response for an instance, including its local dependencies, stale verification agents, and the progress and outcomes of periodic work.

## ADDED Requirements

### Requirement: Instance health is public and reports all checks

`GET /api/v1/system/health` SHALL require no session or token and SHALL include `Access-Control-Allow-Origin: *`. It SHALL return 200 with JSON diagnostics when all health checks pass. It SHALL return 503 with an `application/problem+json` response whose code is `instance_unhealthy` when a health check fails. Both responses SHALL include the results of every dependency check and every registered periodic task, even when several checks fail together. The diagnostics SHALL use stable check identifiers and machine-readable outcomes rather than arbitrary internal error messages.

#### Scenario: A monitor checks a healthy instance without credentials

- **WHEN** an unauthenticated client from another origin checks an instance whose health checks pass
- **THEN** the response is 200, permits that origin through wildcard CORS and includes every check and task

#### Scenario: Several dependencies fail together

- **WHEN** more than one dependency or task fails its health check
- **THEN** health answers 503 with code `instance_unhealthy` and identifies every failed check while retaining the other results
- **AND** the response still includes wildcard CORS

### Requirement: Local dependencies and stale agents determine health

Instance health SHALL fail when the data volume's usage cannot be read, its used space is at least 95 percent of its total space, the internal database cannot answer a connectivity check, a required client tool bundle is unusable, or active verification agents have no contact within the allowed five minutes. For an agent that never sent a heartbeat, age SHALL be measured from its creation. Deleted agents SHALL be excluded. Failure to query agent freshness SHALL itself fail that check. A client tool bundle marked nonessential SHALL NOT by itself fail instance health.

#### Scenario: Disk usage reaches the threshold

- **WHEN** data-volume usage reaches 95 percent
- **THEN** health answers 503 and the disk check identifies the threshold failure

#### Scenario: A local dependency is unavailable

- **WHEN** disk inspection fails, the internal database cannot answer, or a required client tool is missing or unusable
- **THEN** health answers 503 and identifies the corresponding check without disclosing its internal error

#### Scenario: An agent becomes stale

- **WHEN** an active agent's last contact, or creation when it never contacted the server, is more than five minutes old
- **THEN** health answers 503 and reports the number of stale agents
- **AND** deleting that agent removes its contribution to the check

#### Scenario: A new agent has not sent a heartbeat

- **WHEN** an agent was created less than five minutes ago and has never sent a heartbeat
- **THEN** it does not make instance health fail

### Requirement: Periodic work exposes progress independently of its outcome

The diagnostics SHALL list every registered periodic task in stable name order, including tasks that have no work to do. Each entry SHALL expose its interval, permitted pass duration, current running state, latest start time when present, latest completion time when present, and the outcome of the last completed pass when present. Starting another pass SHALL NOT erase the previous completion or outcome. A pass that fails or panics SHALL still record its completion and permit future passes. Reading task diagnostics SHALL NOT wait for a running pass to finish.

#### Scenario: A task is still running

- **WHEN** a monitor checks health while a task has not returned
- **THEN** health returns without waiting for that task and identifies its running state and start time
- **AND** any previous completed result remains visible

#### Scenario: A reporting attempt fails

- **WHEN** the telemetry receiver is unavailable but reporting attempts finish and no health check otherwise fails
- **THEN** health remains 200 and shows the failed task outcome

#### Scenario: A task panics and later recovers

- **WHEN** one pass panics and a later pass succeeds
- **THEN** the first pass is recorded as completed with a panic outcome and the later pass replaces it with a success outcome

### Requirement: A task that stops completing eventually fails health

Every registered task SHALL have a positive permitted pass duration that accounts for its legitimate workload. Before its first completion, the task SHALL have until activation time plus that duration to complete. After a completion, its next deadline SHALL be that completion time plus its interval plus its permitted pass duration. A task with a running pass SHALL also fail health when that pass exceeds its permitted duration. Reaching a deadline without the required completion SHALL cause 503 and identify the task. Task health SHALL be based on completed work, not repeated updates of a start time, and a new timely completion SHALL restore freshness regardless of success or failure. These monitoring deadlines SHALL NOT themselves cancel the task.

#### Scenario: A task never completes its first pass

- **WHEN** the first-completion deadline is reached without a completed pass
- **THEN** health answers 503 and names that task

#### Scenario: A task stops after a completed pass

- **WHEN** the next-completion deadline is reached without another completion
- **THEN** health answers 503 even if the last completed pass succeeded

#### Scenario: A legitimate pass lasts longer than its interval

- **WHEN** a pass exceeds its scheduling interval but remains within its permitted duration and completion deadline
- **THEN** that task does not make health fail

#### Scenario: A completed failed pass restores freshness

- **WHEN** a stale task completes with an error and other checks pass
- **THEN** health returns to 200 while retaining the failed outcome in its task diagnostics

### Requirement: Public health diagnostics do not disclose sensitive internals

The health response SHALL NOT contain raw errors, stack traces, SQL, filesystem paths, connection settings, credentials, agent names or tokens. It SHALL identify failed checks and jobs through stable codes and task names, and stale agents through a count. Detailed failures SHALL be available in the application logs with the corresponding request or task identifiers.

#### Scenario: A failed task returns an error containing a secret

- **WHEN** a completed task's error contains a connection address or a credential
- **THEN** public diagnostics expose its failure outcome without the error text or secret

#### Scenario: A stale agent has a sensitive name

- **WHEN** an agent with a sensitive name becomes stale
- **THEN** health reports a stale-agent count without that name
