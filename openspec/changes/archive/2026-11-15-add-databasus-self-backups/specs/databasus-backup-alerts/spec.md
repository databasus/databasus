## Purpose

Warns the instance owner when Databasus backups fail, become incomplete or stale, or are disabled by removing all destinations, using the existing system mail service and the in-app incident state.

## ADDED Requirements

### Requirement: Protection health includes failures and missing execution

Databasus SHALL track completed outcomes and the age of the last successful snapshot per current destination. A dump failure or failure to deliver any copy SHALL report failure; success in only some destinations SHALL report partial protection. Each destination SHALL retain the earliest scheduled occurrence not yet covered by a successful copy whose snapshot time is at or after that occurrence. Its overdue deadline SHALL be that retained due time plus 15 minutes, including when no job ran or a job is stuck. Later scheduled occurrences, polls, retries and schedule edits SHALL NOT advance or erase that outstanding deadline. A successful delivery SHALL cover every occurrence up to its snapshot time; any later already-due occurrence SHALL become the next outstanding one. A pending initial or resumed backup SHALL not claim protection. Schedule changes SHALL affect future occurrences without clearing outstanding ones or resetting historical successful-copy ages.

#### Scenario: Scheduler never starts a due backup

- **WHEN** a configured scheduled occurrence has no successful copy after its 15-minute grace period
- **THEN** the instance reports an overdue-backup incident without requiring a job failure event

#### Scenario: Only one destination succeeds

- **WHEN** a backup completes with one successful destination out of two
- **THEN** the incident reports partial protection and identifies the unsuccessful destination without claiming all copies failed

#### Scenario: Five-minute schedule never starts a backup

- **WHEN** a five-minute schedule first becomes due at 12:00 and no successful copy is produced for that destination
- **THEN** its overdue deadline remains 12:15 despite the 12:05, 12:10 and 12:15 occurrences
- **AND** it reports overdue at 12:15 rather than granting a fresh grace period

#### Scenario: A fresh copy covers several outstanding occurrences

- **WHEN** that destination accepts a copy whose snapshot time is 12:12
- **THEN** the 12:00, 12:05 and 12:10 occurrences are covered and a due 12:15 occurrence has deadline 12:30
- **AND** late delivery of a snapshot taken before 12:00 cannot clear the 12:00 deadline

### Requirement: Removing the last destination immediately reports stopped protection

A committed configuration change from at least one selected destination to none SHALL immediately create a stopped-protection incident. It SHALL bypass initial onboarding eligibility, setup deferral and the normal overdue grace period. A failed or rolled-back configuration save SHALL create no such incident. An initially unconfigured installation SHALL not send stopped-protection email. Re-adding a destination SHALL request an immediate backup and keep a resuming incident until at least one new copy succeeds; failures of remaining selected destinations SHALL continue to be reported independently.

#### Scenario: Final selected destination is removed

- **WHEN** root admin successfully removes the last selected storage
- **THEN** stopped protection becomes visible immediately and an email is eligible for delivery when mail prerequisites exist

#### Scenario: Final removal cannot be saved

- **WHEN** saving the removal of the last storage fails
- **THEN** the previous configuration remains active and no stopped-protection notification is created

### Requirement: System mail supplements the persistent in-app incident

With system SMTP configured and a usable address on the root-admin account, each new backup incident SHALL request a system email to that account. Without either prerequisite, it SHALL remain visible in-app and SHALL NOT be reported as emailed. No workspace notifier or recipient selection SHALL be required. Emails SHALL carry the condition, last successful snapshot time when known and safe destination identifiers, but no credentials, key, signed links or raw provider output. SMTP acceptance SHALL be recorded separately from backup success. Mail failure SHALL not change a successful archive result or suppress the in-app warning.

#### Scenario: SMTP is absent

- **WHEN** a backup incident starts without system SMTP
- **THEN** root admin sees the incident and missing-SMTP guidance, and no email is recorded as sent

#### Scenario: Mail server rejects delivery

- **WHEN** the mail server rejects an incident email
- **THEN** the backup incident remains visible with a mail-delivery failure and successful storage copies remain successful

### Requirement: Incident delivery survives restarts without flooding the owner

The first email SHALL be eligible immediately after incident creation. Repeated failures, retries, status polling and restarts SHALL update the existing incident rather than request an email for every event. Continuing incidents SHALL request at most one successfully accepted reminder per 24 hours. Failed mail attempts SHALL retry with bounded backoff and a recorded next attempt rather than on every poll. A newly eligible SMTP configuration or root-admin address SHALL allow an unresolved unsent incident to be delivered. Recovery SHALL clear the resolved condition and stale pending mail; a later genuine recurrence SHALL be eligible for a new initial email. At-least-once delivery around an interrupted SMTP acknowledgment SHALL be documented rather than claimed as exactly once.

#### Scenario: Storage retries and application restarts

- **WHEN** repeated delivery failures and a restart occur during one already emailed incident
- **THEN** no additional reminder is accepted before the 24-hour reminder window

#### Scenario: SMTP becomes available during an incident

- **WHEN** an unresolved incident has never been emailed and system SMTP becomes configured
- **THEN** its initial email becomes eligible without requiring another backup failure
