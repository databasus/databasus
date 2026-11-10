## Context

See `proposal.md` for motivation and scope. This design is needed because sequence state and transactional schema changes require different recovery expectations.

| Evidence | Consequence |
| --- | --- |
| `docs/architecture.md:275` delegates physical backup format, incremental combination and replay to PostgreSQL's native tools. | Exercise the existing downloaded restore set and recovery script rather than implement a second restore path. |
| `backend/internal/app/apptest/physicaltest/source_markers.go:19` creates a marker table without generated IDs. | Row markers cannot establish preservation of defaults, sequence state or catalog definitions. |
| `backend/internal/app/apptest/physicaltest/restore_chain.go:11` and `:37` restore a FULL or a FULL with two incrementals with streaming disabled. | Keep those scenarios and add coverage for changes present only in later streamed WAL. |
| `backend/internal/app/apptest/physicaltest/pitr_chain.go:10` already builds incrementals, chooses a source-clock target and waits for contiguous archived WAL. | Reuse those operations, without adding unrelated schema work to every caller of the marker fixture. |
| `backend/internal/app/apptest/physicaltest/restore_rls.go:15` compares source rows and RLS configuration with a running restored cluster. | Reuse this connection-based observation pattern for a broader, bounded set of objects. |
| `backend/internal/app/apptest/physicaltest/source_clock.go:34` places margins around a target; `backend/internal/app/apptest/physicaltest/recovery_script.go:49` formats the target with whole-second precision. | Preserve the margins and place commits strictly on either side of each serialized target. |
| `backend/internal/app/apptest/physicaltest/restored_cluster.go:72` waits until recovery has finished. | Perform generated-ID inserts only after the restored cluster becomes writable. |

The implementation follows the root `AGENTS.md`, `backend/AGENTS.md`, the testing skill and the test-layer boundaries in `docs/testing.md`. Source objects are prepared through SQL because Databasus has no API to create user tables. Application resources, backup requests and restore downloads continue through the existing test application's API helpers.

## Goals / Non-Goals

**Goals:** Distinguish backup combination from later WAL replay, establish the transaction boundary of schema recovery, and provide SQL preparation and assertions reusable by future HA tests through a PostgreSQL connection.

**Non-Goals:** Introduce a schema-diff framework, assert every PostgreSQL catalog field, or equate recovered sequence positions with an exact source snapshot. The wider exclusions are in `proposal.md`.

## Decisions

### Reuse the physical restore harness and separate SQL checks from orchestration

Keep the new fixture under `backend/internal/app/apptest/physicaltest/`, split into domain units for schema recovery and sequence recovery. Its preparation and assertion functions take the fixture's object names and a `*pgx.Conn`; they do not require a `Team`, container or backup catalog. Export only the operations later callers need. Name objects uniquely when a source can be shared, and quote SQL identifiers safely.

The existing harness continues to own source creation, backups, WAL waits, downloads, recovery-script execution and target cleanup. New entry points in `pg17` and `pg18` delegate to the same scenario runners. No production module imports the fixture. This preserves the test-support placement and engine/domain boundaries in `docs/architecture.md` and `backend/AGENTS.md`.

Alternative rejected: copying SQL into version-specific or future HA tests would allow their assertions to drift. A configurable pipeline or new recovery framework would add lifecycle and abstraction work for a small fixed matrix.

### Use explicit stages and four recoveries per engine version

Use these top-level tests in each version package:

- `TestRestorePhysicalSchemaAndSequences`: latest recovery from FULL plus WAL, and latest recovery from FULL plus two incrementals plus WAL.
- `TestRestorePhysicalSchemaAndSequencesToTargetTime`: recovery before and after the schema transaction commit.

Each scenario gets an isolated application, source and restore target. Independent scenarios run in parallel; ordered SQL phases within one scenario remain sequential. Existing marker, RLS, SSH and host-restore scenarios retain their coverage.

```text
baseline --> FULL --> A --> B --> C --> D --> archived WAL
baseline --> FULL --> A --> INCR1 --> B --> INCR2 --> C --> D --> archived WAL

C: BEGIN --> execute DDL --> T_before --> COMMIT --> T_after
```

| Stage | Representative operations |
| --- | --- |
| Baseline | Create a table with a serial primary key and stored values for a later type conversion, an explicit logged sequence used by a column default, and disposable objects for rename/drop checks. |
| A, after FULL | Create a schema, an identity table, an added column with a default, an index and a constraint; insert generated IDs and exercise sequence calls and rollback. |
| B, between incrementals | Rename a column, convert an integer column to text using an expression that forces a table rewrite, rename a table and drop a disposable object. Verify converted rows. |
| C, after the final backup | Create an object available only through later WAL; alter a logged sequence's parameters, perform controlled forward sequence operations, and include transactional rename/drop changes whose prior definitions must survive early PITR. |
| D, after the later target | Commit further changes to disposable objects so latest recovery and the later PITR target have distinguishable schemas. |

For FULL plus WAL, omit both incrementals and keep all post-FULL stages. For the incremental route and both PITR routes, take the final incremental before either target. Never take a later backup that could absorb C or D. Assert the completed FULL and incremental counts through the catalog: zero incrementals for the first route, two for the others.

C uses an explicit transaction. A second source connection supplies the clock while C is open, so `T_before` follows its SQL execution but precedes its commit. Read committed expectations for the preceding stage before opening C to avoid waiting on its locks. Place `T_after` strictly after the commit and D strictly after that target. Include a separate rolled-back transaction with create, alter and drop operations; its changes must be absent from every later recovered state.

Alternative rejected: changing objects only before the last incremental would let a test pass without exercising subsequent WAL. A separate recovery for every SQL command would multiply container and backup cost without testing another restore route.

### Observe semantic catalog definitions and exercise the recovered objects

Read a bounded catalog projection for the named fixture objects: schemas and relations, column names and types, nullability and defaults, identity mode, index definitions and validity, constraint definitions, sequence definitions and column dependencies. Use schema-qualified names and decoded definitions rather than raw OIDs or a dump of the entire cluster. Capture committed stage observations before later mutations and verify them against explicit fixture expectations before using them as the recovery reference.

After recovery, compare the required stage's rows and semantic definitions. Require old names and deleted objects to be absent. For the rewritten column, check both its final type and the transformed values. Exercise valid and invalid writes to establish that recovered defaults and constraints work, handling rejected writes without poisoning the connection's next transaction.

At the end of a successful recovered-target check, use disposable fixture objects to establish observation sensitivity: remove a representative added column and change a sequence cache, then show that the same catalog observations report those differences from the saved state. Do this after all recovery assertions and before cleanup, without building a general fault-injection framework.

Alternative rejected: row counts cannot detect a missing column, incorrect default or inactive constraint. Whole-catalog snapshots include unrelated system state and make failures harder to interpret.

### Separate sequence definitions from recovered positions

Cover serial, identity and explicit sequence defaults. Include `CACHE 1` and a larger cache such as 32, both pre-FULL and post-FULL objects, direct `nextval`, forward `setval` with both values of `is_called`, and committed `ALTER SEQUENCE` changes. Read type, start, increment, bounds, cycle, cache, default and dependency information through the fixture's catalog projection.

Use ascending, noncycling sequences with generous bounds. All explicit repositioning moves forward. Track committed generated IDs and a known forward floor for each relevant stage. A latest recovery must produce IDs beyond the restored committed IDs and respect a recovered forward `setval`; merely avoiding an immediate duplicate does not establish that `setval` survived. For each recovered sequence, perform the first generated-ID insert before any standalone allocation probes and require its ID to exceed the largest restored committed ID. Check every subsequent allocation against that bound and the stage's forward floor, including allocations from separate connections for the cached case. Within a controlled connection, check the recovered increment without requiring the first post-recovery value to equal the last source value plus that increment.

Exercise rolled-back `nextval` and `setval` separately from transactional `ALTER SEQUENCE ... RESTART`. On the live source, confirm the calls' rollback behavior under controlled conditions, then commit subsequent generated-ID rows. After recovery, require the rolled-back rows to be absent and the sequence to remain usable. For rolled-back sequence DDL, require the committed parameter definition to survive. Do not assert that aborted sequence calls have an exact durable position after recovery, or that sequence values at early PITR exclude every call from a later-committing transaction.

If a disputed exact-position assertion becomes necessary, investigate it using native recovery from the same base and WAL files at the same target and PostgreSQL version. Use a separate recovery copy, and observe it before any generated-ID insert changes its state. This is a bounded diagnostic step, not a mandatory second restore matrix or permission to weaken a reproduced failure. Prefer the documented semantic assertions when they already decide correctness.

Alternative rejected: `next ID = max(id) + 1`, exact `last_value` equality and treating sequence calls like row writes would report legitimate PostgreSQL behavior as data loss. Checking uniqueness alone would miss a lost forward repositioning.

Sources: [sequence functions](https://www.postgresql.org/docs/18/functions-sequence.html), [sequence caching](https://www.postgresql.org/docs/18/sql-createsequence.html), [transactional sequence restart](https://www.postgresql.org/docs/18/sql-altersequence.html).

### Prove WAL reachability before requesting recovery

Reuse `sourceTime`, `targetTimeWithMargin`, WAL rotation and `waitForReplayableThroughLSN`. Record a position after D commits, close the containing segment and wait for contiguous archived coverage through that position. Request recovery only after that condition holds. Both PITR tests must have the later commit available, so absence of later DDL cannot be explained by an incomplete archive.

Targets are chosen after the final backup completes. Recovery uses the database restore token, downloaded set and served script; generated-ID checks wait for `pg_is_in_recovery() = false`. Source and target processes and files remain owned by existing cleanup helpers.

Alternative rejected: fixed sleeps or a green backup status provide no proof that the required WAL has been published. A target within the chosen base backup is invalid for native recovery. See [PostgreSQL recovery requirements](https://www.postgresql.org/docs/18/continuous-archiving.html).

## Risks / Trade-offs

- Sequence gaps or WAL prelogging mistaken for loss -> Compare definitions, committed-ID safety and controlled forward floors; use native recovery only for unresolved exact-position questions.
- C holds locks while its target is chosen -> Use a separate clock connection and capture the preceding stage before opening C; give every operation a bounded context.
- Extra recoveries increase test time -> Use four scenarios per version and reuse the harness. Measure actual runs before changing legitimate timeouts; keep existing condition-based waits.
- A post-backup change is accidentally captured by a new backup -> Assert backup counts and complete all selected backups before C and both PITR targets.
- Fixture expectations accidentally accept a missing definition -> Check explicit source-stage expectations, recovered catalog values and the bounded sensitivity mutations.
- A test exposes a production defect -> Keep the failing reproduction, repair only the responsible path, rerun its regression and affected package tests, and update these artifacts if the repair changes the implementation details. Unrelated HA work remains excluded.

## Migration Plan

No deployment or data migration is required. Land the SQL fixture with the first PostgreSQL 17 FULL-plus-WAL scenario, extend it to incrementals and PITR, then enable the same runners on PostgreSQL 18. Document the coverage and focused commands in `docs/testing.md`. Run the targeted e2e tests, `mise run test`, backend formatting and lint before implementation review. Reverting this change removes test support and the newly explicit specification guarantees without changing persisted data or API shapes.

## Verification

Local verification on November 10, 2026 passed:

- `mise run doctor`: the Codex review client and its trusted project hooks are available.
- After the Fedora preparation in `docs/testing.md`, `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences' ./internal/modules/backups/e2e/postgresqlphysical/pg17 ./internal/modules/backups/e2e/postgresqlphysical/pg18`: all eight recoveries passed. The final runs took 78.600 seconds for PostgreSQL 17 and 78.686 seconds for PostgreSQL 18 while the general test suite also ran. They include first-insert and subsequent-allocation safety, catalog sensitivity mutations and assertions that rolled-back sequence writes leave no rows.
- A temporary negative probe reset the recovered accounts sequence to 1 before the allocation assertion. The first generated-ID insert failed with SQLSTATE `23505` on `accounts_pkey`, confirming that the assertion detects reuse of a restored ID. The probe was removed before the final successful matrix run.
- `mise run backend:lint`: generated-code checks, API contract checks, formatting, Go lint, architecture checks and the module-file check passed.
- `mise run test`: both backend groups passed; the frontend passed 194 test files and 2,864 tests.
- `openspec validate verify-physical-schema-and-sequences --strict` and `git diff --check`: passed.

The scenarios reproduced no loss of schema objects or sequence behavior. Application behavior code did not require a repair. Exact sequence-position comparison with a second native recovery was unnecessary because the documented semantic assertions decided every case. Mandatory implementation review remains a separate step from these checks.
