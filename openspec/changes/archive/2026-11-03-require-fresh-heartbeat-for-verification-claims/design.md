## Context

See `proposal.md` for the problem and the approved behavior break. Root `AGENTS.md`, `backend/AGENTS.md`, the API-design skill and the testing skill constrain this change. `docs/architecture.md` places verification domain decisions in the verification module, validates HTTP input through its contracts and takes behavioral time from the injected clock.

- `backend/internal/modules/verification/internal/verifications/claim_service.go:15` validates capacity and immediately attempts assignment; neither it nor its internal claim function at line 33 checks heartbeat freshness.
- `backend/internal/modules/verification/internal/verifications/scheduler_service.go:141` requeues or fails a running job if its agent has never heartbeated or the last accepted heartbeat is older than the five-minute threshold in `constants.go:8`. The exact boundary remains fresh because the comparison is strict.
- `backend/internal/modules/verification/internal/agents/service.go:203` stores capacity and `last_seen_at` only for accepted heartbeat capacity. The gigabyte-only rejection test in `backend/internal/modules/verification/agent_contract_test.go:54` requires refused requests to preserve both.
- `backend/internal/modules/verification/internal/agentapi/handler.go:54` maps a nil assignment to `204`; `backend/api/agent.yaml:56` currently describes that response only as no work fitting available capacity.
- `backend/internal/modules/verification/protocol_test.go:34` provides a shared successful-claim helper without first sending a heartbeat. Such setup will need to meet the new public precondition.
- `backend/internal/modules/verification/internal/httpapi/agent_handler.go:10` returns registered-agent count, while `frontend/src/pages/instance-settings/model/agentStatus.ts:5` already distinguishes display statuses based on age. Neither is an assignment-admission mechanism.

## Goals / Non-Goals

**Goals:** Stop new assignment side effects before querying and locking pending work when an agent lacks a fresh accepted heartbeat. Preserve normal assignment, authentication, validation and recovery behavior for eligible agents.

**Non-Goals:** No new persisted eligibility state, heartbeat validity flag, API error code, online-agent count or UI policy. Do not reinterpret a later malformed request as revoking a previously accepted heartbeat. The real 3.64.0 executable tests remain owned by the companion change.

## Decisions

### 1. Use the existing accepted-heartbeat timestamp and five-minute threshold

An authenticated agent is eligible when `LastSeenAt` is present and is not before `now - staleAgentThreshold`. Read `now` from the verification service's injected clock. The existing scheduler threshold and strict comparison define the boundary: exactly five minutes is eligible; any greater age is ineligible. Use one small private predicate in the verification package for this shared decision if extracting it makes the claim and reaper agree without introducing another policy or dependency.

Keep heartbeat validation as it is. Failed validation, missing gigabyte-to-megabyte conversion and auth failures do not update `LastSeenAt`. If an earlier accepted heartbeat is still fresh, it remains the basis of admission until it expires. On an upgrade, an old agent may therefore retain eligibility briefly; this change prevents indefinite assignment consumption, not immediate rejection based on binary version.

Rejected alternatives: checking whether an HTTP request arrived accepts incompatible clients; treating `CreatedAt` as a heartbeat admits agents that never proved protocol contact; using the UI's 90-second online label changes the recovery policy; adding a validity column or resetting last-seen on a rejected request adds state and contradicts the existing preservation requirement.

### 2. Enforce admission in the verification domain service before assignment side effects

Keep authentication and contract validation ahead of the domain call. In `ClaimVerification`, preserve its nonnegative-capacity validation and then check the authenticated agent's accepted timestamp before calling the internal assignment function. The authenticated agent is already loaded from persistence for each request; use that snapshot and the clock at admission. A heartbeat accepted concurrently after this check can make the next claim eligible, without requiring synchronization across heartbeat and claim requests.

Return no assignment for an ineligible agent. Do not list running backups, lock candidate rows, assign ownership, increment attempts or record a new contact for that request. The existing handler returns an empty `204`. Document the additional `204` condition in `agent.yaml`, regenerate with `mise run generate-api` and describe it alongside stale-run recovery in `docs/architecture.md`.

Keep negative-capacity claims invalid even for ineligible agents. Authentication failures retain `401` and rate limits retain `429`; they must not become indistinguishable successful no-work responses. Streams and reports retain their existing ownership and lifecycle checks.

Rejected alternatives: admission in the handler puts a domain rule at the HTTP boundary; checking only in the agent cannot protect against the already released binary; assigning and immediately requeuing still consumes work and attempts; a new error code requires protocol handling changes without improving this polling interaction.

### 3. Keep registration, availability and queueing as separate observable concerns

Only assignment admission changes. Preserve the registered-agent count and administrative list, so an administrator can find, repair or remove an unavailable agent. Preserve the ability to enqueue manual or scheduled work while no agent is eligible; that work waits for a working agent under existing queue and timeout rules. Do not change deep-health reporting, UI labels or the reaper's treatment of already running jobs.

Add **Assignment eligibility** to the Product table in `docs/glossary.md`: whether an authenticated verification agent has accepted contact fresh enough to be considered for new assignments. Distinguish it from registration, which keeps the agent listed and counted, and capacity, which determines whether a particular verification fits its resource budget. Reference the existing `lastSeenAt` and `claimVerification` contract names rather than introducing a stored eligibility field; keep the timeout policy in the architecture and contract.

Rejected alternatives: hiding expired agents makes recovery harder; redefining the existing count breaks its registered-agent contract and changes frontend flows; refusing enqueue would make recovery depend on users submitting work again.

### 4. Reproduce failures through HTTP and advance the test clock

Start with module API tests for no accepted heartbeat and an expired heartbeat, with a fitting pending verification. Run them on current code and observe `200` assignment where `204` is required before implementing the guard. Assert pending status, absent owner, unchanged attempts and unchanged agent state after denial. Create all entities through the API.

Cover fresh contact, exact five-minute age and just beyond the boundary in isolated applications with an injected clock. Cover gigabyte-only rejected contact, repeated rejected contact after earlier success, a rejected request within the original valid window, and recovery after a valid heartbeat. Test a healthy second agent receiving the work and authenticated/invalid-request precedence. Include regression assertions for list/count visibility, enqueue while no agent is eligible and stream/report ownership behavior after expiry before a reaper pass.

Adapt successful-claim setup explicitly immediately after the guard passes its focused regression tests, before running broader protocol or module checks. Do not make the agent creation fixture heartbeat automatically: that would conceal the never-seen case. A helper may send a heartbeat before a successful claim only when its test owns that setup; stale-admission tests call the raw claim client. For repeated claims, report actual owned verification IDs, so setup heartbeats do not accidentally invoke dropped-job recovery. Retain raw claims for tests of malformed requests, authentication and no accepted heartbeat.

Rejected alternatives: direct repository tests miss the contract and handler behavior; fixed sleeps slow down boundary coverage and can be flaky; globally refreshing contact before every raw claim makes the negative scenarios impossible and can hide defects.

## Risks / Trade-offs

- Agent startup runs heartbeat and claim independently -> the first claim may return `204`; its normal polling loop retries after a successful heartbeat. Verify existing agent end-to-end scenarios with the full suite.
- Previously implicit test setup claims without heartbeat -> update each successful-claim fixture explicitly and rerun verification and agent coverage; do not weaken admission to keep old tests green.
- Request-scoped authentication can read the agent just before a concurrent heartbeat -> conservative denial lasts until the next claim; evaluate eligibility once at admission rather than holding a lock for an HTTP-wide lease.
- A fresh historical heartbeat can survive a server upgrade -> preserve its original expiration and let rejected heartbeats fail to renew it; immediate invalidation is a different product rule and remains out of scope.
- Both changes touch `agent.yaml` and architecture prose -> keep the changes separate and additive; preserve the compatibility clarification when adding the admission rule, whichever is applied first.

## Migration Plan

No migration or agent rollout is necessary. Existing accepted timestamps immediately govern claims after deployment. Previously unseen or expired agents wait until a valid heartbeat; working agents recover through their normal heartbeat loop. Rollback to the previous server restores unconditional authenticated claims, with the known assignment-consumption risk. Complete at the pre-commit boundary after contract generation, relevant lint/format checks, `mise run test` and mandatory review.
