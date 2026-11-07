## Context

See [proposal.md](proposal.md) for the motivation and [the usage spec](specs/backup-usage/spec.md) for the behavior contract.

- Physical pages already return `totalUsageMb` (`backend/api/openapi.yaml:5929`), calculated from completed physical backups and WAL stored sizes (`backend/internal/modules/backups/internal/physical/catalog/catalog_repository.go:17`). The screen accepts it as paged-list metadata but renders it above the actions (`frontend/src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx:67`, `:176`).
- Logical pages have only backups and a cursor (`backend/internal/modules/backups/internal/logical/service.go:227`; `backend/api/openapi.yaml:6765`). Their table has an existing Size help tooltip (`frontend/src/pages/databases/ui/logical-backups/LogicalBackupsTable.tsx:51`).
- Both backup screens poll at 1000 ms (`frontend/src/pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx:23`; `frontend/src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx:25`). `usePolledPagedList` already pauses periodic refresh while hidden and refreshes on visibility changes (`frontend/src/shared/hooks/usePolledPagedList.ts:24`).
- The current backup formatter uses up to two decimal places (`frontend/src/pages/databases/lib/formatBackupSize.ts:1`). Shared byte formatting already uses base 1024 and the active number formatter (`frontend/src/shared/lib/formatBytes.ts:1`).
- Saved-storage details render the name above settings (`frontend/src/pages/storages/ui/StorageComponent.tsx:270`), while the Storage response has no usage (`backend/api/openapi.yaml:4477`). Storages expose batch lookup and a transfer-blocking lookup to other modules (`backend/internal/modules/storages/module.go:103`).

This design is constrained by root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`. In particular, [module ownership and dependency direction](../../../docs/architecture.md#modules) keep backup catalog queries in backups, and [frontend boundaries](../../../docs/architecture.md#frontend) prohibit importing one page slice from another. No architecture exception or instruction-file change is needed.

## Goals / Non-Goals

**Goals:**

- Use one eligibility definition for database and storage aggregates, with a fixed query count independent of the number of backups or databases.
- Deliver aggregate metadata through the existing list refresh lifecycle and keep storage usage independent of storage editing.
- Keep API changes additive and preserve per-backup formatting and existing request contracts.

**Non-Goals:**

- A new persisted counter, reconciliation job, provider scan or source-database connection.
- A new real-time transport, backup progress model or general resource polling framework.

## Decisions

### 1. Calculate totals inside the backups module

Add a focused usage service and repository under `backups/internal/usage`, wired through `backups/module.go`. The repository reads only tables owned by backups: logical backups, physical backups and physical WAL segments. Sum completed backup sizes and committed WAL stored sizes directly, without joining backups to their dependencies or expanding WAL ranges. Filter by the requested database ID or the recorded storage ID. Coalesce an empty sum to zero.

Use the same repository eligibility rules for the existing physical page, the logical page and the storage read. Remove the superseded chain-only total forwarding method/query when its callers have moved, rather than retaining duplicate implementations. Database list services keep their existing database authorization before reading their aggregate.

Storage reads execute in a transaction: load the storage through `GetStorageBlockingTransfer`, reject a missing storage, authorize its workspace with `workspaces.Authorize` and ActionView, then read its aggregate before releasing the lock. This reuses the existing transfer guard so a concurrent transfer cannot detach the authorization check from the storage's ownership. The handler makes one usage-service call and maps its result.

**Alternatives rejected:** Provider measurements would count unrelated files and require credentials or connectivity. Client-side sums miss unloaded and filtered records. Persisted counters add synchronization and repair work. Putting backup SQL or a dependency on backups into storages violates the established module direction.

### 2. Extend list metadata and add a separate storage read

In `backend/api/openapi.yaml`:

- Add required numeric `totalUsageMb` to `LogicalBackupPage`, with the same units and database-wide scope as `PhysicalBackupPage`.
- Add `GET /storages/{storageId}/backup-usage`, operationId `getStorageBackupUsage`, tagged `backups`, returning 200 with a small response containing required `totalUsageMb`.
- Document viewer access, no-session 401, hidden/missing-resource 404, insufficient-permission 403, malformed-ID 400 and unexpected-failure 500, all using Problem and a default Problem response. Do not return internal error text.
- Document that totals count catalog records rather than actual provider consumption, and that filters and pages do not restrict a database total.

Regenerate code with `mise run generate-api`. Implement the operation in the backups strict HTTP server even though its URL identifies a storage. The storage page calls the generated operation through a small facade in its own slice. No public module-contract addition is needed for internal usage calls.

**Alternatives rejected:** Adding totals to every Storage response would couple storage CRUD/list requests to backup aggregation or reverse the module dependency. A separate database-total request would duplicate the lists' existing polling and complicate synchronization. The new storage read avoids both costs. Existing requests, fields and endpoints remain valid; no compatibility shim or request migration is introduced.

### 3. Keep aggregate formatting reusable and presentation local

Put the small, pure aggregate-size formatter in `shared/lib`, because both database and storage pages use it. It accepts the recorded MB value and the active `useLocale().formatNumber`, uses the existing base-1024 convention, and applies whole MB or up to one decimal GB. Reuse existing numeric formatting rather than introducing a library. Unit choice precedes rounding, so an amount just below 1024 MB remains an MB value. Keep current per-backup row/card formatting intact.

Pass total metadata into both table components. Use an inline, nonwrapping header group: translated Size, a small gap, then `text-xs text-muted font-normal` for the total. Keep logical Size help accessible without moving the value below Size. Initial loading and first-page failure omit the value. At mobile card breakpoints show one translated usage line above the cards.

Place the storage usage line outside the name and settings editing branches, directly below the name area. Editing or saving the name must not mount a new usage session. Keep one whole-sentence translation key for Total usage with a size placeholder; use all seven locale dictionaries and remove a superseded physical-only key only after every caller has moved. Numbers follow the selected language, while MB and GB remain technical symbols.

**Alternatives rejected:** Importing the formatter from the database page into the storage page violates slice boundaries. Changing the existing row formatter would expand the requested presentation change. Putting totals below Size contradicts the requested single-line layout. A separate prominent statistic duplicates the header and takes extra space.

### 4. Refresh totals through the owning screen's lifecycle

For logical backups, adapt the response to `usePolledPagedList` metadata exactly as physical backups already do, and render the accepted metadata in the table/cards. Both screens keep their existing 1000 ms interval. A changed total must publish even when row merging produces unchanged items and cursor; do not derive it from the merged rows or memoize it on row identity. Preserve existing generations, first-page failure behavior, silent background list errors and visibility handling. Periodic and visibility-triggered refreshes run only for a ready list whose current first page succeeded. A failed initial load or reload continues to require explicit Retry; changing visibility must not bypass that gate.

Storage details own a small usage polling hook in the storages page slice. Start once the saved storage is loaded; poll every 1000 ms while visible and trigger a fresh read when visibility returns only if the usage session is ready and not awaiting explicit Retry. Neither a visibility event nor an interval tick retries a failed initial usage load or a failed background usage read. Use the existing session/context identity checks and active-effect guards so earlier requests can settle without publishing into a later session. A storage, workspace or session change resets usage and invalidates pending reads. Cleanup ends timers/listeners and makes late responses inert. Keep at most one usage request in flight; schedule another interval only when it can be read without overlap.

An initial storage usage failure shows inline translated feedback with retry. A transient refresh failure keeps the last number with a translated refresh-failure indication and stops polling until retry succeeds. A current 401/403/404 hides the last value. Neither usage refresh nor retry reloads the storage editor, resubmits a draft or resets a connection test.

Totals follow catalog eligibility: a running upload is excluded, and completion or committed WAL becomes visible on the next successful poll. Deletions reduce the aggregate as soon as the catalog records disappear, regardless of file cleanup progress.

**Alternatives rejected:** Polling the complete Storage response risks overwriting drafts and performs unrelated work. A new shared polling framework is unnecessary for one small screen-owned read. WebSockets/SSE introduce connection management when the backup lists already poll once per second. Counting running progress would change the existing physical usage meaning and require a different progress model.

### 5. Verify observable aggregates, refresh and geometry

Module API tests use the real application and generated client. Cover every supported database kind, completed versus excluded statuses, unpublished WAL, shared WAL counted once, empty totals, mixed database content in a storage, storage isolation, recorded attribution, filters/cursors, catalog deletion and access refusal. Create fixtures through the API; use SQL only for completed backup/WAL catalog states that the API cannot create without running an actual backup. Do not start backup pipelines for this query-only behavior.

Screen tests use real locale providers and finite HTTP fixtures. Cover metadata-only refreshes, completion/deletion/WAL responses, stale resource replies, loading/failure/retry, unmount and setup-cleanup-setup, hidden/visible transitions, draft preservation, all supported database kinds/providers, compact rounding and a live language switch. Explicitly prove that visibility events and interval ticks do not retry failed database first pages or storage usage sessions awaiting Retry. Run dictionary placeholder/completeness checks.

The general test run exposed a race in the existing WAL last-backup-time test: an idle catalog can precede the request scheduler publishing its full backup. Wait through the API for both the initial and requested full backups to be present and completed before comparing timestamps; retain the original completion assertions.

Extend production browser UI fixtures/scenarios for physical and logical headers and saved-storage details. Assert that Size and its value share a line, the storage line precedes settings, both themes remain readable, and mobile/200-percent enlarged text does not lose the total. Update affected logical-page fixtures and give every rendered storage detail an explicit usage fixture.

**Alternatives rejected:** DOM-only assertions cannot establish single-line geometry. Pure formatter tests cannot prove screen placement or accepted metadata refreshes. Full backup engine/provider runs would test unrelated execution paths and add cost to catalog-read coverage.

### 6. Define backup usage in the glossary with the API change

Add a Product row in `docs/glossary.md` for Backup usage, mapping it to `totalUsageMb`. Define it as the sum of catalog-recorded stored sizes of retained completed backups and committed archived WAL for one database or storage, distinct from the provider's actual consumption. The API and usage spec hold the eligibility details; the glossary keeps the short shared definition.

**Alternatives rejected:** Defining the term only in the endpoint description leaves it absent from the project's common vocabulary. Calling it disk usage would conflate backup catalog totals with actual filesystem or provider measurements.

## Risks / Trade-offs

- Catalog usage differs from actual stored bytes, especially during cleanup or for sidecar/encryption overhead. Mitigation: define the catalog meaning in the API/spec and avoid capacity or filesystem claims.
- Repeated aggregate reads can become expensive for large catalogs. Mitigation: use direct aggregate queries with a fixed query count, no per-backup lookups, no row downloads and no provider traffic; retain existing one-second cadence rather than adding database requests.
- Required logical-page metadata changes typed fixtures and generated clients. Mitigation: regenerate every contract consumer and update all affected finite fixtures in the same implementation; keep request shapes and existing fields unchanged.
- A late response could publish another storage's usage or disturb an edit. Mitigation: resource/session identity checks, effect cleanup, screen-owned usage state and race-focused tests.
- A one-second interval is a refresh cadence, not a delivery guarantee during network failure or hidden-page pauses. Mitigation: specify the next successful current response and explicit storage failure/retry behavior.

## Migration Plan

No persisted data migration is needed. Ship the regenerated server/client contract and the frontend together in the existing single-image deployment. Rollback uses the prior image with unchanged tables and backup records. This change creates no deployment task; implementation ends after verification and review with the working tree ready to commit.
