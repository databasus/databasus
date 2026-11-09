## Context

See `proposal.md` for the maintenance problem and approved scope. Version 3.67.0 is already released and is the last version with built-in Google Drive support.

- `backend/internal/modules/storages/module.go:136` registers the storage providers, including Google Drive at line 142. Its implementation and encrypted settings live under `storages/internal/providers/googledrive`.
- `backend/api/openapi.yaml:4619` defines the settings union and discriminator; line 4690 defines `GoogleDriveStorageSettings`. Generated backend and frontend clients expose that variant.
- `backend/internal/app/app.go:69` opens the internal database and line 76 starts migrations before constructing the modules. `backend/internal/lib/database/migrate.go:15` documents that each migration commits independently, and line 76 implements the refusal of a newer schema. A failure in the last migration does not undo earlier migrations from the same startup.
- `backend/migrations/20250629143714_add_google_drive_storage.sql:5` creates `google_drive_storages`; `backend/migrations/20261026120000_move_storages_into_module.sql:23` renames it to `storages_google_drive`. The parent table remains `storages`, with provider identity in `type`.
- `frontend/src/app/ui/App.tsx:50` routes the separate storage OAuth callback. The account sign-in callback is separately registered at line 49. `docs/google-drive-storage-authorization.md:3` describes the editor-owned draft and token exchange.
- `website/app/i18n.ts:56` includes the translated setup route. The website also has an old redirect at `website/app/(en)/google-drive-storage/page.tsx:3`.
- `backend/e2e/upgrade/previous_release_storages_test.go:138` creates a Google Drive storage for the general successful-upgrade scenario. `backend/migrations/storages_migration_test.go:178` also creates one for historical secret-preservation coverage. Both need deliberate changes when full migration now refuses that provider.

The design follows `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`. The relevant architecture constraints are module ownership and the public contract in `docs/architecture.md:59`, migration safety in `docs/architecture.md:137`, provider registration in its Engines and providers section, and frontend state ownership in its Frontend section. The API design skill governs contract deletion and regeneration; frontend design, localization and FSD skills govern removal of the storage UI and its public exports.

## Goals / Non-Goals

**Goals:**

- Preserve a usable 3.67.0 database when normal application startup refuses the upgrade.
- Remove production integration code while retaining only the data checks needed to protect old installations.
- Keep the generic migration runner independent of storage types and make the refusal reachable from both server and console initialization.
- Make the absence of Google Drive observable in API validation, UI choices and generated public materials.

**Non-Goals:**

- Support a 3.67.0 binary after migrations have successfully completed.
- Bypass existing storage-in-use and pending-cleanup rules when an operator removes a storage in 3.67.0.
- Treat arbitrary rclone configuration as a built-in Google Drive storage or inspect its secrets for provider-specific blocking.
- Introduce a new storage settings representation, migration framework or authorization abstraction.

## Decisions

### 1. Add a read-only startup check before the migration runner

Expose a narrowly scoped upgrade check through the storages module's `module.go`; keep its implementation and queries in a separate logical unit under that module's `internal/`. Application composition calls it immediately after opening the internal database and before `database.Migrate`, key migration, module initialization or background work. It takes the database dependency without requiring construction of the full storages module, decrypted credentials or the new schema, following the existing standalone `ProbeLocalStorage` pattern in the public contract.

The check first determines whether the parent storage table exists. An empty installation passes. Otherwise, it looks for any parent row with `type = 'GOOGLE_DRIVE'`, without filtering by workspace, backup assignment or the existence of a provider row. It returns the retirement error before any schema or migration-history mutation. Database inspection failures also stop startup; they must not be treated as proof that no blocking storage exists. The check remains read-only after successful removal and does not depend on the obsolete provider table existing.

This keeps table knowledge within its owning module, as `docs/architecture.md` and `backend/AGENTS.md` require. The existing newer-schema refusal and migration serialization remain in force. The check is not a replacement for the migration runner's session lock; the independent SQL guard below rechecks at the schema-change boundary. Running the old serving instance concurrently with an upgrade remains outside the existing single-instance deployment model.

**Alternatives rejected:** A guard only in the final SQL migration allows earlier migrations to commit and can prevent a 3.67.0 restart. Storage-specific SQL in `lib/database` violates the domain boundary. Constructing the full module before migrations requires schema and dependencies that may not yet exist. Reversing every already-applied migration introduces a downgrade mechanism outside this task.

### 2. Use an independent transactional removal migration

Create the SQL file with goose, leaving shipped historical migrations intact. The migration first checks the parent storage discriminator and raises an exception when Google Drive remains. Only after the check passes does it drop the obsolete settings table, without `CASCADE`. Account for the old and renamed settings layouts in catalog checks where necessary, and verify upgrade from an older dump as well as from the immediate predecessor. The migration must not depend on the relative order of other new migrations in this change.

Do not delete parent storages, backup references, queued file obligations or remote files to satisfy the precondition. A direct goose execution receives the same recovery guidance, but it cannot promise reversal of other migrations the operator has already applied separately. The migration's own changes and history entry remain transactional. Its Down section restores only the empty pre-removal settings schema and constraints; this does not establish a supported application downgrade.

Historical migration tests that prove Google Drive data was preserved during an earlier rename must stop at that historical migration rather than applying the new removal migration. General full-upgrade fixtures contain supported providers only. Separate refusal tests exercise old parent rows, including missing provider settings, before historical repairs can delete them.

**Alternatives rejected:** Editing old migrations to erase the provider breaks installed and restored databases. An unconditional drop loses credentials. Cascading deletion hides retained references. Keeping an empty provider implementation preserves the integration surface the user asked to remove.

### 3. Give one explicit recovery instruction and avoid credentials

Use this English diagnostic for the preflight and the SQL exception:

> Google Drive support ended after Databasus 3.67.0. Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, or continue using 3.67.0.

The checks need only existence, so they neither decrypt nor render client secrets, token JSON or serialized settings. Preserve normal logging redaction. Application startup closes the connection on refusal and exits before HTTP or jobs start. The operator sees the error in the existing startup output; no new error screen is needed.

**Alternatives rejected:** Saying support ended in 3.67.0 misidentifies the release that still supports it. Telling users only to delete storages omits the need to restart the supporting image. Printing settings for diagnosis creates unnecessary credential exposure. Logging a warning and continuing strands existing storages on an unsupported version.

### 4. Remove the provider from the contract before removing implementations

Delete `GoogleDriveStorageSettings` and its union and discriminator entries in `backend/api/openapi.yaml`, then run `mise run generate-api`. Delete the backend provider package, registry entry, provider type constant, settings conversion and connection-test cases. Remove the built-in provider's imports and run `go mod tidy` to remove only dependencies that are no longer reachable. The retained rclone provider imports `github.com/rclone/rclone/backend/all` (`backend/internal/modules/storages/internal/providers/rclone/provider.go:10`), whose Drive backend still requires `google.golang.org/api`; retain that module as an indirect dependency. Retain `golang.org/x/oauth2` and other dependencies used for account sign-in or rclone as well.

Authenticated requests carrying the removed discriminator to create, update or unsaved-test operations receive the existing `400 invalid_request` contract error. Use raw HTTP test requests where generated clients cannot express the removed variant, as the testing skill permits. No retained settings alias, API variant or special compatibility parser is added. Existing authentication and authorization checks remain applicable.

**Alternatives rejected:** Retaining the contract variant implies continued support and leaves generated consumers able to create it. Rejecting it only in a provider implementation leaves dead API and registry code. Removing every Google or OAuth dependency also breaks account sign-in.

### 5. Delete storage authorization as a complete frontend unit

Remove the storage OAuth page and route, the editor authorization hook, attempt and relay, test-only channel helpers, Google Drive edit and detail components, storage logos and type mappings, defaults, validation branches and unused public exports. Delete its dictionary keys in English and every translated dictionary together. Remove the website-link mapping for its deleted setup page.

Keep the existing account OAuth feature, `/auth/callback`, shared session policy and generic navigation untouched except where obsolete storage-specific references must be removed. An old storage callback falls through ordinary unknown-route handling and exchanges no code. Preserve storage creation from both the storage section and backup configuration. Update App, storage screen and browser fixtures to cover the remaining provider list and the retired callback without preserving a Drive-specific UI suite or image references.

**Alternatives rejected:** Hiding the option retains the difficult flow and unused code. Keeping the callback as a retirement page adds a compatibility surface the task does not need. Moving the relay to shared keeps machinery without a consumer and conflicts with frontend ownership rules.

### 6. Remove public claims across every language and generated discovery surface

Delete the English Google Drive setup page, its localized dispatcher and all six content copies, the older English redirect and all setup screenshots. Remove references in navigation, translated-path registration, sitemap configuration, `llms.txt`, structured data, metadata, provider counts and every affected product or comparison page. Remove README mentions in English and all six translations. Remaining lists and sentences must stay grammatical and describe supported providers accurately. Removed addresses follow the ordinary static-site 404 behavior, without new redirects.

Delete `docs/google-drive-storage-authorization.md`, remove its architecture section and glossary entry, and update the storage examples in the glossary. Keep `docs/architecture.md`, `docs/testing.md` and other current documentation free of explanations about historical Google Drive support, its removal or recovery from the upgrade refusal. Recovery guidance belongs in startup and migration diagnostics and their behavioral specifications; regression coverage remains in the tests and upgrade-test specification. Remove the dead authorization-document link in the root `AGENTS.md` and the Google Drive brand-label example in `website/AGENTS.md`; these are consistency fixes, not new instructions. Search the whole tracked repository for remaining integration references. Allow historical migrations and their focused tests, retirement guards and their tests, and OpenSpec change history and retirement requirements. Do not rewrite archived change history.

**Alternatives rejected:** Removing only the setup page leaves claims and broken links in translations and metadata. Keeping redirects leaves Drive-specific website behavior. Rewriting historical migrations or archived specs to achieve a literal zero-match search destroys necessary upgrade or historical context.

### 7. Separate supported upgrades from the anchored refusal regression

The general upgrade scenario continues to start from the newest published release and exercises every provider supported by the current version. It no longer seeds built-in Google Drive. Add a separate scenario using the actual `databasus/databasus:v3.67.0` release, resolving and pinning the fixture image identity during implementation. Reuse the upgrade suite's isolated volumes, naming, labels, startup observation and cleanup, with no external Google account or HTTP calls.

The regression writes an account, a workspace, an unused Drive storage with synthetic credentials and a supported storage through the old API. It stops the old application, captures read-only snapshots of migration history and relevant persisted records, and starts the candidate. It observes process exit and the recovery diagnostic, then compares snapshots without emitting secrets. It restarts 3.67.0 on the same volume, signs in, verifies records, deletes the unused Drive storage through the old API, stops 3.67.0 and starts the candidate successfully. Check preserved supported records and the absence of the obsolete settings table. Read-only SQL is appropriate for migration history and encrypted values that the API cannot return.

Add focused startup and migration tests for unused storages, another workspace, missing settings, old layouts, fresh install, direct SQL refusal, retry and secret-free diagnostics. Update existing storage UI browser coverage rather than relying only on compilation. Run the complete upgrade command because the test harness is changed.

**Alternatives rejected:** Dropping Drive from the success fixture alone does not prove refusal or rollback. A synthetic SQL database alone does not prove the actual 3.67.0 image restarts. Following the moving newest release for this regression would eventually erase the only supporting fixture. Real Google authorization makes a local retirement check depend on external credentials and services.

## Risks / Trade-offs

- **A user cannot delete a storage that retained backups or backup configs reference.** Existing deletion rules remain in force. Recovery requires resolving those references in 3.67.0, or staying on that release; no force-delete behavior is added.
- **A direct goose invocation can apply other migrations before the removal guard fails.** Its error provides the same guidance, but only application preflight guarantees an unchanged upgrade input. Preserve that boundary in the behavioral specifications and test both paths separately.
- **Historical fixture helpers currently apply every migration.** Limit preservation tests with Drive records to the historical boundary and keep full-upgrade coverage for all remaining providers, so refusal tests do not replace preservation checks.
- **Website removal spans many translated pages and generated links.** Search source and static output, build the site, verify the remaining localized routes and assert retired paths are not emitted.
- **The anchored image adds a release dependency to the upgrade suite.** Pin the real release identity, use existing image loading and cleanup, and fail clearly if it cannot be obtained rather than silently skipping the scenario.

## Migration Plan

1. Before upgrading, operators who want to proceed use 3.67.0 to resolve backup references and delete every built-in Google Drive storage. Operators who retain those storages remain on 3.67.0.
2. Normal startup on a database with any remaining Google Drive storage refuses before new migrations. Returning to 3.67.0 on the same unchanged volume restores the prior application behavior.
3. On a database without Google Drive storages, or on a fresh install, startup applies migrations, drops the obsolete settings table and serves the reduced provider set.
4. After successful migration, the existing one-way upgrade policy applies. This change provides no downgrade path or data conversion back to Google Drive.
