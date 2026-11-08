## RENAMED Requirements

- FROM: `### Requirement: Sign-in refuses cleanly and discloses nothing about the accounts an instance holds`
- TO: `### Requirement: Sign-in explains refusals without disclosing other accounts`

## MODIFIED Requirements

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

## ADDED Requirements

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
