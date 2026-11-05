# Test coverage of the move

The baseline before the move, from `pnpm --dir frontend test`, is 155 test files and 1793 tests.

Every test file keeps its cases. Paths in the table are under `frontend/src`.

| Test file | Final path | Retained guarantee |
| --- | --- | --- |
| `entities/backups/physical/api/physicalBackupsApi.test.ts` | `pages/databases/api/physicalBackupsApi.test.ts` | A restore token request sends the backup or point-in-time subject |
| `features/backups/physical/lib/restoreCommands.test.ts` | `pages/databases/lib/physical-backups/restoreCommands.test.ts` | The restore commands use the right container paths and optional arguments for each PostgreSQL version |
| `features/backups/physical/monitoring/lib/chartRows.test.ts` | `pages/databases/lib/wal-stream-monitoring/chartRows.test.ts` | Chart rows keep gaps, markers and the minimum number of points |
| `features/backups/physical/monitoring/lib/easedTexts.test.ts` | `pages/databases/lib/wal-stream-monitoring/easedTexts.test.ts` | Amounts move between old and new values, and a changed sentence is replaced at once |
| `features/backups/physical/monitoring/lib/fullBackupAction.test.ts` | `pages/databases/lib/wal-stream-monitoring/fullBackupAction.test.ts` | The full backup button appears only for a user who can manage the database and only when a reason calls for it |
| `features/backups/physical/monitoring/lib/fullBackupRequestController.test.ts` | `pages/databases/lib/wal-stream-monitoring/fullBackupRequestController.test.ts` | The button stays busy until the refresh after a request lands, and a failure frees it |
| `features/backups/physical/monitoring/lib/keepingUpSentences.test.ts` | `pages/databases/lib/wal-stream-monitoring/keepingUpSentences.test.ts` | The keeping-up sentences explain the difference, backlog and upcoming limits |
| `features/backups/physical/monitoring/lib/liveOverlay.test.ts` | `pages/databases/lib/wal-stream-monitoring/liveOverlay.test.ts` | Newer live values overlay the newest sample, and older ones leave it unchanged |
| `features/backups/physical/monitoring/lib/monitoringFigures.test.ts` | `pages/databases/lib/wal-stream-monitoring/monitoringFigures.test.ts` | The figures show the last save and the rates, and unknown rates stay unknown rather than zero |
| `features/backups/physical/monitoring/lib/monitoringTab.test.ts` | `pages/databases/lib/wal-stream-monitoring/monitoringTab.test.ts` | The Monitoring tab appears only for a physical database in WAL streaming mode |
| `features/backups/physical/monitoring/lib/pathStages.test.ts` | `pages/databases/lib/wal-stream-monitoring/pathStages.test.ts` | Stage tones follow the most severe reason, and amount progress is calculated per stage |
| `features/backups/physical/monitoring/lib/reasonTexts.test.ts` | `pages/databases/lib/wal-stream-monitoring/reasonTexts.test.ts` | Reasons and tips get the right sentences and values, with broken reasons first |
| `features/backups/physical/monitoring/lib/restoreTimeline.test.ts` | `pages/databases/lib/wal-stream-monitoring/restoreTimeline.test.ts` | The restore timeline draws ranges, lost stretches and the unsaved tail so that the bar adds up to 100% |
| `features/backups/physical/monitoring/lib/roundedDuration.test.ts` | `pages/databases/lib/wal-stream-monitoring/roundedDuration.test.ts` | Durations round correctly, and time differences handle future and invalid times |
| `features/backups/physical/monitoring/lib/verdictBanner.test.ts` | `pages/databases/lib/wal-stream-monitoring/verdictBanner.test.ts` | The banner states the verdict, the main problem, its link and the restorable moment |
| `features/backups/physical/monitoring/lib/walStreamLiveController.test.ts` | `pages/databases/lib/wal-stream-monitoring/walStreamLiveController.test.ts` | Live readings follow the interval and tab visibility, and they survive failures, overlap and stop |
| `features/backups/physical/monitoring/lib/walStreamRefreshController.test.ts` | `pages/databases/lib/wal-stream-monitoring/walStreamRefreshController.test.ts` | Refreshes follow the interval, tab visibility and period, and they ignore failed and stale responses |
| `features/backups/physical/monitoring/ui/WalStreamMonitoringSections.test.tsx` | `pages/databases/ui/wal-stream-monitoring/WalStreamMonitoringSections.test.tsx` | The Monitoring tab sections show the verdict, figures, timeline, path and technical details |
| `features/backups/physical/ui/PhysicalBackupsComponent.test.tsx` | `pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx` | An old refresh does not replace current usage, and failed refreshes keep the rows and try again without an alert |
| `features/backups/physical/ui/PhysicalBackupTypeCellComponent.test.tsx` | `pages/databases/ui/physical-backups/PhysicalBackupTypeCellComponent.test.tsx` | The type cell shows the badge, and WAL ranges show their first and last segments |
| `features/backups/physical/ui/ShowPhysicalBackupConfigComponent.test.tsx` | `pages/databases/ui/backup-config/ShowPhysicalBackupConfigComponent.test.tsx` | A saved weekday 0 is shown as Sunday |
| `features/verification/agents/ui/VerificationAgentsComponent.test.tsx` | `pages/instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx` | The agent screen reveals a new token once and clears it after confirmation, shows reported memory, and suppresses an old delete or poll completion after session replacement |
| `features/verification/config/ui/EditVerificationConfigComponent.test.tsx` | `pages/databases/ui/verification/EditVerificationConfigComponent.test.tsx` | A weekday chosen before a time converts with the time, and a saved weekday 0 is shown as Sunday |
| `features/verification/config/ui/ShowVerificationConfigComponent.test.tsx` | `pages/databases/ui/verification/ShowVerificationConfigComponent.test.tsx` | A saved weekday 0 is shown as Sunday |
| `features/verification/runs/ui/VerificationsComponent.test.tsx` | `pages/databases/ui/verification/VerificationsComponent.test.tsx` | The list follows `nextCursor` and shows the after-backup trigger, and failed refreshes keep the rows and try again without an alert |
| `pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx` | `pages/databases/ui/backup-config/EditPhysicalBackupConfigComponent.test.tsx` | Schedules are labeled and validated separately, and the storage selection and draft survive nested storage creation |

## Removed cases

Task 4.1 removes two cases of `frontend/src/shared/lib/test/frontendArchitecture.test.ts`, because `NESTED_SLICES` and `MIGRATION_GROUPS` no longer exist:

- `nested group`, a row of `rejects $name dependencies when they violate a boundary`. Its fixtures now resolve to one slice, `features/backups`, so the import no longer crosses a boundary.
- `separates new sibling scenarios inside an existing migration group`. Its fixtures now also resolve to one slice, `entities/backups`, so the sibling folders are no longer separate slices.

## Tests changed beyond their imports

`pages/databases/ui/backup-config/EditPhysicalBackupConfigComponent.test.tsx` no longer passes `renderStorageSelection`, because the editor renders `BackupStorageSelection` itself. Its assertions are unchanged.

`tooling/styleChecks.test.ts` names the new path of the WAL palette file. Its assertions are unchanged.
