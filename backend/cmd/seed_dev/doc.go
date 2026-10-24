// Command seed_dev fills the local development app with what a developer or an agent needs to
// look at a screen right after start: an account to sign in with, a workspace with a local storage
// and two PostgreSQL databases whose backups are already running. `mise run dev` starts it once the
// backend accepts connections. Run it on its own with `go run ./cmd/seed_dev` in backend/ to
// restore an item that was deleted from the running app.
//
// It talks to the backend on http://localhost:4005 through the public API, so every item passes
// the same validation as in the web interface. The production image builds only ./cmd/databasus and
// never contains this command.
//
// What it makes sure exists, finding each item by its name:
//
//   - the account test@email.com with the password Q1234567;
//   - the workspace "Dev" and, in it, the local storage "Local";
//   - "Logical PG16": logical backups of the shared logical test fixture on
//     TEST_LOGICAL_POSTGRES_16_PORT (5004), database testdb, daily, keeping the 3 newest backups;
//   - "WAL PG18": WAL-streaming physical backups of the live WAL backup source on
//     WAL_BACKUP_POSTGRES_18_PORT (5018), signed in as WAL_BACKUP_DB_USER, keeping the 2 newest
//     chains.
//
// Missing items are created; a database gets its backups switched on with "Local" and a first
// backup right away. An item that already exists is left exactly as it is, including one a
// developer edited, and an existing account is only ever signed in to. The seed refuses to run
// unless ENV_MODE in .env is development. It exits with a non-zero status and the backend's
// reason when the backend does not answer or reports itself unhealthy, when it cannot sign in as
// the account or register it, when the instance requires two-factor sign-in, and when the backend
// refuses to create an item.
//
// Fixing by hand what the seed leaves alone:
//
//   - Recreating the wal-backup-postgres-18 container drops the replication slot of "WAL PG18",
//     and its backups start failing. Delete "WAL PG18" in the web interface and run the seed again.
//   - Recreating the backend-db container forgets "WAL PG18" but leaves its replication slot on
//     the WAL backup source, which then keeps WAL without limit. Find the inactive slot named
//     databasus_slot_* in pg_replication_slots there and drop it with pg_drop_replication_slot.
//   - The backend reports itself unhealthy once 95% of the disk is used, and the seed then stops
//     with that reason. Free disk space and run the seed again.
package main
