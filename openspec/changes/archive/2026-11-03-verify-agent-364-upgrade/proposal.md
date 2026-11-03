## Why

The current server rejects the heartbeat sent by verification agent 3.64.0, so continued verification depends on the running agent downloading and restarting into the current binary. Existing self-update coverage rebuilds current sources with an older version number, leaving the released agent's upgrade path and recovery from a failed binary replacement unverified before release.

## What Changes

- Extend upgrade coverage with the actual verification-agent binary distributed in release 3.64.0, obtained from a pinned release image and checked for its version.
- Exercise the background update after observing the old heartbeat's rejection, then require an accepted heartbeat and a completed verification of a real backup with the replacement binary.
- Exercise a binary directory that the agent cannot write, verify that self-update fails without silently replacing or restarting the binary, and prove that manual replacement and restart restore verification.
- Repair a WAL shutdown test setup race exposed by the required full run, preserving its existing slot-retention and restart-time assertions.
- Correct the unconditional previous-release compatibility claim in `backend/api/agent.yaml`, regenerate the API artifacts, and qualify the shared-contract claim in `docs/architecture.md` with successful self-update.
- Document automatic update requirements, failure symptoms and manual recovery in the repository documentation. Website documentation and translations are excluded at the user's request.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `upgrade-test`: Require coverage of the released 3.64.0 verification agent's background upgrade, failed replacement, manual recovery and a completed backup verification after recovery.

## Impact

- Governed by root `AGENTS.md` and `backend/AGENTS.md`, with the testing and API-design skills applying to the affected tests and contract description.
- Affects `backend/e2e/upgrade/`, its release-fixture preparation, WAL shutdown test setup in `backend/internal/modules/backups/physicalapitests/walstreaming/`, `backend/api/agent.yaml`, generated API artifacts where regeneration changes them, `docs/architecture.md` and `docs/testing.md`.
- No new request format or compatibility shim. The memory-field break already exists; this change verifies and explains recovery from it.
- The companion change `require-fresh-heartbeat-for-verification-claims` owns withholding assignments from agents without a fresh accepted heartbeat. This change must pass before and after that restriction; its failure scenario does not require the old agent to receive assignments.

## Out of Scope

- Changing assignment eligibility, retry limits, heartbeat intervals, cancellation delivery or the stale-agent timeout.
- Accepting `maxRamGb` heartbeats, restoring removed API shapes or modifying the historical agent.
- New interface controls, alerts, automatic permission repair or changes to agent registration counts.
- A general historical-release matrix, release deployment, commits or publishing.
- Website pages, website documentation and translations.
