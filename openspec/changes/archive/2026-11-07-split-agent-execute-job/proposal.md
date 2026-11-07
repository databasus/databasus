## Why

The verification agent's `executeJob` mixes resource ownership, restore preparation, restore verdicts and statistics collection across 384 lines. Separating those responsibilities will make future agent changes easier to review while preserving every current outcome; this remains a P3 maintenance change suitable for the next work on the agent.

## What Changes

- Keep `executeJob` as the lifecycle owner and a readable sequence of domain steps.
- Extract private functions for container preparation, archive restoration and restored-database verification, reusing the existing download and report functions.
- Preserve cancellation, disk-budget handling, failure diagnostics, restore exit-code classification, TimescaleDB sequencing and resource cleanup.
- Verify the refactor through the existing agent tests and real agent end-to-end suite without changing their scenarios.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is an internal refactor with `skip_specs: true`. The existing `backup-verification` and `secret-handling` requirements continue to apply without amendment.

## Impact

- Primary code: `backend/internal/agent/runner/runner.go`; private execution helpers may live alongside it within the same package.
- Governing instructions: root `AGENTS.md` and `backend/AGENTS.md`; architecture constraints come from `docs/architecture.md`, especially "Long-running work" and "Verification agent".
- Verification: existing agent tests, including `backend/internal/agent/runner/` and `backend/internal/agent/e2e/`.
- No API, dependency, configuration, schema or persisted-format changes. No breaking changes.
- Documentation additions are limited to this change's planning artifacts; existing architecture and product specifications stay applicable.

## Out of Scope

- Refactoring `app.New`, the agent claim loop, other agent packages or backend orchestration.
- Changing verdicts, report fields or messages, retries, cancellation policy, disk-budget policy or diagnostic sanitization.
- Adding a stage framework, new public interfaces, new configuration or new test scenarios solely for the extraction.
- Enforcing a target function length or splitting code that already represents one coherent responsibility.
