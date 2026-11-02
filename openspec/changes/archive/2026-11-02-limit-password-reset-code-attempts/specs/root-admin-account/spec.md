## ADDED Requirements

### Requirement: A password reset code accepts five attempts

A password reset code SHALL accept at most five attempts to complete a reset. After five attempts it SHALL stop working, and the correct value SHALL get the same refusal as a wrong code. The count SHALL survive a failed attempt and a server restart, and simultaneous attempts SHALL NOT check the code more than five times in total. A new code SHALL start with all five attempts.

#### Scenario: Correct code after five wrong attempts

- **WHEN** somebody enters a wrong code five times for an address and then enters the correct code
- **THEN** the reset is refused as an invalid reset code
- **AND** the password is unchanged

#### Scenario: Correct code after four wrong attempts

- **WHEN** somebody enters a wrong code four times for an address and then enters the correct code
- **THEN** the password is replaced

#### Scenario: A failed attempt stays counted

- **WHEN** somebody enters a wrong code and the request is refused
- **THEN** the stored count for that code has grown by one

#### Scenario: Simultaneous wrong attempts

- **WHEN** more than five wrong attempts for one code arrive at the same time
- **THEN** the correct code is refused afterwards
- **AND** the stored count for that code does not exceed five

#### Scenario: A new code after an exhausted one

- **WHEN** a code has been exhausted, the user requests a new one, and somebody enters a wrong code four times and then the new code
- **THEN** the new code replaces the password

#### Scenario: Simultaneous correct attempts

- **WHEN** two requests with the same correct code and different new passwords arrive at the same time
- **THEN** exactly one is answered as a completed reset and the other is refused as an invalid reset code
- **AND** the account signs in with the password of the accepted request

### Requirement: Only the newest password reset code completes a reset

Only the newest code issued for a user SHALL be able to complete a reset. An older code SHALL be refused from the moment a newer one exists, and SHALL stay refused after the newest code is exhausted or used. An older code SHALL NOT replace the password after a completed reset.

#### Scenario: Older code while the newest one is usable

- **WHEN** a user requests two codes and somebody enters the first
- **THEN** the reset is refused as an invalid reset code

#### Scenario: Older code after the newest one is exhausted

- **WHEN** a user requests two codes, the second is exhausted, and somebody enters the first
- **THEN** the reset is refused as an invalid reset code

#### Scenario: Older code after a completed reset

- **WHEN** a user requests two codes, completes a reset with the second, and somebody enters the first
- **THEN** the reset is refused as an invalid reset code
- **AND** the password stays the one set by the completed reset

### Requirement: The status, error code and detail of an answer to an entered password reset code disclose nothing about the accounts an instance holds

When the new password is long enough, a wrong, used, exhausted, expired or older code and an address without an active account SHALL all get the same status, error code and detail, so a caller learns nothing about the accounts from them.

#### Scenario: Unknown address, inactive account and wrong code

- **WHEN** somebody enters a code for an address without an account, a code for a deactivated account's address and a wrong code for an active account's address
- **THEN** all three answers have the same status, error code and detail

### Requirement: The new password is processed only after the reset code is verified

The server SHALL NOT process the new password beyond checking its minimum length until the code is verified, so that a caller without a valid code cannot make the server hash a password. A password shorter than the minimum SHALL be refused naming the password field, whatever the code, and that refusal SHALL NOT spend an attempt of the code.

#### Scenario: Wrong code with a password the server cannot process

- **WHEN** somebody enters a wrong code together with a new password longer than the server can hash
- **THEN** the reset is refused as an invalid reset code

#### Scenario: Wrong code with a short password

- **WHEN** somebody enters a wrong code together with a new password shorter than the minimum
- **THEN** the request is refused as invalid, naming the new password field
- **AND** the stored count for that code does not grow
