## Why

The embedded rclone integration brings a large dependency graph, increases the server binary's size and vulnerability exposure, and needs substantial integration code for its abstractions. Remove it completely while giving existing installations a safe refusal and a recovery path through a released version that can still manage their storages.

## What Changes

- **BREAKING**: Delete the `RCLONE` storage settings variant from the public API, backend provider and registration, frontend editor and details, dictionaries, icons and provider-only fixtures and tests. Remove its admin-only management policy rather than retaining a compatibility adapter.
- Inspect both `GOOGLE_DRIVE` and `RCLONE` storages before any pending migration. Refuse startup and application-initializing console commands with one diagnostic naming every blocking type. Google Drive alone requires 3.67.0, rclone alone requires 3.67.1, and both require 3.67.0, where the operator can remove both before retrying directly. The operator may instead remain on the indicated release. No intermediate release runs automatically.
- Add a transactional removal migration that independently refuses remaining rclone storages and, only when none remain, drops their settings schema. Preserve historical migrations and the existing Google Drive removal migration.
- Remove rclone and dependencies used only by it, and delete the architecture check's third-party test-library exceptions. Enforce absence from production dependencies through executable checks, not comments about removed support.
- Replace rclone and Dropbox claims throughout the website and the English README and its six translations with supported destinations such as SFTP, FTP and Azure Blob Storage. Preserve the surrounding markup and approximately the existing text length in prominent summaries. Remove unsupported provider lists, keywords and the 70+ providers claim, including metadata, structured data, comparison pages and machine-readable listings.
- Describe only current behavior in project documentation and comments. Keep refusal and recovery instructions in runtime diagnostics and behavioral specifications, without adding removal notices to current documentation. Preserve shared streaming, retry, encryption and cleanup behavior used by remaining providers.
- Cover independent and combined refusals with released-image upgrade regressions anchored to 3.67.0 and 3.67.1, including unchanged data on refusal and a successful direct retry after removal through the old API.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storages`: Remove rclone settings, permissions and UI; reject its API discriminator and replace unsupported public storage claims.
- `internal-database-migrations`: Aggregate both blocking storage types before migrations, select a common recovery release, and safely remove the rclone settings schema.
- `upgrade-test`: Preserve only supported providers in successful general upgrades and prove independent and combined refusal and recovery with the actual supporting releases.
- `backend-code-checks`: Remove third-party test-library exceptions and reject rclone in production binary dependencies.
- `workspace-access`: Remove the rclone-specific exception from workspace member permissions.
- `frontend-workspace-screens`: Retain resource behavior and access policy for supported providers without the rclone-specific management rule.
- `secret-handling`: Remove rclone configuration from secret-retention requirements while preserving other providers' rules.
- `backup-usage`: Limit storage-detail coverage to supported providers.
- `physical-backups/wal-archiving`: Remove rclone from performance coverage and cancellation exceptions while retaining guarantees for supported providers.

## Impact

Affected areas include `backend/api/openapi.yaml` and generated servers and clients, the storages module, startup checks, SQL migrations, upgrade fixtures and regressions, architecture checks, `backend/go.mod` and `go.sum`, frontend storage slices and browser scenarios, interface dictionaries, website content and metadata, README translations, `docs/architecture.md` and `docs/glossary.md`. Remove only dependencies no remaining feature uses; preserve Google account sign-in and its OAuth dependencies. Measure binary size with the same build settings before and after removal; an exact size ratio or CVE reduction is not assumed.

The change follows root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`. No new agent convention is needed.

### Out of scope

- Automatic conversion, export, transfer or deletion of existing storages, backup records, credentials or remote files.
- A replacement generic provider, an external rclone process, a plugin, a compatibility settings variant or a deprecation screen.
- Bypassing storage-in-use or pending-cleanup rules when removing a storage through the supporting release.
- Account OAuth, other supported providers and unrelated dependency upgrades or security suppressions.
- Downgrading a database after migrations have completed successfully, or automatic installation of intermediate releases.
- Rewriting archived OpenSpec changes or historical migrations to erase their references.
