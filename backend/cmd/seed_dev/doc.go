// Command seed_dev fills the local development app with what a developer or an agent needs to
// look at a screen right after start: an account to sign in with, a workspace with a local storage
// and two PostgreSQL databases whose backups are already running. `mise run run` starts it once the
// backend accepts connections. Run it on its own with `go run ./cmd/seed_dev` in backend/ to
// restore an item that was deleted from the running app.
//
// Account sign-in, registration and seeded items use the public API on http://localhost:4005 and
// pass the same validation as the web interface. After authentication, a local users-module
// operation connects through DATABASE_DSN and makes the dev account the bootstrap administrator.
// The production image builds only ./cmd/databasus and never contains this command.
//
// What it makes sure exists, finding each item by its name:
//
//   - the account test@email.com with the password Q1234567, the ADMIN role and the sole
//     bootstrap-administrator designation;
//   - the workspace "Dev" and, in it, the local storage "Local";
//   - "Logical PG16": logical backups of the shared logical test fixture on
//     TEST_LOGICAL_POSTGRES_16_PORT (5004), database testdb, daily, keeping the 3 newest backups;
//   - "WAL PG18": WAL-streaming physical backups of the live WAL backup source on
//     WAL_BACKUP_POSTGRES_18_PORT (5018), signed in as WAL_BACKUP_DB_USER, keeping the 2 newest
//     chains.
//
// Missing items are created; a database gets its backups switched on with "Local" and a first
// backup right away. Existing items and backup settings stay as they are. The only privilege
// change gives the dev account ADMIN and the bootstrap designation; the previous holder keeps
// ADMIN, its password and its settings. The assignment and its audit event commit together, and
// repeating an already-correct assignment creates no transfer event. The seed refuses to run
// unless ENV_MODE is development. It exits with a non-zero status and the backend's
// reason when the backend does not answer or a required health check fails, when it cannot sign in as
// the account or register it, when the instance requires two-factor sign-in, and when the backend
// refuses to create an item. Stale verification agents alone produce a warning and do not stop the
// seed; their records and the instance's reported health stay unchanged.
//
// Missing DATABASE_DSN, connection failure or an authenticated ID/email that does not identify an
// active account in that database stops the seed before it creates other items. A failed transfer
// preserves the previous privileges; a later item-creation failure leaves a completed transfer in
// place. Passwords, profiles and account settings are never reset.
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
