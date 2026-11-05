# Physical backup screens refactoring coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-02, before this change: 168 files, 1929 tests passed.

## Ownership map

Paths are relative to `frontend/src/pages/databases`.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `lib/physical-backups/restoreCommands.test.ts` | Same file, unchanged | The restore commands stay as they are |
| `lib/wal-stream-monitoring/chartRows.test.ts` | Same file, unchanged | Chart rows |
| `lib/wal-stream-monitoring/easedTexts.test.ts` | Same file, unchanged | Eased texts |
| `lib/wal-stream-monitoring/fullBackupAction.test.ts` | Same file, unchanged | When the full backup button is offered |
| `lib/wal-stream-monitoring/fullBackupRequestController.test.ts` | Same file | The request's busy state and reports |
| `lib/wal-stream-monitoring/keepingUpSentences.test.ts` | Same file, unchanged | Keeping-up sentences |
| `lib/wal-stream-monitoring/liveOverlay.test.ts` | Same file, unchanged | Live values over the statistics |
| `lib/wal-stream-monitoring/monitoringFigures.test.ts` | Same file, unchanged | Monitoring figures |
| `lib/wal-stream-monitoring/monitoringTab.test.ts` | Same file, unchanged | Which databases have the tab |
| `lib/wal-stream-monitoring/pathStages.test.ts` | Same file, unchanged | Path stage tones |
| `lib/wal-stream-monitoring/reasonTexts.test.ts` | Same file, unchanged | Reason and tip texts |
| `lib/wal-stream-monitoring/restoreTimeline.test.ts` | Same file, unchanged | The restore timeline |
| `lib/wal-stream-monitoring/roundedDuration.test.ts` | Same file, unchanged | Rounded durations |
| `lib/wal-stream-monitoring/verdictBanner.test.ts` | Same file, unchanged | The verdict banner |
| `lib/wal-stream-monitoring/walStreamLiveController.test.ts` | Same file, unchanged | The live schedule |
| `lib/wal-stream-monitoring/walStreamRefreshController.test.ts` | Same file, unchanged | The refresh schedule |
| `ui/physical-backups/PhysicalBackupsComponent.test.tsx` | Same file | Late refreshes, failing polls and a failed refresh after a cancel |
| `ui/physical-backups/PhysicalBackupTypeCellComponent.test.tsx` | Same file | The type badge and the WAL range segments |
| `ui/wal-stream-monitoring/WalStreamMonitoringSections.test.tsx` | Same file | The monitoring sections |

## Tests changed beyond their imports

- `lib/wal-stream-monitoring/fullBackupRequestController.test.ts`: `frees a busy button on the first landed refresh even when it never sent the request` is now `reports no change on a landed refresh when no request was sent`, because a controller no longer starts over during a request. Two tests for `stopReporting` are new.
- `ui/wal-stream-monitoring/WalStreamMonitoringSections.test.tsx`: the arrow's markup expectation reads `text-muted` instead of `text-gray-400`.
- `ui/physical-backups/PhysicalBackupsComponent.test.tsx`: `shows no alert and refreshes again when the refresh after a successful cancel fails` presses the named cancel button instead of the icon image.

New scenarios: `ui/wal-stream-monitoring/WalStreamMonitoringComponent.test.tsx`, `ui/physical-backups/PhysicalRestoreComponent.test.tsx`, the new groups of `ui/physical-backups/PhysicalBackupsComponent.test.tsx`, the dark theme run of `frontend/browser/scenarios/wal.spec.ts` and `frontend/browser/scenarios/physicalBackups.spec.ts`.

## Guards

These new tests already held on the unchanged code, so they pin behavior this change must keep rather than a defect:

- `ui/wal-stream-monitoring/WalStreamMonitoringComponent.test.tsx`: `keeps the message and the busy button when the browser tab is hidden during the request`.
- `ui/physical-backups/PhysicalRestoreComponent.test.tsx`: `requests one token for a backup and shows commands without a target time`, `requests a token for the latest moment on open and for the chosen time afterwards`, `shows the reason with its hint and no commands when the newest request is refused` and `ignores an answer after the screen closes and starts its own request when it opens again`.
- `ui/physical-backups/PhysicalBackupsComponent.test.tsx`: `offers a member the backup start`, `reports a refused start once, keeps the rows and leaves the start usable`. `changes nothing when the config answers with a failure after the user left` passed its assertions on the unchanged code, but that run reported the screen's unhandled rejection.

## Browser findings

`frontend/browser/scenarios/physicalBackups.spec.ts` found two defects, fixed here:

- The `FULL` badge's text had a contrast of 4.22 on a 15% accent fill in the light theme, so the fill is 10%, with a contrast of 4.54, and `INCREMENTAL` is 5%.
- Closing a restore dialog lost the focus, because the dialog was unmounted before its close animation could return it. The list's backup restore, point-in-time restore and error details dialogs now stay mounted while closed and render their content only while open. The delete `ConfirmationComponent` restores focus on its own when it unmounts, so it stays as it was.

## Final

Final `pnpm --dir frontend test` on 2026-11-02: 170 files, 1969 tests passed. `pnpm --dir frontend test:browser --project=ui`: 102 passed.
