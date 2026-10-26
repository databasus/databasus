## Why

`notifiers` is the first of the three provider families in `docs/architecture.md` to move into the target layout, and storages and engines will copy its registry. Today it is an old feature with a `switch` on the notifier type in five places, a 125-line `NotifierRepository.Save` that writes seven GORM associations, controllers that answer every error as 400 with the internal error text, and a response that echoes the request body instead of the saved notifier. Every sender of a notification loops over the database's notifiers itself: the logical backuper, the physical backuper twice, the WAL supervisor twice, the healthcheck and the verification run. Two defects come from this shape. A send re-saves the whole notifier with its provider row after the network call, so an edit the user saves while a notification is on its way is overwritten. And the webhook, Discord and Teams providers send without a timeout, while backups and healthchecks send synchronously, so a hanging endpoint holds up the sender. Milestone 6 of the backend rewrite plan moves the feature into `internal/modules/notifiers` with its operations in the contract, as one change.

## What Changes

- The module `internal/modules/notifiers` appears with `module.go`, `Deps`, `New` and its code under `internal/`. It owns notifiers, their provider settings, test sends and delivery. Its services open transactions through the `database.Transactor` from `Deps`, check access through `workspaces.Authorize` and write audit events through `lib/audit` in the same transaction.
- Each of the seven providers (email, Telegram, Slack, Discord, Teams, Mattermost, webhook) becomes a package under `internal/providers` that holds its settings, its validation, its send and the repository of its own table. A registry keyed by type is the only place that knows every provider. The `switch` statements, the seven `Preload` calls and the GORM associations are deleted.
- `module.go` offers `Notify(ctx, notifierIDs, notification)`: one call that loads the notifiers in one batch, delivers to each in the background with a timeout, and records each result with a single-column `UPDATE` of `last_send_error`. The seven loops in the old code are replaced by this call. It also offers a batch lookup of notifiers by ID, the notifier types for telemetry, the handler of `workspaces.WorkspaceDeleting`, and the event `NotifierTransferring`.
- Every provider gets a send timeout, and no provider sends with `context.Background()`.
- 8 operations enter `backend/api/openapi.yaml` under the tag `notifiers`: list with a cursor, create, read, update, delete, test a saved notifier, test unsaved settings, and transfer to another workspace. The provider settings are a `oneOf` with the discriminator `type`. Secrets are `writeOnly`, and an omitted secret keeps its stored value on update, a webhook header value included. A response carries the saved notifier.
- A notifier's type is fixed when it is created. An update with settings of another type answers 400 `invalid_request`, and the edit form shows the type of a saved notifier without a selector. The form offers a type change today, but the backend never applies it, so nothing that works is lost.
- Deleting a notifier that a database uses answers 409 `in_use`, from the database's `RESTRICT` reference instead of a counter registered in code. Transferring a notifier answers 409 `in_use` while a database outside the target workspace uses it. The contract gains the code `in_use`.
- The old features reach the module through a bridge in `features/notifiers`: the installed module, and the test helpers behind the `testhelpers` tag. The old database model keeps its `notifiers` field in the `/databases` API, filled from the module's batch lookup instead of a GORM many-to-many association. The database transfer moves the database before its notifiers. It skips a notifier that another database still uses, which stays in its workspace, and returns every other error instead of ignoring it.
- The provider tables become `notifiers_<type>`. The migration replaces `notifier_type varchar(50)` with `type` under a `CHECK`, adds a composite foreign key `(notifier_id, type)` from every provider table, a `CHECK` on every enum column, `NOT NULL` on `workspace_id`, and turns the webhook headers into `JSONB` and its accepted notification types into `accepted_categories text[]`. Rows that would break the new constraints are repaired first.
- The frontend calls the 8 operations through the generated client and types. The hand-written models of `entity/notifiers` are deleted, the edit and show screens read the settings union, and the new code gets a translation in the six interface dictionaries.
- The tests of `notifiers` become module API tests through `apptest` and the contract-checking client, with a fake endpoint per provider. The upgrade test writes one notifier of every type before the upgrade and checks them after it.
- **BREAKING** for users' scripts: the notifier operations change shape. The list takes `workspaceId` instead of `workspace_id` and returns a page with `items` and `nextCursor`. A notifier carries `type` and `settings` instead of `notifierType` and one field per provider. Saving an existing notifier is `PUT /notifiers/{notifierId}` instead of `POST /notifiers` with an ID. The test of unsaved settings moves from `/notifiers/direct-test`. A test send answers its outcome in the body. Errors are `Problem` bodies.
- **BREAKING** for the schema: seven tables are renamed, one column is replaced, two columns change type, and constraints are added.

## Capabilities

### New Capabilities

- `notifiers`: creating, changing, testing, transferring and deleting a notifier, its secrets, and how a notification is delivered and its failure recorded.

### Modified Capabilities

- `workspace-access`: the notifier operations now answer access refusals with 404 `not_found` and 403 `permission_denied`, like the workspace operations.
- `upgrade-test`: the test writes one notifier of every type, attached to the database, instead of one webhook notifier.

## Out of scope

- Moving `databases`, `backups`, `healthcheck`, `verification` or `telemetry` into modules. They call the module through the bridge and keep their own errors and routes, including `/databases/notifier/{id}/is-using` and `/databases/notifier/{id}/databases-count`.
- Renaming `database_notifiers`. It belongs to `databases` and is renamed in milestone 8.
- Running the database transfer in one transaction. The transfer belongs to the backup configs and is rebuilt in milestone 9; this change only reorders it and stops ignoring the notifier errors other than `in_use`.
- Delivery guarantees beyond today's: no retry, no outbox, no persistence of a notification across a restart.
- New providers, new notification categories and changes to the texts of notifications.
- Changing who may do what. A viewer can still send a test notification, as today.

## Impact

- Backend: new `backend/internal/modules/notifiers`; `backend/api/openapi.yaml`; `backend/internal/lib/apierror` (the `in_use` code); `backend/internal/app` (construction, routes, the `WorkspaceDeleting` and `NotifierTransferring` subscriptions, the shutdown step for deliveries) and `backend/internal/app/apptest`; `backend/internal/features/notifiers` reduced to the bridge and the test helpers; the notification senders and their mocks in `features/backups`, `features/healthcheck` and `features/verification`; the notifier field, repository and transfer of `features/databases`; the database transfer in `features/backups/config/logical` and `physical`; `features/telemetry`; the test files that use `notifiers.CreateTestNotifier`, listed in design.md Context; `backend/e2e/upgrade`.
- Frontend: `frontend/src/entity/notifiers`, the screens of `features/notifiers`, the notifier parts of `features/databases`, the six dictionaries in `frontend/src/shared/i18n/locales`, and the tests of the Telegram screens.
- Migrations: one migration renames the provider tables, repairs rows, converts two webhook columns and adds the constraints.
- Running instances: notifiers keep their settings, secrets and database attachments. A provider row whose type differs from its notifier's is deleted. A notifier without its provider row cannot send or even load today, so it is deleted together with its database attachments. A notifier without a workspace takes the workspace of a database that uses it, or is deleted when none does.
- Documents: `temp/refactor/plan.md` (the checklist and the table of temporary relaxations), `docs/glossary.md` (notification, notification category, delivery), `docs/architecture.md` (the row of `lib/database`).
- Rules the change answers to: the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
