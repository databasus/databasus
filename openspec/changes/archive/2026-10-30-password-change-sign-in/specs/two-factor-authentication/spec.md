## ADDED Requirements

### Requirement: A password change requires a complete new sign-in

After a successful password change in the profile, the interface SHALL clear its current session, display the ordinary sign-in screen and tell the user that the password changed and signing in again is required. It SHALL NOT submit the new password automatically or report a completed sign-in before authentication completes. This SHALL apply whether the instance requires a second factor or not.

A refused or failed password change SHALL leave the current session and profile open, display the failure, and SHALL NOT display password-change success.

#### Scenario: Password changed with a required second factor

- **WHEN** a signed-in user changes their password on an instance requiring a second factor
- **THEN** the interface displays the sign-in screen and the password-change message
- **AND** it starts no sign-in and sends no code until the user submits the ordinary sign-in form

#### Scenario: Completing sign-in with the new password

- **WHEN** the user signs in with the new password after changing it and a second factor is required
- **THEN** the ordinary emailed-code step is shown and the interface becomes authenticated only after that step succeeds

#### Scenario: Password changed without a second factor

- **WHEN** a signed-in user changes their password while a second factor is not required
- **THEN** the interface still displays the sign-in screen and requires a new sign-in

#### Scenario: Password change is refused

- **WHEN** a password change fails validation or the server refuses it
- **THEN** the profile remains open with its current session and displays the failure

### Requirement: A password change grants no authentication shortcut

The session issued before a successful password change SHALL no longer authorize requests. A subsequent password sign-in SHALL meet the same bot challenge and second-factor requirements as any other new sign-in. Neither the password-change response nor a pending sign-in SHALL authorize the interface.

Invalidation SHALL hold even when the password changes in the same second or at the exact same clock instant as its previous setting. An earlier pending sign-in SHALL NOT complete or be reused after replacement, even when its emailed code is correct. Every successful replacement SHALL invalidate earlier authentication state; a rejected or rolled-back replacement SHALL NOT do so.

#### Scenario: Earlier token is used again

- **WHEN** a client sends an authenticated request using a session issued before the password changed
- **THEN** the request is refused as unauthenticated

#### Scenario: Password changes within the same second

- **WHEN** the password is replaced in the same second as its previous setting and a client reuses an earlier token
- **THEN** that token is refused as unauthenticated and a completed new sign-in with the new password authorizes requests

#### Scenario: Multiple replacements at one clock instant

- **WHEN** a user completes a new sign-in and changes the password again without any clock advancement
- **THEN** every session issued before the latest replacement is refused and a completed sign-in with the latest password works

#### Scenario: Earlier emailed code after replacement

- **WHEN** the password changes at the same clock instant while a pending sign-in for the earlier password has a correct emailed code
- **THEN** that pending sign-in cannot issue a session and the user starts again with the new password

#### Scenario: Replacement does not commit

- **WHEN** a password replacement is refused or its transaction rolls back
- **THEN** the prior session and usable pending sign-in retain their validity

#### Scenario: Bot protection is enabled

- **WHEN** the user returns to sign-in after changing their password on an instance with bot protection enabled
- **THEN** the ordinary bot challenge is required before a new password sign-in is accepted
