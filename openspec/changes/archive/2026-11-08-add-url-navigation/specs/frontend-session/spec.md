## MODIFIED Requirements

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

## ADDED Requirements

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
