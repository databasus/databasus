## Why

The verification agent's protocol is still written by hand in two places. The server side is the old Gin code in `backend/internal/features/verification`: two controllers, the agent token middleware, their DTOs and a bridge to five temporary methods of `verification/module.go` that `verification-module` left for it (`backend/internal/modules/verification/module.go:62-68`). The agent side is `backend/internal/agent/api`, with its own DTOs and a `resty` client. Nothing checks that the two copies agree, and the old routes skip the contract guard that every other route of the target layout passes.

The copies have already drifted on units. The heartbeat sends memory as `MaxRAMMb / 1024` (`backend/internal/agent/heartbeat/heartbeat.go:118`), so an agent given less than 1024 MB reports 0 GB. The agents list then hides its memory altogether (`frontend/src/features/verification/agents/ui/VerificationAgentsComponent.tsx:45`), and the claim, which already sends megabytes, disagrees with the heartbeat.

Milestone 12e of the backend rewrite plan describes the protocol once in `backend/api/agent.yaml`, generates the server, the agent's client and a contract-checking test client from it, and deletes the old agent code. Everything before it in milestone 12 is done, so this change also removes the last of `internal/features/verification`.

## What Changes

- **`backend/api/agent.yaml`** describes the four operations the agent calls: the heartbeat, the claim, the backup stream and the report. Their paths stay exactly as agents of the previous release call them. The agent token goes in `Authorization`, and a verification that no longer belongs to the agent answers `410`. Errors are `Problem` bodies with their own list of codes, which adds `verification_gone`.
- **The server** is generated into `internal/modules/verification/internal/agentapi` and mounted by `internal/app` on a second route group, behind a contract guard for `agent.yaml`. The module's agents service checks the agent's token and the per-agent rate limit. The rate limiter comes through `Deps` instead of the old global.
- **Clients.** `internal/lib/agentclient` is the generated client the agent uses. `internal/lib/agentclienttest` wraps it for tests and checks every response against `agent.yaml`. The response check of `internal/lib/apiclient` moves to a shared test package, so both test clients use one implementation.
- **The agent** switches from `resty` and its own DTOs to `lib/agentclient`. It keeps its retry policy, the report retry budget, the idle watchdog of the backup stream and its two hand-written calls to `/system/version` and `/system/verification-agent`. The agent no longer imports `resty`; it remains an indirect dependency required by rclone's Proton Drive backend.
- **Server failures stop dropping jobs.** Today the backup stream answers `410` to every error, a database failure included, and the report answers `400` to every error except "gone". In both cases the agent drops the job until the backend reclaims it. After this change only a verification that is no longer running or no longer the agent's answers `410`, a server failure answers `500`, and the agent retries it.
- **Memory in megabytes.** The heartbeat sends `maxRamMb`. The server still accepts `maxRamGb` from an agent of the previous release and uses `maxRamMb` when both arrive. The column `verification_agents.max_ram_gb` becomes `max_ram_mb`, with existing values multiplied by 1024.
  - **BREAKING** in the user API: the `VerificationAgent` schema of `openapi.yaml` replaces `maxRamGb` with `maxRamMb`. The field was added by `verification-module`, which has not been released, so no released script reads it. The agents list shows the memory with the unit that fits, such as "512 MB RAM" or "4 GB RAM".
  - **BREAKING** for the telemetry collector at `metrics.databasus.com`: an agent's entry sends `maxRamMb` instead of `maxRamGb`. The collector lives outside this repository and must accept the new field before the release that carries this change.
- **Removed:** `internal/features/verification` with the old controllers, the agent middleware, the bridge and the DTOs `agents/dto.go` and `runs/dto.go`; `internal/agent/api/dto.go`; the temporary agent methods of `verification/module.go`; the agent routes in `internal/app/routes.go`.
- **Tests.** The protocol API tests of the verification module switch from `CallOldAPI` to `agentclienttest`. A request the contract cannot describe, such as an agent ID that is not a UUID, uses a raw HTTP request, as the testing skill allows. New API tests cover the heartbeat in megabytes, the heartbeat of the previous release and the `500` of a server failure. The two memory assertions of the agent end-to-end update scenario switch to megabytes. No other part of the end-to-end suite changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: new requirements for the agent's authentication, its memory in megabytes, the heartbeat of an agent of the previous release, the checks of the capacity and the report, and when the backup stream and the report answer that a verification is gone.
- `api-contract`: the repository holds two contracts, the user API and the agent API. Both are generated by one command, checked by the lint and checked by their test clients.
- `api-http-server`: the access mark applies to the operations of the user API contract. Agent operations authenticate with the agent's token instead of a session.

## Out of scope

- Redesigning the agent protocol by the `api-design` skill. Its paths, request bodies and success responses stay as they are, because an agent of the previous release calls them until it updates.
- Describing `/system/version` or `/system/verification-agent` in `agent.yaml`, and moving the agent binary download out of `internal/features/system/agent`. Both paths never change, and the download controller moves with the rest of `features/system` in milestone 13.
- Removing the `maxRamGb` compatibility field from the heartbeat. The next release removes it, and this change adds that item to milestone 13 of the rewrite plan.
- Changing the telemetry collector, and moving telemetry into a module.
- Changing the claim, capacity, scheduling, retry or notification rules.
- Changing how the agent restores, verifies or updates itself.
- Removing rclone's indirect dependency on `resty` or changing storage providers to eliminate it.

## Impact

- **Contracts:** new `backend/api/agent.yaml`, embedded by `backend/api/api.go`; `backend/api/openapi.yaml` (`VerificationAgent.maxRamMb`); `backend/api/vacuum.yaml` or its counterpart for `agent.yaml`, with the documented exception for the frozen path segments.
- **Backend, new:** `internal/modules/verification/internal/agentapi`, `internal/lib/agentclient`, `internal/lib/agentclienttest` and the shared response check.
- **Backend, changed:**
  - `internal/modules/verification`: the agent authentication, the heartbeat in megabytes, the `410`/`500` split, `RateLimiter` in `Deps`, and no temporary methods in `module.go`;
  - `internal/lib/apiclient` on the shared response check, and `internal/lib/apierror` with `verification_gone`;
  - `internal/app`: the second route group, the module's `Deps`, and `apptest` with an agent client and `MaxRAMMb`;
  - `internal/agent/{api,heartbeat,runner,upgrade,start}` on `lib/agentclient`;
  - `internal/features/telemetry`: the agent entry in megabytes;
  - a migration of `verification_agents.max_ram_gb` and its test in `backend/migrations`;
  - `mise.toml`: the contract lint covers `agent.yaml`.
- **Backend, deleted:** `internal/features/verification`, `internal/agent/api/dto.go` and `internal/agent/api/enums.go`. `github.com/go-resty/resty/v2` becomes an indirect dependency required by rclone. `github.com/hashicorp/go-retryablehttp` takes over the agent's retries and becomes a direct dependency instead of an indirect one.
- **Frontend:** the generated types, `VerificationAgentsComponent.tsx` with its test, and the `ram` string of the six interface dictionaries.
- **Running instances:** the migration converts the stored memory of every agent to megabytes. Agents of the previous release keep working without an update: the paths do not change, the server accepts their heartbeat, and they read only the status code of an error.
- **Documents:** `docs/architecture.md` (agent operations, their authentication and where their server lives), `docs/testing.md` (the agent protocol tests no longer use `CallOldAPI`), `docs/glossary.md` (the two contracts, the agent test client and the shared contract check), the `api-design` skill (where `agent.yaml` is served, and that frontend texts are needed only for codes of `openapi.yaml`), and the rewrite plan in `temp/refactor/plan.md`.
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
