## ADDED Requirements

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
