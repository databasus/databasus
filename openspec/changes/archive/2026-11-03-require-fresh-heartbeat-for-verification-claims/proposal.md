## Why

An authenticated verification agent can claim work without ever sending an accepted heartbeat, or after its last accepted heartbeat has expired. An incompatible agent whose self-update fails can therefore consume assignments that the scheduler subsequently requeues or fails instead of leaving them available for a working agent.

## What Changes

- **BREAKING**: Require an accepted heartbeat no more than five minutes old before assigning new work. Remove the unconditional assignment path for authenticated agents that have never heartbeated successfully or whose accepted heartbeat is older than five minutes; add no bypass or compatibility fallback.
- Return the existing empty `204` claim response for an otherwise valid request from an ineligible agent, leaving pending work and its attempts unchanged.
- Keep rejected heartbeats from establishing or refreshing eligibility. A rejected heartbeat preserves the last accepted timestamp and capacity; an earlier accepted heartbeat remains usable until it expires.
- Allow claims again after a valid heartbeat, without re-registering the agent or changing its token.
- Cover the absent, rejected, fresh, exact-boundary, expired and recovered heartbeat cases, and show that a healthy second agent can claim work withheld from an ineligible agent.
- Define assignment eligibility in `docs/glossary.md`, distinguishing admission to new work from agent registration and capacity. Document its condition in the agent contract and architecture, regenerate API artifacts, and adapt test setup to report a valid heartbeat before a claim that is expected to succeed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: Make a fresh accepted heartbeat a prerequisite for new verification assignments, with observable no-work and recovery behavior.

## Impact

- Governed by root `AGENTS.md` and `backend/AGENTS.md`, with the testing and API-design skills applying.
- Affects the verification module's claim service and HTTP tests, agent protocol test setup, `backend/api/agent.yaml`, generated API artifacts where needed, `docs/architecture.md` and `docs/glossary.md`.
- Changes claim behavior for protocol clients that previously claimed before reporting their capacity. Production agents already heartbeat independently; a startup race returns no work and recovers on a later claim.
- No migration, new API field or error code is needed. The registered-agent list and count retain their management meaning; ineligible agents are excluded from new assignments, not deleted or hidden.
- Can be implemented independently of `verify-agent-364-upgrade`. That change owns the real released-binary fixture and self-update/recovery tests.

## Out of Scope

- Invalidating a still-fresh accepted heartbeat immediately because a later request is malformed, rejected, rate-limited or unauthenticated.
- Changing the five-minute timeout, the scheduler's existing stale-run recovery, authentication or request validation precedence.
- Blocking backup streams or reports solely because the owner's heartbeat is old; existing ownership and lifecycle checks continue to govern them.
- Redefining registered-agent counts, hiding offline agents, changing interface status labels or requiring an online agent before a user can enqueue a verification.
- Version negotiation, compatibility shims, changes to historical binaries or changes to the self-update mechanism.
