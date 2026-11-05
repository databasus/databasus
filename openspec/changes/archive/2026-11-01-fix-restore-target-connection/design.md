## Context

See proposal.md for why. The current state that shapes the approach:

- The restore dialog builds its target as a copy of the backed-up database with only the host, user and password cleared (`frontend/src/features/restores/ui/RestoresComponent.tsx:34-43`). The copy keeps the database's ID, port, database name, SSH tunnel, TLS certificates and engine flags.
- The dialog reuses the database form through `EditDatabaseSpecificDataComponent` with `isSaveToApi={false}` (`RestoresComponent.tsx:156-168`). Only the PostgreSQL logical form receives `isRestoreMode` (`frontend/src/features/databases/ui/edit/EditDatabaseSpecificDataComponent.tsx:134-155`). The MySQL, MariaDB and MongoDB forms do not know they serve a restore.
- The forms decide "new or saved database" by `database.id`: the password is optional for a saved one (`EditPostgreSqlLogicalSpecificDataComponent.tsx:371`, `EditMySqlSpecificDataComponent.tsx:188`), SSH secrets show as stored (`EditPostgreSqlLogicalSpecificDataComponent.tsx:809-813`), the SSL mode is not suggested (`:127`), and a certificate the settings hold shows masked (`:299`).
- The connection test sends `databaseId: database.id` (`frontend/src/features/databases/ui/edit/toDatabaseSettingsTarget.ts:14`). With it the server fills omitted secrets from that database and needs the member role (`backend/api/openapi.yaml:1807-1812`).
- Every form shows its submit button only after a successful test (`EditPostgreSqlLogicalSpecificDataComponent.tsx:774-797`, `EditMySqlSpecificDataComponent.tsx:382-403`, the same in the MariaDB and MongoDB forms).
- The restore start builds the target only from its request body (`backend/internal/modules/backups/internal/httpapi/restore_target.go:13-110`). It connects and detects the version before answering, and refuses an unreachable target with `400 database_connection_failed` (`backend/api/openapi.yaml:2687-2692`). The problem detail carries the server's reason (`backend/internal/modules/backups/internal/restore/errors.go:46-52`), and the interface shows that detail (`frontend/src/shared/i18n/translateApiError.ts:66-70`).
- The dialog closes only on an accepted start and shows a refusal inline (`RestoresComponent.tsx:109-116`, `:170-177`). The existing screen test already pins that a refusal keeps the entered host (`RestoresComponent.test.tsx`, "shows $code inline in the target form").
- When the dispatcher receives a submission with `isSaveToApi={false}`, it first replaces its own draft with the submitted database (`EditDatabaseSpecificDataComponent.tsx:58-63`). Each form resets its pending and test state whenever its `settings` input changes (`EditPostgreSqlLogicalSpecificDataComponent.tsx:266-275`, `EditMySqlSpecificDataComponent.tsx:167-174`, `EditMariaDbSpecificDataComponent.tsx:173`, `EditMongoDbSpecificDataComponent.tsx:188`).

Constraints come from root `AGENTS.md` and `frontend/AGENTS.md`: component structure, the FSD boundaries of `docs/architecture.md#frontend`, screen tests through the HTTP boundary only, and no copy outside the dictionaries.

## Goals / Non-Goals

**Goals:**

- One rule decides what the target form starts with: the defaults of a new database of the backup's engine.
- All four logical engines follow the same restore flow.
- A pending start cannot be submitted twice, and a refused start leaves the form editable with every entered value.

**Non-Goals:**

- Changing how saved databases are edited, created or tested.
- Moving the database form or the restore dialog to their final FSD owners. The baseline entries for `features/restores/ui/RestoresComponent.tsx` stay unchanged.

## Decisions

### The target draft is a new database of the backup's engine

The dialog builds its draft with an empty ID, the source's `workspaceId` and `createDefaultDatabaseSettings(database.settings.type)` (`frontend/src/entities/databases/model/createDefaultDatabaseSettings.ts:73-75`). This is the same shape the creation wizard starts from (`frontend/src/features/databases/ui/CreateDatabaseComponent.tsx:37-44`). The draft builder is `createRestoreTargetDraft` in `frontend/src/features/restores/ui/createRestoreTargetDraft.ts`, next to `toRestoreTarget.ts`, as its own logical unit: both translate between a database form and a restore target.

Every `database.id` check in the forms then takes its "new database" branch: the password becomes required, no secret shows as stored, the SSL mode is suggested by host, and a test request, if one were made, would carry no `databaseId`.

Alternatives rejected:

- Clearing more fields of the copied database. The current code already does this, and the tunnel, certificate, port and database name leaked anyway. Every future settings field would leak the same way.
- Keeping the ID and adding `isRestoreMode` branches to each `database.id` check. That touches five checks in four forms, and one missed branch reopens the leak. The ID is the wrong fact: the target is not the saved database.
- Keeping convenient fields such as the database name or a non-default port. They spare some typing when the target mirrors the source, but they break the single rule. A field the user did not type can then point a destructive restore at an unintended database.

### Restore mode starts the restore without a separate test

In restore mode each logical form renders the submit button directly, enabled once its required fields are filled. The dispatcher forwards `isRestoreMode` to the MySQL, MariaDB and MongoDB forms as it already does to PostgreSQL. In every form, restore mode never calls the settings test.

The restore start is already the authoritative check: it connects, detects the version and returns the server's reason. A separate test would duplicate that check, and the member-only test would keep a viewer out of a flow the `workspace-access` spec grants them.

Alternatives rejected:

- Keeping the test step on a new-database draft. It works for members, but a viewer would still get `403`, and a passing test would add nothing the start does not check again.
- Letting a viewer test unsaved settings without `databaseId`. It changes the API access rule, its spec and backend tests to restore a step the start already performs. The user chose to drop the step.
- A dedicated restore target form per engine. It would duplicate four connection forms, their SSH and TLS handling and their validation.

### The form awaits its caller and stays pending until the start answers

The `onSaved` callback may return a promise. In the `isSaveToApi={false}` path a form sets its saving state, awaits `onSaved` and clears the state afterwards, so the button shows loading and accepts no second submission. The restore dialog returns the promise of its start, which already turns a refusal into inline feedback and never rejects.

For the pending state to survive, the dispatcher stops replacing its draft before it hands an `isSaveToApi={false}` submission to the caller. Otherwise the replaced `settings` input triggers the form's reset effect, which clears the pending state mid-request. The replacement serves only the limited-user dialog of the saving path. The other `isSaveToApi={false}` caller, the creation wizard, unmounts the form as soon as it receives the submission (`CreateDatabaseComponent.tsx:157-160`), so it does not notice the change.

Alternatives rejected:

- A pending flag owned by the dialog and passed down as a prop. It adds a prop to four forms that already own an `isSaving` state for the same purpose.
- Guarding the second click only with a ref inside the dialog. It would block the request, but the button would look ready while the start is pending.

### Refusal feedback stays with the dialog

The dialog keeps its inline alert and its rule that only an accepted start closes the form. Entered values stay in the form's own state, because nothing replaces its `settings` input during or after a submission. No copy changes: the submit label and the target description already exist in all six dictionaries.

## Risks / Trade-offs

- [The start now carries the connection check, which can take until the driver's connection timeout] → The button shows loading for the whole start, and the existing refusal message names the server's reason.
- [A user restoring into a mirror of the source types the port and database name again] → Accepted as the cost of one rule, as above. The dialog still keeps the last entered target while it stays open.
- [Dropping the dispatcher's draft replacement could affect the creation wizard] → The wizard unmounts the form on submission. No wizard screen test exists yet, so task 3.2 adds one that submits the settings step and checks that the wizard advances with the submitted settings.
- [Two quick clicks before the loading state renders] → A screen test clicks twice and expects exactly one start request. If React batching lets the second click through, the form also ignores a submission while one is pending.
