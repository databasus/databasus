## ADDED Requirements

### Requirement: Password sign-in explains its refusal and offers the applicable correction path

The sign-in form SHALL translate the server's specific refusal in every supported interface language. Unknown-address errors SHALL identify the email field and incorrect-password errors SHALL identify the password field. Other refusals SHALL appear as form-level feedback. The form SHALL retain entered values, SHALL NOT automatically repeat sign-in, and SHALL keep the existing registration, OAuth and password-recovery controls available according to instance configuration.

Incomplete-registration feedback SHALL direct the user to finish sign-up with the submitted address. Password-absence feedback SHALL explicitly say that no password is set. When that feedback recommends Google, GitHub or both, it SHALL name those providers and use the existing provider buttons; it SHALL NOT initiate OAuth automatically. When no provider is recommended, the form SHALL point to existing password recovery if email delivery is configured, and otherwise SHALL direct the user to an administrator. Recovery guidance SHALL NOT promise that mail was sent or that the account is active. Feedback SHALL never suggest another account's address.

A language change SHALL retranslate visible feedback without losing the draft. Editing either credential after a refusal SHALL clear that refusal and its cause-specific guidance. Leaving the form or superseding the authentication attempt SHALL continue to prevent an obsolete response from publishing feedback or activating a session.

#### Scenario: The submitted email does not exist

- **WHEN** the server refuses a password sign-in because the submitted address is unknown
- **THEN** the email field reports that no account was found and asks the user to check the address
- **AND** both entered values remain available for correction

#### Scenario: The submitted password is incorrect

- **WHEN** the server refuses a password sign-in because the password does not match
- **THEN** the password field reports an incorrect password, and the existing recovery control remains available when configured
- **AND** no session is activated and no sign-in is repeated automatically

#### Scenario: Registration is incomplete

- **WHEN** the server identifies an invited account that has not completed registration
- **THEN** the form explains that registration is incomplete and directs the user to sign up with that same email address
- **AND** the existing sign-up action opens the registration flow

#### Scenario: The account has no password and a recommended provider

- **WHEN** the server identifies an account without a password and recommends Google, GitHub or both
- **THEN** the form explains the missing password and names only the recommended providers
- **AND** the user can initiate the existing provider sign-in explicitly

#### Scenario: No provider is recommended and email delivery is configured

- **WHEN** an account has no password, no provider is recommended, and the instance supports password recovery by email
- **THEN** the form explains that password recovery can be used to set a password and offers the existing recovery action
- **AND** it does not claim that a code was sent or that the account is active

#### Scenario: Neither a recommended provider nor email recovery is available

- **WHEN** an account has no password, no provider is recommended, and email recovery is unavailable
- **THEN** the form explains the missing password and directs the user to an administrator
- **AND** it does not recommend an unavailable provider or recovery action

#### Scenario: Deactivation is verified

- **WHEN** the server reports deactivation after verifying the password
- **THEN** the form explains that the account is deactivated and directs the user to an administrator
- **AND** it does not report an incorrect password

#### Scenario: The language changes while a refusal is visible

- **WHEN** the user changes interface language with sign-in refusal feedback visible
- **THEN** the refusal and its guidance appear in the selected language without losing the entered credentials

#### Scenario: Credentials change after a refusal

- **WHEN** the user edits either credential after a sign-in refusal
- **THEN** the previous refusal and its cause-specific guidance disappear without submitting another request
