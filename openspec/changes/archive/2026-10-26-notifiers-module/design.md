## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach. It assumes `workspaces-module` and `audit-log-views` have landed: `workspaces.Authorize`, `workspaces.WorkspaceDeleting` with its wiring in `internal/app/wiring.go`, the bridge pattern of `features/workspaces`, and `RESTRICT` on `fk_notifiers_workspace_id`.

**Operations.** `NotifierController` (`internal/features/notifiers/controller.go:20-28`) serves seven routes. `POST /notifiers` creates a notifier, or updates one when the body carries an ID, and answers with the request body. `GET /notifiers?workspace_id=` returns every notifier of a workspace as an array. `GET /notifiers/:id`, `DELETE /notifiers/:id`, `POST /notifiers/:id/test`, `POST /notifiers/:id/transfer` and `POST /notifiers/direct-test` complete the set. Errors go out as `{"error": err.Error()}`, with 403 for the permission errors and 400 for everything else, a failed test send included. A test needs only `ActionView`, so a viewer may test.

**Model and repository.** `Notifier` (`internal/features/notifiers/model.go:20`) has one pointer field per provider with a GORM `foreignKey` association, and `getSpecificNotifier` (`:116`) switches on the type and panics on an unknown one. `NotifierRepository.Save` (`repository.go:14`) runs two `switch` statements and an `Omit` of seven associations. `FindByID` and `FindByWorkspaceID` run seven `Preload` calls. `Delete` runs one more `switch`. `Notifier.Update` (`model.go:80`) copies the incoming type and then updates only the provider row of that type. When the type changed, the existing row of the new type is nil, so the new settings are never applied, although the edit form offers a type selector for a saved notifier. No user can rely on changing the type, so the module fixes the type at creation instead, as the `notifiers` spec says.

**Providers.** Seven packages under `internal/features/notifiers/models/<name>` each define a GORM model with `Validate`, `Send`, `HideSensitiveData`, `Update` and `EncryptSensitiveData`, all taking a global `encryption.FieldEncryptor`. The webhook stores its headers and its accepted notification types as JSON in `TEXT` columns and decrypts its URL and body template in a GORM `AfterFind` hook that reads the global encryptor. Since commit `2c0a2940` the URL is no longer encrypted on save; rows saved before that still hold `enc:` values. Webhook header values are hidden on read, and `WebhookNotifier.Update` replaces the stored headers with the incoming ones, so saving the edit form, which receives the headers without values, stores empty values. The webhook, Discord and Teams providers send through a client without a timeout, and eight sends across the providers use `context.Background()`. Slack posts to a hard-coded `https://slack.com/api/chat.postMessage`, and Telegram defaults to `https://api.telegram.org`. `models/url_credentials.go` strips credentials from webhook URLs in errors, and `telegram/bot_api_url.go` and `telegram/proxies.go` hold the Telegram rules of the `telegram-notifier-endpoint` spec.

**Sending.** `NotifierService.SendNotification` (`service.go:297`) cuts the message to 2000 characters, reloads the notifier with every association, sends, and saves the whole notifier, provider row included, to record `last_send_error`. Seven loops call it: `backuping/logical/backuper.go:261`, `backuping/physical/backuper.go:756` and `:789`, `backuping/physical/wal_supervisor.go:381` and `:486`, `healthcheck/attempt/check_database_health_uc.go:219`, all synchronously, and `verification/runs/service.go:739`, one goroutine per notifier. The callers take the notifiers from `databases.Database.Notifiers`, a GORM many-to-many over `database_notifiers` (`internal/features/databases/model.go:37`), and each filters by its own `SendNotificationsOn`. Five interfaces describe the sender: `NotificationSender` in `notifiers`, `backups/backups/core/logical`, `backups/backups/backuping/physical` and `verification/runs`, and `HealthcheckAttemptSender` in `healthcheck/attempt`, with testify mocks in `backuping/logical/mocks.go` and `healthcheck/attempt/mocks.go`.

**References from databases.** `database_notifiers` (`backend/migrations/20250605090323_init.sql:168`) references `notifiers` with `ON DELETE RESTRICT` and `databases` with `CASCADE`. Even so, `DeleteNotifier` asks `NotifierDatabaseCounter`, which `databases/di.go:50` registers through a setter, and refuses with its own error. `TransferNotifierToWorkspace` takes a `transferingWithDbID` to allow a notifier used only by the database being moved. The database transfer in `backups/config/logical/service.go:205` and its twin in `backups/config/physical/service.go` move the notifiers before the database and ignore their errors (`_ =`). The old `databases` code also reads notifiers for its routes `/databases/notifier/:id/is-using` and `/databases/notifier/:id/databases-count`, for the workspace check on database create and update (`databases/service.go:179`), and for `UpdateDatabaseNotifiers`. The `/databases` responses carry the parent fields of each attached notifier, because the many-to-many `Preload` loads no provider rows.

**Other readers.** `telemetry` reads every notifier to collect their types (`telemetry/service.go:396`). The deletion tests of the `workspaces` module create a webhook notifier through the old `POST /notifiers` and count notifiers through `/notifiers?workspace_id=` (`internal/modules/workspaces/helpers_test.go:253` and `:286`).

**The schema.** `notifiers` has `id`, `name`, `notifier_type varchar(50)`, `last_send_error` and a nullable `workspace_id`. The seven provider tables `telegram_notifiers`, `email_notifiers`, `webhook_notifiers`, `slack_notifiers`, `discord_notifiers`, `teams_notifiers` and `mattermost_notifiers` have `notifier_id` as primary key with a deferrable `CASCADE` reference to `notifiers`. Only `email_notifiers.security` has a `CHECK`.

**Tests.** `internal/features/notifiers/controller_test.go` (1978 lines) drives the old routes. Model tests cover the webhook, Mattermost, Telegram, Discord and email providers and the URL credential stripping. 24 files of other features call `notifiers.CreateTestNotifier`, `RemoveTestNotifier` or `GetWebhookStub` 230 times: ten in `backups/backups`, three in `backups/config`, seven in `verification/runs`, and one each in `databases`, `healthcheck/attempt`, `restores` and `restores/restoring`. Five helpers of `databases/testing.go` and 19 places in tests build `Notifiers: []notifiers.Notifier{...}`. The upgrade test writes one webhook notifier through the previous release and reads it through `GET /notifiers?workspace_id=` (`e2e/upgrade/current_release_checks_test.go:291`).

**Frontend.** `entity/notifiers` has a hand-written `notifierApi` and 33 model and validation files, with tests for the Telegram and Mattermost validation and the email port rule. `features/notifiers` has 25 files: the list, the card, the show screens and one edit component per provider, with tests for the Telegram screens. `features/databases` reads `database.notifiers` in the edit, show and transfer screens, and calls the two old `databases` notifier routes.

**Rules in force.** The module lands in the target layout and passes every linter and `archlint`. `docs/architecture.md` fixes the registry, the provider-owned table and repository, the parent's transaction, the batch load per type, the composite key `(id, type)`, `CHECK` on enums, `JSONB` only for opaque data, `text[]` for lists, the single-column `UPDATE`, decryption where a secret is used, `RESTRICT` with 409 `in_use`, and goroutines only through `lib`. The api-design skill fixes the paths, the workspace filter, cursor pages, the `oneOf` settings with `type`, `writeOnly` secrets kept on `PUT`, and the Problem codes. The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `notifiers` is a module with its operations in the contract, and adding a provider takes a package, a migration, a registry entry, a contract schema and its conversion in the module's `httpapi`.
- One call notifies the notifiers of a database. No sender waits for a delivery, and no delivery can overwrite the user's settings.
- The old features keep their routes and answers and reach the module only through the bridge.

**Non-Goals:**

- A shared provider framework for storages. Milestone 7 copies the shape and extracts what is common once two families exist.
- A general event library. `NotifierTransferring` follows the subscription pattern of `WorkspaceDeleting`.
- Moving the old `databases` notifier routes into the contract. They move with `databases` in milestone 8.

## Decisions

### 1. The module's packages and its contract

```
internal/modules/notifiers/
  module.go          Module, Deps, New and the aliases below
  internal/
    notification/    NotifierType, Category and Notification
    notifier/        Notifier, NotifierTransferring; create, list, read, update, delete, transfer,
                     test; the parent repository and the outcome UPDATE
    delivery/        Notify: background sends with a timeout through notifier.Service
    providers/       the Provider interface, the registry and Endpoints
      email/ telegram/ slack/ discord/ teams/ mattermost/ webhook/
    httpapi/         the generated strict server and its handlers
    module/          the Module implementation that ties the services together
```

The types the contract exposes are declared in the internal packages and aliased in `module.go`, as `workspaces.Workspace` aliases `workspace.Workspace`, because `internal/module` returns them and cannot import the root package that imports it:

```go
// internal/notification
type NotifierType string // EMAIL, TELEGRAM, SLACK, DISCORD, TEAMS, MATTERMOST, WEBHOOK

// Category says what a notification is about; a webhook can accept only some categories.
type Category string // ALL, BACKUP_SUCCESS, BACKUP_FAILED, HEALTHCHECK_SUCCESS, ...

type Notification struct {
	Category Category
	Heading  string
	Message  string
}

// internal/notifier
type Notifier struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	Name          string
	Type          notification.NotifierType
	LastSendError *string
}

// Transfer names the notifier to move and the workspace it moves to.
type Transfer struct {
	NotifierID        uuid.UUID
	TargetWorkspaceID uuid.UUID
}

// Transferring is published inside the transfer's transaction, before the notifier moves.
type Transferring struct {
	Transfer
}

type TransferringHandler func(ctx context.Context, event Transferring) error

// internal/providers
type Endpoints struct {
	SlackPostMessageURL string
	TelegramBotAPIURL   string
}
```

`module.go`:

```go
type (
	Type                        = notification.NotifierType
	Category                    = notification.Category
	Notification                = notification.Notification
	Notifier                    = notifier.Notifier
	NotifierTransfer            = notifier.Transfer
	NotifierTransferring        = notifier.Transferring
	NotifierTransferringHandler = notifier.TransferringHandler
	Endpoints                   = providers.Endpoints
)

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// Notify returns at once; each notifier gets its own delivery in the background.
	Notify(ctx context.Context, notifierIDs []uuid.UUID, notification Notification)
	// GetNotifiers leaves out the IDs that match no notifier.
	GetNotifiers(ctx context.Context, notifierIDs []uuid.UUID) ([]Notifier, error)
	// GetNotifierTypes returns the type of every notifier of the instance, for telemetry.
	GetNotifierTypes(ctx context.Context) ([]Type, error)
	TransferNotifier(ctx context.Context, actor users.Actor, transfer NotifierTransfer) error
	OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error
	// SubscribeToNotifierTransferring is called by internal/app while it builds the application.
	SubscribeToNotifierTransferring(handler NotifierTransferringHandler)
	// WaitForDeliveries returns once every started delivery has ended, or with ctx's error.
	WaitForDeliveries(ctx context.Context) error
}

type Deps struct {
	DB              *gorm.DB
	Transactor      *database.Transactor
	Logger          *slog.Logger
	AuditLog        *audit.Log
	FieldEncryptor  *crypto.FieldEncryptor
	Users           users.Module
	Workspaces      workspaces.Module
	DeliveryTimeout time.Duration
	Endpoints       Endpoints
	PublicURL       string
}
```

`Users` gives the audit events their author. `PublicURL` is the instance's public address: the email provider takes the host of its default SMTP greeting from it when a notifier sets no `heloName`, as the old email sender did.

`notification` is a package of its own because the providers need its types and `notifier` imports the providers. Inside it the notifier type is `NotifierType`, so `notification.NotifierType` does not read as the type of a notification; outside the module it is `notifiers.Type`. `Category` keeps today's values of `notifier_models.NotificationType`, `ALL` included, because the webhook's stored accepted categories and the backup and verification configs use them. The concept has one name everywhere: `Category` in Go, the enum schema `NotificationCategory` and the webhook field `acceptedCategories` in the contract, the column `accepted_categories`, and the generated `NotificationCategory` on the frontend. `docs/glossary.md` gets "notification category", "notification" and "delivery", see decision 15.

*Rejected: `Notify(ctx, databaseID, notification)`.* `notifiers` sits below `databases` and cannot read which notifiers a database uses. The caller already holds the IDs.

*Rejected: `TransferNotifier` only behind HTTP.* The database transfer of the old backup configs moves notifiers with the actor's rights, so it needs the same operation with the same checks; the bridge calls it.

### 2. A provider owns its settings, its table and its send

```go
type Provider interface {
	GetType() notification.NotifierType
	Validate(settings Settings) error
	// MergeStoredSecrets fills the secrets the incoming settings left out from the stored ones.
	MergeStoredSecrets(merge SecretMerge) (Settings, error)
	Send(ctx context.Context, settings Settings, message notification.Notification) error
	GetRepository() SettingsRepository
}

// SecretMerge names both sides, so a swapped call cannot keep the stored settings over the edit.
type SecretMerge struct {
	Incoming Settings
	Stored   Settings
}

type SettingsRepository interface {
	Upsert(ctx context.Context, notifierID uuid.UUID, settings Settings) error
	ListByNotifierIDs(ctx context.Context, notifierIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
}
```

A provider row is deleted with its parent by the `CASCADE` of the composite key, so the repository has no delete. `MergeStoredSecrets` returns an error because the webhook merges header values by name and refuses settings of another provider.

Each provider package holds its settings struct with plain fields, its GORM row type for `notifiers_<type>`, its repository through `database.Conn`, its validation, and its send. `Settings` is an interface with one method, `GetType()`, so the registry can check that a value belongs to the provider it is handed to. The registry is a map from `NotifierType` to `Provider`, built once in `module.New` from the seven constructors. It is the only place that names all seven, and an unknown type in a stored row is an error of the load, not a panic.

A provider encrypts its secrets with the `FieldEncryptor` it gets in its constructor before `Upsert`, and decrypts them inside `Send`, so a decrypted secret never leaves `Send`. `Send` returns a decryption error, which the `secret-handling` spec asks for. The `AfterFind` hook and the `storedFieldsDecryptionErr` field go.

The webhook URL and body template are not secrets, but releases before `2c0a2940` stored them encrypted. The webhook repository decrypts them when it loads a row; a value without the `enc:` prefix passes through `Decrypt` unchanged. When decryption fails, the loaded field is empty and the row carries the error, so a read shows the notifier with that field empty, as the `notifiers` spec says, and `Send` refuses with the error. Saving the notifier again writes the values in clear text.

The provider rules move with their code: `url_credentials.go` into a small `providers/redaction` package that the webhook, Discord, Teams and Mattermost providers share, the Bot API URL and proxy rules into `telegram`. The existing model tests move with them and keep testing those rules as unit tests, because they are pure functions with many cases.

The conversion between a provider's settings and its contract schema lives in `httpapi`, one file per provider, because a provider importing the generated types would import the package that imports it. The generated discriminator of the `oneOf` picks the file. The registry and these conversions are the only two places that list the providers, and both sit inside the module's family code. Task 6.1 amends "Engines and providers" in `docs/architecture.md` to say so: adding a provider also takes its conversion in the module's `httpapi`, because a provider cannot import the generated types, and the dispatch over the `oneOf` belongs to the family, which `httpapi` of the family's own module is part of.

*Rejected: generics over the settings type.* A registry of `Provider[S]` values of different `S` cannot sit in one map without an interface anyway, and the conversion to and from the contract types is per provider either way.

### 3. Every send has a context and a timeout

Each provider sends with the context `Notify` or the test gives it, and every HTTP client is built once per provider with `Timeout` set and the transport the provider uses today: the environment proxy, the Telegram proxy, the Mattermost TLS setting. The delivery wraps each send in `context.WithTimeout(DeliveryTimeout)`, which `internal/app` sets to 30 seconds, the timeout Slack and Mattermost already use. SMTP sends take the same context through `lib/mailer`. `ExternalEndpoints` in `internal/app/environment.go` gains the Slack API URL and the official Telegram Bot API URL, so `apptest` can point both at fakes.

### 4. The parent repository saves both rows in one transaction

`notifier.Service` opens one transaction per write. The parent repository inserts or updates the `notifiers` row, then hands the settings to the provider's repository. A list loads one page of parents, groups their IDs by type and calls `ListByNotifierIDs` once per type that occurs on the page, so a page costs at most eight queries.

An update whose settings have another type than the stored notifier is refused with 400 `invalid_request` before anything is written. The type never changes after creation, so the composite key of decision 6 needs no `ON UPDATE` action, and an update always merges into a provider row of the same type.

The parent repository is unexported and belongs to `notifier`, like every repository. `delivery` reaches the table through two exported methods of `notifier.Service`: `ListForDelivery(ctx, ids)`, which returns the notifiers with their settings in one batch per type, and `RecordSendOutcome(ctx, notifierID, sendErr)`. `testNotifier` calls `RecordSendOutcome` inside its own package. `last_send_error` is written only by `UPDATE notifiers SET last_send_error = ? WHERE id = ?`, never by saving the row. An update of the settings does not touch it, so the outcome of a delivery and the user's edit never overwrite each other.

The list is ordered by `name, id` with the opaque cursor of the workspace list. That cursor and `isForeignKeyViolation` move from `internal/modules/workspaces/internal/workspace` into `lib/database` as `database.NameCursor` and `database.IsForeignKeyViolation`, so both modules use one copy; the row of `database` in the table of `docs/architecture.md` grows to "the database pool, transactions, constraint errors and keyset cursors".

### 5. Delivery runs in the background of the module

`Notify` removes duplicate IDs, cuts the message to 2000 characters, and starts one goroutine on a `background.Group` that the module owns. That goroutine loads the notifiers through `notifier.Service.ListForDelivery`, then starts one goroutine per notifier on the same group. Each sends with its own timeout and records the outcome through `RecordSendOutcome`, then logs a failure with `notifier_id` and the category, as `SendNotification` does today. The context of every delivery is `database.WithoutTransaction(context.WithoutCancel(ctx))`. `WithoutCancel` keeps the end of the request or the backup that asked from cancelling the delivery, and keeps the logging attributes of `ctx`. It also keeps every other value, the caller's transaction included, so the new helper `database.WithoutTransaction` in `lib/database/transaction.go` removes it, and the load and the outcome `UPDATE` run on the pool. A unit test checks that a repository call through the returned context does not join the transaction. Callers call `Notify` only after their transaction commits, because a message cannot be taken back; task 2.4 checks each of the seven old call sites for an enclosing `Transaction` and moves the call after it. A notifier deleted before its delivery loads is skipped.

`internal/app` adds a shutdown step `notification deliveries` with `WaitForDeliveries` after `running tasks` and before `background loops`, so the deliveries of the last backups get the rest of the drain. A delivery still running when the drain ends dies with the process, as today. Console commands send nothing, and `WaitForDeliveries` returns at once when nothing was started.

*Rejected: keep `Notify` synchronous and put the timeout on each send.* A healthcheck pass would still wait up to 30 seconds per hanging notifier, and the verification already sends in the background for that reason.

*Rejected: a persistent outbox.* A delivery lost to a restart is lost today too, and an outbox needs a table, a worker and retry rules. The proposal leaves delivery guarantees out of scope.

### 6. The schema

In one goose file:

1. The seven provider tables are renamed to `notifiers_email`, `notifiers_telegram`, `notifiers_slack`, `notifiers_discord`, `notifiers_teams`, `notifiers_mattermost` and `notifiers_webhook`, with their primary keys and references renamed to the new prefix.
2. Repairs, before any constraint:
   - a notifier with an unknown type, or without a row in the provider table of its type, is deleted together with its rows in `database_notifiers`; it cannot load or send today;
   - a provider row whose notifier has another type is deleted;
   - a notifier without a workspace takes the workspace of a database that uses it, the earliest by database ID, and is deleted when no database uses it.
3. `notifiers.notifier_type` is renamed to `type`, becomes `text` and gets `ck_notifiers_type` over the seven values. `workspace_id` becomes `NOT NULL`. `uq_notifiers_id_type` is a unique constraint on `(id, type)`. `idx_notifiers_workspace_id_name_id` serves the list.
4. Every provider table gets `type text NOT NULL DEFAULT '<TYPE>'` with `CHECK (type = '<TYPE>')`, and its reference becomes `FOREIGN KEY (notifier_id, type) REFERENCES notifiers (id, type) ON DELETE CASCADE`. `CASCADE` stays, because both tables belong to the module.
5. Enum checks: `webhook_method` in `POST` and `GET`, `delivery_mode` of Mattermost over its two values, and the existing email `security` check renamed to the new prefix.
6. `notifiers_webhook.headers` becomes `JSONB` through `headers::jsonb`, with `'[]'` for an empty or `NULL` value. `accept_notification_types` is renamed to `accepted_categories` and becomes `text[]` through `jsonb_array_elements_text`, with `'{ALL}'` as default and a `CHECK` that every element is one of the category values.

Encrypted header values stay as they are inside the JSON, as the migration rules require. The down section reverses the renames, the types and the constraints, and serves development only.

*Rejected: keep the JSON in `TEXT`.* `docs/architecture.md` asks for `JSONB` for opaque data and `text[]` for lists, and the plan names both columns.

*Rejected: stop the migration on a notifier without its provider row.* An instance would not start for a notifier that cannot work anyway. Deleting it loses nothing that works.

### 7. Eight operations

| Operation | Method and path | Answers |
|---|---|---|
| `listNotifiers` | `GET /notifiers?workspaceId&limit&after` | 200 `NotifierPage`, 404 |
| `createNotifier` | `POST /notifiers` | 201 `Notifier`, 400, 403, 404 |
| `getNotifier` | `GET /notifiers/{notifierId}` | 200 `Notifier`, 404 |
| `updateNotifier` | `PUT /notifiers/{notifierId}` | 200 `Notifier`, 400, 403, 404 |
| `deleteNotifier` | `DELETE /notifiers/{notifierId}` | 204, 403, 404, 409 |
| `testNotifier` | `POST /notifiers/{notifierId}/test` | 200 `NotifierTestResult`, 404 |
| `testNotifierSettings` | `POST /notifier-tests` | 200 `NotifierTestResult`, 400, 404 |
| `transferNotifier` | `POST /notifiers/{notifierId}/transfer` | 204, 403, 404, 409 |

`Notifier` has `id`, `workspaceId`, `name`, `lastSendError` and `settings`. `NotifierSettings` is a `oneOf` of seven schemas with the discriminator `type`. Each provider schema lists its fields with the names the frontend uses today, `type` included, except the webhook's accepted categories, which become `acceptedCategories` with items of the enum schema `NotificationCategory`, per decision 1. Its secrets are `writeOnly` and optional; `createNotifier` requires them through the provider's validation, which answers 400 `invalid_request` with the field in `errors`. A webhook header is `{name, value}` with `value` `writeOnly`. The SMTP security of the email schema is the enum `SmtpSecurity` with `TLS`, `STARTTLS` and `NONE`, like every enum of the contract; the conversion maps it to the lowercase values the column and its `CHECK` keep. `NotifierCreate` adds `workspaceId` to `name` and `settings`; `NotifierUpdate` has `name` and `settings`. The Telegram topic `threadId` stays an integer, because Telegram issues it; the api-design skill and the `databasus-id-is-uuid` rule of `backend/api/vacuum.yaml` exempt an integer ID of another system from `format: uuid`.

`testNotifierSettings` takes `workspaceId`, an optional `notifierId` whose stored secrets fill the omitted ones, and `settings`. It is a resource of its own, `/notifier-tests`, because `POST /notifiers/test` would sit next to `/notifiers/{notifierId}` and read like a notifier with the ID `test`. It needs `ActionView` in `workspaceId`. A `notifierId` that does not exist or belongs to another workspace answers 404 `not_found` before anything is sent, so no stored secret leaves its workspace.

`NotifierTestResult` is `{isDelivered, error}`. A failed test is the answer to the question the user asked, not a broken request, and the channel's error is what the user needs to fix the settings. The frontend shows `error` as the channel's message under a translated heading.

The contract gains `in_use` with status 409 in `ProblemCode` and `apierror.Code`.

*Rejected: a failed test as a `Problem` with 502.* `detail` is meant for logs, so the frontend would have to show it anyway, and the frontend retries 502.

### 8. Access goes through `Authorize`

Every operation on a notifier loads it first and answers 404 `not_found` when it is missing. An operation addressed by a notifier ID replaces a `not_found` from `Authorize` with the same notifier `not_found` Problem it gives for a missing notifier, so a foreign notifier and a missing one answer with the same body apart from the ID, as the `workspace-access` spec asks. Then it calls `workspaces.Authorize` with `ActionView` for reads and tests, and `ActionEditContent` for writes. The list and create authorize the `workspaceId` of the request. A transfer authorizes `ActionEditContent` in the source and then in the target, so a user outside the source learns nothing about the notifier, and a viewer of the target gets 403. Refusals are the Problem values of `Authorize`, returned unchanged.

### 9. A transfer asks the databases through an event

`TransferNotifier` takes a `NotifierTransfer` and runs in one transaction: it checks access, publishes `NotifierTransferring` to the subscribed handlers in order, updates `workspace_id` with one `UPDATE`, and writes the audit event with both workspace names. `internal/app` subscribes the old `databases` service through the bridge. Its handler counts the databases that use the notifier and are not in the target workspace, and returns the Problem `in_use` when there is any.

The transaction first locks the notifier row with `SELECT ... FOR UPDATE`. The insert into `database_notifiers` takes `FOR KEY SHARE` on that row for its reference check, and the two conflict, so an attach whose transaction is still open either commits before the count sees it or waits until the transfer commits. The old attach checks the notifier's workspace before its own transaction, so an attach that checked, then waited for a transfer, still inserts. Milestone 8 moves that check into the attach's transaction.

The database transfer of the old backup configs, logical and physical, moves the database first and then its notifiers. With the database already in the target workspace, the handler lets its notifiers through, so the parameter `transferingWithDbID` goes away. A notifier that another database outside the target workspace still uses is refused with `in_use`; the transfer skips it, so it stays in its workspace and keeps serving that database, as the old transfer did. Every other error of a notifier's transfer is returned instead of ignored. The transfer is still not one transaction; milestone 9 rebuilds it.

*Rejected: a `workspace_id` column in `database_notifiers` with composite keys to both parents.* The schema would hold the rule itself, but a database transfer would have to defer two keys to the end of a transaction that the old code does not have, and the table belongs to `databases`, which moves later.

### 10. Deletion relies on the reference

`deleteNotifier` loads, authorizes and deletes the parent in one transaction; the provider row goes by `CASCADE`. A foreign key violation from `database_notifiers` becomes the Problem `in_use`. `NotifierDatabaseCounter`, `SetNotifierDatabaseCounter` and `DatabaseService.GetNotifierAttachedDatabasesIDs` are deleted.

`OnWorkspaceDeleting` deletes the workspace's notifiers inside the deletion's transaction. `internal/app/wiring.go` subscribes the module's handler where the old service's handler stood, after `databases`, so the order of the `workspace-deletion` spec stays.

Audit events go through `lib/audit` in the transaction of the write, with today's messages: created, updated, renamed, deleted and transferred.

### 11. The old code reaches the module through a bridge

`features/notifiers` keeps:

- `Install(module)` and `GetModule()`, with a module built from the old access points in a `testhelpers` build, as `workspaces_bridge` does;
- the test helpers of decision 13, behind the `testhelpers` tag.

The old `databases.Database` replaces `Notifiers []notifiers.Notifier` with `Notifiers []DatabaseNotifier`, a type of its own with `gorm:"-"` and the JSON fields `id`, `workspaceId`, `name`, `notifierType` and `lastSendError` that `/databases` returns today. Its repository reads and writes the IDs in `database_notifiers` with plain queries instead of the many-to-many association, and its service fills the other fields from one `GetNotifiers` call per request, a list included. The create and update checks take the workspace from `GetNotifiers`. `UpdateDatabaseNotifiers` takes IDs. The routes `/databases/notifier/:id/is-using` and `.../databases-count` check access with `GetNotifiers` and `workspaces_bridge.Access.IsActorAllowed`.

The five sender interfaces become one method, `Notify(ctx, notifierIDs, notification)`, and the seven loops become one call with `database.GetNotifierIDs()`. The testify mocks follow the new method. `telemetry` calls `GetNotifierTypes`.

`temp/refactor/plan.md` gets a row in the table of temporary relaxations: the old features notify and read notifiers through the bridge, and the old `databases` keeps `database_notifiers` without the module prefix, until milestone 8.

*Rejected: return only `notifierIds` from `/databases`.* The database screens would then load every notifier of the workspace to show three names, and the `/databases` shape belongs to milestone 8, which describes those operations in the contract.

### 12. Settings versus last send error in a test

`testNotifier` sends with the stored settings and records the outcome through `RecordSendOutcome` of decision 4. `testNotifierSettings` merges, validates and sends without writing anything. Neither runs in a transaction while it waits for the channel.

### 13. The old test helpers write SQL

`notifiers.CreateTestNotifier(workspaceID)` inserts a webhook notifier pointing at the shared webhook stub into `notifiers` and `notifiers_webhook`, and returns a `notifiers.Notifier` of `module.go`. `RemoveTestNotifier` deletes the parent. `GetWebhookStub` stays. The 19 places that build `Notifiers: []notifiers.Notifier{...}` switch to `[]databases.DatabaseNotifier{{ID: notifier.ID}}` through `gofmt -r` rules where the shape allows and by hand elsewhere.

*Rejected: let the helpers call the module.* The module needs an actor and a workspace member, which the old tests do not set up for notifiers, and the SQL helpers of `users` and `workspaces` already set the pattern.

### 14. The frontend calls the generated client

`entity/notifiers/api` is rewritten on the generated client, with `notifierApi.listAllNotifiers(workspaceId)` following `nextCursor` with `limit` 100, because the database form and the notifier screen need every notifier of the workspace. The hand-written models of `entity/notifiers` are deleted, apart from the label keys, the logo lookup and the validation functions, which move onto the generated types. The hand-written `NotificationType` enum and its label keys become the generated `NotificationCategory` and `NOTIFICATION_CATEGORY_LABEL_KEYS`, and the webhook form edits `acceptedCategories`. The edit component keeps one sub-form per provider and switches on `settings.type`. The type selector is offered only for a new notifier; a saved notifier shows its type as text. Secrets are empty in the form of a saved notifier, and an empty secret is left out of the request. The test button calls `testNotifierSettings` with the notifier's ID when editing and shows `error` of a failed result.

The database screens read their own `DatabaseNotifier` type from `entity/databases`, since `/databases` is not in the contract yet. The new code `in_use` gets a `translateApiError` key and a translation in every dictionary.

Screen tests on `mockApi` cover: the edit form of a saved notifier offers no type selector, saving a renamed Slack notifier sends no bot token, a webhook header keeps its name and sends no value, a failed test shows the channel's error, and deleting a used notifier shows the `in_use` text. The existing Telegram tests move to the new types.

### 15. Tests and terms

The tests in `internal/modules/notifiers` use `apptest` and the contract-checking client, one file per concept: notifier lifecycle, secrets, the fixed type, list, deletion, transfer, tests, delivery. Each provider has a delivery test against a fake: an `httptest` server for the webhook, Discord, Teams, Mattermost, Slack and Telegram, the last two through the new `ExternalEndpoints`, and a Mailpit container from `util/testing/containers` for email, because the email provider speaks SMTP itself and the mail fake of `apptest` stands in only for the instance's own mailer. A delivery test waits for `lastSendError` or the fake's request through `testwait.Until`. `apptest` gets `CreateNotifier(tb, session, workspaceID, settings)`, an option `WithNotificationDeliveryTimeout` for the scenario of an endpoint that never answers, and `App.AttachNotifierToDatabase(tb, apptest.NotifierAttachment{DatabaseID, NotifierID})` that inserts into `database_notifiers` through `ExecuteSQL`, because no contract operation attaches one yet. It also gets `WithNotifierEndpoints` for the Slack and Telegram fakes, `App.Notify`, which hands a notification to the module the way the old senders do, `InsertDatabase`, which writes a PostgreSQL database row for the deletion and transfer tests, `CallOldAPI` and `CreateLocalStorage` for the database transfer through the old backup config route, and `EncryptSecret` for rows inserted in the stored shape. The shutdown scenario runs through the server test in `internal/app`.

The deletion tests of the `workspaces` module switch their helpers `createWebhookNotifier` and `countContent` to `apptest.CreateNotifier` and `listNotifiers`.

The requirement "An upgrade keeps every notifier's secrets" is tested in two parts. The migration test of task 2.2 inserts rows of the old shape with encrypted secrets, applies the migration and shows that every secret column holds the same bytes. The API test of task 3.3 inserts a webhook row with an encrypted header value in the stored shape through `ExecuteSQL` and shows that a test send carries the decrypted value.

The Telegram save and read tests of the old `controller_test.go` (`Test_SaveTelegramNotifier_WithBotAPIURL_ReturnedInClearAndClearedByUpdate` and `Test_SaveTelegramNotifier_WithUnusableBotAPIURL_RefusedAndStoredNotifierUnchanged`) and its Telegram test-send tests are ported onto the new operations from the file as it stands before this change, because task 2.4 deletes it with the old controller.

`docs/glossary.md` gets three terms:

- "notification": a message about an event of a database, with a category, a heading and a text, which Databasus sends to the notifiers of that database;
- "notification category": what a notification is about, such as a failed backup, which a webhook notifier can filter on;
- "delivery": one send of one notification to one notifier, ended by a timeout, whose outcome becomes the notifier's last send error.

## Risks / Trade-offs

- [The migration deletes notifiers it cannot repair] → They have no provider row or an unknown type, so they cannot load today. The release notes name the rule.
- [A delivery outlives its sender] → It holds no transaction and no lock, and the shutdown step waits for it.
- [Old tests asserted synchronous sends through testify mocks] → The mocks record `Notify`, which the callers still call synchronously; only the module's real delivery is asynchronous.
- [The settings union widens the frontend diff] → The sub-forms keep their fields and names; only the path to them changes from `notifier.slackNotifier` to `notifier.settings`.
- [Seven providers in one change] → They share one interface and one repository shape, and each keeps its existing unit tests for its rules.

## Migration Plan

1. The migration runs at startup before the server accepts requests. The notifier tables are small, so the renames, repairs and conversions finish in milliseconds.
2. The upgrade is one-way, as `docs/architecture.md` says for every migration. Recovery from a failed upgrade means restoring the internal database from a backup taken before it.
3. The upgrade test writes a notifier of every type through the previous release, attaches all of them to its database, and checks after the upgrade that each is listed with its non-secret settings and still attached.
