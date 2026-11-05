## Purpose

The audit log records who did what in a Databasus instance. This capability covers what an audit event keeps about its author, how long events are kept, and how events written before an upgrade read afterwards.

## ADDED Requirements

### Requirement: An audit event keeps its author as they were at the time of the action

An audit event with an author SHALL keep the author's user ID, email address and name as they were when the action happened. Every audit log view SHALL show the author from the event itself. A later change of the account's email address or name, a deactivation of the account or the removal of the account SHALL NOT change what an earlier event shows.

#### Scenario: Author changes their name

- **WHEN** a user performs an action that writes an audit event and then changes their name
- **THEN** the audit log shows that event with the name the user had when the action happened

#### Scenario: Author changes their email address

- **WHEN** a user performs an action that writes an audit event and then changes their email address
- **THEN** the audit log shows that event with the email address the user had when the action happened

#### Scenario: Event without an author

- **WHEN** an action that runs without a signed-in user writes an audit event
- **THEN** the audit log shows that event without an author

### Requirement: Audit events written before the upgrade keep their author

After an upgrade from a release that did not keep the author in the event, every existing audit event whose author account still exists SHALL show that account's email address and name as they were at the moment of the upgrade. An existing event that already had no author SHALL stay without one. No existing event SHALL be lost or change its message, time or workspace.

#### Scenario: Upgrade with existing events

- **WHEN** an instance with audit events written by existing accounts is upgraded
- **THEN** each of those events shows its author's email address and name
- **AND** its message, time and workspace are the same as before the upgrade

#### Scenario: Upgrade with an event that has no author

- **WHEN** an instance holds an audit event without an author before the upgrade, such as an event whose author account was removed earlier
- **THEN** after the upgrade that event is still listed, without an author, with its message, time and workspace unchanged

### Requirement: Audit events are kept for one year

The instance SHALL delete audit events older than one year. It SHALL check for such events at least once an hour, and SHALL keep every event that is one year old or younger.

#### Scenario: Event older than one year

- **WHEN** an audit event was written more than one year ago and the hourly cleanup runs
- **THEN** that event is no longer listed in any audit log view

#### Scenario: Event younger than one year

- **WHEN** an audit event was written less than one year ago and the hourly cleanup runs
- **THEN** that event is still listed
