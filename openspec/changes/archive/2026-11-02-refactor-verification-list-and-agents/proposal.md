## Why

This change closes frontend milestone 7. The verification list, its detail drawer and the agent screen hold 129 occurrences of the style baseline, and some of their failures mislead the user:

- A failed first page of verifications reads as "no verifications", after a blocking alert.
- The detail drawer shows a blocking alert over its own error state, which gives no reason.
- A failed first load of agents reads as an empty list, and because the screen reports every failed load, a server outage repeats an error message every 15 seconds.

The list and the drawer also format durations with two different functions, so a restore that took one hour and two minutes reads `1h 2m 0s` in one and `62m 0s` in the other. The verification list and the agent screen each hold several concepts in one file.

## What Changes

- A failed first page of verifications shows its reason with a retry instead of the empty text. A failed next page keeps the rows and shows its reason once, and a failed cancel uses context-bound feedback.
- A failed drawer load shows its translated reason with a retry and no alert.
- The agent screen moves onto the shared polled paged list, which already provides the first-load failure, the retry and a silent refresh. A failed first load shows its reason with a retry, and Create stays disabled during any list load or deletion and after a failed load. While that failure is shown, only the retry reloads. A failed background refresh keeps the rows and shows nothing. The refresh no longer pauses behind an open dialog, because it cannot change what a dialog holds. A confirmed deletion reloads the list, as the logical list does, which adds one list request and a brief loading state.
- Durations use the shared `formatDuration`, so the list and the drawer agree. A verification shorter than a minute now reads `0m 5s`, as the physical backup list does.
- The verification list and the agent screen split by concept. Apart from that reload, their API calls, one-time token, install commands, polling intervals and page size stay as they are. Icon actions become keyboard buttons with names.
- Raw colors become semantic roles, which removes the milestone's last entries from `frontend/style-baseline.json`.
- `docs/frontend-design.md` stops describing these domain screens as unmigrated.

Nothing breaks an API, a schema or a config.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: the verification list and its details end their loads recoverably.
- `backup-verification`: the agent screen ends its loads recoverably, a failed refresh gives no feedback, and the interface shows an agent's token only once.

## Impact

This is the last change of frontend milestone 7, after `refactor-physical-backup-screens`. The root `AGENTS.md` and `frontend/AGENTS.md` apply, with the frontend-design and testing skills, and the frontend-localization skill if a control needs a new name.

Affected areas:

- `frontend/src/pages/databases/ui/verification`, for the list and the drawer;
- `frontend/src/pages/instance-settings/ui/verification-agents`;
- `frontend/style-baseline.json`, a new browser scenario, `docs/frontend-design.md`, and `temp/refactor/frontend-plan.md`, which marks milestone 7 done.

There is no backend change, migration or new dependency.

### Out of Scope

- Polling intervals, and any change to when a verification starts or how an agent reports.
- Real agent pipelines in a browser test, which stay in `mise run agent:test` and the backend suites.
- The remaining `ToastHelper` entries in `style-baseline.json` and the baselines themselves, which milestone 8 removes.
- The older uses of "verification runs" in `openspec/specs/backup-verification/spec.md` and `docs/architecture.md`, which disagree with the glossary and are reported separately.
