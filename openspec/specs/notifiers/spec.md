# notifiers Specification

## Purpose
Defines the notification channels of a workspace: how a notifier is created, changed, tested, moved to another workspace and deleted, how its secrets are kept, and how a notification reaches every notifier of a database without holding up the work that sent it.

## Requirements

### Requirement: A saved notifier is returned without its secrets

Creating a notifier SHALL answer `201` with the notifier as it was saved: its ID, workspace, name, type, settings and last send error. Reading a notifier, listing notifiers, creating one and changing one SHALL never return a secret. The secrets are the Telegram bot token and proxy URL, the SMTP password, the Slack bot token, the Discord channel webhook URL, the Teams webhook URL, the Mattermost webhook URL and bot token, and the value of every webhook header. A webhook header SHALL still be returned with its name. Every other setting SHALL be returned as it was saved, the Telegram Bot API URL, the webhook URL and the webhook body template included. A webhook URL or body template that an older release stored encrypted SHALL be returned decrypted. When it cannot be decrypted with the current master key, the read SHALL return that field empty and SHALL still return the notifier, in the list as well, so the user can enter the value again.

#### Scenario: Creating a Telegram notifier

- **WHEN** a member of a workspace creates a Telegram notifier with a bot token, a chat ID and a proxy URL
- **THEN** the response is `201` with the notifier's ID, name, type and chat ID, and the body contains neither the bot token nor the proxy URL

#### Scenario: Reading a webhook notifier with headers

- **WHEN** a member reads a webhook notifier that has the header `Authorization` with a value
- **THEN** the response lists the header `Authorization` without its value, and shows the webhook URL

#### Scenario: Webhook URL stored encrypted by an older release

- **WHEN** a member reads a webhook notifier whose URL an older release stored encrypted with the current master key
- **THEN** the response shows the URL in clear text

#### Scenario: Webhook URL that cannot be decrypted

- **WHEN** a member lists the notifiers of a workspace that has a webhook notifier whose stored URL cannot be decrypted with the current master key
- **THEN** the list loads and shows that notifier with an empty URL

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

### Requirement: A saved notifier keeps its type

A notifier's type SHALL be fixed when it is created. Changing a notifier with settings of another type SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The edit form SHALL show the type of a saved notifier without letting the user change it. A user who needs another channel type SHALL create a new notifier.

#### Scenario: Slack notifier sent with Telegram settings

- **WHEN** a member changes a saved Slack notifier and sends Telegram settings
- **THEN** the response is `400` with `code` `invalid_request`, and the notifier is still the same Slack notifier with its settings

#### Scenario: Editing a saved notifier in the form

- **WHEN** a member opens the edit form of a saved Telegram notifier
- **THEN** the form shows the type Telegram and offers no way to choose another type

### Requirement: Invalid settings are refused without a change

Creating or changing a notifier with an empty name, a missing required setting or a setting its provider rejects SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The provider rules that exist today SHALL keep applying, such as the Telegram Bot API URL rules of the `telegram-notifier-endpoint` spec.

#### Scenario: Webhook without a URL

- **WHEN** a member creates a webhook notifier without a URL
- **THEN** the response is `400` with `code` `invalid_request`, and the workspace has no new notifier

#### Scenario: Invalid change of a saved notifier

- **WHEN** a member changes an email notifier and sends an empty SMTP host
- **THEN** the response is `400` with `code` `invalid_request`, and the notifier keeps its previous SMTP host

### Requirement: Notifiers are listed per workspace in name order

The notifiers of a workspace SHALL be listed in pages, ordered by name and then by ID. A page SHALL hold at most the requested number of notifiers and SHALL point to the next page while more notifiers remain. Listing SHALL need the workspace's ID.

#### Scenario: Workspace with more notifiers than one page

- **WHEN** a workspace has three notifiers named `b`, `a` and `c`, and a member lists them two per page
- **THEN** the first page holds `a` and `b` and points to a next page, and the next page holds `c` and points to none

### Requirement: A notifier a database uses cannot be deleted

Deleting a notifier that a database uses SHALL get `409` with `code` `in_use` and SHALL change nothing. Deleting a notifier no database uses SHALL answer `204`, and the notifier SHALL no longer be listed.

#### Scenario: Notifier attached to a database

- **WHEN** a member deletes a notifier that one database of the workspace uses
- **THEN** the response is `409` with `code` `in_use`, and the notifier is still listed and still notifies for that database

#### Scenario: Unused notifier

- **WHEN** a member deletes a notifier that no database uses
- **THEN** the response is `204`, and the notifier is no longer listed

### Requirement: A notifier moves to another workspace only with its databases

Transferring a notifier SHALL require the right to change content in both the notifier's workspace and the target workspace. A transfer SHALL get `409` with `code` `in_use` and SHALL change nothing while a database outside the target workspace uses the notifier. Transferring a database together with its notifiers SHALL move the database and those notifiers, and SHALL leave each notifier attached to it.

#### Scenario: Unused notifier to another workspace

- **WHEN** a user who is a member of two workspaces transfers an unused notifier from the first to the second
- **THEN** the notifier is listed in the second workspace and no longer in the first

#### Scenario: Notifier still used in its workspace

- **WHEN** a member transfers a notifier that a database of its current workspace uses
- **THEN** the response is `409` with `code` `in_use`, and the notifier stays in its workspace

#### Scenario: Target workspace where the user is a viewer

- **WHEN** a member of the notifier's workspace who is only a viewer of the target workspace transfers the notifier
- **THEN** the response is `403` with `code` `permission_denied`, and the notifier stays in its workspace

#### Scenario: Database transferred with its notifier

- **WHEN** a member transfers a database to another workspace and asks to take its notifiers along
- **THEN** the database and its notifier are both in the target workspace, and the database still uses the notifier

### Requirement: A test send reports its outcome

Testing a saved notifier SHALL send a test message through it and SHALL answer `200` with the outcome: delivered, or not delivered together with the error the channel gave. A viewer of the workspace SHALL be allowed to test. The test SHALL record its outcome as the notifier's last send error, as a scheduled notification does.

#### Scenario: Test through a working channel

- **WHEN** a viewer tests a saved webhook notifier whose endpoint answers `200`
- **THEN** the response is `200` and reports the message as delivered, and the endpoint received one request

#### Scenario: Test through a failing channel

- **WHEN** a member tests a saved webhook notifier whose endpoint answers `500`
- **THEN** the response is `200` and reports the message as not delivered with an error that names the status, and the notifier's last send error shows that error

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

### Requirement: A notification does not hold up the work that sends it

A backup, a WAL stream, a healthcheck or a verification that notifies the notifiers of a database SHALL continue without waiting for the deliveries. Every notifier SHALL get its own delivery, so a slow or failing notifier SHALL NOT delay or prevent the others. A delivery that has not finished after 30 seconds SHALL be stopped and SHALL count as failed. A delivery to Slack SHALL instead have 3 minutes, so it can wait out a rate limit, while each of its requests stays limited to 30 seconds. A Slack rate limit that asks for a longer wait than the delivery or the test send has left SHALL fail it at once. A test send SHALL keep the 30-second limit for every notifier type. A message longer than 2000 characters SHALL be cut to 2000 characters before it is sent.

#### Scenario: Webhook endpoint that never answers

- **WHEN** a database has a webhook notifier whose endpoint accepts the connection and never answers, and a Discord notifier, and its backup fails
- **THEN** the backup is recorded as failed without waiting for the webhook, the Discord notifier receives the failure message, and after 30 seconds the webhook notifier's last send error says the delivery timed out

#### Scenario: Long message

- **WHEN** a database's webhook notifier receives a notification whose message is 2500 characters long
- **THEN** the webhook endpoint receives exactly the first 2000 characters of the message

#### Scenario: Notifier that cannot be reached

- **WHEN** one of a database's two notifiers points at an address that refuses connections, and its healthcheck reports the database as unavailable
- **THEN** the other notifier receives the message

#### Scenario: Slack asks to wait thirty seconds

- **WHEN** Slack answers a delivery with a rate limit and `Retry-After: 30`, and accepts the next attempt
- **THEN** the message is delivered and the notifier's last send error stays empty

#### Scenario: Slack asks for a wait longer than the delivery has left

- **WHEN** Slack answers a delivery with a rate limit and `Retry-After: 3600`
- **THEN** the delivery fails at once and the notifier's last send error says Slack rate-limited the message

#### Scenario: Slack does not answer

- **WHEN** the Slack endpoint accepts a delivery's request and never answers
- **THEN** the request ends after 30 seconds, and the notifier's last send error is set and says the request timed out

#### Scenario: A Slack test send is rate-limited beyond its limit

- **WHEN** a member tests a saved Slack notifier and Slack answers with a rate limit and `Retry-After: 60`
- **THEN** the response is `200` and reports the message as not delivered without waiting out the wait, and the last send error says Slack rate-limited the message

#### Scenario: A Slack test send gets no answer

- **WHEN** a member tests a saved Slack notifier whose endpoint accepts the request and never answers
- **THEN** the test ends at the 30-second limit and reports the message as not delivered

### Requirement: Each delivery records its outcome without touching the settings

After every delivery, the notifier's last send error SHALL hold the error of that delivery, or nothing when it succeeded. Recording it SHALL change nothing else about the notifier, so a change the user saves while a delivery is in progress SHALL be kept. The recorded error SHALL NOT contain a secret of the notifier.

#### Scenario: Edit saved during a delivery

- **WHEN** a notification to a webhook notifier is in progress and a member saves a new name and URL for that notifier before the delivery ends
- **THEN** after the delivery the notifier has the new name and URL, and its last send error reflects the delivery

#### Scenario: Failure followed by success

- **WHEN** a notifier's delivery fails and its next delivery succeeds
- **THEN** the notifier first shows the failure as its last send error, and shows no last send error after the second delivery

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

### Requirement: Deliveries in progress finish during shutdown

When the backend shuts down, deliveries that have already started SHALL be given the time of the shutdown drain to finish, and SHALL record their outcome when they do. A delivery still running when the drain ends SHALL be abandoned with the process.

#### Scenario: Shutdown during a delivery

- **WHEN** the backend receives SIGTERM while a notification to a webhook endpoint that answers after two seconds is in progress
- **THEN** the endpoint receives the request, and the notifier's last send error is empty after the backend stops

### Requirement: Notifier editing measures changes against its baseline

The notifier editor SHALL derive whether a draft has changes from its initial or last acknowledged saved values. Returning all editable values to that baseline SHALL remove the save action. Initialization, selecting another notifier and reopening a cancelled editor SHALL establish the appropriate baseline without carrying over the previous editing session. A new notifier SHALL still require valid settings and an explicit successful test before creation.

#### Scenario: A user reverses an edit

- **WHEN** a user changes a saved notifier field and then returns it to its baseline value
- **THEN** the editor no longer offers to save an unchanged draft

#### Scenario: A user reopens an editor

- **WHEN** a user cancels a notifier draft and reopens the saved notifier
- **THEN** the editor starts from the saved values without the cancelled draft or its test result

### Requirement: A notifier settings test authorizes only the active settings

The editor SHALL allow a changed draft to be saved only after an explicitly requested successful test of its active settings in its current workspace and notifier context. Changing delivery settings SHALL invalidate previous test success and any in-flight result for the previous settings. Changing only the display name, interface language or theme SHALL not invalidate an already successful settings test. A late success or failure from an obsolete request SHALL not enable saving or overwrite current feedback. Testing SHALL never be retried automatically.

#### Scenario: Settings change before a successful test returns

- **WHEN** a user starts a settings test, edits the delivery settings and then receives the earlier success
- **THEN** saving remains unavailable until the current settings pass a new explicit test

#### Scenario: A display name changes after a successful test

- **WHEN** the active settings have passed their test and the user changes only the display name
- **THEN** the editor can save the valid changed draft without another test message

#### Scenario: An obsolete request fails

- **WHEN** a test of a previous draft fails after the user switches notifier or editing session
- **THEN** the failure does not alter the current editor or show feedback belonging to the previous session

### Requirement: Save completion preserves newer notifier edits

The editor SHALL acknowledge the values submitted by a successful save while preserving any newer changes entered during that request. After a successful creation, it SHALL retain the returned notifier ID and treat later saves as updates of that notifier, including when newer edits keep the original creation editor open. A completion for an obsolete editing session SHALL not close or replace the active editor or assign its saved ID to that editor. Failed saves SHALL retain the current draft and leave save or retest available as appropriate. Repeated activation while a save is pending SHALL send at most one save request.

The editor SHALL prevent changing provider type while creation is pending, while allowing changes to the name and settings of the submitted type. After successful creation the provider type SHALL remain immutable, including in the still-open editor. If creation fails without a saved notifier ID, the unsaved draft SHALL permit type selection again.

#### Scenario: A user edits while saving

- **WHEN** a user submits a tested draft, edits another value while it is saving and then receives a successful response
- **THEN** the newer edit remains visible and is not marked as saved by that response
- **AND** the editor stays open with the appropriate save or test action for the newer draft

#### Scenario: A creation succeeds while newer edits remain

- **WHEN** a user submits a new tested notifier, edits its name or settings of the submitted type before creation succeeds and then receives the created notifier's ID
- **THEN** the editor retains that ID and the newer edits without treating them as saved or closing
- **AND** after any required explicit test, the next save updates that same ID rather than creating another notifier
- **AND** the sequence creates exactly one notifier

#### Scenario: Creation locks type until the editor becomes saved

- **WHEN** a user submits tested SMTP settings and the creation response is still pending
- **THEN** the editor prevents choosing another provider type while allowing edits to the name and SMTP fields
- **AND** after success with newer edits, the editor retains the returned ID and presents SMTP as an immutable type even though the original creation editor remains open
- **AND** subsequent explicit tests and updates use that ID and SMTP settings, without another creation

#### Scenario: Failed creation unlocks the unsaved type

- **WHEN** the pending SMTP creation fails without a saved notifier ID
- **THEN** the unsaved draft keeps its current values and permits another provider type to be selected
- **AND** changing that type invalidates the previous settings test

#### Scenario: A user cancels newer edits after creation succeeds

- **WHEN** creation succeeds with newer edits still open and the user cancels those newer edits
- **THEN** the created notifier remains available with the values acknowledged by creation
- **AND** cancellation neither deletes it nor creates another notifier

#### Scenario: A save fails

- **WHEN** the active save fails
- **THEN** the current values remain in the editor with localized feedback and the applicable next action

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
