## MODIFIED Requirements

### Requirement: A code is sent only after the password is verified

The instance SHALL verify the password and admit the account to password sign-in before generating or sending any code. A sign-in refusal for an unknown address, an incorrect password, incomplete registration, an account without a password or verified deactivation SHALL send no mail and SHALL create no new pending sign-in. A refused password attempt SHALL NOT alter or invalidate an existing pending sign-in; its expiry, reuse and verification SHALL remain governed by the existing pending sign-in requirements. The refusal SHALL give the same status, specific error code and guidance as it does with the second factor off, so the answer does not disclose whether the second factor is on. Distinguishing the existence of the submitted address and the other specific refusal causes is intentional.

#### Scenario: Wrong password

- **WHEN** someone submits a known address with an incorrect password
- **THEN** no mail is sent, no new pending sign-in is created, and the response identifies an incorrect password just as it does with the second factor off

#### Scenario: Wrong password after a pending sign-in was issued

- **WHEN** an account already has a live pending sign-in from a successful password step and a subsequent password attempt fails
- **THEN** the failed attempt creates no new pending sign-in, sends no mail and leaves the existing pending sign-in unchanged
- **AND** a subsequent correct password step reuses that pending sign-in, and its previously issued code remains usable under the existing expiry and verification rules

#### Scenario: Unknown address

- **WHEN** someone submits an address that belongs to no account
- **THEN** no mail is sent, no new pending sign-in is created, and the response identifies an unknown address just as it does with the second factor off

#### Scenario: Incomplete registration or missing password

- **WHEN** someone submits an invited account's address or an account without a password
- **THEN** no mail is sent, no new pending sign-in is created, and the response gives the same respective cause and guidance as it does with the second factor off

#### Scenario: Correct password for an inactive account

- **WHEN** someone submits an inactive account's correct password while the second factor is required
- **THEN** the response identifies deactivation without sending mail or creating a new pending sign-in
