//go:build testhelpers

package databases_bridge

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/util/testing/containers"
)

// CreateTestMysqlDatabase inserts a MySQL 8.0 database at host:port with the credentials of the
// containers package. The caller owns the server.
func CreateTestMysqlDatabase(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) databases.Database {
	databaseID := insertTestDatabase("test-mysql "+uuid.NewString(), databases.TypeMysql, workspaceID, notifier)
	execute(`INSERT INTO databases_mysql (database_id, version, host, port, username, password, database)
		VALUES (?, '8.0', ?, ?, ?, ?, ?)`,
		databaseID, host, port, containers.MysqlUsername, containers.MysqlPassword, containers.MysqlDatabase)

	return getTestDatabase(databaseID)
}
