package pgrestore

// timescaledb_pre_restore sets a database-level GUC (ALTER DATABASE ... SET
// timescaledb.restoring='on') and stops background workers. The separate pg_restore
// connection inherits the GUC. timescaledb_post_restore resets it and restarts workers.
func GetTimescaleDBPreRestoreStatements() []string {
	return []string{"CREATE EXTENSION IF NOT EXISTS timescaledb", "SELECT timescaledb_pre_restore()"}
}

func GetTimescaleDBPostRestoreStatements() []string {
	return []string{"SELECT timescaledb_post_restore()"}
}
