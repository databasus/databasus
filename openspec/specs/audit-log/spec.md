# Audit Log Specification

## Purpose

The audit log records who did what in a Databasus instance. This capability covers what an audit event keeps about its author, how long events are kept, how events written before an upgrade read afterwards, and who may read which audit log view and how the views page.

## Requirements

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

### Requirement: Each audit log view has its own readers

The instance SHALL offer three audit log views, and each SHALL be readable only by its readers:

- the global view, with the events of the whole instance, SHALL be readable only by instance admins;
- a user's view, with the events that user authored, SHALL be readable by that user and by instance admins;
- a workspace's view, with the events of that workspace, SHALL be readable by its members of every role and by instance admins.

A workspace's view narrowed to one author SHALL follow the rule of the workspace's view and list only that author's events in the workspace. A user who is not a reader SHALL get `403` with `code` `permission_denied` for the global view and for another user's view. A user who is not a member of a workspace SHALL get `404` with `code` `not_found` for its view, the same answer as for a workspace that does not exist.

#### Scenario: User who is not an instance admin reads the global view

- **WHEN** a signed-in user who is not an instance admin reads the global audit log
- **THEN** the response is `403` with `code` `permission_denied`, and no event is returned

#### Scenario: User reads their own view

- **WHEN** a user who is not an instance admin reads the audit log of their own actions
- **THEN** the response lists the events that user authored and no event of another author

#### Scenario: User reads another user's view

- **WHEN** a user who is not an instance admin reads the audit log of another user
- **THEN** the response is `403` with `code` `permission_denied`

#### Scenario: Viewer reads the workspace's view

- **WHEN** a viewer of a workspace reads its audit log
- **THEN** the response lists the events of that workspace and no event of another workspace

#### Scenario: Instance admin reads the global view

- **WHEN** an instance admin reads the global audit log
- **THEN** the response lists the events of every workspace and the events outside any workspace

#### Scenario: Instance admin reads another user's view

- **WHEN** an instance admin reads the audit log of another user
- **THEN** the response lists the events that user authored and no event of another author

#### Scenario: Instance admin reads the view of a workspace they are not a member of

- **WHEN** an instance admin who is not a member of a workspace reads its audit log
- **THEN** the response lists the events of that workspace and no event of another workspace

#### Scenario: Member narrows the workspace's view to one author

- **WHEN** a viewer of a workspace reads its audit log narrowed to another member as the author
- **THEN** the response lists that member's events in the workspace and no other event

#### Scenario: Outsider reads a workspace's view

- **WHEN** a user who is not a member of a workspace reads its audit log
- **THEN** the response is `404` with `code` `not_found`

### Requirement: Audit log views list the newest events first and page without gaps

Every audit log view SHALL list events newest first and SHALL return them in pages. A page SHALL say whether more events follow and how to ask for them, and SHALL NOT report a total number of events. A client that follows the pages from the first to the last SHALL receive every event that existed when it read the first page exactly once, even when new events are written while it pages.

#### Scenario: Following the pages to the end

- **WHEN** a view holds more events than the requested page size and a client follows the pages to the last one
- **THEN** every event is listed exactly once, newest first, and the last page says no more events follow

#### Scenario: New event while paging

- **WHEN** a client has read the first page of a view, a new event is then written, and the client reads the next page
- **THEN** the next page starts with the event that followed the last event of the first page, and no event of the first page appears again

### Requirement: An audit event shows its workspace

An audit event that belongs to a workspace SHALL show the workspace's ID and its current name in every view. An event of a workspace that has since been deleted SHALL show the workspace's ID without a name. An event that belongs to no workspace SHALL show no workspace.

#### Scenario: Workspace renamed after the event

- **WHEN** an event is written in a workspace and the workspace is renamed afterwards
- **THEN** the global view shows that event with the workspace's new name

#### Scenario: Workspace deleted after the event

- **WHEN** an event is written in a workspace and the workspace is deleted afterwards
- **THEN** the global view still lists the event, with the workspace's ID and without a name

#### Scenario: Event outside any workspace

- **WHEN** an instance admin changes the instance settings, which writes an event that belongs to no workspace
- **THEN** the global view lists that event without a workspace

### Requirement: A sent password reset code writes an audit event

When the instance mail server accepts a password reset code for an account, the audit log SHALL gain an event that belongs to no workspace and names the account's email address. The event SHALL have no author, because anyone can request a code for any address. The event SHALL NOT contain the code. A request that sends no code SHALL write no event.

A failure to write this event SHALL be logged and SHALL NOT change the answer to the request, so the answer keeps telling nothing about whether the address has an account.

#### Scenario: A code is sent

- **WHEN** someone requests a password reset for an active user with a deliverable address and the mail server accepts the message
- **THEN** an instance admin reading the global audit log sees an event without an author and without a workspace that names that address and does not contain the code

#### Scenario: The address has no account

- **WHEN** someone requests a password reset for an address without a user
- **THEN** the answer is the same `204` as for an existing user and no event is written

#### Scenario: The mail server refuses the message

- **WHEN** a code is issued for a user and the mail server refuses the message
- **THEN** the answer is the same `204` as for a sent code and no event is written

#### Scenario: The event cannot be written

- **WHEN** a code is sent and the audit event cannot be written
- **THEN** the answer is the same `204` as for an address without a user, and the failure is logged
