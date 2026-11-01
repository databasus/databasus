## Why

A monthly schedule can silently lose backups. The scheduler treats a day the current month does not have as a slot in the next month, so a schedule on day 31 runs in seven months of the year and produces no error in the other five. The interface makes this reachable for ordinary choices: schedules are stored in UTC, so "the 1st at 02:00" in Moscow is saved as day 31 or 30, depending on the month it was saved in.

The interface also shows and writes the wrong day. In November a schedule saved for day 31 shows as day 1, and in February as day 3. Editing the time then writes that wrong day back, and choosing day 31 in November saves day 1. Monthly verification schedules share all of this.

This goes first in frontend milestone 6, because it protects backups on its own and does not depend on the larger moves that follow.

## What Changes

- A monthly schedule whose day is past the end of the current month runs on that month's last day, at its time of day. The length of a month no longer skips a run. This applies to logical backups, physical backups and verification, which share one schedule rule.
- The interface converts a monthly day between UTC and local time without depending on the current month. The length of a month never changes the shown day, and a day round-trips through an edit unchanged. Only a daylight saving change can move the shown day, together with the shown time.
- The `dayOfMonth` description in `backend/api/openapi.yaml` states the last-day rule, and the generated code is regenerated from it. The field, its range and all operations stay the same.
- Nothing breaks for API callers. The one behavior change for existing configs is intended: a schedule on day 29, 30 or 31 now also runs in shorter months, on their last day.
- There is no migration. Saved days stay as they are. A day the old interface saved wrongly now shows as the day it really runs, and the user can correct it.

## Capabilities

### New Capabilities

- `recurring-schedules`: when a monthly schedule runs in a month shorter than its day, and how the interface shows and saves that day whatever the current month is.

### Modified Capabilities

None.

## Impact

This is the first change of frontend milestone 6 in `temp/refactor/frontend-plan.md`. The root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md` apply, with the api-design and testing skills.

Affected areas:

- `backend/internal/lib/schedule`, the rule the logical and physical backup schedulers and the verification scheduler share;
- `backend/api/openapi.yaml` and the code `mise run generate-api` produces from it, for the description only;
- `frontend/src/shared/time/utils.ts`. All six consumers (the logical, physical and verification editors and their show views) call it without changing;
- `temp/refactor/frontend-plan.md`, which records the milestone's split and the decisions behind it.

There is no database migration, new dependency or dictionary change. The change has a clear rollback: reverting the commit restores the old rule, and saved configs are untouched either way.

### Out of Scope

- Storing a time zone in a schedule. Without one, a local day near a month boundary can still land one day off in a shorter month, as `design.md` explains.
- Weekly schedule conversion, which does not overflow and is tested when the schedule unit is extracted.
- Catching up a monthly slot missed across a month boundary, such as an outage from the slot to the end of the month. That limit exists for every day of the month today and stays as it is.
- The rest of milestone 6. It follows as three later changes, written after this one so that each starts from the checkout as it then stands:
  - a mechanical move of logical backups and restores into `pages/databases`, with flat entities;
  - one schedule unit and editor for the logical and physical configs, with stricter schedule validation and shared field labels;
  - the logical config editor and the logical backup list and restore fixes, including the wizard draft reset, the Continue button after Back and the restore dialog crash when its backup disappears.
