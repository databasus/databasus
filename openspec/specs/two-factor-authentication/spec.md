# Two-Factor Authentication Specification

## Purpose

Lets the owner of an instance require a second, emailed factor before a password sign-in succeeds, with the preconditions that keep the switch from locking everyone out and a console path back when mail delivery stops working.

## Requirements

### Requirement: An administrator controls the second factor for the whole instance

The instance SHALL offer a single global setting that requires a second factor for password sign-in. It SHALL be off until an administrator turns it on, SHALL apply to every account on the instance once on, and SHALL be changeable only by an administrator.

Turning the setting on SHALL NOT end sessions that already exist. An access token issued before the change SHALL keep working until it expires or a password change invalidates it.

#### Scenario: Default state

- **WHEN** an instance is deployed and claimed for the first time
- **THEN** the second factor is off and sign-in works with an address and a password alone

#### Scenario: A non-administrator tries to change it

- **WHEN** a member submits a settings update that turns the second factor on
- **THEN** the update is refused and the setting is unchanged

#### Scenario: A session that predates the switch

- **WHEN** an administrator turns the second factor on while another user is already signed in
- **THEN** that user's session keeps working

### Requirement: The second factor cannot be turned on unless the instance can deliver codes

Turning the setting on SHALL be refused unless both conditions hold: the instance has a configured mail server, and every active administrator holds a syntactically valid email address. The refusal SHALL name which condition failed and, when an administrator address is the problem, which account it is.

The interface offering the setting SHALL explain the mail server requirement and link to the published documentation for configuring it. It SHALL decide whether the mail server is configured from the instance's own answer, so that the interface and the refusal never disagree about the same instance.

#### Scenario: No mail server

- **WHEN** an administrator turns the setting on while the instance has no mail server configured
- **THEN** the change is refused with a message naming the missing mail server
- **AND** the interface offers a link to the documentation for configuring it

#### Scenario: An administrator has no usable address

- **WHEN** an administrator turns the setting on while another active administrator still holds a value that is not an email address, such as the login `admin` an instance created before the administrator account was reworked
- **THEN** the change is refused, the message names that account, and the setting stays off

#### Scenario: Both conditions satisfied

- **WHEN** an administrator turns the setting on with a mail server configured and every active administrator holding a real address
- **THEN** the change is accepted and takes effect for the next sign-in

#### Scenario: The interface and the instance agree

- **WHEN** an administrator opens the settings screen
- **THEN** whether the setting is offered or explained as unavailable follows the same judgement about the mail server that a refusal would apply

#### Scenario: Switching off after the mail server is gone

- **WHEN** the second factor is on and the instance has since lost its mail server
- **THEN** an administrator who is still signed in can switch the setting off from the settings screen, because only turning it on is refused

### Requirement: Password sign-in asks for an emailed code when the second factor is on

With the setting on, a correct address and password SHALL NOT by itself produce an access token. The instance SHALL instead send a six-digit numeric code to the account's address and answer with an identifier for that pending sign-in. Presenting that identifier together with the matching code SHALL produce the access token.

The code SHALL be generated from a cryptographically secure source, SHALL be stored so that reading the stored form does not reveal it, and SHALL be usable once.

The message carrying the code SHALL state the code, SHALL say that it stops working ten minutes after it was sent, and SHALL say that requesting another code replaces it.

This applies to every account on the instance, including the recorded bootstrap administrator.

#### Scenario: Completing a two-step sign-in

- **WHEN** a user submits a correct address and password on an instance with the second factor on
- **THEN** no access token is returned, a six-digit code arrives at that address, and the response identifies the pending sign-in
- **AND** submitting that identifier with the received code returns an access token

#### Scenario: The bootstrap administrator signs in

- **WHEN** the recorded bootstrap administrator signs in with a correct password
- **THEN** a code is required exactly as for any other account

#### Scenario: Reusing a code

- **WHEN** a user submits a code that has already completed a sign-in
- **THEN** the attempt is rejected and no second token is issued

#### Scenario: The same code submitted twice at once

- **WHEN** two requests carrying the same correct code for the same pending sign-in arrive at the same time
- **THEN** exactly one of them returns an access token

#### Scenario: What the message says

- **WHEN** a code arrives at an account's address
- **THEN** the message carries the six digits, says the code stops working in ten minutes, and says that requesting another code replaces it

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

### Requirement: A pending sign-in expires, tolerates few wrong guesses, and can be resent

A pending sign-in SHALL expire ten minutes after its code is issued. It SHALL be destroyed after five incorrect code submissions, after which the user starts again from the password step. An expired, destroyed or already-used pending sign-in SHALL never produce a token. Submissions that arrive at the same time SHALL count against the same five, so no more than five codes are ever checked against one pending sign-in.

The user SHALL be able to request a new code for a pending sign-in. A resend SHALL supersede the pending sign-in it replaces: the previous code SHALL stop working, and the answer SHALL carry the identifier that names the new one. Resends SHALL be limited to one a minute for the same account; a request inside that minute SHALL be refused without sending mail.

Codes SHALL be limited to five an hour for the same account however they were requested, so that repeating the password step cannot send more mail than resending would. A request beyond that limit SHALL be refused without sending mail and without leaving a usable pending sign-in, and the caller SHALL be told that too many codes have been requested, rather than being shown a code screen for a message that will never arrive. This SHALL be the only hourly limit on codes, so that no second limit can disagree with it.

Repeating the password step while a pending sign-in for that account is still live - not expired, not used, not destroyed by wrong guesses, and started with the password the account holds now - SHALL return that same pending sign-in and SHALL send no further mail. Password steps that arrive at the same time SHALL produce one pending sign-in and one message between them. Reaching the code screen again therefore costs nothing from the hourly limit, and the code already delivered stays the one that works.

A pending sign-in SHALL be kept for as long as the hourly limit counts it and SHALL NOT be kept after that. Being kept SHALL never make an expired, used or destroyed pending sign-in usable.

#### Scenario: Code expires

- **WHEN** a user submits a correct code eleven minutes after it was issued
- **THEN** the attempt is rejected and no token is issued

#### Scenario: Guessing the code

- **WHEN** someone submits five incorrect codes for the same pending sign-in
- **THEN** the pending sign-in is destroyed, and the correct code submitted afterwards is also rejected

#### Scenario: Guessing the code in parallel

- **WHEN** someone submits ten incorrect codes for the same pending sign-in at the same time
- **THEN** five of them are checked, the pending sign-in is destroyed, and the other five are refused without being checked

#### Scenario: Requesting another code

- **WHEN** a user whose code has not arrived requests a resend more than a minute after the last one
- **THEN** a new code is sent, the answer carries the identifier that names it, and the previous code no longer works

#### Scenario: Resending too often

- **WHEN** a user requests a second resend within the same minute
- **THEN** the request is refused and no mail is sent

#### Scenario: Reaching the code screen again

- **WHEN** a user who has reached the code screen loses the pending sign-in it was showing, by reloading the page or opening the instance on another device, and submits the correct password again while the first code is still live
- **THEN** the instance answers with the same pending sign-in, sends no second message, and the code already in the mailbox still works

#### Scenario: Signing in again after changing the password

- **WHEN** a user whose pending sign-in is still live changes the password, for example after an unexpected code revealed that someone else knows it, and then signs in with the new password
- **THEN** the instance sends a fresh code and answers with a new pending sign-in, instead of handing back the one the old password started, which can no longer be completed

#### Scenario: Restarting the password step to send more mail

- **WHEN** someone who knows an account's password lets each pending sign-in lapse and repeats the password step until that account has received five codes within an hour, then completes it once more
- **THEN** no further mail is sent, no usable pending sign-in is produced, and the response says that too many codes have been requested

#### Scenario: Old pending sign-ins are not kept

- **WHEN** pending sign-ins are older than the hour their limits are counted over
- **THEN** the instance no longer holds them

### Requirement: The verification step re-checks what the password step checked

Before issuing an access token, the instance SHALL confirm that the account behind a pending sign-in may still be let in: it still exists, it is still active, and its password has not changed since the pending sign-in was created. A pending sign-in that fails any of these SHALL be refused, whatever code it carries.

A pending sign-in created while the second factor was on SHALL still be completable after the setting has been switched off.

#### Scenario: The account is deactivated while the code is in flight

- **WHEN** an administrator deactivates an account after its password step succeeded but before its code is submitted
- **THEN** submitting the correct code is refused and no access token is issued

#### Scenario: The password changes while the code is in flight

- **WHEN** the account's password is changed after its password step succeeded but before its code is submitted
- **THEN** submitting the correct code is refused and no access token is issued

#### Scenario: The setting is switched off while the code is in flight

- **WHEN** an operator switches the second factor off after a user's password step succeeded
- **THEN** that user can still complete the sign-in with the code they received

### Requirement: The second step is protected like the first

The endpoints that verify a code and request a resend SHALL carry the same automated-abuse protection the instance applies to password sign-in and to the password-reset code request, including its human-verification challenge when the instance has one configured.

Both requests SHALL name the pending sign-in by the identifier the instance returned most recently, from the password step or from a resend, and SHALL NOT accept an email address as a way to name it. Knowing an address therefore gives no way to spend another account's attempts, resends or hourly allowance.

#### Scenario: Human verification is configured

- **WHEN** an instance with human verification configured receives a code verification or a resend request without a valid challenge answer
- **THEN** the request is refused before any code is checked and before any mail is sent

#### Scenario: Naming an account by its address

- **WHEN** somebody who knows another user's address tries to verify a code or request a resend for that account, without an identifier the instance issued to that account
- **THEN** the request has no way to name the account, so it consumes none of that account's attempts, resends or hourly allowance

### Requirement: Sign-in fails closed when the code cannot be sent

If the instance cannot send the code, the sign-in SHALL fail with a response that says the code could not be sent, and SHALL NOT issue an access token. An instance whose mail server has stopped working therefore admits nobody through password sign-in while the setting is on.

#### Scenario: Mail server unreachable

- **WHEN** a user submits a correct address and password while the mail server refuses the message
- **THEN** the response states that the code could not be sent and carries no access token

### Requirement: Sign-in through an external identity provider is not covered by the second factor

Sign-in through Google or GitHub SHALL continue to issue an access token directly, with no emailed code, whether or not the setting is on. This is a deliberate exception: those providers run their own multi-factor checks, and an instance that allows them accepts that the second factor guards only password sign-in.

Any interface or documentation that describes the second factor SHALL NOT claim it covers external identity providers.

#### Scenario: Signing in with Google while the second factor is on

- **WHEN** a user completes Google sign-in on an instance with the second factor on
- **THEN** an access token is issued without an emailed code

### Requirement: An operator can switch the second factor off from the host

An operator with shell access to the running instance SHALL be able to turn the global setting off with a single command, so that an instance whose mail server has died can be entered again without editing its database by hand. The command SHALL report what it changed, and SHALL succeed without complaint when the setting is already off.

A run that switches the setting off SHALL be recorded in the audit log, so that the log explains why an instance stopped asking for codes and the host is not a way to move a security switch unobserved.

The command SHALL be available only to a caller who can already execute commands inside the running instance, the same trust level the password-reset and administrator-listing commands require.

#### Scenario: Recovering from a dead mail server

- **WHEN** an operator who cannot receive codes runs the disable command on the host
- **THEN** the setting is off and the next password sign-in completes without a code

#### Scenario: Running it twice

- **WHEN** the operator runs the disable command on an instance where the setting is already off
- **THEN** the command reports no change and exits successfully

#### Scenario: The disable command leaves a trace

- **WHEN** the operator runs the disable command on an instance where the setting is on
- **THEN** the audit log holds an entry recording that the second factor was switched off from the host

### Requirement: The second factor is documented as part of the product's security story

The published documentation SHALL state that the instance can require an emailed second factor at sign-in, in the security material of both the repository readme and the website - the website's dedicated security page as well as the security question answered on its home page - in every language each ships. The console command that switches the second factor off SHALL be documented beside the other recovery commands, so an owner locked out by a broken mail server finds it where they already look for password recovery.

#### Scenario: A reader looks up what protects sign-in

- **WHEN** a reader consults the published security material in any supported language
- **THEN** it states that sign-in can require an emailed code, and does not claim that this covers sign-in through an external identity provider

#### Scenario: An owner locked out by a broken mail server

- **WHEN** that owner reads the published password and recovery documentation
- **THEN** it names the command that switches the second factor off and shows how to run it

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
