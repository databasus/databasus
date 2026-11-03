# frontend-session Specification

## Purpose

Keep account transitions predictable and prevent obsolete requests or unrelated OAuth callbacks from replacing or terminating the user's current session.

## Requirements

### Requirement: Session transitions clear the previous account's interface state

Completing sign-in SHALL activate the returned session. Logout, a successful password change requiring sign-in again, and refusal of the current session SHALL return the user to ordinary sign-in without requiring a full page reload. These transitions SHALL clear session-owned account data, selected screen, workspace selection, drafts and pending sign-in state. A later account SHALL receive only its own accepted startup data. Language and theme preferences SHALL survive. Ending an already ended session or disposing an absent subscription SHALL not disturb another active subscriber.

#### Scenario: A user signs out and another signs in

- **WHEN** a user signs out with a workspace and profile open and another user completes sign-in
- **THEN** the previous account's content and selection do not appear in the new session
- **AND** the new account starts with its own permitted screens and workspaces while language and theme remain selected

#### Scenario: The current session is refused

- **WHEN** an authorized request belonging to the active session receives an unauthorized response
- **THEN** the session ends once and the interface shows ordinary sign-in
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
