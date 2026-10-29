## Why

Verification is the last product feature before milestone 13 that still lives entirely in the old code. `internal/features/verification` holds about 8.6 thousand lines with its tests, built on package-level getters in `di.go`, Gin controllers that answer errors with the internal error's text, as `400` for configs and verifications and as `500` for most agent failures, and `storage.GetDb()` with transactions that do not join the caller's. It checks workspace access through the `workspaces_bridge` and `databases_bridge` and creates the "verification gone" error from text in two places (`backend/internal/features/verification/runs/repository.go:167`, `service.go:414`) and compares it by its text in one (`agent_controller.go:124`). Its scheduler is a ticker loop of its own, and its schedule uses `internal/features/intervals`, which exists only for it.

Milestone 12d2 of the backend rewrite plan moves verification into `internal/modules/verification` on the schema that `verification-schema` already prepared and the upgrade test already checks. The agent protocol stays served by the old code until `agent-contract`, because the protocol gets its own contract `agent.yaml` there, and the agent end-to-end tests from `agent-e2e-on-apptest` keep it in check through this move.

## What Changes

- **The module.** `internal/modules/verification` appears with `module.go`, `Deps`, `New` and its code under `internal/`: agents, configs, verifications, the scheduler job and the HTTP handlers. Repositories run through `database.Conn(ctx)`. Services open transactions with the `Transactor` from `Deps`, write audit events in the same transaction, check workspace access through `workspaces.Authorize`, read databases through `databases.Module`, backups and their files through `backups.Module`, and send notifications through `notifiers.Module.Notify` after the commit.
- **Reactions instead of wiring in the old code.** The module handles `LogicalBackupCompleted`, `LogicalBackupsDeleting`, `DatabaseCopied` and `DatabaseDeleting`. The handlers that `verification-schema` added to the old code move into it. A database copy copies only a saved config, because a missing one reads as the default. The scheduler becomes a job of `background.Runner`, and its schedule is a `lib/schedule` schedule. `internal/features/intervals` is deleted.
- **"Verification gone" becomes a sentinel error** in the module's `errors.go`, which the old agent controller answers with `410` as today.
- **11 user operations enter `backend/api/openapi.yaml` under the tag `verification`**, designed by the `api-design` skill:
  - the number of agents, for any signed-in user;
  - create, list, rotate the token of and delete an agent, marked `x-access: admin`;
  - read and save the verification config of a database;
  - start a manual verification, which answers `202`, cancel one, read one, and list the verifications of a database newest first with a cursor.

  Errors are `Problem` bodies with a stable `code`.
- **The agent protocol stays as it is until `agent-contract`.** The heartbeat, the claim, the backup stream, the report and the agent token check stay in `internal/features/verification`, on temporary methods of `module.go`, one per route. Their paths, bodies and status codes do not change, so agents of the previous release keep working.
- **Consumers.** The system health check and telemetry read agents and configs through the module's contract. The methods they need stay in it until milestone 13 moves those consumers. The methods of `backups/module.go` that the old verification code used become ordinary methods of the contract for the module.
- **The frontend** calls the 11 operations through the generated client. `verificationAgentApi.ts`, `verificationConfigApi.ts` and `verificationRunsApi.ts` use it, the hand-written models of `entity/verification/*/models` give way to the contract's types apart from the label dictionaries, and errors reach the user through `translateApiError`. The verification list follows the cursor with its existing infinite scroll.
- **Tests.** The controller and service tests of `internal/features/verification` become API tests of the module on `apptest`. The agent protocol tests become API tests through `CallOldAPI` against the old controller. The pure logic of capacity and retries keeps unit tests. The user calls of the agent end-to-end helpers switch from `CallOldAPI` to the generated client. `SeedTestLogicalBackup` is deleted, because the tests make backups through the API. A `coverage.md` in the change maps every old test to its replacement.
- **The upgrade test** reads the agent, the config and the verification written before the upgrade through the new operations.
- **BREAKING** for users' scripts: every user route of verification moves and changes shape. `/verification/agents`, `/verification-config/{databaseId}` and `/verifications/...` give way to the operations of design.md, the list takes a cursor instead of `limit` and `offset` and no longer returns a total, the manual start answers `202`, and errors are `Problem` bodies. The agent's routes and the agent binary do not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: new requirements for the user operations: who may manage agents, how a manual verification starts and what it refuses, cancelling, and the paged list of a database's verifications.
- `workspace-access`: a new requirement that the verification operations of a workspace's databases follow the workspace roles, with the same answers as the other content operations.

## Out of scope

- `agent.yaml`, the generated agent server and client, the removal of `resty` from the agent, the memory unit of the heartbeat and the removal of the old agent controller and its DTOs. That is `agent-contract`.
- Any change to the agent protocol or the agent binary.
- Hard deletion of agents; an agent's row keeps its `deleted_at`, as `verification-schema` decided.
- Moving the system health check and telemetry into modules, which milestone 13 does. They switch only their source of verification data.
- Changing scheduling, retry, capacity or notification rules. The module keeps them as they are, with the same constants.
- New screens or a new design of the verification screens beyond what the generated types and the cursor require.

## Impact

- **Backend, new:** `internal/modules/verification`.
- **Backend, changed:**
  - `backend/api/openapi.yaml`, the generated code and `internal/lib/apierror`, with the new code `backup_not_completed`;
  - `internal/modules/databases/module.go` and `internal/modules/backups/module.go`: one batch read each, for the engine versions and the latest completed logical backups of several databases;
  - `internal/modules/backups/module.go`: the comments of the methods the old verification used, which now serve the module;
  - `internal/app`: construction, routes, the job, the subscriptions in `wiring.go`, and `internal/app/apptest/verification.go` on the generated client;
  - `internal/features/verification`: reduced to the agent routes, the agent token check and their DTOs, on `module.go`, plus a bridge that hands the module to the old consumers;
  - `internal/features/system/healthcheck` and `internal/features/telemetry`;
  - `internal/features/backups/config`: `SeedTestLogicalBackup` is deleted, and the bridge stays for telemetry, the health check and `internal/app` until milestone 13;
  - `internal/features/intervals`: deleted;
  - `backend/e2e/upgrade`: the checks of the current code.
- **Frontend:** `frontend/src/entity/verification`, the screens of `frontend/src/features/verification`, the manual start in `frontend/src/features/backups/logical/ui/LogicalBackupsComponent.tsx`, and, in the six interface dictionaries, the label of the after-backup trigger, which the screens lack today, and the text of `backup_not_completed`.
- **Not affected:** the schema, the agent and its protocol, and every stored agent, config and verification.
- **Running instances:** nothing to migrate. Agents of the previous release keep working, because the protocol does not change.
- **Documents:** `docs/glossary.md` (the verification config, and the API name of the verification agent) and `temp/refactor/plan.md` (the checklist, the table of temporary relaxations, and the correction that `internal/features/backups/config` stays until milestone 13). `docs/architecture.md` does not change: the module follows its rules, and its module table already lists verification.
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
