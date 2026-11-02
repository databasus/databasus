## MODIFIED Requirements

### Requirement: An upgrade keeps every notifier's secrets

Upgrading Databasus SHALL keep every stored secret of every notifier as it was, encrypted values included, so each notifier sends with the secrets saved before the upgrade. The one exception is a webhook header row with an empty name, which an older release could save and never sent. The upgrade SHALL remove such a row with its value, and SHALL keep the other headers of the webhook in their order.

#### Scenario: Webhook header and Telegram token after the upgrade

- **WHEN** an instance has a webhook notifier with an encrypted header value and a Telegram notifier with an encrypted bot token, and it is upgraded to this release
- **THEN** both stored values are unchanged, and a test send through the webhook notifier carries the header value saved before the upgrade

#### Scenario: Webhook with a nameless header row after the upgrade

- **WHEN** an instance has a webhook notifier with a header `Authorization`, a header row with an empty name and a header `X-Team`, and it is upgraded to this release
- **THEN** reading the notifier lists `Authorization` and `X-Team` in that order and no other header, and both keep their stored values

#### Scenario: Delivery through a webhook that had a nameless header row

- **WHEN** a webhook notifier had a header row with an empty name before the upgrade, and a notification is sent through it after the upgrade
- **THEN** the webhook endpoint receives the request with the headers that have names

#### Scenario: Saving a webhook that had a nameless header row

- **WHEN** a member changes a webhook notifier that had a header row with an empty name before the upgrade, and sends the headers the read returned
- **THEN** the change answers `200`, and the next notification carries the stored values of those headers

#### Scenario: Webhook with only nameless header rows after the upgrade

- **WHEN** a webhook notifier has only header rows with an empty name, and the instance is upgraded to this release
- **THEN** reading the notifier lists no headers, and the webhook endpoint receives its notifications

#### Scenario: Webhook with an invalid non-empty header name after the upgrade

- **WHEN** a webhook notifier has a stored header `X Team`, and the instance is upgraded to this release
- **THEN** reading the notifier still lists `X Team`, and a change that sends the header back answers `400` with the code `invalid_request`

## ADDED Requirements

### Requirement: A webhook header name must be a valid HTTP header name

Creating a webhook notifier, changing one and sending a test through unsaved settings SHALL refuse a header name that is not a valid HTTP header name with `400`. A valid name is an RFC 9110 token: one or more ASCII letters, digits or the characters ``!#$%&'*+-.^_`|~``. It therefore has no spaces. A refused change SHALL leave the stored notifier as it was. The backend SHALL NOT trim a name to make it valid. The upgrade SHALL NOT repair a stored non-empty invalid name such as `X Team`.

#### Scenario: Creating a webhook with a space in a header name

- **WHEN** a member creates a webhook notifier with the header name `X Team`
- **THEN** the response is `400` with the code `invalid_request`, and no notifier is created

#### Scenario: Changing a webhook to an invalid header name

- **WHEN** a member changes a webhook notifier and sends a header whose name is a single space or is empty
- **THEN** the response is `400` with the code `invalid_request`, and the next notification through it carries the headers saved before

#### Scenario: Testing unsaved settings with an invalid header name

- **WHEN** a member sends a test through unsaved webhook settings with the header name `X Team`
- **THEN** the response is `400` with the code `invalid_request`, and no request reaches the webhook

#### Scenario: A valid header name is accepted

- **WHEN** a member creates a webhook notifier with the header name `X-Api_Key.1`
- **THEN** the notifier is created, and its notifications carry that header

#### Scenario: A header name with other token characters is accepted

- **WHEN** a member creates a webhook notifier with the header name `X~Team!1`
- **THEN** the notifier is created, and its notifications carry that header

### Requirement: The webhook form refuses an invalid header name

When the user leaves a header name field, the webhook form SHALL remove spaces at the start and the end of the name and SHALL send the name without them. Leaving a field whose name has no such spaces SHALL NOT count as a change of settings, so an earlier successful test stays valid. Spaces at the edges SHALL NOT count as an error. The form SHALL show an error at every header row whose name is empty or is not a valid HTTP header name. While such a row exists, saving and sending a test SHALL be unavailable. The error SHALL be shown in the language of the interface.

#### Scenario: Typing a space in a header name

- **WHEN** a user types `X Team` as a header name in the webhook form
- **THEN** the row shows the error, and saving and sending a test are unavailable

#### Scenario: Spaces around a header name

- **WHEN** a user types ` X-Team ` as a header name in the webhook form and leaves the field
- **THEN** the field shows `X-Team`, the row shows no error, and saving sends the name `X-Team`

#### Scenario: Leaving a header name field without changing it

- **WHEN** a test through the webhook form has succeeded, and the user clicks into a header name field and leaves it without changing the name
- **THEN** saving stays available

#### Scenario: A saved webhook with a stored invalid header name

- **WHEN** a user opens a webhook notifier with the stored header `X Team`
- **THEN** the row shows the error, and saving and sending a test are unavailable

#### Scenario: A header name of only spaces

- **WHEN** a user types only spaces as a header name in the webhook form and leaves the field
- **THEN** the field is empty, the row shows the error, and saving and sending a test are unavailable

#### Scenario: Adding a header row and leaving its name empty

- **WHEN** a user adds a header row in the webhook form and leaves its name empty
- **THEN** the row shows the error at once, and saving and sending a test are unavailable

#### Scenario: The error follows the interface language

- **WHEN** the interface language is not English and a header row has an invalid name
- **THEN** the row shows the error in that language

#### Scenario: Fixing or removing the row

- **WHEN** a user corrects the invalid name to `X-Team`, or removes the row
- **THEN** the error disappears, and saving and sending a test become available again
