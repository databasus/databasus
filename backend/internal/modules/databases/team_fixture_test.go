package databases_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// team is an instance with a workspace owned by a user who is not an instance admin.
type team struct {
	app       *apptest.App
	admin     apptest.Session
	owner     apptest.Session
	workspace apiclient.Workspace
}

func newTeam(t *testing.T, options ...apptest.Option) team {
	t.Helper()

	app := apptest.New(t, options...)
	admin := app.CreateAdmin(t)
	owner := app.CreateNonAdminUser(t, admin)

	return team{
		app:       app,
		admin:     admin,
		owner:     owner,
		workspace: apptest.CreateWorkspace(t, owner, "Team "+uuid.NewString()[:8]),
	}
}

func (tm team) addMember(t *testing.T, role apiclient.AssignableWorkspaceRole) apptest.Session {
	t.Helper()

	user := tm.app.SignUp(t, apptest.NewTestEmail("member"))
	apptest.AddWorkspaceMember(t, tm.owner, tm.workspace.Id, user.Email, role)

	return user
}

func (tm team) signUpNonMember(t *testing.T) apptest.Session {
	t.Helper()

	return tm.app.SignUp(t, apptest.NewTestEmail("non-member"))
}

func (tm team) createDatabaseNamed(t *testing.T, name string, settings apiclient.DatabaseSettings) apiclient.Database {
	t.Helper()

	return createDatabase(t, tm.owner, apiclient.DatabaseCreate{
		WorkspaceId: tm.workspace.Id,
		Name:        name,
		NotifierIds: []uuid.UUID{},
		Settings:    settings,
	})
}

func (tm team) createDatabase(t *testing.T, settings apiclient.DatabaseSettings) apiclient.Database {
	t.Helper()

	return apptest.CreateDatabase(t, tm.owner, tm.workspace.Id, settings)
}

func (tm team) listDatabaseIDs(t *testing.T) []uuid.UUID {
	t.Helper()

	listedDatabases := listDatabases(t, tm.owner, tm.workspace.Id)

	databaseIDs := make([]uuid.UUID, 0, len(listedDatabases))
	for _, listedDatabase := range listedDatabases {
		databaseIDs = append(databaseIDs, listedDatabase.Id)
	}

	return databaseIDs
}

func (tm team) readDatabase(t *testing.T, databaseID uuid.UUID) apiclient.Database {
	t.Helper()

	return getDatabase(t, tm.owner, databaseID)
}

func (tm team) countRows(t *testing.T, query string, values ...any) int {
	t.Helper()

	var rowCounts []int
	tm.app.QuerySQL(t, &rowCounts, query, values...)
	require.Len(t, rowCounts, 1)

	return rowCounts[0]
}

// insertMysqlDatabase writes a MySQL database of the workspace directly, because creating one
// through the API needs a MySQL server and the test only needs a saved database of that type.
func (tm team) insertMysqlDatabase(t *testing.T) uuid.UUID {
	t.Helper()

	databaseID := uuid.New()
	tm.app.ExecuteSQL(t, "INSERT INTO databases (id, workspace_id, name, type) VALUES (?, ?, ?, 'MYSQL')",
		databaseID, tm.workspace.Id, "MySQL "+databaseID.String()[:8])
	tm.app.ExecuteSQL(t, `INSERT INTO databases_mysql (database_id, version, host, port, username, password, database)
		VALUES (?, '8.0', '127.0.0.1', 3306, 'backup', ?, 'shop')`,
		databaseID, tm.app.EncryptSecret(t, "mysql-password"))

	return databaseID
}
