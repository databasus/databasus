## Context

The databases page renders one card per database (`frontend/src/features/databases/ui/DatabaseCardComponent.tsx`). Each card fetches its own backup config to learn its storage (`DatabaseCardComponent.tsx:33`), so a workspace with N databases costs N extra requests. No endpoint returns counts or sizes across databases: the only aggregate is the per-database physical total `GetTotalUsageMBByDatabase` (`backend/internal/features/backups/backups/core/physical/service/service.go:403`). Logical backups have counts but no sums (`backend/internal/features/backups/backups/core/logical/repository.go`). Healthcheck attempts can only be read per database and per period.

The frontend has no URL router for pages. The main screen switches between sidebar tabs held in component state, and the databases tab is the default (`frontend/src/widgets/main/MainScreenComponent.tsx:44` before this change).

## Goals / Non-Goals

**Goals:**
- One request per workspace returns everything the dashboard table and tiles need.
- The numbers match what the existing pages report for the same database.
- Global admins see installation-wide totals; nobody else does.

**Non-Goals:**
- Storage usage reported by the providers, history, charts or caching (see the proposal's out-of-scope list).

## Decisions

### A dedicated dashboard feature that reads through other features' services

The new `backend/internal/features/dashboard` package owns no table. It asks each owning feature's service for batched totals: logical backups, physical backups, both backup configs, healthcheck configs and attempts, and databases. Each batched method groups by `database_id` in one query per table, in the raw-SQL style of `GetLastBackupTimesByDatabaseIDs` (`physical/service/service.go:428`).

Rejected alternatives:
- **Adding the statistics to `GET /databases`.** It would make the most-called list heavier for every page that uses it, and mix presentation totals into the database model.
- **Querying the backup tables from the dashboard package.** `backend/AGENTS.md` forbids injecting another feature's repository. The owning package also knows the status semantics, such as which physical statuses count as failed.
- **Per-database calls in a loop.** A workspace with many databases would cost six queries per database instead of six per request.

### Authorization by delegating to the databases list

`DashboardService.GetWorkspaceDashboard` starts with `DatabaseService.GetDatabasesByWorkspace` (`backend/internal/features/databases/service.go:366`). That call already checks workspace access, hides secrets and fills physical last-backup times. A non-member therefore gets exactly the databases list's 400 response.

Rejected alternative: calling `CanUserAccessWorkspace` again in the dashboard service. It would run the same membership query twice and could drift from the list's rule.

### Admin-only installation route through middleware

`GET /dashboard/installation` sits on a router group with `RequireRole(UserRoleAdmin)`, as the verification agents routes do (`backend/internal/features/verification/agents/controller.go:23`).

Rejected alternative: a role check inside the service. The rule applies to the whole route and does not depend on any entity, which is the case the middleware exists for.

### Query parameter, not path parameter

The workspace route is `GET /dashboard?workspace_id=`, matching `GET /databases?workspace_id=` (`backend/internal/features/databases/controller.go:27`), `/storages` and `/notifiers`.

Rejected alternative: `/dashboard/workspaces/:id`. No other workspace-scoped read in the backend uses that shape.

### Counting rules

- Physical total size = successful full + successful incremental + all WAL segment sizes. This is the `GetTotalUsageMBByDatabase` formula, so the dashboard matches the physical backups page's total.
- The physical backup count covers full and incremental rows only. The paginated list's `CountBackups` (`physical/service/service.go:544`) also counts WAL segments. That count feeds pagination, while a streaming database uploads thousands of segments, which would drown the number the user reads as "backups".
- Physical "failed" = `ERROR` + `CHAIN_BROKEN`. In-progress and canceled rows count toward the total only, as they do for logical backups.

Rejected alternative: counting only successful backups. The user asked for the number of backups, and a failing database should show that failures exist.

### Recent attempts through a window query

`FindRecentByDatabaseIDs` ranks attempts with `ROW_NUMBER() OVER (PARTITION BY database_id ORDER BY created_at DESC)`, served by the existing `idx_healthcheck_attempts_database_id_created_at` index (`backend/migrations/20250704095930_add_healthckeck.sql:40`).

Rejected alternative: reusing the period filter of the healthcheck panel. The number of attempts in a period depends on each database's check interval, so a fixed period gives rows of wildly different lengths.

### Frontend placement

The API client and types live in `features/dashboard`, because the dashboard is their only consumer (`frontend/AGENTS.md`, "single use - consumer slice"). The health badge moves to `entity/databases/ui` and the attempt strip to `entity/healthcheck/ui`, because both the dashboard and an existing feature render them. Features may not import each other. The two copies of `formatSize` (`LogicalBackupsComponent.tsx:521`, `PhysicalBackupsComponent.tsx:48`) become `shared/lib/formatSizeMb`, which follows the physical copy's module-level shape.

Rejected alternative: a new `entity/dashboard` slice. `frontend/AGENTS.md` calls a single-consumer entity premature.

## Risks / Trade-offs

- **Large WAL tables:** every dashboard load sums every WAL segment of the workspace's physical databases. The same cost already exists in `GetTotalUsageMBByDatabase`, and the sums use the `database_id` indexes. Caching stays out of scope until it shows up as a problem.
- **Installation query:** it scans every backup table without a filter. It is admin-only and loads once a minute at most per open dashboard.
- **Retention shrinks history:** hard-deleted backups vanish from the counts, so the dashboard reports what is stored now, not lifetime activity.

## Migration Plan

No migration. Deploying the new version adds the routes and makes the dashboard the landing page. Rolling back removes both.
