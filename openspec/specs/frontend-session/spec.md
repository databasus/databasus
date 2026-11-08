# frontend-session Specification

## Purpose

Keep account transitions predictable and prevent obsolete requests or unrelated OAuth callbacks from replacing or terminating the user's current session.

## Requirements

### Requirement: Session transitions clear the previous account's interface state

Completing sign-in SHALL activate the returned session. Logout, a successful password change requiring sign-in again, and refusal of the current session SHALL return the user to ordinary sign-in without requiring a full page reload. These transitions SHALL clear session-owned account data, selected screen, workspace selection, drafts, pending sign-in state and any return destination belonging to that session; the address SHALL return to `/`. Replacing an active session SHALL also reset the address and addressed selections before loading the replacement account. A later account SHALL receive only its own accepted startup data. A new deep link opened while signed out SHALL create a separate sign-in navigation intent under the deep-link sign-in requirement, rather than restoring a previous account's interface state. Language and theme preferences SHALL survive. Ending an already ended session or disposing an absent subscription SHALL not disturb another active subscriber.

#### Scenario: A user signs out and another signs in

- **WHEN** a user signs out with a workspace and profile open and another user completes sign-in
- **THEN** the previous account's content, selection and return destination do not appear in the new session
- **AND** the new account starts with its own permitted screens and workspaces while language and theme remain selected

#### Scenario: An active session is replaced within a shared workspace

- **WHEN** an active session is replaced with another account that can access the same workspace
- **THEN** the previous account's database and tab address is cleared before the replacement account loads
- **AND** the new account opens its own default destination without inheriting the previous selection or filters

#### Scenario: The current session is refused

- **WHEN** an authorized request belonging to the active session receives an unauthorized response
- **THEN** the session ends once and the interface shows ordinary sign-in at `/`
- **AND** no page-reload loop or repeated session-ending feedback occurs

#### Scenario: Password verification fails without a session

- **WHEN** a signed-out user submits incorrect sign-in credentials
- **THEN** the sign-in form shows the translated refusal and retains its correction path
- **AND** the application does not reload or start another sign-in request automatically

#### Scenario: A subscription is disposed more than once

- **WHEN** an authentication consumer unsubscribes repeatedly or its setup is repeated after cleanup
- **THEN** other consumers continue receiving transitions and the current consumer receives each transition only while subscribed

### Requirement: Obsolete requests cannot end or repopulate the current session

Authentication refusal SHALL apply only to the still-active session that authorized the request. Work started under an ended or replaced session SHALL not publish account data, completion callbacks or feedback into the current interface. An abandoned sign-in attempt SHALL not activate a session after a newer attempt or session transition supersedes it.

#### Scenario: An old session receives a late unauthorized response

- **WHEN** a request authorized by an earlier session receives an unauthorized response after a new sign-in has completed
- **THEN** the newer session remains active with its token and account data intact
- **AND** the old response causes no navigation, reload or stale feedback

#### Scenario: Startup finishes after logout

- **WHEN** startup data for an account arrives after that account signs out
- **THEN** it does not restore the account's screen, workspaces or capabilities

#### Scenario: A replaced sign-in finishes late

- **WHEN** an earlier sign-in attempt completes after it was abandoned for a newer attempt
- **THEN** the earlier completion does not replace the active attempt or session

### Requirement: OAuth sign-in callbacks belong to one browser-bound attempt

Starting GitHub or Google sign-in SHALL create an unpredictable state bound to that provider, the initiating tab, the redirect address and a ten-minute lifetime. Only a callback carrying the matching state and an authorization code within that lifetime SHALL start code exchange with the recorded provider and redirect address. The attempt SHALL be consumed before exchange and SHALL permit at most one exchange. Missing, malformed, mismatched, replayed or expired attempts SHALL fail before exchange and offer a fresh sign-in. Provider refusal SHALL produce translated feedback without exchange. Every failure the callback finds before exchange SHALL use one translated message saying that sign-in did not finish and that the user should start again, and SHALL NOT attribute the failure to an invalid provider. Attempt metadata SHALL contain no password, client secret, access token or authorization code. Failure to create secure attempt metadata SHALL prevent navigation to the provider.

#### Scenario: A provider returns a matching callback

- **WHEN** either supported provider returns a code and matching state for a live attempt in its initiating tab
- **THEN** exactly one exchange uses that attempt's provider and redirect address and successful completion activates the session
- **AND** the attempt cannot be reused

#### Scenario: The callback is unrelated to a live attempt

- **WHEN** callback state is missing, incorrect, already consumed, expired or has no valid local attempt, including an old provider-name state, or the authorization code is missing
- **THEN** no code exchange occurs, the message says that sign-in did not finish and the user can restart sign-in

#### Scenario: Callback processing is started again

- **WHEN** callback processing repeats while its first exchange is pending
- **THEN** no second exchange starts and only the current callback lifecycle can accept the first completion

#### Scenario: A provider refuses authorization

- **WHEN** a provider returns a refusal for the current attempt
- **THEN** the attempt ends, no exchange occurs and the translated message says that sign-in did not finish and offers a fresh sign-in

#### Scenario: The server refuses the exchange

- **WHEN** a code exchange fails with a server refusal or a network failure
- **THEN** the translated reason for that failure is shown and the user can restart sign-in

#### Scenario: Attempt storage is unavailable

- **WHEN** secure state generation or tab-local attempt storage fails
- **THEN** the interface reports the failure and does not start an unbound authorization flow

### Requirement: Session transitions discard session-owned request state

Ending or replacing a session SHALL discard that session's reusable request results and completed action state, request cancellation of its ordinary pending reads, and stop its automatic refresh work. A newly activated session SHALL load its own data even when it belongs to the same user and selects the same workspace and resource. A pending action from the ended session SHALL NOT publish data, schedule follow-up reads, show feedback, close a dialog or navigate in the new session. Discarding frontend request state SHALL NOT be treated as proof that an already submitted server action was cancelled. Language, theme and the application-version reload cooldown SHALL retain their existing lifetimes.

#### Scenario: Cached resource data exists when the user signs out

- **WHEN** a user signs out while resource data and a pending ordinary read belong to that session
- **THEN** the results and completed action state are discarded, the read is signalled for cancellation and the session's automatic refresh work stops
- **AND** sign-in shows no resource content from the ended session

#### Scenario: The same user signs back in

- **WHEN** the same user signs back in and opens the same workspace and resource
- **THEN** a fresh read obtains data authorized by the newly activated session
- **AND** results from the previous session cannot complete that load

#### Scenario: A previous session's mutation succeeds late

- **WHEN** a mutation submitted under an earlier session succeeds after a new session activates
- **THEN** it causes no data update, follow-up read, feedback, dialog closure or navigation in the new session
- **AND** the new session remains active regardless of the old request's outcome

#### Scenario: A session transition interrupts polling

- **WHEN** a session is replaced while a polling read is pending
- **THEN** no later tick belongs to the previous session and its pending response cannot repopulate the current interface
- **AND** language, theme and the version-reload cooldown remain selected

### Requirement: A new deep link survives its sign-in flow safely

Opening an application screen link while signed out SHALL preserve that destination through password, two-factor and supported OAuth sign-in. After successful sign-in, the application SHALL validate the destination against the new account's available resources and show it or its canonical available fallback. The sign-in return SHALL replace the authentication destination rather than adding callback or sign-in entries to application history.

Return destinations SHALL be restricted to recognized application screen paths on the same origin, with no credentials, tokens, authorization codes or arbitrary query data. External addresses, protocol-relative addresses and authentication or storage callback addresses SHALL NOT be accepted as return destinations. The destination SHALL belong to the initiating tab and current sign-in intent, be consumed once, and SHALL NOT outlive an abandoned intent, a completed sign-in or an account-ending transition. OAuth attempt state, provider binding, expiry and single-use exchange requirements SHALL remain intact.

#### Scenario: Password and two-factor sign-in return to the link

- **WHEN** a signed-out user opens an accessible database Configuration link and completes password sign-in and required two-factor verification
- **THEN** that database's Configuration address and screen appear after access validation

#### Scenario: OAuth returns to the link

- **WHEN** a signed-out user opens a database link and completes a valid supported OAuth flow in the initiating tab
- **THEN** the validated destination or its available fallback opens and the callback address is replaced

#### Scenario: The signed-in account cannot access the linked resource

- **WHEN** a deep-link sign-in completes with an account that cannot access the requested workspace or database
- **THEN** the application shows that account's available fallback and changes the address to it without displaying the linked resource

#### Scenario: A return destination attempts external navigation

- **WHEN** a return destination is external, protocol-relative or names a callback
- **THEN** it is rejected and successful sign-in resolves through the ordinary default destination

#### Scenario: A newer sign-in supersedes an earlier destination

- **WHEN** an abandoned sign-in flow later completes after a newer sign-in intent has replaced it
- **THEN** the earlier flow changes neither the active account nor the current destination

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
