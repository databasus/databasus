# Root Admin Account Specification

## Purpose

Describes how a Databasus instance gets the administrator account its owner works from: how that account is created on a new instance, what happens to the account older instances were shipped with, how the instance keeps recognizing it once its email address changes, which powers it holds over other administrators, and how an owner who has forgotten the address recovers it from the host.

## Requirements

### Requirement: A new instance creates no account until somebody registers

A started instance SHALL NOT create any account on its own. An instance that holds no account SHALL offer account creation as the way in, and SHALL say that the account being created administers the instance.

Creating that account SHALL require nothing the owner cannot supply locally: no delivered email, no outbound network access and no configured mail server.

An instance that already holds an account SHALL offer the ordinary way in - signing in, and creating an account where the registration policy allows it - and SHALL NOT repeat that the account being created administers the instance.

The instance SHALL NOT offer any way to set a password on an account that already exists without either authenticating as that account or proving control of its address with an emailed reset code. Claiming an instance is account creation; recovering an account whose password is lost is the emailed reset or a command run inside the instance.

#### Scenario: Opening a freshly deployed instance

- **WHEN** an owner opens a freshly started instance that has no mail server configured and no network access
- **THEN** the instance offers to create an account, saying that it will administer the instance
- **AND** no account exists on the instance until that form is submitted

#### Scenario: Setting up without any mail server

- **WHEN** the owner submits the form on that instance
- **THEN** the account is created and the owner is signed in as an administrator, with no email sent

#### Scenario: Opening an instance that already holds accounts

- **WHEN** a visitor opens an instance on which at least one account exists
- **THEN** the entry screen offers signing in, and says nothing about the first account administering the instance

#### Scenario: An anonymous caller tries to set a password on an existing account

- **WHEN** an unauthenticated caller attempts to set a password on an account that already exists, without an emailed reset code
- **THEN** the instance refuses, whether or not that account currently has a password

### Requirement: The first account on an empty instance administers it

The first account created on an instance that holds no other account SHALL receive the administrator role and SHALL be recorded as the instance's bootstrap administrator. This SHALL hold however the account is created, including a first sign-in through an external identity provider, and SHALL NOT depend on whether open registration is enabled.

An instance SHALL never record more than one bootstrap administrator, including when two accounts are created at the same moment.

An account created by registration SHALL carry a syntactically valid email address, so that the administrator of an instance always has an address the product can deliver to. An account created through an external identity provider carries the address that provider reports for it.

Recording an account as the bootstrap administrator SHALL be visible in the audit log, so the log explains where that account's privileges came from.

**BREAKING**: registration previously accepted any non-empty value as the email address. A value that is not an email address is now refused.

#### Scenario: The first registration takes the instance

- **WHEN** the first person registers on an instance that holds no account
- **THEN** that account holds the administrator role and is the recorded bootstrap administrator
- **AND** the audit log holds an entry naming that account as the instance's administrator

#### Scenario: The first sign-in through an external provider takes the instance

- **WHEN** the first person to reach an empty instance signs in through Google or GitHub instead of registering
- **THEN** the account created for them holds the administrator role and is the recorded bootstrap administrator

#### Scenario: A later registration is an ordinary account

- **WHEN** somebody registers on an instance that already holds an account
- **THEN** the new account holds the member role and the recorded bootstrap administrator is unchanged

#### Scenario: Two people register at the same moment

- **WHEN** two accounts are created concurrently on an empty instance
- **THEN** exactly one of them is recorded as the bootstrap administrator

#### Scenario: Registering with a value that is not an address

- **WHEN** somebody submits a registration whose email is not a valid email address
- **THEN** the registration is refused and no account is created

### Requirement: An upgraded instance keeps the account its owner already uses

An instance upgraded from a version that created an account with the login `admin` at startup SHALL carry the bootstrap-administrator record onto the account its owner already signs in with, so the upgrade changes nothing that owner can observe.

An upgraded instance whose account was never claimed - it has no password and no linked external identity, so nobody has ever signed in with it - and which holds no other account SHALL be left in the state of a new instance: that account is removed, and the first account created afterwards administers the instance.

An instance whose account was never claimed while other accounts already exist SHALL keep that account and record it as the bootstrap administrator. No upgrade SHALL leave an instance that holds accounts without a recognized bootstrap administrator, and the owner of such an instance SHALL be able to take that account over from the host, the same way a lost password is recovered.

#### Scenario: Upgrading an instance in use

- **WHEN** an instance whose owner has been signing in as `admin` is upgraded
- **THEN** that account is the recorded bootstrap administrator
- **AND** the owner signs in exactly as before, with the address `admin` and the same password

#### Scenario: Upgrading an instance nobody ever set up

- **WHEN** an instance that was deployed but never claimed, and holds nothing else, is upgraded
- **THEN** it holds no account, and the first account created on it administers it

#### Scenario: Upgrading an instance that was never claimed but already holds accounts

- **WHEN** an instance whose placeholder account was never signed in with, but which holds other accounts, is upgraded
- **THEN** that placeholder account is kept and recorded as the bootstrap administrator
- **AND** the owner can give it a password from the host and then move it to a real address

### Requirement: The instance recognizes its bootstrap administrator independently of that account's email address

The instance SHALL record which account is the bootstrap administrator when it creates that account, and SHALL keep recognizing it by that record. Authorization decisions SHALL use that recorded identity rather than the text of any account's email address. The local development seed specified by `local-development-workflow` is the sole exception to preserving this designation: after authenticating the configured dev account, it SHALL explicitly transfer the designation to that account. An email change alone SHALL NOT transfer privileges. This covers, at minimum, every privilege the requirement below reserves for the bootstrap administrator.

#### Scenario: The bootstrap administrator keeps its role after changing address

- **WHEN** the bootstrap administrator changes their email address
- **THEN** the instance still treats that account as the bootstrap administrator

#### Scenario: Restarting the instance

- **WHEN** the backend is restarted without running the local development seed
- **THEN** no account is created, and the recorded bootstrap administrator is unchanged

### Requirement: The bootstrap administrator keeps its powers over other administrators after changing address

Granting or revoking the administrator role, deactivating an account that holds it, and reactivating one SHALL remain reserved for the bootstrap administrator, and SHALL be permitted on the basis of the instance's record of which account that is. Changing the bootstrap administrator's email address SHALL NOT remove any of these powers, and SHALL NOT grant them to any other account.

A production host-console exception is creating a new account with the administrator role from the host console, which SHALL be permitted to a caller who can execute commands inside the running instance without the bootstrap administrator's involvement. It sits at the trust level of the console password reset, which can already take over the bootstrap administrator's own account. The exception covers only new accounts: the console SHALL NOT grant or revoke the role of an existing account, and SHALL NOT deactivate or reactivate one.

The local development seed has the separate, narrow exception specified by `local-development-workflow`: it SHALL grant the dev account the administrator role and transfer the bootstrap designation after successful authentication. The previous bootstrap administrator SHALL retain its administrator role. Ordinary account-management operations and production console commands SHALL NOT expose this transfer.

An overlapping account-management request SHALL NOT apply privileges or bootstrap-protection decisions made stale by this transfer. When the transfer completes before that request's protected administration checks, the request SHALL use the new bootstrap designation: it SHALL NOT demote or deactivate the new holder or allow the previous holder to manage administrators. When administration completes first, the seed SHALL validate the resulting account state before assigning privileges and SHALL refuse an inactive target.

#### Scenario: Development seeding transfers administrator-management powers

- **WHEN** the local development seed transfers the bootstrap designation from another administrator to the dev account
- **THEN** the dev account can grant or revoke another account's administrator role
- **AND** the previous holder remains an administrator but cannot manage administrator roles
- **AND** the audit log identifies the transfer

#### Scenario: An overlapping request cannot demote the new bootstrap administrator

- **WHEN** a request to demote the dev account overlaps a development-seed transfer and the transfer commits before the request's protected administration checks
- **THEN** the request is refused and the dev account remains the active bootstrap administrator with the administrator role

#### Scenario: An overlapping request cannot deactivate the new bootstrap administrator

- **WHEN** a request to deactivate the dev account overlaps a development-seed transfer and the transfer commits before the request's protected administration checks
- **THEN** the request is refused and the dev account remains the active bootstrap administrator with the administrator role

#### Scenario: The previous holder cannot keep its powers through an overlapping request

- **WHEN** the previous bootstrap administrator starts a request to grant another account the administrator role and the seed transfer commits before that request's protected administration checks
- **THEN** the request is refused and the third account's role is unchanged

#### Scenario: Deactivation completes before the seed transfer

- **WHEN** another administrator deactivates the dev account after seed authentication but before the seed validates it for transfer
- **THEN** the seed refuses the assignment and preserves the previous bootstrap administrator

#### Scenario: Appointing an administrator after the address has changed

- **WHEN** the bootstrap administrator, having changed its address, grants the administrator role to another user
- **THEN** the role is granted

#### Scenario: Another administrator still cannot manage administrators

- **WHEN** an account that holds the administrator role but is not the bootstrap administrator tries to grant the role, revoke it, or deactivate or reactivate an account holding it
- **THEN** every one of those attempts is refused, whatever address the bootstrap administrator currently holds

#### Scenario: The bootstrap administrator cannot be stripped of its role

- **WHEN** any account attempts to revoke the administrator role from the bootstrap administrator or to deactivate it
- **THEN** the attempt is refused, so the instance is never left without a recognized bootstrap administrator

#### Scenario: The console cannot promote an existing account

- **WHEN** an operator runs the console create command with the administrator role for an address that a member account already holds
- **THEN** the command fails, and that account is still a member

### Requirement: The bootstrap administrator can change their email address

The bootstrap administrator SHALL be able to change their email address through the same profile update every other user has, with no field locked and no explanation of why it is locked. The instance SHALL accept only a syntactically valid email address, and SHALL refuse an address another account already holds.

After the change, the administrator SHALL sign in with the new address, and the previous value SHALL no longer be accepted for that account. The administrator's existing session SHALL survive the change.

#### Scenario: An upgraded instance moves off the placeholder address

- **WHEN** the bootstrap administrator of an upgraded instance submits a real email address in place of `admin`
- **THEN** the change is accepted, and the audit log holds an entry naming the old and the new address
- **AND** the next sign-in succeeds with the new address and fails with `admin`

#### Scenario: Editing the address on the profile screen

- **WHEN** the bootstrap administrator opens their profile
- **THEN** the email field is editable and carries no notice that the address cannot be changed

#### Scenario: Rejecting a value that is not an address

- **WHEN** an administrator submits a new email value that is not a valid email address
- **THEN** the change is rejected and the stored address is unchanged

#### Scenario: Rejecting an address another account already holds

- **WHEN** the bootstrap administrator submits an address that belongs to another user
- **THEN** the change is rejected and the stored address is unchanged

### Requirement: An administrator can reset their password by email

The ordinary password reset SHALL work for the bootstrap administrator on the same terms as for any other user: a code is sent to the stored address, and it expires, is single-use and is rate-limited exactly as it is elsewhere. On a new instance this holds from the moment the account is created; on an upgraded one it holds as soon as the owner replaces the placeholder address.

A reset requested while an upgraded account still holds the placeholder address `admin` SHALL NOT deliver a code, because no mail server can reach that address. The caller SHALL get the same answer as for any other address, as the requirement on password reset requests says.

#### Scenario: Resetting the administrator's password by email

- **WHEN** the bootstrap administrator requests a password reset for their stored address
- **THEN** a reset code is sent to that address
- **AND** submitting the code sets a new password

#### Scenario: Reset requested for the placeholder address

- **WHEN** somebody requests a password reset for the address `admin` on an upgraded instance whose administrator still holds it
- **THEN** the answer is the same as for any other address
- **AND** no code is sent

### Requirement: Sign-in explains refusals without disclosing other accounts

A failed password sign-in SHALL distinguish an unknown email address, an incorrect password, incomplete registration and an account with no password. The response SHALL carry a stable, specific error code for its cause. These refusals SHALL answer 401 and SHALL issue no access token. An account without a password SHALL be refused cleanly and SHALL leave the instance serving other requests.

For an account with no password, the response SHALL distinguish guidance to use Google, GitHub, either provider, or neither provider. A provider SHALL be recommended only when it is linked to that account and configured on the instance. A stored password SHALL remain usable regardless of linked OAuth providers.

A failed sign-in SHALL NOT disclose another account's address, suggest the `admin` login, or reveal how many accounts the instance holds. This applies regardless of how many accounts there are. An owner who does not know their administrator address retains the existing host-console recovery path.

**BREAKING**: the shared `invalid_credentials` answer is replaced by specific refusal codes. Clients can distinguish whether the submitted address has an account; no compatibility alias for the removed code is retained.

#### Scenario: Mistyped address on a single-user instance

- **WHEN** an anonymous visitor attempts to sign in to an instance that holds only its administrator, using an address that does not exist
- **THEN** the response identifies the submitted address as unknown
- **AND** it does not name another address, mention the login `admin` or reveal the account count

#### Scenario: Password sign-in for an account that has no password

- **WHEN** somebody submits the password form for an account without a password
- **THEN** the response identifies the absence of a password and recommends only linked, configured OAuth providers
- **AND** the instance keeps answering every other request

#### Scenario: Unknown address and wrong password

- **WHEN** one sign-in uses an address that belongs to no account and another uses an existing account's address with a wrong password
- **THEN** both receive 401 with different codes identifying the respective causes

#### Scenario: Invited account that has not finished sign-up

- **WHEN** somebody signs in with the address of an account that was invited and has not finished sign-up
- **THEN** the response identifies incomplete registration rather than an incorrect password or a missing password

#### Scenario: Deactivated account with a wrong password

- **WHEN** somebody signs in to a deactivated account with a wrong password
- **THEN** the response is the same as for an active account with a wrong password

#### Scenario: Deactivated account with its correct password

- **WHEN** somebody signs in to a deactivated account with its correct password
- **THEN** the response says that the account is deactivated and carries no access token

#### Scenario: Both linked providers are available

- **WHEN** an account has no password and is linked to both Google and GitHub, and both providers are configured
- **THEN** the refusal recommends either Google or GitHub

#### Scenario: A linked provider is unavailable

- **WHEN** an account has no password and one or more of its linked providers are no longer configured
- **THEN** the refusal recommends only its remaining linked, configured providers
- **AND** it recommends no OAuth provider if none meets both conditions

#### Scenario: A configured provider is not linked

- **WHEN** an account has no password and an instance-configured provider is not linked to that account
- **THEN** the refusal does not recommend that provider

#### Scenario: OAuth account also has a password

- **WHEN** an active account has a password as well as an OAuth link and its correct password is submitted
- **THEN** password sign-in proceeds to the session or required second-factor step

### Requirement: An owner can recover the administrator addresses from the host

An operator with shell access to the running instance SHALL be able to list every account holding the administrator role, seeing for each one its email address, its display name, when it was created, whether it is still active, and whether it is the bootstrap administrator. The active state is part of the answer because a deactivated administrator explains why an address does not let its owner in. The listing SHALL print no password, no password hash, no token and no other secret.

The listing SHALL be available only to a caller who can already execute commands inside the running instance, which is the same trust level the existing password-reset command requires.

#### Scenario: Recovering a forgotten address

- **WHEN** an owner who cannot remember which address to sign in with runs the administrator listing command on the host
- **THEN** the output names every administrator account with its address, display name, creation date and active state, and marks the bootstrap administrator
- **AND** the output contains no password material

#### Scenario: Listing on an instance nobody has set up

- **WHEN** the listing command runs on an instance where no account has been created yet
- **THEN** it reports that the instance has no administrator, and lists nothing

### Requirement: Setting up and recovering an instance are documented

The product documentation SHALL describe how an instance gets its administrator - the first account created on it - and SHALL describe how to create an account from the host console, how to reset an administrator password and how to recover a forgotten administrator address, in the same place and in every language the documentation ships. The description of account creation SHALL say that the first account created from the console on an empty instance becomes its administrator, that a repeated run with the same address and role changes nothing, and that the password is then rotated with the console reset. The public documentation SHALL also answer the question of what to do when the administrator address or password is lost, and link to those steps.

Documented command examples SHALL NOT present `admin` as the address an owner is expected to use, and the documentation SHALL NOT describe the instance as holding a user named `admin`. Where the placeholder address is mentioned at all, it SHALL be as the account an instance created before this change still carries until its owner replaces it.

#### Scenario: An owner searches the documentation

- **WHEN** an owner who cannot sign in reads the published password documentation
- **THEN** it explains both resetting the password and listing the administrator addresses
- **AND** the same guidance is reachable from the questions answered on the home page

#### Scenario: An operator automates the first administrator

- **WHEN** an operator who deploys instances from scripts reads the published password documentation in any of the six published languages
- **THEN** it shows the command that creates an account from the console with its parameters
- **AND** it says the command changes nothing when run again for the same account, and that the password is rotated with the reset command

#### Scenario: Reading about roles

- **WHEN** a reader consults the access-management documentation in any of the six published languages
- **THEN** it describes the first account created on the instance as its administrator, without claiming that account's address is `admin`

### Requirement: An operator can create an account from the host

An operator with shell access to the running instance SHALL be able to create an account by running a console command that takes the account's email address, its password and its role, administrator or member, and optionally a display name. When no display name is given, the account SHALL be named after the part of its address before the `@`.

The created account SHALL be active and hold the given password at once: it SHALL sign in with that address and password without completing any registration or invitation step, and the console password reset SHALL change its password the way it does for any other account.

The command SHALL create the account whether or not the instance allows open registration.

The command SHALL refuse, creating nothing, when the address is not a syntactically valid email address, when the password is shorter than eight characters, or when the role is missing or is neither administrator nor member.

Every account the command creates SHALL be recorded in the audit log with its address and role, and the entry SHALL say that the account was created from the host console.

#### Scenario: Provisioning an administrator on an instance that already holds accounts

- **WHEN** an operator creates an account with the administrator role from the console on an instance that already holds accounts
- **THEN** the account holds the administrator role and is not the bootstrap administrator
- **AND** it signs in with the given address and password
- **AND** the audit log holds an entry naming its address and role and saying it was created from the host console

#### Scenario: Provisioning while open registration is off

- **WHEN** open registration is switched off and an operator creates a member account from the console
- **THEN** the account is created and signs in

#### Scenario: Rotating the password of a provisioned account

- **WHEN** an operator resets the password of an account created from the console, using the console password reset
- **THEN** the account signs in with the new password and is refused with the old one

#### Scenario: Omitting the display name

- **WHEN** an operator creates the account `ops@example.com` without a display name
- **THEN** the account's display name is `ops`

#### Scenario: Refusing invalid input

- **WHEN** an operator runs the command with a value that is not an email address, a password of seven characters, no role, or a role other than administrator or member such as `owner`
- **THEN** the command fails, says what is wrong, and no account is created

### Requirement: An account created from the host claims an empty instance under the first-account rule

An account created from the console on an instance that holds no account SHALL be that instance's bootstrap administrator, exactly as a first registration is, and the audit log SHALL hold the same entry naming it as the instance's administrator.

On an instance that holds no account, the command SHALL refuse to create a member account and SHALL say that the first account must be an administrator. A member as the first account would leave the instance with nobody able to grant the administrator role.

When the command and a registration create the first account at the same moment, exactly one account SHALL become the bootstrap administrator. If the console-created account loses, it SHALL be created as an ordinary account with the role the operator asked for.

#### Scenario: Provisioning the first administrator of a fresh instance

- **WHEN** an operator creates an administrator account from the console on an instance that holds no account
- **THEN** that account is the recorded bootstrap administrator
- **AND** the administrator listing marks it as the bootstrap administrator
- **AND** a person who registers afterwards gets the member role

#### Scenario: The console and a registration claim an empty instance at the same moment

- **WHEN** an operator creates an administrator account from the console while a registration claims the same empty instance
- **THEN** exactly one account is the recorded bootstrap administrator
- **AND** the console-created account exists and holds the administrator role

#### Scenario: Refusing a member as the first account

- **WHEN** an operator creates a member account from the console on an instance that holds no account
- **THEN** the command fails, says the first account must be an administrator, and the instance still holds no account

### Requirement: Creating the same account from the host again changes nothing

Running the command for an address that already belongs to an active account holding the requested role and a password SHALL succeed. It SHALL report that the account already exists, that nothing changed and that its password was not changed, and SHALL leave that account's password, name and sessions untouched. A deployment can therefore run the command on every rollout.

Running the command for an address that belongs to any other account SHALL fail and change nothing. This covers an account holding the other role, a deactivated account, an invited account that has not completed sign-up, and an account without a password, such as one created through Google or GitHub.

#### Scenario: Re-running provisioning on every deployment

- **WHEN** an operator runs the same create command twice, with a different password the second time
- **THEN** the second run succeeds and reports that nothing changed and that the password was not changed
- **AND** the account still signs in with the first password

#### Scenario: The address belongs to an account with another role

- **WHEN** an operator creates an administrator account for an address that a member account already holds
- **THEN** the command fails, and that account is still a member with its old password

#### Scenario: The address belongs to a deactivated account

- **WHEN** an operator creates an account for an address whose account is deactivated
- **THEN** the command fails, and the account stays deactivated

#### Scenario: The address belongs to an invited account

- **WHEN** an operator creates an account for an address that was invited and has not completed sign-up
- **THEN** the command fails, and the account stays invited and without a password

#### Scenario: The address belongs to an account without a password

- **WHEN** an operator creates an administrator account for an address whose active administrator account signs in through Google or GitHub and has no password
- **THEN** the command fails, and the account still has no password

### Requirement: Creating an account from the host stays at host trust level and prints no secret

Creating an account without registration SHALL be available only to a caller who can already execute commands inside the running instance, the same trust level the console password reset requires. The instance SHALL NOT expose it over the network.

The command's output and log lines SHALL NOT contain the password or its hash.

#### Scenario: Reading the command's output

- **WHEN** an operator creates an account from the console, runs the same command again, and runs it once more with input it refuses
- **THEN** none of the three outputs and no log line contains the password or a password hash

### Requirement: The console password reset refuses short passwords

The console password reset SHALL refuse a password shorter than eight characters and leave the stored password unchanged, matching the minimum every other way of setting a password enforces. It SHALL keep accepting any stored address, including the placeholder address `admin` that an upgraded instance can still carry.

#### Scenario: Resetting to a short password

- **WHEN** an operator resets an account's password from the console to a value of seven characters
- **THEN** the command fails, and the account still signs in with its old password

#### Scenario: Resetting the placeholder account

- **WHEN** an operator resets the password of the account whose address is `admin` to a password of at least eight characters
- **THEN** the account signs in with the address `admin` and the new password

### Requirement: A password reset request discloses nothing about the accounts an instance holds

A request for a password reset code SHALL give the same successful answer whether the address belongs to an active account, to an account that is not active, or to no account. Only an active account SHALL receive a code. A request refused by the rate limit SHALL answer that the limit was reached. The limit counts requests per address whether or not an account exists, so that answer reveals nothing either.

#### Scenario: Reset requested for an account that is not active

- **WHEN** somebody requests a reset code for the address of a deactivated account
- **THEN** the answer is the same as for an active account
- **AND** no code is sent

#### Scenario: Reset requested for an unknown address

- **WHEN** somebody requests a reset code for an address that belongs to no account
- **THEN** the answer is the same as for an active account
- **AND** no message is sent

#### Scenario: Rate limit for known and unknown addresses

- **WHEN** somebody sends three reset requests within an hour for an address that belongs to no account and three for an active account's address, and then a fourth request for each
- **THEN** both fourth requests get the same status and the same error code, saying that the limit was reached

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

### Requirement: Password sign-in reveals deactivation only after password verification

Password sign-in SHALL report account deactivation only after successfully verifying that account's stored password. Before that proof, the status, error code, text and recommended actions SHALL NOT distinguish an inactive account from an otherwise equivalent active account. An inactive account with no password SHALL receive the same password-absence guidance as an equivalent active account, without revealing deactivation. Existing provider-authenticated OAuth behavior SHALL remain unchanged.

#### Scenario: An inactive account has no password

- **WHEN** password sign-in is attempted for an inactive account with no password
- **THEN** the answer is the same password-absence refusal and guidance given for an active account with the same provider links
- **AND** it does not identify the account as deactivated

#### Scenario: A password does not match an inactive account

- **WHEN** password sign-in is attempted for an inactive account with an incorrect password
- **THEN** the response identifies an incorrect password and gives no indication of deactivation

#### Scenario: A password matches an inactive account

- **WHEN** password sign-in verifies an inactive account's correct password
- **THEN** the response is 403 `user_deactivated`, without issuing a session or creating a new pending sign-in

### Requirement: Specific sign-in refusals remain behind existing attempt controls

Human verification, when configured, and the existing limit of ten attempts per submitted address within a minute SHALL run before account-specific sign-in refusals. The controls SHALL count unknown addresses, incomplete registrations, accounts without passwords and incorrect passwords alike. Once the limit is reached, the response SHALL be 429 `rate_limited` without an account-specific reason.

#### Scenario: Any refusal exhausts the address limit

- **WHEN** ten sign-in attempts for the same address have been accepted by the limiter within a minute, regardless of their refusal cause
- **THEN** the next attempt receives 429 `rate_limited` rather than a specific account refusal

#### Scenario: Human verification fails

- **WHEN** a sign-in request fails the configured human-verification challenge
- **THEN** it receives the human-verification refusal without an account-specific reason
