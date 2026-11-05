# google-drive-storage-oauth Specification

## Purpose

Let users authorize Google Drive storage from the existing form without exposing credentials in navigation or accepting an unrelated authorization response.

## Requirements

### Requirement: Authorization navigation contains no storage credentials

Google Drive authorization SHALL carry only an unpredictable attempt identifier as state. Client secrets, access tokens, refresh tokens and the storage draft SHALL NOT appear in authorization or callback URLs, persisted browser storage, application logs or diagnostics. Entered secrets and exchanged tokens SHALL remain in the initiating editor's memory until sent in the body of an explicitly requested existing storage operation. The Google code exchange SHALL send the client secret only in its request body to Google's fixed token endpoint. The interface SHALL show authorization status rather than render token contents.

#### Scenario: A new storage starts authorization

- **WHEN** a user authorizes a Google Drive draft containing a client secret
- **THEN** the navigation contains an opaque random state and contains no client secret, token or serialized draft
- **AND** the draft and credentials are not written to browser storage

#### Scenario: Google returns tokens

- **WHEN** the initiating editor successfully exchanges a code
- **THEN** tokens remain in that editor's memory and are absent from callback messages, rendered status, navigation and logs

### Requirement: Authorization results belong to a current single-use attempt

The interface SHALL accept an authorization result only for an attempt initiated by the current editor, account session, workspace and client settings. An attempt SHALL expire ten minutes after creation and SHALL be consumed before code exchange. Malformed, absent, incorrect, future-dated, expired or consumed state SHALL cause no exchange and no storage mutation. Starting another attempt, changing client settings, closing the editor, replacing the session or leaving its workspace SHALL invalidate the previous attempt and its pending completion. Changing only the storage name, language or theme SHALL preserve the current attempt and newer form input.

#### Scenario: A callback is replayed

- **WHEN** the same successful callback is delivered twice, including during callback setup replay
- **THEN** at most one code exchange occurs and no duplicate storage operation occurs

#### Scenario: State does not match or has expired

- **WHEN** a response has a missing, incorrect, future-dated or at-least-ten-minute-old attempt state
- **THEN** it causes no token exchange and no change to the current draft or saved storage

#### Scenario: An older exchange finishes after the editor changes

- **WHEN** an exchange finishes after its editor, session, workspace, client settings or authorization attempt has been replaced
- **THEN** it changes no active draft, authorization status, selection or feedback

#### Scenario: Presentation or the storage name changes while authorization is pending

- **WHEN** the user changes the storage name, language or theme while the current attempt is pending
- **THEN** authorization can finish for its original client settings and the newer name remains

### Requirement: The callback resumes the initiating editor without receiving its secrets

The existing Google storage callback address SHALL relay only the code or provider refusal and attempt identity through communication confined to the application origin. It SHALL NOT reconstruct the draft from URL data, obtain credentials from another tab's storage or create a storage independently. The initiating editor SHALL receive the result even when the provider's navigation has removed its direct relationship to the popup. A callback without a matching live editor SHALL terminate with translated restart guidance rather than remain loading.

#### Scenario: A form opened from backup settings is authorized

- **WHEN** its Google callback is accepted
- **THEN** authorization status updates in that same form and the surrounding config draft and workspace remain unchanged

#### Scenario: No initiating editor remains

- **WHEN** a callback opens after the initiating tab has reloaded or closed
- **THEN** it reaches a bounded error state with restart guidance and sends no token-exchange or storage request

### Requirement: Authorization leaves persistence under explicit user control

Successful authorization SHALL only update the initiating draft. Testing and saving SHALL use the existing storage operations, permissions and secret-omission behavior. New authorization SHALL require a usable refresh token before being offered for persistence. Reconnecting a saved storage SHALL require an explicitly entered client secret; no saved secret SHALL be returned to populate the form. Cancellation, refusal, missing refresh token, exchange failure or a failed save SHALL preserve previously saved credentials. A rename without reconnection SHALL require no secret re-entry or automatic authorization.

#### Scenario: Authorization succeeds before saving

- **WHEN** Google authorization succeeds for a new storage
- **THEN** no storage is created until the user explicitly tests the current settings and saves

#### Scenario: Reconnection fails

- **WHEN** authorization is refused, code exchange fails or the new response lacks a usable refresh token
- **THEN** translated feedback permits a fresh explicit attempt, and the saved storage keeps its existing credentials

#### Scenario: A saved storage is renamed

- **WHEN** the user renames a saved Google Drive storage without reconnecting it
- **THEN** testing and saving retain its stored credentials without requesting them from the API or starting authorization

### Requirement: Browser authorization failures have bounded recovery

A blocked popup, unsupported same-origin communication, abandoned authorization, provider refusal or transport failure SHALL leave the initiating draft available and offer cancellation or an explicit fresh attempt. Waiting SHALL end no later than attempt expiry; cancellation SHALL release pending work. The callback SHALL remove its query parameters from the current history entry before relaying data, starting callback-owned network work or showing diagnostic content. Failure SHALL NOT trigger automatic code-exchange or storage-mutation retries, nor expose raw provider payloads.

#### Scenario: The popup is blocked

- **WHEN** the browser refuses to open authorization
- **THEN** pending state ends with translated feedback and the draft remains editable

#### Scenario: The user abandons authorization

- **WHEN** no callback arrives and the attempt expires, or the user cancels it
- **THEN** pending state ends, late callbacks are ignored and a fresh attempt can be started

### Requirement: The old URL-encoded draft is not accepted after upgrade

Authorization callbacks using the old serialized-draft state SHALL be rejected without a compatibility parser. Upgrading SHALL preserve the callback address and every saved storage credential. Attempts whose initiating editor did not survive the update SHALL require fresh authorization.

#### Scenario: An old callback arrives after an update

- **WHEN** a callback carries the previous JSON-state format
- **THEN** no exchange or save occurs and the user receives restart guidance
