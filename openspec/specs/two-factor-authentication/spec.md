# Two-Factor Authentication Specification

## Purpose

Lets the owner of an instance require a second factor, an emailed code or a code from an authenticator app, before a password sign-in succeeds, with the preconditions that keep the switch from locking everyone out and a console path back when mail delivery stops working.

## Requirements

### Requirement: An administrator controls the second factor for the whole instance

The instance SHALL offer a single global setting that requires a second factor for password sign-in. It SHALL be off until an administrator turns it on, SHALL apply to every account on the instance once on, and SHALL be changeable only by an administrator.

When the setting is on, the administrator SHALL choose its method: an emailed code or an authenticator app. The interface SHALL present this as the switch that turns the requirement on and, under it, the choice of method. An account with a connected authenticator app proves the second factor with the app under either method.

Turning the setting on or changing its method SHALL NOT end sessions that already exist. An access token issued before the change SHALL keep working until it expires or a password change invalidates it. A pending sign-in created before the change SHALL be completed by the step it was created with.

#### Scenario: Default state

- **WHEN** an instance is deployed and claimed for the first time
- **THEN** the second factor is off and sign-in works with an address and a password alone, except for accounts that connected an authenticator app

#### Scenario: A non-administrator tries to change it

- **WHEN** a member submits a settings update that turns the second factor on or changes its method
- **THEN** the update is refused and the setting is unchanged

#### Scenario: A session that predates the switch

- **WHEN** an administrator turns the second factor on while another user is already signed in
- **THEN** that user's session keeps working

#### Scenario: Changing the method

- **WHEN** an administrator changes the method from the emailed code to the authenticator app
- **THEN** existing sessions keep working, the next password sign-in of an account without an app asks for an app setup, and an emailed code already sent still completes its pending sign-in

### Requirement: The second factor cannot be turned on unless the instance can deliver codes

Requiring the emailed code method SHALL be refused unless both conditions hold: the instance has a configured mail server, and every active administrator holds a syntactically valid email address. The refusal SHALL name which condition failed and, when an administrator address is the problem, which account it is. These conditions SHALL be checked both when the setting is turned on with this method and when the method changes to it.

Requiring the authenticator app method SHALL have no mail preconditions and SHALL be accepted on an instance without a mail server.

The interface offering the setting SHALL explain the mail server requirement of the emailed code method and link to the mail server section of the same settings screen, where the documentation for configuring it is linked. It SHALL decide whether the mail server is configured from the instance's own answer, so that the interface and the refusal never disagree about the same instance. The authenticator app method SHALL be offered whether or not a mail server is configured.

#### Scenario: No mail server

- **WHEN** an administrator requires the emailed code method while the instance has no mail server configured
- **THEN** the change is refused with a message naming the missing mail server
- **AND** the interface offers a link to the mail server section of the same settings screen

#### Scenario: No mail server, app method

- **WHEN** an administrator requires the authenticator app method while the instance has no mail server configured
- **THEN** the change is accepted

#### Scenario: An administrator has no usable address

- **WHEN** an administrator requires the emailed code method while another active administrator still holds a value that is not an email address, such as the login `admin` an instance created before the administrator account was reworked
- **THEN** the change is refused, the message names that account, and the setting stays as it was

#### Scenario: Both conditions satisfied

- **WHEN** an administrator requires the emailed code method with a mail server configured and every active administrator holding a real address
- **THEN** the change is accepted and takes effect for the next sign-in

#### Scenario: The interface and the instance agree

- **WHEN** an administrator opens the settings screen
- **THEN** whether the emailed code method is offered or explained as unavailable follows the same judgement about the mail server that a refusal would apply

#### Scenario: Switching off after the mail server is gone

- **WHEN** the emailed code method is required and the instance has since lost its mail server
- **THEN** an administrator who is still signed in can switch the setting off or move it to the authenticator app method from the settings screen, because only requiring the emailed code method is refused

### Requirement: Password sign-in asks for an emailed code when the second factor is on

When the instance requires the emailed code method and the account has no connected authenticator app, a correct address and password SHALL NOT by itself produce an access token. The instance SHALL instead send a six-digit numeric code to the account's address and answer with an identifier for that pending sign-in. Presenting that identifier together with the matching code SHALL produce the access token.

The code SHALL be generated from a cryptographically secure source, SHALL be stored so that reading the stored form does not reveal it, and SHALL be usable once.

The message carrying the code SHALL state the code, SHALL say that it stops working ten minutes after it was sent, and SHALL say that requesting another code replaces it.

This applies to every account on the instance without a connected authenticator app, including root admin.

#### Scenario: Completing a two-step sign-in

- **WHEN** a user without a connected app submits a correct address and password on an instance that requires the emailed code method
- **THEN** no access token is returned, a six-digit code arrives at that address, and the response identifies the pending sign-in
- **AND** submitting that identifier with the received code returns an access token

#### Scenario: The bootstrap administrator signs in

- **WHEN** root admin without a connected app signs in with a correct password
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

This requirement governs a pending sign-in that waits for an emailed code. A pending sign-in that waits for an app code or for an app setup follows its own requirements.

A pending sign-in SHALL expire ten minutes after its code is issued. It SHALL be destroyed after five incorrect code submissions, after which the user starts again from the password step. An expired, destroyed or already-used pending sign-in SHALL never produce a token. Submissions that arrive at the same time SHALL count against the same five, so no more than five codes are ever checked against one pending sign-in.

The user SHALL be able to request a new code for a pending sign-in. A resend SHALL supersede the pending sign-in it replaces: the previous code SHALL stop working, and the answer SHALL carry the identifier that names the new one. Resends SHALL be limited to one a minute for the same account; a request inside that minute SHALL be refused without sending mail.

Codes SHALL be limited to five an hour for the same account however they were requested, so that repeating the password step cannot send more mail than resending would. A request beyond that limit SHALL be refused without sending mail and without leaving a usable pending sign-in, and the caller SHALL be told that too many codes have been requested, rather than being shown a code screen for a message that will never arrive. This SHALL be the only hourly limit on emailed codes, so that no second limit can disagree with it.

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

If the instance cannot send the emailed code to an account that has no connected authenticator app, the sign-in SHALL fail with a response that says the code could not be sent, and SHALL NOT issue an access token. While the emailed code method is required, an instance whose mail server has stopped working therefore admits no account without an app through password sign-in. Accounts with a connected app, and every account under the authenticator app method, sign in without mail.

#### Scenario: Mail server unreachable

- **WHEN** a user without a connected app submits a correct address and password while the emailed code method is required and the mail server refuses the message
- **THEN** the response states that the code could not be sent and carries no access token

#### Scenario: Mail server unreachable for an account with an app

- **WHEN** a user with a connected app submits a correct address and password while the emailed code method is required and the mail server is down
- **THEN** the answer asks for an app code and no mail is attempted

### Requirement: Sign-in through an external identity provider is not covered by the second factor

Sign-in through Google or GitHub SHALL continue to issue an access token directly, with no emailed code, no app code and no app setup, whether or not the setting is on, whichever method it requires and whether or not the account has a connected authenticator app. This is a deliberate exception: those providers run their own multi-factor checks, and an instance that allows them accepts that the second factor guards only password sign-in.

Any interface or documentation that describes the second factor SHALL NOT claim it covers external identity providers.

#### Scenario: Signing in with Google while the second factor is on

- **WHEN** a user completes Google sign-in on an instance with the second factor on
- **THEN** an access token is issued without an emailed code

#### Scenario: Signing in with GitHub with a connected app

- **WHEN** a user with a connected authenticator app completes GitHub sign-in on an instance that requires the app method
- **THEN** an access token is issued without asking for an app code

### Requirement: An operator can switch the second factor off from the host

An operator with shell access to the running instance SHALL be able to turn the global setting off with a single command, so that an instance whose mail server has died can be entered again without editing its database by hand. The command SHALL report what it changed, and SHALL succeed without complaint when the setting is already off.

Switching the setting off SHALL NOT disconnect any authenticator app: an account with a connected app keeps being asked for its code. An owner who cannot produce an app code or a recovery code SHALL use the host command that resets one account's app, which the `authenticator-app` spec describes.

A run that switches the setting off SHALL be recorded in the audit log, so that the log explains why an instance stopped asking for codes and the host is not a way to move a security switch unobserved.

The command SHALL be available only to a caller who can already execute commands inside the running instance, the same trust level the password-reset and administrator-listing commands require.

#### Scenario: Recovering from a dead mail server

- **WHEN** an operator who cannot receive codes runs the disable command on the host
- **THEN** the setting is off and the next password sign-in of an account without an app completes without a code

#### Scenario: Running it twice

- **WHEN** the operator runs the disable command on an instance where the setting is already off
- **THEN** the command reports no change and exits successfully

#### Scenario: The disable command leaves a trace

- **WHEN** the operator runs the disable command on an instance where the setting is on
- **THEN** the audit log holds an entry recording that the second factor was switched off from the host

#### Scenario: Disabling does not remove apps

- **WHEN** the operator runs the disable command and a user with a connected app then signs in
- **THEN** that user is still asked for an app code

### Requirement: The second factor is documented as part of the product's security story

The published documentation SHALL state that the instance can require a second factor at sign-in, either as an emailed code or as a code from an authenticator app, that any user can connect an authenticator app, and that recovery codes replace a lost app. It SHALL say this in the security material of both the repository readme and the website - the website's dedicated security page as well as the security question answered on its home page - in every language each ships. The console commands that switch the second factor off and reset one account's authenticator app SHALL be documented beside the other recovery commands, so an owner locked out by a broken mail server or a lost phone finds them where they already look for password recovery.

#### Scenario: A reader looks up what protects sign-in

- **WHEN** a reader consults the published security material in any supported language
- **THEN** it states that sign-in can require an emailed code or an authenticator app code, and does not claim that this covers sign-in through an external identity provider

#### Scenario: An owner locked out by a broken mail server

- **WHEN** that owner reads the published password and recovery documentation
- **THEN** it names the command that switches the second factor off and shows how to run it

#### Scenario: An owner who lost the phone

- **WHEN** an owner without the phone or the recovery codes reads the published password and recovery documentation
- **THEN** it names the command that resets one account's authenticator app and shows how to run it

### Requirement: A password change requires a complete new sign-in

After a successful password change in the profile, the interface SHALL clear its current session, display the ordinary sign-in screen and tell the user that the password changed and signing in again is required. It SHALL NOT submit the new password automatically or report a completed sign-in before authentication completes. This SHALL apply whether the instance requires a second factor or not, and whether the account has a connected authenticator app or not.

A refused or failed password change SHALL leave the current session and profile open, display the failure, and SHALL NOT display password-change success.

#### Scenario: Password changed with a required second factor

- **WHEN** a signed-in user changes their password on an instance requiring a second factor
- **THEN** the interface displays the sign-in screen and the password-change message
- **AND** it starts no sign-in and sends no code until the user submits the ordinary sign-in form

#### Scenario: Completing sign-in with the new password

- **WHEN** the user signs in with the new password after changing it and the account has a second factor
- **THEN** the step the password answer names is shown, whether an emailed code, an app code or an app setup, and the interface becomes authenticated only after that step succeeds

#### Scenario: Password changed without a second factor

- **WHEN** a signed-in user without a connected app changes their password while a second factor is not required
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

### Requirement: Password sign-in chooses the second factor for the account

After the password step succeeds, the instance SHALL choose the second factor for that account in this order:

1. The account has a connected authenticator app: the sign-in waits for an app code or a recovery code, whatever the instance requirement is, and no mail is sent.
2. The instance requires the emailed code method: the sign-in waits for an emailed code as before.
3. The instance requires the authenticator app method: the sign-in waits for a new app setup.
4. The instance requires no second factor: the session is issued.

The answer to the password step SHALL tell the client which of these steps follows, so the interface can show the matching screen. This order applies to every account, including root admin.

Repeating the password step SHALL hand back a live pending sign-in only when it waits for the step this order chooses now. When the chosen step has changed since that pending sign-in was created, because the method changed or the account connected or lost an app, the password step SHALL start a pending sign-in for the step chosen now. The earlier pending sign-in SHALL stay completable by its own step until it expires or is used, as the requirement about changing the setting says, except that a pending sign-in waiting for an app code becomes unusable when that app is disconnected, replaced or reset.

#### Scenario: A connected app on an instance without a requirement

- **WHEN** a user with a connected app signs in with a correct password while the instance requires no second factor
- **THEN** no session is issued yet, no mail is sent, and the answer asks for an app code

#### Scenario: A connected app under the emailed code method

- **WHEN** a user with a connected app signs in while the instance requires the emailed code method
- **THEN** the answer asks for an app code and no mail is sent, even if the mail server is down

#### Scenario: No app under the emailed code method

- **WHEN** a user without an app signs in while the instance requires the emailed code method
- **THEN** a code is emailed exactly as before

#### Scenario: No app under the app method

- **WHEN** a user without an app signs in with a correct password while the instance requires the authenticator app method
- **THEN** no session is issued and the answer asks the user to set up an app

#### Scenario: The method changes while an emailed code is live

- **WHEN** a user without an app has a live pending sign-in with an emailed code, an administrator changes the method to the authenticator app, and the user submits the correct password again
- **THEN** the answer asks for an app setup instead of handing back the emailed-code pending sign-in, and the emailed code still completes the earlier pending sign-in until it expires

#### Scenario: The user connects an app while an emailed code is live

- **WHEN** a user has a live pending sign-in with an emailed code, connects an app from a session in another browser, and then submits the correct password again
- **THEN** the answer asks for an app code and no mail is sent

### Requirement: A pending sign-in can wait for an app code

A pending sign-in that waits for an app code SHALL accept a current app code or an unused recovery code of the account and SHALL then issue the session. A wrong code SHALL be refused with the same answer a wrong emailed code gets. The pending sign-in SHALL expire ten minutes after it was created and SHALL be destroyed after five wrong codes, after which the user starts again from the password step. Submissions that arrive at the same time SHALL count against the same five. Each wrong code SHALL also count toward the per-account limit the `authenticator-app` spec sets.

Such a pending sign-in sends no mail, so it SHALL offer no resend and SHALL NOT count toward the five codes an hour that emailed codes are limited to. Repeating the password step while it is still live, started with the password the account holds now, SHALL return the same pending sign-in.

It SHALL stay usable only while the account keeps the app that was connected when it was created. Disconnecting, replacing or resetting the app SHALL make it unusable. The re-checks of the account, its status and its password that apply to every pending sign-in SHALL apply to it as well.

The operation that completes it SHALL carry the same automated-abuse protection as the code verification step, and SHALL name the pending sign-in only by the identifier the instance returned.

#### Scenario: Completing a sign-in with the app

- **WHEN** a user submits the current app code for a live pending sign-in
- **THEN** the session is issued

#### Scenario: The app code step expires

- **WHEN** a user submits a correct app code eleven minutes after the password step
- **THEN** the attempt is refused and no session is issued

#### Scenario: Guessing the app code

- **WHEN** someone submits five wrong codes for the same pending sign-in
- **THEN** the pending sign-in is destroyed, and the correct code submitted afterwards is also rejected

#### Scenario: Reaching the app code screen again

- **WHEN** a user reloads the app code screen and submits the correct password again within ten minutes
- **THEN** the answer names the same pending sign-in, and a current app code completes it

#### Scenario: The app is reset while the sign-in waits

- **WHEN** an administrator resets the user's app after the password step and before the code is submitted
- **THEN** submitting a code that the old app generated is refused and no session is issued

#### Scenario: The account is deactivated while the app code is in flight

- **WHEN** an administrator deactivates the account after its password step and before the app code is submitted
- **THEN** submitting the correct app code is refused and no session is issued

#### Scenario: No resend for an app code

- **WHEN** a client asks to resend the code of a pending sign-in that waits for an app code
- **THEN** the request is refused with 409 `pending_sign_in_not_resendable` and no mail is sent

### Requirement: A required app setup completes the sign-in

While the instance requires the authenticator app method, a user without a connected app SHALL set one up between the password step and the session. The answer to the password step SHALL name a pending sign-in that waits for the setup, and SHALL itself carry the setup secret, its `otpauth://` address and the time the setup expires. Each such password step SHALL start a new setup with a new secret, replacing the account's unconfirmed setup and the pending sign-in that waited for it, so that every secret appears in exactly one answer.

Confirming the setup SHALL be a separate operation that names the pending sign-in by its identifier and carries a code from the new app. A correct code SHALL connect the app, SHALL return the new recovery codes once, and SHALL issue the session in the same answer. The interface SHALL offer no way to skip the setup.

The pending setup SHALL expire thirty minutes after it was created, giving the user time to install an app, and SHALL be destroyed after five wrong codes. Two confirmations with the same correct code at the same time SHALL connect one app and issue one session.

The re-checks of the account, its status and its password that apply to every pending sign-in SHALL apply to the pending setup. A pending setup created while the app method was required SHALL still be completable after the requirement was switched off or changed. The confirming operation SHALL carry the same automated-abuse protection as the code verification step.

#### Scenario: First sign-in after the requirement was turned on

- **WHEN** a member without an app signs in with a correct password on an instance that requires the app method, scans the shown QR code and submits the app's code
- **THEN** the app is connected, the recovery codes are shown once, and the member is signed in

#### Scenario: Reloading the setup screen

- **WHEN** a user on the setup screen reloads the page and submits the correct password again
- **THEN** a new secret and QR code are shown, a code generated from the earlier secret no longer confirms the setup, and a code from the new secret does

#### Scenario: Leaving the setup unfinished

- **WHEN** a user leaves the setup screen without submitting a correct code
- **THEN** no session is issued and the account has no connected app

#### Scenario: The setup expires

- **WHEN** a user submits a correct code thirty-one minutes after the password step that started the setup
- **THEN** the attempt is refused, no app is connected and no session is issued

#### Scenario: Guessing during the setup

- **WHEN** someone submits five wrong codes for the same pending setup
- **THEN** it is destroyed, and a correct code submitted afterwards is refused

#### Scenario: Confirming twice at once

- **WHEN** two confirmations with the same correct code for the same pending setup arrive at the same time
- **THEN** one app is connected and exactly one of them returns a session and recovery codes

#### Scenario: The account changes during the setup

- **WHEN** the account is deactivated, or its password is changed, after the password step and before the setup is confirmed
- **THEN** confirming with a correct code is refused, no app is connected and no session is issued

#### Scenario: The requirement changes during the setup

- **WHEN** an administrator switches the requirement off or to the emailed code method after the password step and before the setup is confirmed
- **THEN** confirming with a correct code still connects the app and issues the session

#### Scenario: Instance without a mail server

- **WHEN** the instance has no mail server and requires the app method
- **THEN** every account can still sign in by setting up or using an app
