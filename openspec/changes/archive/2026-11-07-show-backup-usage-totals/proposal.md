## Why

Only physical PostgreSQL backups currently show their total recorded size, and the separate usage line takes space above the backup controls. Users need a compact total beside the Size column label for every database, and a corresponding total on each storage's detail page.

## What Changes

- Show the database's total immediately after Size on the same line, in smaller, muted, regular-weight text. Remove the physical list's separate Total usage line.
- Cover physical PostgreSQL and logical PostgreSQL, MySQL, MariaDB and MongoDB backups. Preserve a compact total above backup cards on mobile.
- Format totals as whole MB below 1024 MB and GB with at most one decimal place above that threshold, without trailing decimal zeroes. Examples in English: `104 MB`, `10.5 GB`, `10 GB`.
- Refresh database totals with the existing one-second backup list polling, even when only the total changes.
- Show a localized Total usage line directly beneath a saved storage's name, using the same compact format and refreshing while its detail page is visible.
- Calculate totals from retained completed backup records and committed WAL records in Databasus. Attribute storage usage by each record's storage ID, independently of a database's current backup configuration.
- Extend logical backup list responses with `totalUsageMb` and add an authenticated storage backup-usage read operation. These are additive response/API changes; existing request shapes remain unchanged.

## Capabilities

### New Capabilities

- `backup-usage`: Catalog-based database and storage backup totals, compact localized presentation, live refresh and authorized reads.

### Modified Capabilities

None. Existing requirements for backup ordering, filtering, pagination, authorization and load failures continue to apply.

## Impact

- Backend: the backups module, its logical and physical repositories and services, `backend/api/openapi.yaml`, and generated server/client types. Storage lookup uses the storages module's public contract; the storages module does not depend on backups.
- Frontend: logical and physical backup lists and tables, the saved-storage detail page, a shared compact size formatter, and all seven interface dictionaries where copy changes.
- Documentation: add Backup usage to `docs/glossary.md`, mapped to `totalUsageMb`, as the catalog-recorded size of eligible backups and WAL rather than actual provider consumption.
- Verification: module API tests, screen tests, dictionary checks and production browser UI scenarios for placement and responsive behavior.
- Governing instructions: root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`. The approach follows the existing module and frontend boundaries; no architectural rule changes are required.

## Out of Scope

- Measuring provider capacity, scanning storage contents, or reporting actual filesystem/bucket consumption.
- Counting unfinished uploads, failed/canceled backups, untracked files, metadata sidecars or pending physical file deletions.
- Changing backup execution, retention, cleanup, storage assignment, or per-backup row size formatting.
- Adding usage to storage list cards, quotas, history, charts, push notifications, WebSockets or server-sent events.
- Request-shape breaks, schema migrations, compatibility shims, deployment or Git publishing.
