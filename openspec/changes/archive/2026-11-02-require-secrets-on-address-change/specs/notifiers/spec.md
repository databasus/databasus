## MODIFIED Requirements

### Requirement: Changing a notifier keeps the secrets the request leaves out

Changing a notifier SHALL replace its name and settings with those in the request, except a secret the request leaves out, which SHALL keep its stored value, unless the request changes the notifier's resource address as the `secret-handling` spec describes. A webhook header sent without a value SHALL keep the stored value of the header with the same name, under the same condition. A header the request does not list SHALL be removed. A change that succeeds with the Telegram proxy turned off SHALL clear the stored proxy URL.

#### Scenario: Renaming a Slack notifier

- **WHEN** a member changes the name of a Slack notifier and sends its chat ID without the bot token
- **THEN** the notifier has the new name, and a test send through it still authenticates with the stored bot token

#### Scenario: Editing a webhook notifier with a header

- **WHEN** a member changes the method of a webhook notifier and sends its header `Authorization` without a value
- **THEN** the next notification through it carries the header `Authorization` with the value saved before

#### Scenario: Changing the URL of a webhook notifier with a header

- **WHEN** a member changes the URL of a webhook notifier and sends its header `Authorization` without a value
- **THEN** the response is `400` with `code` `invalid_request` and a field error for that header's value, and the notifier keeps its URL and its header value

#### Scenario: Removing a webhook header

- **WHEN** a member changes a webhook notifier and leaves its header `X-Team` out of the request
- **THEN** the next notification through it carries no `X-Team` header

#### Scenario: Turning the Telegram proxy off with the bot token

- **WHEN** a member turns the proxy of a Telegram notifier off and sends the bot token
- **THEN** the response is `200`, and the stored proxy URL is cleared

#### Scenario: Turning the Telegram proxy off without the bot token

- **WHEN** a member turns the proxy of a Telegram notifier off and leaves out the bot token
- **THEN** the response is `400` with `code` `invalid_request` and a field error for the bot token, and the notifier keeps its proxy URL

### Requirement: Unsaved settings can be tested before saving

Testing settings that are not saved yet SHALL send a test message built from the request's settings and SHALL answer with the outcome, like a test of a saved notifier. When the request names a saved notifier of the same workspace, a secret the request leaves out SHALL be taken from that notifier, unless the request changes the notifier's resource address as the `secret-handling` spec describes. When it names a notifier that does not exist or belongs to another workspace, the test SHALL get `404` with `code` `not_found`, and no message SHALL leave the backend. Such a test SHALL change no saved notifier, its last send error included. Settings that a save would refuse SHALL get `400` with `code` `invalid_request`, and no message SHALL leave the backend.

#### Scenario: New notifier tested from the form

- **WHEN** a member fills in a Discord notifier in the form and tests it before saving
- **THEN** the test message reaches the entered webhook URL, and the workspace has no new notifier

#### Scenario: Edited notifier tested with its stored secret

- **WHEN** a member edits the chat ID of a saved Telegram notifier and tests the form without entering the bot token
- **THEN** the test message is sent to the new chat with the stored bot token, and the saved notifier keeps its old chat ID

#### Scenario: Saved notifier tested at another address

- **WHEN** a member tests unsaved Telegram settings that name a saved Telegram notifier and another Bot API URL, without the bot token
- **THEN** the response is `400` with `code` `invalid_request`, and no request leaves the backend

#### Scenario: Notifier of another workspace named in the test

- **WHEN** a member of one workspace tests unsaved Slack settings without a bot token and names a Slack notifier of another workspace
- **THEN** the response is `404` with `code` `not_found`, and no request leaves the backend

#### Scenario: Invalid unsaved settings

- **WHEN** a member tests unsaved Telegram settings whose Bot API URL is `ftp://telegram.example.com`
- **THEN** the response is `400` with `code` `invalid_request`, and no request leaves the backend

### Requirement: Notifier editing preserves stored-secret omission

The form pilot SHALL preserve the existing distinction between an omitted stored secret and an explicitly supplied replacement. Loading, renaming, changing language/theme or testing a saved notifier SHALL not manufacture a secret value or send a display placeholder as a credential. Diagnostic and browser-test artifacts SHALL contain no real credentials. While the resource address of a saved notifier in the form differs from the saved one, as the `secret-handling` spec defines it, the form SHALL require each secret that the settings in the form use before it offers the test and the save, SHALL NOT show the kept-secret placeholder or the notice that saved header values may be left empty, and SHALL show under each such empty field a message that the address changed and the secret must be entered again. When the address returns to the saved value, the form SHALL show the placeholder and the notice again, and the message SHALL disappear. A value the user typed SHALL stay in the field. A secret that the settings in the form do not use SHALL stay optional.

#### Scenario: A saved SMTP notifier keeps its password

- **WHEN** a user tests and saves changes to an existing SMTP notifier at the same SMTP host and port without entering a replacement password
- **THEN** both requests omit the password and retain the existing credential through the established API behavior

#### Scenario: A user replaces a secret

- **WHEN** a user explicitly supplies a replacement secret and tests those settings
- **THEN** the test and subsequent save use that replacement without substituting a hidden-value placeholder

#### Scenario: SMTP host changed in the form

- **WHEN** a user changes the SMTP host of a saved notifier that has an SMTP user and leaves the password empty
- **THEN** the message appears under the password field, the kept-secret placeholder is gone, and the test and save controls are unavailable

#### Scenario: SMTP host returned to the saved value

- **WHEN** a user renames a saved notifier, changes its SMTP host and then restores the saved host
- **THEN** the kept-secret placeholder returns, the message disappears, and the test control is available

#### Scenario: SMTP host changed without an SMTP user

- **WHEN** a user changes the SMTP host of a saved notifier and clears the SMTP user
- **THEN** no password is required, and the test control is available

#### Scenario: Webhook URL changed in the form

- **WHEN** a user changes the URL of a saved webhook notifier with a saved header and leaves the header's value empty
- **THEN** the message appears under the header's value, the notice that saved values may be left empty is not shown, and the test and save controls are unavailable

#### Scenario: Webhook URL returned to the saved value

- **WHEN** a user renames a saved webhook notifier with a saved header, changes its URL and then restores the saved URL
- **THEN** the notice returns, the message disappears, and the test control is available

#### Scenario: Telegram Bot API URL changed in the form

- **WHEN** a user changes the Bot API URL of a saved Telegram notifier and leaves the bot token empty
- **THEN** the message appears under the bot token field, and the test control is unavailable

#### Scenario: Telegram proxy changed in the form

- **WHEN** a user types a proxy URL or toggles the proxy of a saved Telegram notifier and leaves the bot token empty
- **THEN** the message appears under the bot token field, and the test control is unavailable

#### Scenario: Telegram chat changed in the form

- **WHEN** a user changes the chat ID of a saved Telegram notifier and leaves the bot token empty
- **THEN** the kept-secret placeholder stays, and the test control is available

#### Scenario: Mattermost server URL changed in the bot mode

- **WHEN** a user changes the server URL of a saved Mattermost notifier in the bot mode and leaves the bot token empty
- **THEN** the message appears under the bot token field, and the test control is unavailable

#### Scenario: Mattermost server URL changed in the webhook mode

- **WHEN** a saved Mattermost notifier in the webhook mode has a changed server URL in the form
- **THEN** no bot token is required, and the test control is available

## ADDED Requirements

### Requirement: Only a user who may change notifiers sends stored secrets elsewhere

Testing unsaved notifier settings SHALL need the right to change the workspace's content. A viewer SHALL get `403` with `code` `permission_denied`, and no message SHALL leave the backend. A viewer SHALL keep the test of a saved notifier, because it sends only to the saved address.

#### Scenario: Viewer tests settings with a saved notifier's ID

- **WHEN** a viewer of a workspace tests webhook settings that name an endpoint of the viewer's choice and the ID of a saved webhook notifier of the workspace
- **THEN** the response is `403` with `code` `permission_denied`, and the endpoint receives no request

#### Scenario: Viewer tests new settings

- **WHEN** a viewer of a workspace tests unsaved Discord settings without naming a saved notifier
- **THEN** the response is `403` with `code` `permission_denied`, and no request leaves the backend

#### Scenario: Viewer tests a saved notifier

- **WHEN** a viewer of a workspace tests one of its saved notifiers
- **THEN** the response is `200` and reports the outcome of the test send
