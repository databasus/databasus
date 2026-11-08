## Why

With GFS configured to keep two daily, one weekly and one monthly backup, each new backup takes both the weekly and monthly slots. Cleanup therefore keeps only the two daily backups and loses the older restore points the user requested. Increasing daily retention to seven does not repair this behavior.

## What Changes

- Keep hourly and daily representatives for the configured recent calendar periods, including the period of the newest successful backup.
- Keep weekly, monthly and yearly representatives for the configured number of completed calendar periods, in addition to the current period's protected candidate. A candidate is the newest successful backup in the current period and can be replaced only by a newer successful backup.
- Select each level independently and keep the union of its choices. One backup may satisfy several levels without creating duplicate files.
- Use UTC calendar boundaries and Monday-start weeks. Anchor the calendar to the newest successful backup, preserving protection during a pause in successful backups.
- Apply the same selection to logical backups and completed physical full backups. Preserve the consumers' existing restore, running-work, grace-period, chain and WAL protections.
- Explain these rules in the API descriptions and the logical and physical GFS help, in every interface language. Cover repeated cleanup after each backup, rather than testing only a complete historical list.
- Define GFS candidate, GFS representative and GFS reference instant in `docs/glossary.md`, distinguishing the protected backup of a current long period from the backup retained for a completed period.
- **BREAKING behavior:** existing GFS settings acquire the completed-period interpretation without a settings migration or opt-in. Remove the rolling-duration cutoffs, hierarchical window capping and current-period-inclusive weekly/monthly/yearly slot counting; keep no compatibility mode. Request and response shapes, stored settings and defaults do not change. Previously deleted backups cannot be recovered by this fix.

## Capabilities

### New Capabilities

- `backup-retention/gfs`: Calendar selection shared by logical and physical full-backup retention, including completed periods, protected candidates, overlaps and gaps.

### Modified Capabilities

- `logical-backups`: Make GFS cleanup follow the shared calendar rules while preserving other policies and deletion protections.
- `frontend-database-screens`: Explain completed periods, current candidates and overlapping retention levels in both GFS help popups.

## Impact

- Backend: `internal/lib/gfs`, logical retention, physical full-backup retention, their tests, and `backend/api/openapi.yaml` descriptions with regenerated outputs.
- Frontend: existing GFS help and retention descriptions in all seven interface dictionaries, with affected screen and dictionary checks. No new controls or settings.
- Documentation: `docs/glossary.md` definitions for the three GFS concepts introduced in the contract and help, using the shared GFS specification as the behavior authority.
- Specifications: the new shared GFS capability and deltas for logical cleanup and frontend help. No architecture change or new dependency.
- Rules: root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md` apply. Planning follows the API, testing, frontend design and localization skills.

### Out of scope

- Backup scheduling, separate weekly/monthly backup jobs, changes to retention defaults or validation.
- Count and time-period policies, physical chain selection, incremental dependency deletion, WAL retention and grace periods.
- Restoring already deleted backups, persisted tier labels, historical candidate records or new storage tables.
- Website or README rewrites, new interface controls, commits and deployment.
