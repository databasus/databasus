## Context

See proposal.md for motivation. The state this design starts from, with paths relative to `frontend/src`. The line references are to `features/databases/ui/edit`, which `move-database-connection-form` moves to `features/database-connection/ui` without changing content:

- The form has three scenarios and infers them. `EditPostgreSqlLogicalSpecificDataComponent.tsx:127` sets `hasUserChosenSslMode` from `database.id`, line 375 makes the password optional for the same reason, and `toDatabaseSettingsTarget.ts` names the database when an ID exists. `features/restores/ui/createRestoreTargetDraft.ts` therefore needs an empty ID to stay safe.
- `EditPostgreSqlLogicalSpecificDataComponent.tsx:207-212` accepts a successful test with no check that the settings are still the tested ones. The other four forms do the same.
- `EditDatabaseSpecificDataComponent.tsx:58-80` saves the database, then checks the limited user, and calls `onSaved` only on some paths. Closing the dialog at line 98, going back at line 105 or a failed check at line 77 never reports the save to the page.
- `pages/databases/ui/CreateDatabaseComponent.tsx:159-174` renders the form on its step. The form holds typed input locally, so Back loses it, and each mount resets `hasUserChosenSslMode`.
- `EditDatabaseBaseInfoComponent.tsx` calls `changeDatabaseType` on every tile click. Choosing another engine and returning resets the settings, and clicking the highlighted PostgreSQL tile on a physical database resets them too.
- The certificate section reads the `settings` prop (`EditPostgreSqlLogicalSpecificDataComponent.tsx:300`), so a typed certificate in a reopened wizard step or restore target appears as stored. A saved certificate also keeps its mask after SSL is turned off and on, although the draft no longer holds it. The save then removes the certificate, and `backend/internal/modules/databases/internal/providers/postgresql_tls_settings.go:34` keeps only an omitted client key.
- `EditPostgreSqlPhysicalSpecificDataComponent.tsx:576-579` resets only the approval on an SSH change, while line 129 shows the helper that also clears the failed result.
- `EditMySqlSpecificDataComponent.tsx:351` and the matching MariaDB and MongoDB sections show exclusions in restore mode. `features/restores/ui/toRestoreTarget.ts:36-73` sends none of them.
- `EditPostgreSqlLogicalSpecificDataComponent.tsx:116-124` opens Advanced for saved values but ignores restore options.
- The MySQL and MariaDB forms differ only in type names, parser, label and two MariaDB options. Their parsers differ in accepted schemes, the format hint and comments. MariaDB has no parser test file.
- `entities/databases/api/databaseApi.ts:50-75` already selects the editable fields and omits empty secrets, for any object of the right shape.
- `features/backups/logical/ui/EditLogicalBackupConfigComponent.tsx:237` and `features/backups/physical/ui/EditPhysicalBackupConfigComponent.tsx:291` branch on `database.id`, so the wizard cannot drop `Database` for these steps until milestones 6 and 7.
- `features/edit-storage` and `features/edit-notifier` already separate draft policy from `shared/lib/RequestRevision.ts`, as `docs/architecture.md` ("Request identity is shared; acceptance policy is local") requires. `useStorageDraft` guards late answers with `isCurrentSession` and a context revision.

The approach is bound by root `AGENTS.md` (logical units, naming, comments, no compatibility shims), `frontend/AGENTS.md` (component structure, AntD, feedback and localization skills, generated aliases with their owner) and `docs/architecture.md#frontend` (public APIs, no same-layer imports, state near its lifecycle, cleanup invalidates late results).

## Goals / Non-Goals

**Goals:**

- A scenario's rules come from its draft type, so an impossible combination does not compile.
- One owner holds each draft, and containers own saving and navigation.
- Every behavior change is pinned by a test that fails on the code before it.

**Non-Goals:**

- A shared form engine. `useStorageDraft` remains a storage policy, and this hook is a smaller database policy beside it.
- Restructuring the backup config editors, the restore list or `LogicalBackupsComponent` beyond one prop.

## Decisions

### 1. Three draft types discriminated by `scenario`

```
DatabaseCreateDraft  scenario: create   workspaceId, name, notifierIds, settings, hasUserChosenSslMode
DatabaseEditDraft    scenario: edit     workspaceId, databaseId, name, notifierIds, savedSettings, settings, hasUserChosenSslMode
RestoreTargetDraft   scenario: restore  settings (no physical), hasUserChosenSslMode
```

`entities/databases` holds the types, because the form, the page and `features/restores` all name them, while instances stay in the containers. A new `toDatabaseSettingsTarget` there accepts only the create and edit drafts, and it replaces the feature's private function when the old forms are deleted, so a restore target cannot produce a test request or name a database. The create and edit drafts are structurally valid `DatabaseCreate` and `DatabaseUpdate` values, so containers pass them directly to `databaseApi`, whose tests cover payloads and secret omission.

`hasUserChosenSslMode` starts false for create and restore, and true for edit. `changeDatabaseCreateDraftType` resets the settings to the new type's defaults and sets it back to false, so the new type derives its SSL mode again. The wizard calls it only when the type differs at Continue, which replaces `changeDatabaseType`.

Rejected: one draft with an optional `databaseId` and flags. It keeps the combination that caused the leak legal and leaves the guarantee to a test. Plain `settings` plus separate scenario props was also rejected, because the props would need to stay consistent by convention. `toDatabaseCreate` and `toDatabaseUpdate` were rejected because `databaseApi` already does their work. Placing the create and edit drafts in the feature was rejected, because `toDatabaseSettingsTarget` is a domain conversion that the page's limited user form also needs.

### 2. The form is controlled and containers own the work

```
                 pages/databases (composes everything)
        +----------------+----------------------+
        |                |                      | render prop
        v                v                      v
     wizard       settings section     LogicalBackups -> Restores
   CreateDraft       EditDraft           RestoreTargetDraft,
   Back/Continue   Cancel, PUT,          restore options,
   limited user    limited user dialog   startRestore
        |                |                      |
        +----------------+----------------------+
                         | draft, onDraftChange, onSubmit,
                         | submitLabel, secondaryActions
                         v
          features/database-connection
```

`DatabaseConnectionForm` receives the draft, reports changes, and awaits `onSubmit` with a draft snapshot. A rejected `onSubmit` shows its translated reason inline and keeps the input. `secondaryActions` holds Cancel or Back, so the form knows no navigation. For a `RestoreTargetDraft` the form's props also require `restoreOptions` and `onRestoreOptionsChange`. `onConnectionErrorChange` stays, because the wizard widens itself for a physical diagnostic.

Rejected: an uncontrolled form with `initialDraft`. Going back from the connection step would still lose input. Separate field sections composed by each container were also rejected, because three containers would then repeat the wiring of five engine forms.

### 3. Containers replace a draft only while the form is unmounted

The wizard's steps and the section's limited user dialog already unmount the form. A new mount starts without test approval, so a draft a container replaces can never inherit one. The section therefore rebases its draft on the save's answer only when the dialog opens and unmounts the form, or when the section closes.

Rejected: a revision field in the draft that containers increment, and a remount key. Both add a rule every container must follow, and neither is needed.

### 4. A test hook with two kinds of change

`useDatabaseConnectionTest` owns `isTesting`, `isTested`, the failed result and three `RequestRevision` values, for connection settings, test requests and context. A connection change invalidates the settings revision and clears the failed result. Changes such as schema selection, exclusions and user mapping go through a second setter that leaves approval alone, as they do today. Session changes and unmount invalidate pending work. Physical `backupType` counts as a connection change, as it does today.

Rejected: comparing tested and current values. `docs/architecture.md` requires a change away and back to withdraw approval. Copying `useStorageDraft` was also rejected, because it owns saving, create-then-update and baselines that the containers own here. Plan item 5.3 mentions a name change, but the name is not on this form, so only the second setter applies.

### 5. Inputs are disabled during a submission

The form wraps its fields in AntD `ConfigProvider` with `componentDisabled` while `onSubmit` is pending. In the settings section that promise covers the `PUT` and the check right after it. When it settles, the form has either been refused and stays with its input, or it unmounts because the section closed or the dialog opened. Input typed while it is pending has nowhere to go, so the form does not accept it.

Rejected: the snapshot merge of `useStorageDraft`. It serves an editor that stays open after a save. Leaving fields enabled was also rejected, because input typed during a successful save would silently disappear.

### 6. The section reports a save at once and ignores late answers

A page-local `EditDatabaseConnectionSection` runs the `PUT` and calls a new `onDatabaseSaved`, which updates the page and leaves the section open. It then checks the limited user, still inside the pending `onSubmit`:

- If no limited user is advisable, it closes.
- If the check fails, it shows the translated reason through context-bound feedback and closes.
- If a limited user is advisable, it rebases its draft on the save's answer and shows the existing dialog, which unmounts the form.

Inside the dialog, `onLimitedUserNotSuggested` and a confirmed skip close the section. A failed second check keeps the dialog with its choices, as it does today. Close and Go back return to the form with the saved values. A created limited user fills the draft, and the user tests and saves again, as today.

The section captures the session and a mount revision before the `PUT`, and it checks them after each answer, so a late answer neither calls `onDatabaseSaved` nor opens the dialog. The page gives the section `key={database.id}`, so selecting another database unmounts it.

Rejected: reporting the save only when the dialog ends. That is the current defect. Keeping the section open after a failed check was also rejected, because the settings are saved and the suggestion is only advice.

### 7. MySQL and MariaDB share one editor and parser

`EditMySqlCompatibleConnectionComponent` reads `settings.type` for the label and for the MariaDB options, and it passes the type to `MySqlCompatibleConnectionStringParser.parse`. That parser owns the scheme list and format hint, and it keeps the MySQL and new MariaDB test suites. The row spacing difference between the two forms is dropped. The word "family" is not used, because `docs/architecture.md` uses it for backend registries.

Rejected: a separate descriptor object. Its fields already have owners in `settings.type`, `DATABASE_TYPE_LABEL_KEYS` and the parser. Keeping two thin wrappers was also rejected, because they would add files with no behavior. Merging MongoDB was rejected too, because SRV and `directConnection` make it a different grammar and form.

### 8. PostgreSQL shares units, not one block

Pure functions own the host-to-SSL rule and clearing certificates. `PostgresqlSslModeField` and `PostgresqlCertificateFields` are separate components, because logical shows certificates under Advanced and physical on the main form. The certificates appear as stored only in the edit scenario and only while `settings.sslClientCert` equals `savedSettings.sslClientCert`. Choosing Replace clears them from the draft, so the mask goes away with no separate replacing state. Supabase suggestions, the localhost hint and exclusions stay with logical. Backup strategy and replication diagnostics stay with physical.

Rejected: one combined connection and TLS component. It would force one layout onto both forms and change one of them for no reason. Restoring the saved certificates when SSL is turned on again was also rejected, because the form would need a second copy of the saved values to reconcile, and the user can paste them.

### 9. Restore keeps its own conversion, options and draft factory

`docs/architecture.md` forbids same-layer imports, and `RestoreTarget` and `PostgresqlRestoreOptions` belong to `entities/restores`. Restore options therefore stay out of `RestoreTargetDraft`. `RestoresComponent` holds them as a separate controlled value of the existing alias, and both `features/restores` and `features/database-connection` import that alias from `entities/restores`.

`toRestoreTarget` and `createRestoreTargetDraft` stay in `features/restores`, their only consumer. The physical type is narrowed once: `createRestoreTargetDraft(type)` takes only a database type and returns `undefined` for a physical one, and `RestoresComponent` offers no target form then. That path is unreachable, because the page shows the logical backup list only for other types. `toRestoreTarget` then takes a draft whose settings exclude physical, so its `undefined` branch goes away.

Rejected: placing the conversion in `entities/databases`, as plan item 5.3 suggested, because it would import a sibling entity. Holding restore options in the draft was also rejected, because it needs a second alias of a restore concept, which `docs/architecture.md` forbids.

### 10. Restore receives the form through a render prop

`renderRestoreTargetForm` goes from the page through `LogicalBackupsComponent` to `RestoresComponent`, like the existing `renderStorageSelection`. The page passes `(options) => <DatabaseConnectionForm {...options} />`. Restore's start refusal now shows through `onSubmit`, so its separate `restoreStartError` state is removed. The draft and restore options stay in `RestoresComponent`, which lives while the backup's dialog is open, and Advanced opens on mount when either holds a non-default value. A start captures the session and a mount revision, so an answer after the dialog closed or the session changed is ignored.

Tests that need the real form move to `pages/databases/ui/RestoreTargetComposition.test.tsx`, because the composition lives on the page. Restore list and cancel scenarios stay in the feature.

Rejected: a shared restore form feature imported by restores. It remains a same-layer import. Lifting the restore dialog into the page was also rejected, because milestone 6 owns that split.

### 11. The wizard adapts its draft once and compares types on Continue

`toUnsavedDatabase`, private to the wizard, builds the `Database` with an empty ID for the backup config and notifier steps. The form receives the draft. `CreateLimitedUserComponent` is added to `pages/databases/ui`, built from `CreateReadOnlyComponent`. It takes the create or edit draft, and the entity's `toDatabaseSettingsTarget` is its target. `CreateReadOnlyComponent` and the feature's private `toDatabaseSettingsTarget` stay for the old connection form until the containers have moved, and they are deleted with it. Its props become `onLimitedUserCreated`, `onLimitedUserNotSuggested`, `onGoBack` and `onSkipped`, while dictionary keys stay.

The base-info form receives the name and engine and reports them only on Continue, and it keeps the selection locally. The PostgreSQL type form receives the type. Clicking the highlighted PostgreSQL tile keeps the selection. Their dead `isSaveToApi`, `isShowName` and `isShowEngine` branches are deleted. Milestones 6 and 7 delete the adapter along with the backup editors' `database.id` branches, which `temp/refactor/frontend-plan.md` records.

Rejected: widening the backup editors' props here. That adds type guards to files milestones 6 and 7 are about to split, and it does not remove the `id` checks. Discarding input on every tile click was also rejected, because it loses input for a selection the user reversed.

### 12. Feedback and labels follow the design documents

Failed tests render through one `ConnectionTestFeedback` unit. It holds the raw server error and translates its own text at render time, as `docs/frontend-localization.md` requires. It shows the `databases.edit.ipWhitelistHint` hint for every variant except physical, as today. The successful test keeps its title and description through a context-bound notification, as the storage and notifier editors do. Field rows link labels with `controlId`, as `AdvancedFieldRowComponent` does, and the import control becomes a `Button`. The 63 `direct-color` entries go away by using theme classes.

Rejected: keeping `alert` until milestone 8. These lines are rewritten here, and root `AGENTS.md` makes that refactoring part of the task.

## Risks / Trade-offs

- [Rewriting five forms can lose an option or rule] → Characterization tests per engine and scenario pass on the old code first, and they exercise containers, so the rewrite does not change them.
- [A new `RestoreTargetFormOptions` import would multiply the milestone 7 baseline entry of `LogicalBackupsComponent`] → The type shares the existing import statement, and lint confirms the entry's count.
- [The restore draft keeps a typed password in memory while the dialog is open] → This matches current behavior and stays in memory only. Closing the dialog discards it, and a spec scenario pins that.
- [The empty-ID adapter survives this change] → It stays private to the wizard, no form receives its result, and the plan names its removal.
- [Hiding exclusions changes what the restore form offers] → Those fields never reached the request, so no restore result changes.
- [A removed client certificate leaves its stored key on the server] → This is existing server behavior outside this change, and the form no longer hides that the certificate goes away.
