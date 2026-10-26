## Purpose

Defines the notification channels of a workspace: how a notifier is created, changed, tested, moved to another workspace and deleted, how its secrets are kept, and how a notification reaches every notifier of a database without holding up the work that sent it.

## ADDED Requirements

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

Changing a notifier SHALL replace its name and settings with those in the request, except a secret the request leaves out, which SHALL keep its stored value. A webhook header sent without a value SHALL keep the stored value of the header with the same name. A header the request does not list SHALL be removed. Turning the Telegram proxy off SHALL clear the stored proxy URL.

#### Scenario: Renaming a Slack notifier

- **WHEN** a member changes the name of a Slack notifier and sends its chat ID without the bot token
- **THEN** the notifier has the new name, and a test send through it still authenticates with the stored bot token

#### Scenario: Editing a webhook notifier with a header

- **WHEN** a member changes the URL of a webhook notifier and sends its header `Authorization` without a value
- **THEN** the next notification through it carries the header `Authorization` with the value saved before

#### Scenario: Removing a webhook header

- **WHEN** a member changes a webhook notifier and leaves its header `X-Team` out of the request
- **THEN** the next notification through it carries no `X-Team` header

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

Testing settings that are not saved yet SHALL send a test message built from the request's settings and SHALL answer with the outcome, like a test of a saved notifier. When the request names a saved notifier of the same workspace, a secret the request leaves out SHALL be taken from that notifier. When it names a notifier that does not exist or belongs to another workspace, the test SHALL get `404` with `code` `not_found`, and no message SHALL leave the backend. Such a test SHALL change no saved notifier, its last send error included. Settings that a save would refuse SHALL get `400` with `code` `invalid_request`, and no message SHALL leave the backend.

#### Scenario: New notifier tested from the form

- **WHEN** a member fills in a Discord notifier in the form and tests it before saving
- **THEN** the test message reaches the entered webhook URL, and the workspace has no new notifier

#### Scenario: Edited notifier tested with its stored secret

- **WHEN** a member edits the chat ID of a saved Telegram notifier and tests the form without entering the bot token
- **THEN** the test message is sent to the new chat with the stored bot token, and the saved notifier keeps its old chat ID

#### Scenario: Notifier of another workspace named in the test

- **WHEN** a member of one workspace tests unsaved Slack settings without a bot token and names a Slack notifier of another workspace
- **THEN** the response is `404` with `code` `not_found`, and no request leaves the backend

#### Scenario: Invalid unsaved settings

- **WHEN** a member tests unsaved Telegram settings whose Bot API URL is `ftp://telegram.example.com`
- **THEN** the response is `400` with `code` `invalid_request`, and no request leaves the backend

### Requirement: A notification does not hold up the work that sends it

A backup, a WAL stream, a healthcheck or a verification that notifies the notifiers of a database SHALL continue without waiting for the deliveries. Every notifier SHALL get its own delivery, so a slow or failing notifier SHALL NOT delay or prevent the others. A delivery that has not finished after 30 seconds SHALL be stopped and SHALL count as failed. A message longer than 2000 characters SHALL be cut to 2000 characters before it is sent.

#### Scenario: Webhook endpoint that never answers

- **WHEN** a database has a webhook notifier whose endpoint accepts the connection and never answers, and a Discord notifier, and its backup fails
- **THEN** the backup is recorded as failed without waiting for the webhook, the Discord notifier receives the failure message, and after 30 seconds the webhook notifier's last send error says the delivery timed out

#### Scenario: Long message

- **WHEN** a database's webhook notifier receives a notification whose message is 2500 characters long
- **THEN** the webhook endpoint receives exactly the first 2000 characters of the message

#### Scenario: Notifier that cannot be reached

- **WHEN** one of a database's two notifiers points at an address that refuses connections, and its healthcheck reports the database as unavailable
- **THEN** the other notifier receives the message

### Requirement: Each delivery records its outcome without touching the settings

After every delivery, the notifier's last send error SHALL hold the error of that delivery, or nothing when it succeeded. Recording it SHALL change nothing else about the notifier, so a change the user saves while a delivery is in progress SHALL be kept. The recorded error SHALL NOT contain a secret of the notifier.

#### Scenario: Edit saved during a delivery

- **WHEN** a notification to a webhook notifier is in progress and a member saves a new name and URL for that notifier before the delivery ends
- **THEN** after the delivery the notifier has the new name and URL, and its last send error reflects the delivery

#### Scenario: Failure followed by success

- **WHEN** a notifier's delivery fails and its next delivery succeeds
- **THEN** the notifier first shows the failure as its last send error, and shows no last send error after the second delivery

### Requirement: An upgrade keeps every notifier's secrets

Upgrading Databasus SHALL keep every stored secret of every notifier as it was, encrypted values included, so each notifier sends with the secrets saved before the upgrade.

#### Scenario: Webhook header and Telegram token after the upgrade

- **WHEN** an instance has a webhook notifier with an encrypted header value and a Telegram notifier with an encrypted bot token, and it is upgraded to this release
- **THEN** both stored values are unchanged, and a test send through the webhook notifier carries the header value saved before the upgrade

### Requirement: Deliveries in progress finish during shutdown

When the backend shuts down, deliveries that have already started SHALL be given the time of the shutdown drain to finish, and SHALL record their outcome when they do. A delivery still running when the drain ends SHALL be abandoned with the process.

#### Scenario: Shutdown during a delivery

- **WHEN** the backend receives SIGTERM while a notification to a webhook endpoint that answers after two seconds is in progress
- **THEN** the endpoint receives the request, and the notifier's last send error is empty after the backend stops
