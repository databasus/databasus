## Why

Databasus opens on a card list of databases. Each card shows the health badge, the storage and the time of the last backup, and nothing more. To learn how many backups a database has, how large they are, or how much space the whole workspace uses, a user has to open every database one by one. Admins have no view of the space used across all workspaces at all.

## What Changes

- Add a dashboard page that becomes the page shown after sign-in and after creating a workspace. The databases list stays one click away in the sidebar.
- The dashboard lists every database of the selected workspace in a table with its health status, its last ten healthcheck attempts, its backup counts, the mean and total size of its backups, the time of its last backup and its storage. Phones get one card per database instead of the table.
- Summary tiles above the table show the workspace totals: databases, backups and total backup size.
- Global admins also see an installation-wide tile with the database count, backup count and total backup size across every workspace.
- Add `GET /api/v1/dashboard?workspace_id=<id>` for any workspace member, and `GET /api/v1/dashboard/installation` for global admins only.
- Extract the health status badge, the healthcheck attempt strip and the MB/GB size formatter into shared pieces, so the dashboard and the existing pages render them the same way.

No caller, schema, config or API breaks, so nothing here is BREAKING. Three existing things change:
- The page shown after sign-in and after creating a workspace is the dashboard instead of the databases list.
- The private healthcheck attempt test fixture becomes the exported `CreateTestHealthcheckAttempt`.
- The two near-identical `formatSize` functions in the logical and physical backup lists are replaced by one shared `formatSizeMb`.

## Capabilities

### New Capabilities

- `workspace-dashboard`: the per-workspace and installation-wide backup and healthcheck summary, and who may read each.

### Modified Capabilities

None.

## Impact

- Backend: new `dashboard` feature package and routes, plus batched read methods on the logical and physical backup services, both backup config services, the healthcheck config and attempt services, and the database service. No migration.
- Frontend: new `features/dashboard` slice, a new sidebar entry and icons, new keys in all six interface dictionaries.
- Answers to `backend/AGENTS.md` and `frontend/AGENTS.md` on top of the root `AGENTS.md`.

## Out of scope

- Opening a database's detail page from a dashboard row.
- Storage usage reported by the storage provider, free space per storage, and charts or history over time.
- Caching or background pre-computation of the aggregates.
- Changing the databases card list beyond reusing the extracted badge.
