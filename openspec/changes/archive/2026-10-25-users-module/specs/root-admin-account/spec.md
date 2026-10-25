## MODIFIED Requirements

### Requirement: Sign-in refuses cleanly and discloses nothing about the accounts an instance holds

A failed sign-in SHALL NOT tell the caller that an account with the login `admin` exists on the instance, SHALL NOT suggest signing in under that login, and SHALL NOT reveal how many accounts the instance holds. This applies regardless of how many accounts there are.

A failed password sign-in SHALL give the same answer, with the same status and the same error code, whether the address belongs to no account, to an account that has not finished sign-up, to an account that has no password, or to an account whose password does not match. The answer SHALL say that the account is deactivated only after the password is verified. A caller who does not know an account's password therefore cannot learn from the status or the error code of a sign-in which addresses have accounts.

A sign-in attempt for an account that has no password - one created through an external identity provider, or an unclaimed account on an instance that has not yet been upgraded - SHALL be refused the way an incorrect password is refused, and SHALL leave the instance serving other requests.

**BREAKING**: an instance holding a single user previously answered a sign-in attempt with an unknown address by naming the `admin` login. That response is removed; an owner recovers the address from the console instead.

**BREAKING**: sign-in previously answered an unknown address, an account that had not finished sign-up and a deactivated account each with its own message before checking the password. Those three answers are replaced by the answer for an incorrect password.

#### Scenario: Mistyped address on a single-user instance

- **WHEN** an anonymous visitor attempts to sign in to an instance that holds only its administrator, using an address that does not exist
- **THEN** the response does not mention the login `admin` and does not reveal how many accounts the instance holds

#### Scenario: Password sign-in for an account that has no password

- **WHEN** somebody submits the password form for an account that signs in through Google or GitHub
- **THEN** the attempt is refused in the same way an incorrect password is refused
- **AND** the instance keeps answering every other request

#### Scenario: Unknown address and wrong password

- **WHEN** one sign-in uses an address that belongs to no account and another uses an existing account's address with a wrong password
- **THEN** both get the same status and the same error code

#### Scenario: Invited account that has not finished sign-up

- **WHEN** somebody signs in with the address of an account that was invited and has not finished sign-up
- **THEN** the response is the same as for a wrong password

#### Scenario: Deactivated account with a wrong password

- **WHEN** somebody signs in to a deactivated account with a wrong password
- **THEN** the response is the same as for a wrong password

#### Scenario: Deactivated account with its correct password

- **WHEN** somebody signs in to a deactivated account with its correct password
- **THEN** the response says that the account is deactivated and carries no access token

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

## ADDED Requirements

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
