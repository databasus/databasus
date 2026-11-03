## Why

The project's API rule says a long operation answers 202 with the resource in its in-progress state, yet starting and cancelling a physical backup both answer 202 with an empty body. A client that cancels a backup has to read the list again for an outcome the server already recorded. Start is a different case, because a start only records a request and the scheduler creates the backup later, and nothing written down says this is deliberate. A start can also be accepted with 202 when no backup will follow, because backups are turned off or no storage is set. The contract is cheapest to change before the release, because afterwards each of these becomes a break for users' scripts.

## What Changes

- **BREAKING** Cancelling a physical backup answers `200` with the backup in its final status instead of `202` with no body. Scripts that check for `202` must check for `200`. The `202` response of this operation is deleted, with no shim.
- **BREAKING** Starting a physical backup of a database whose backups are turned off or whose config names no storage answers `400` with `code` `invalid_request` and records no request. Today it answers `202`. Callers that relied on a start being remembered until backups are turned on get an error and must start the backup after turning them on.
- Starting a physical backup keeps answering `202` with no body. `docs/architecture.md` records this as a reasoned exception to the long-operation rule, and the `api-design` skill links to it.
- `totalUsageMb` stays required on every page of the physical backup list. `docs/architecture.md` records the decision, and the `api-design` skill makes clear that its ban on totals concerns paging counts.
- The "Back up now" button on the physical backup list is disabled while the loaded config has backups turned off or names no storage, so the screen does not offer a start the server now rejects. One new translated sentence says what a start needs, because the existing notice speaks only of scheduled backups and does not appear for a config with backups on and no storage.

## Out of scope

- Logical backups, restores and verifications. `cancelLogicalBackup` keeps its `202`, because it returns the backup while it is still `IN_PROGRESS`.
- Creating the backup row inside the start request, and any change to the queue of requested backups that the scheduler, WAL streaming and the storage switch share.
- A new `GET /physical-backups/{backupId}` operation. Nothing in this change needs one.
- Moving `totalUsageMb` to its own operation, and the cost of the physical list query.
- A request left pending when backups are turned off after it was accepted. It still runs if backups are turned on again.
- A new `ProblemCode`. The refusal uses `invalid_request` with a field problem, as `startLogicalBackup` does for a config without a storage.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/operations`: a start with backups turned off or without a storage is refused, and a cancel answers `200` with the backup in its final status.
- `workspace-access`: the member's cancel answers `200`.
- `frontend-database-screens`: the physical backup start is unavailable while the loaded config has backups turned off or names no storage.

## Impact

- **API contract:** `cancelPhysicalBackup` response and description, and the `startPhysicalBackup` description in `backend/api/openapi.yaml`. Generated servers, test clients and frontend types are regenerated with `mise run generate-api`.
- **Backend:** the `backups` module's physical request and cancellation services, its list filter and its HTTP handler, plus the tests and test helpers that expect `202` from a cancel.
- **Frontend:** `PhysicalBackupsComponent`, one key in each of the six interface dictionaries, and the test and browser fixtures that use a config with no storage. The browser check also exposed a flaky enlarged-text scenario, repaired in `enlargeText`, `assertAccountReflow` and the storage type row of `EditStorageComponent`. The API wrapper keeps ignoring the cancel body, and the list still refreshes after a cancel.
- **Documentation:** `docs/architecture.md` and `.agents/skills/api-design/SKILL.md`.
- **Rules this change answers to:** `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
