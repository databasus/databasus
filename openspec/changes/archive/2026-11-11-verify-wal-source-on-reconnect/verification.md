# Verification

The checks use the pinned mise environment on Fedora with Docker available. Direct Go commands run from the repository root after `source scripts/fedora.sh` and `apply_libedit_shim`, with `GIN_MODE=release` and `TESTCONTAINERS_RYUK_DISABLED=true`.

## Reproduction and regression sensitivity

Before the implementation, the initial and reconnect split-routing scenarios reached independent cluster B with working credentials and a matching physical slot. B answered `IDENTIFY_SYSTEM`, and the unprotected receiver sent `START_REPLICATION`; both negative assertions failed. The test did not depend on an authentication or slot error.

After implementation, a temporary mutation replaced the retained replication session's identity comparison with an independent SQL inspection. SQL reached A while replication reached B. Both scenarios failed on an observed `CopyBothResponse` from B, with the diagnostic `foreign server entered CopyBoth replication on connection 6` for initial startup and connection 10 for reconnect. The mutation run exited nonzero after 44.524 seconds. The original session comparison was restored, and the entire WAL API package subsequently passed in 270.345 seconds. No precheck replacement or bypass remains in the implementation.

## Direct package checks

The latest direct engine, WAL API and recovery checks used one `go test -p 2` invocation with the package paths below. This limits simultaneous package processes without filtering scenarios.

| Command | Result |
| --- | --- |
| `go -C backend test -count=1 -timeout 40m ./internal/engines/postgresql/physical/...` | Passed: physical 32.288s, walmath 0.003s |
| `go -C backend test -count=1 -timeout 40m ./internal/modules/backups/physicalapitests/walstreaming` | Passed: 199.823s with session guards, mutation guards and the complete setup deadline |
| `go -C backend test -count=1 -timeout 40m ./internal/modules/backups/e2e/postgresqlphysical/pg17 ./internal/modules/backups/e2e/postgresqlphysical/pg18` | Passed: pg17 110.775s, pg18 36.433s |
| `go -C backend test -count=1 -timeout 40m ./internal/modules/backups ./internal/modules/backups/physicalapitests/catalog` | Passed: backups API 76.686s, catalog API 28.022s |
| `pnpm --dir frontend test src/pages/databases/wal-monitoring src/shared/i18n` | Passed: 23 files, 292 tests |
| `mise run backend:lint` | Passed: 0 issues; generated code, formatting, API rules and architecture checks passed |
| `mise run frontend:lint` | Passed: formatting, architecture, styles, types and generated API types |
| `pnpm --dir frontend build` | Passed; Vite reported its runtime-script and bundle-size warnings |
| `openspec validate verify-wal-source-on-reconnect --strict` | Passed |
| `mise run test` | Passed, exit status 0; frontend 194 files / 2,866 tests, all selected backend packages passed |

The additional affected packages `./internal/modules/backups/physicalapitests/streamstatistics`, `./internal/modules/backups/physicalapitests/backuping` and `./internal/lib/process` passed in the affected-package Go run in 26.734s, 36.416s and 2.023s respectively.

The final full run also passed the physical engine in 77.141s, WAL API in 265.710s, backup API in 44.306s, catalog API in 15.127s, physical backup API in 71.250s, statistics API in 19.914s and migrations in 43.255s. Intermediate runs exposed the cleanup assertion and expectations that disabling deleted streamer state; those checks were corrected as described below. A superseded run was stopped through its normal cleanup before repeating the final run, which completed with exit status 0.

Engine coverage includes native PostgreSQL 17/18 receivers, one available WAL sender, fragmented protocol setup, owner-only socket permissions, rejection of a second local client, cancellation during identification and replication startup, rejected replication, asynchronous server slot release and socket cleanup. Existing API and recovery scenarios exercise mutual TLS and SSH forwarding, including a bastion outage.

API scenarios cover initial and repeated split routing, absent/lost/stolen foreign slots, identity refusal during rebuild, retained refusal across restart and stale claims, disabling streaming, notification settings and throttling, queued and staged A WAL, verified recovery and the separate manual-FULL rule after a backup identity failure. The upgrade test preserves populated saved identities, slot names and backup/WAL references while adding an empty diagnostic field.

Both recovery packages download and reconstruct real bundles. The negative control restores a pre-refusal A recovery point that requires the queued WAL and checks its control rows. The positive control promotes a genuine standby, compares identities, keeps the configured address, rejects an incremental across timelines and restores a replacement FULL with promoted-source rows.

## Repairs exposed by verification

The receiver cleanup test now waits for PostgreSQL to release its slot after the connection closes; immediate inspection raced the server's asynchronous cleanup under the full suite. The existing timeline-history failure test now changes proxy routing instead of corrupting the saved identity, so it can exercise history retry while respecting the new receiver guard. The backup-pause scenario counts completed FULL backups separately from the intentionally refused FULL.

The concurrent-disable regression holds a real receiver's observed terminal refusal before manager persistence, then disables through HTTP. It failed before the repair because the claim row had been deleted (expected one row, found zero; 2.394s). Disabling now retains the claim row, allowing terminal persistence after shutdown drains. The scenario verifies disabled monitoring, restart, re-enable and eventual reception from A. It passed together with configuration and notification scenarios in 33.090s.

The rebuild mutation scenarios let release probes reach A, then script the separate deletion, recreation or consumer-termination connection to B. The recreation case first deletes the slot on A, and the termination case keeps a real B consumer active with Databasus's application-name prefix. The proxy records extended-query commands and SQL identity responses as well as replication events. With guards intact, the scenarios and the original rebuild scenario passed in 78.326s. With only the three mutation guards temporarily bypassed, all three failed on the actual forbidden commands sent to B: `pg_drop_replication_slot`, `pg_create_physical_replication_slot` and `pg_terminate_backend` (78.095s). The original guards were restored.

The repeated-notification scenario observes the retry's B identity response and waits until the manager removes its live run after the notification decision. It drains deliveries without restarting the manager. Temporarily bypassing throttling made it fail on two deliveries instead of one (2.444s); throttling was restored.

Receiver test wrappers resolve their native bundle through `clienttools.FindBundleRoot` and `pgshared.ClientTools`. The latest direct full engine package passed with those paths in 32.288s on this x64 host; no native arm64 run was performed locally.

Retaining the claim changes neither the health level while disabled nor the allowed streamer-state values. The stopped receiver now remains `NOT_RUNNING`, matching the existing rule that `NOT_STARTED` means no receiver has ever run. The configuration, catalog and shutdown checks now assert the retained claim and disabled monitoring instead of requiring deletion of that row. Long-held manual replication consumers set `wal_sender_timeout=0` so an unrelated sender timeout cannot release a slot before the guarded mutation is exercised.

No `mise run test-full` was run.

The native-setup regression holds `START_REPLICATION` after the same replication session has returned the expected identity. Before the setup-deadline repair, the attempt remained blocked until the test's two-minute wait expired (121.129s package run). The receiver now keeps its setup budget until the first upstream `CopyBothResponse`; the regression observes a retryable exit with its caller still active, no streaming callback, removal of the private directory, closure of the old source connection and slot release, followed by actual streaming on a new attempt. The targeted receiver, cancellation and authentication scenarios passed in 31.670s; the complete physical engine package, including the strengthened connection-close assertion, passed in 32.288s.

Real PostgreSQL authentication scenarios call `Source.ReceiveWal` with an incorrect password or a source lacking a replication HBA entry. Both assert `WalReceiverFatal`, the server's typed SQLSTATE (`28P01` or `28000`) and no streaming callback. They cover upstream authentication before local handoff, independently of the synthetic child-stderr classification checks.

After the setup-deadline and authentication coverage changes, `mise run backend:lint` passed with zero issues and `mise run test` completed with exit status 0. The frontend passed all 194 files and 2,866 tests; all selected backend packages passed, including the complete engine and WAL API scenarios. OpenSpec strict validation and the whitespace check passed.
