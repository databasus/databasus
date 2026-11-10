## Why

After a connection loss, a saved PostgreSQL address can lead WAL reception to an unrelated cluster whose credentials, slot name, timeline and WAL filenames match the original source. The current receiver does not compare that connection's cluster identity with Databasus's saved identity, so foreign WAL can enter the original archive; archive contamination and its recovery consequences still need a failing integration reproduction.

## What Changes

- Verify the saved system identifier on the replication session that actually supplies WAL before allowing reception, including every reconnect and the first connection after an ordinary Databasus restart.
- Keep the stock WAL receiver and supply its upstream session through a private local bridge that checks identity before forwarding the session. Preserve upstream authentication, TLS/mTLS, SSH tunneling, cancellation and synchronous WAL writes.
- Check identity on each SQL session before it creates or rebuilds a WAL slot. An identity refusal must not delete or recreate a slot on the foreign cluster.
- Stop foreign WAL before it reaches the original queue, storage or catalog, while retaining existing backups and legitimate queued segments of the original cluster.
- Persist and expose `SYSTEM_IDENTIFIER_MISMATCH` through existing WAL monitoring and notifications. Resume reception through the normal retry mechanism after the expected source returns, without adopting another identifier.
- Prove both rejection and recovery with a switchable TCP endpoint, including SQL-to-A/replication-to-B routing, a restart with queued WAL, a return to A followed by an actual restore, and promotion of a real replica followed by a replacement FULL and restore.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/wal-archiving`: Bind every incoming WAL session to the saved cluster identity, preserve legitimate queued WAL after refusal, and resume against the expected source.
- `physical-backups/wal-slot-rebuild`: Require verified cluster identity before creating or rebuilding a WAL slot; identity refusal takes priority over rebuild triggers.
- `physical-backups/wal-stream-monitoring`: Report a persistent, specific identity mismatch through existing stream health, with correct running state and recovery behavior.
- `postgresql-physical-backup-failover`: Preserve same-cluster promotion, the prohibition on incrementals across timelines, and recovery through a new FULL behind an unchanged endpoint.

## Impact

- Backend: PostgreSQL physical engine connection handling and WAL reception; backup streamer supervision, slot management, persisted streamer diagnostics, monitoring and notifications; an additive migration for the persisted failure reason; API tests and physical recovery fixtures.
- API: add `SYSTEM_IDENTIFIER_MISMATCH` to the existing WAL stream reason enum. Keep endpoints, request shapes and streamer states unchanged. Consumers with exhaustive handling of response enum values must handle the new reason. Regenerate server/client types together; no compatibility aliases or fallback endpoint are introduced.
- Frontend: display the new reason and recovery guidance in existing WAL monitoring, with all interface dictionaries updated. No new screen or control.
- Documentation: describe session identity enforcement and recovery in `docs/architecture.md`. Main specs are updated when this change is synced.
- Dependencies: reuse the existing PostgreSQL protocol client and vendored receiver; no new production dependency or replacement client bundle is planned.
- Governing instructions: root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Out of Scope

- Automatically accepting another cluster, changing the saved identifier, or migrating archives between clusters.
- Validating local WAL headers, checksums or repeated filenames with different bytes; repairing archives contaminated before this change.
- Split-brain resolution or proving branch compatibility between divergent clones with the same identifier.
- PITR across timelines, changes to the existing backup-scheduler mismatch recovery policy, or new FULL scheduling rules.
- A SIGKILL matrix across publication phases, Kubernetes fixtures, new product settings, or a new interface.
- Replacing the native receiver with a custom WAL writer, patching vendored PostgreSQL binaries, or depending on another checkout or temporary task.
