## Why

A review of the backend found defects that change what users get, and the code confirms them. A stopped container leaves running backups and restores unrecorded until the next start, although `logical-backup-outcome` already promises a shutdown fail message. Changing the storage of a physical backup configuration keeps every old backup, while the interface warns that they will be deleted. Background status updates can revert a password the user has just changed. A Google Drive token refresh prints token fragments to standard output, past the logger's masking. The same review found smaller hygiene problems: committed binaries, tests that never run, and dead code. They are cheap to fix together.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md) and [`backend/AGENTS.md`](../../../backend/AGENTS.md). It deletes one file under `agent/verification/`, so [`agent/verification/AGENTS.md`](../../../agent/verification/AGENTS.md) applies to that deletion. It changes no code under `frontend/` or `website/`.

## What Changes

The change has four parts, implemented in this order.

**1. Application shutdown**

- One handler in the application receives `SIGTERM` and `SIGINT`. The two other handlers are deleted, along with the one that was never started.
- The shutdown signal reaches the running logical backups and restores. A logical backup is recorded as failed with `backup interrupted by application shutdown`, and a restore is recorded as failed. Neither is recorded as cancelled.
- Before it exits, the application waits for the running backups and restores to record their outcome, up to a fixed limit. The HTTP drain runs inside the same limit.
- **BREAKING** (internal API): `config.StartListeningForShutdownSignal` and `config.IsShouldShutdown` are deleted. The process-wide shutdown signal can no longer be read as a plain boolean.
- **BREAKING** (operational): the whole application shutdown, the final log flush included, takes at most 8 seconds. Today the HTTP drain alone may take 10 seconds and the log flush 5 more, which exceeds Docker's default stop timeout and leaves the embedded PostgreSQL server no time for a clean shutdown.

**2. Backup and storage consistency**

- A restore reads the backup from the storage the backup was written to. Today it reads from the storage the backup configuration names at the time of the restore.
- Changing the storage of a physical backup configuration deletes the database's physical backups, as the interface already says. It also stops the running physical backups and restarts WAL streaming on the new storage.
- Saving a logical or physical backup configuration no longer tries to save the embedded storage object.
- Recording the last backup time, the last backup error and the health status of a database writes only those columns. The connection settings are no longer rewritten.
- The unused list of backup removal listeners in the logical backup service is deleted. Restores keep their subscription on the backup cleaner, which is the path every deletion takes.

**3. Secrets and HTTP**

- Google Drive token refresh writes nothing to standard output. Refresh events go through the masking logger, without token fragments.
- The master encryption key is loaded or generated once, even when the first uses arrive at the same time.
- A webhook notifier whose URL, body template or header cannot be decrypted fails with an error that names the decryption. Today it sends the ciphertext as the URL.
- The HTTP server closes connections that send their request headers too slowly and closes idle keep-alive connections. It sets no write deadline, so long backup downloads keep working.
- **BREAKING** (HTTP API): an unknown path under `/api/` returns `404` with a JSON body. Today it returns `200` with the frontend's `index.html`.

**4. Repository hygiene**

- The committed binaries `backend/cleanup_test_db` and `agent/verification/mock-server.exe` are deleted and ignored.
- `backend/internal/features/test_once_protection.go` becomes a `_test.go` file, so its five tests run.
- The backend tests under `cmd/` run in `just test` and in the CI `other` group. The `ci-pipeline` and `local-development-workflow` specs already require every backend package to run.
- The `testing.Short()` skip branches are deleted. Nothing passes `-short`, so they never ran.
- **BREAKING** (schema): a migration drops the tables `postgres_monitoring_settings` and `postgres_monitoring_metrics`, which no code reads or writes.
- `AGENTS.md` stops claiming that `revive` fails CI, because no `revive` rule is enabled.

## Capabilities

### New Capabilities

- `application-shutdown`: what the application does with running backups and restores after a stop signal, and how long it may take to exit.
- `backup-storage-assignment`: which storage a restore reads a backup from, what changing a configuration's storage does to existing backups, and that saving a configuration never changes storages.
- `database-status-updates`: background status updates of a database never overwrite settings a user saved.
- `secret-handling`: tokens stay out of the process output, the master key is generated once, and an undecryptable notifier secret fails visibly.
- `api-http-server`: the response to an unknown API path and the connection limits of the HTTP server.

### Modified Capabilities

None. `logical-backup-outcome`, `ci-pipeline` and `local-development-workflow` already require the behavior this change delivers.

## Impact

- Backend: `cmd/main.go`, `internal/config/signals.go`, the logical backup and restore use cases that poll for shutdown, `tasks/cancellation`, the restorer, both backup configuration services and repositories, the physical backup service, `databases` service and repository, the Google Drive storage model, the secret key service, the webhook notifier model, and a new migration.
- Tests: new and changed tests in the packages above. `test_once_protection.go` is renamed, and the physical test bodies lose their `-short` branches.
- Tooling: `scripts/test.sh`, `backend/Justfile` and two `.gitignore` files. CI takes its package list from `backend/Justfile`, so `.github/workflows/ci-release.yml` does not change.
- Documentation: `AGENTS.md`.
- API clients that relied on `200` for an unknown `/api/` path now get `404`. The frontend calls no such path.

## Out of scope

- Deriving the AES key with a KDF. The current key keeps about 106 bits of entropy, and changing the derivation needs a migration that re-encrypts every stored secret.
- The 10-year JWT lifetime. A password change already revokes old tokens, and the session length is a product decision.
- Moving the shared `testing.go` helpers out of production packages. `backend/AGENTS.md` prescribes them, so this is a convention change for a separate refactoring.
- `backend/swagger/` in `.gitignore`. The Dockerfile and CI generate it on purpose.
- Recording the outcome of physical full and incremental backups at shutdown. They keep today's behavior and are marked failed on the next start.
- Other `fmt.Printf` calls that carry no secrets, in `users/controllers/management_controller.go` and `backups/backups/controllers/logical/controller.go`.
- Raising Docker's stop timeout or the Helm chart's termination period.
