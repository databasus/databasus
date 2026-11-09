## Why

Google Drive requires a separate authorization flow and a difficult connection setup, making the integration costly to maintain. Remove built-in Google Drive support while giving existing installations a safe refusal and a clear return path to version 3.67.0, the last release that supports it.

## What Changes

- **BREAKING**: Remove the built-in `GOOGLE_DRIVE` storage type, its API settings variant, backend provider, frontend editor and details, storage OAuth callback and authorization machinery. Delete the integration rather than retaining a compatibility adapter.
- Refuse startup before applying any pending migrations when any workspace has a Google Drive storage. State that support ended after 3.67.0 and tell the operator to return to 3.67.0, remove those storages and retry, or remain on 3.67.0.
- Add a transactional SQL migration that independently refuses existing Google Drive storages before dropping their settings table. Preserve historical migrations needed to install and upgrade older databases.
- Remove Google Drive pages, redirects, images, navigation, metadata and mentions throughout the website and all its locales, and remove mentions from the English README and its six translations.
- Remove obsolete documentation, dictionary entries, tests and dependencies used only by the integration. Keep current documentation free of historical Google Drive support and removal explanations; retain upgrade refusal and recovery guidance in runtime diagnostics and behavioral specifications. Update related specifications and preserve coverage for the remaining providers.
- Split upgrade coverage into successful upgrades without Google Drive and a regression anchored to the actual 3.67.0 image that proves refusal, rollback and a successful retry after removal.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storages`: Remove Google Drive from supported settings and add API, interface and public-documentation removal requirements.
- `internal-database-migrations`: Add refusal before pending migrations, recovery guidance and safe removal of the obsolete settings schema.
- `google-drive-storage-oauth`: Remove all requirements for the retired storage authorization flow.
- `upgrade-test`: Restrict successful preservation coverage to supported storage types and add a 3.67.0 refusal and recovery regression.
- `secret-handling`: Remove requirements specific to Google Drive token refresh and its fixed-address credentials.
- `outbound-http-proxy`: Remove Google Drive from HTTP storage and TLS requirements while preserving the remaining providers' guarantees.
- `backup-usage`: Remove Google Drive from storage detail coverage.
- `physical-backups/wal-archiving`: Remove references to the retired provider from performance and cancellation exceptions.

## Impact

Affected areas are `backend/api/openapi.yaml` and generated clients, the storages module, startup composition, SQL migrations, upgrade tests, frontend storage and authorization slices, dictionaries and browser fixtures, `website/`, `README.md`, `assets/readme/`, and related documents. Remove the built-in provider's imports and run `go mod tidy`; retain `google.golang.org/api` as an indirect dependency required by rclone, and preserve dependencies still used by Google account sign-in.

This change follows the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`. Instruction-file edits are limited to removing references and examples made obsolete by the deleted integration and documentation; no new agent conventions are introduced.

### Out of scope

- Google or GitHub account sign-in and their authorization configuration.
- Restricting providers available through the generic rclone integration.
- Automatic transfer, export or deletion of existing storages, backup records or remote files.
- A replacement provider, a deprecation screen, an OAuth compatibility callback or redirects for removed website pages.
- Making a successfully migrated database compatible with 3.67.0. Rollback is guaranteed for the startup refusal before migration, not after a successful upgrade.
