## Why

The physical restore tests prove that rows survive, but they do not cover a representative set of schema and sequence changes made after a FULL backup. Before reusing these checks in HA scenarios, we need evidence that a restored PostgreSQL 17 or 18 cluster has the intended structure and can generate IDs safely; no loss of these objects has been reproduced yet.

## What Changes

- Add a reusable SQL fixture and assertions for schema changes, sequence definitions and generated IDs, using the existing physical restore harness.
- Exercise FULL plus subsequent WAL without incrementals, FULL plus two incrementals and subsequent WAL, and PITR before and after a transaction that changes the schema.
- Verify catalog definitions, restored rows, rolled-back DDL and new inserts after recovery. Account for PostgreSQL sequence caching and recovery semantics rather than requiring gapless IDs.
- State the restored-schema and sequence guarantees explicitly in the physical backup operations specification.
- Fix production behavior only if a new scenario reproduces a defect, preserving the failing reproduction before the limited repair.

### Out of scope

- Patroni, CNPG, Kubernetes and managed-service compatibility audits.
- Promotion, source identity checks and recovery across timelines.
- Exhaustive DDL coverage or preservation of UNLOGGED object contents.
- New screens, HTTP operations, backup formats, migrations or a general-purpose test framework.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/operations`: Define preservation of committed schema changes and usable sequence configuration through physical recovery, including the transaction boundary of PITR and PostgreSQL's allowance for sequence gaps.

## Impact

- Backend test support under `backend/internal/app/apptest/physicaltest/` and end-to-end scenarios under `backend/internal/modules/backups/e2e/postgresqlphysical/pg17/` and `pg18/`.
- SQL preparation and assertions accept PostgreSQL connections so later HA tests can reuse them without duplicating assertions or taking on this task's backup orchestration.
- `docs/testing.md` gains the fixture's coverage, reuse boundary and focused run command. The application architecture and API contracts remain unchanged unless a reproduced defect requires a limited repair within the agreed behavior.
- The change follows the root `AGENTS.md`, `backend/AGENTS.md` and the testing skill. It adds no dependencies and plans no breaking changes.
