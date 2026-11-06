package logical

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getSupportedVersionImages() []string {
	return []string{
		"postgres:12",
		"postgres:13",
		"postgres:14",
		"postgres:15",
		"postgres:16",
		"postgres:17",
		"postgres:18",
	}
}

// TestSource boots each supported version once and runs every scenario of the package against it.
// The versions run one after another, so only one server is alive at a time (docs/testing.md,
// "Database servers for engines"); the scenarios of a version run in parallel, each in a database
// of its own.
func TestSource(t *testing.T) {
	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			server := startTestServer(t, image)

			for _, scenarioGroup := range []struct {
				name string
				run  func(t *testing.T, server testServer)
			}{
				{name: "TestConnection", run: runTestConnectionScenarios},
				{name: "TestConnection dump privileges", run: runDumpPrivilegeScenarios},
				{name: "TestConnection RLS privileges", run: runRLSPrivilegeScenarios},
				{name: "DetectVersion", run: runDetectVersionScenarios},
				{name: "GetDumpedSizeMb", run: runGetDumpedSizeMbScenarios},
				{name: "GetTimescaleDBVersion", run: runGetTimescaleDBVersionScenarios},
				{name: "ShouldSuggestReadOnlyUser", run: runShouldSuggestReadOnlyUserScenarios},
				{name: "CreateReadOnlyUser", run: runCreateReadOnlyUserScenarios},
			} {
				t.Run(scenarioGroup.name, func(t *testing.T) {
					t.Parallel()
					scenarioGroup.run(t, server)
				})
			}
		})
	}
}

func runTestConnectionScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("returns detected version when role reads every table and sequence", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		backupRole := database.createLoginRole(t, "backup_user")

		database.exec(t,
			`CREATE TABLE backup_test (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`INSERT INTO backup_test (data) VALUES ('test1')`,
			fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO "%s"`, backupRole.Name),
			fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO "%s"`, backupRole.Name),
			fmt.Sprintf(`GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO "%s"`, backupRole.Name),
		)

		detectedVersion, err := database.newSource(backupRole).
			TestConnection(t.Context(), newTestLogger(t), DumpCheck{})

		require.NoError(t, err)
		assert.Equal(t, server.version, detectedVersion)
	})

	t.Run("connects when password contains spaces", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		roleWithSpaces := testRole{Name: "testuser_spaces_" + newNameSuffix(), Password: "test password with spaces"}

		database.exec(t,
			fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, roleWithSpaces.Name, roleWithSpaces.Password),
		)
		database.dropRoleOnCleanup(roleWithSpaces.Name)

		_, err := database.newSource(roleWithSpaces).TestConnection(t.Context(), newTestLogger(t), DumpCheck{})

		assert.NoError(t, err)
	})

	t.Run("returns insufficient permissions error when role cannot read public tables", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		limitedRole := database.createLoginRole(t, "limited_user")

		database.exec(t,
			`CREATE TABLE permission_test (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`INSERT INTO permission_test (data) VALUES ('test1')`,
		)

		_, err := database.newSource(limitedRole).TestConnection(t.Context(), newTestLogger(t), DumpCheck{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("returns password error with wrong password", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)

		wrongPasswordRole := testRole{Name: getAdminRole().Name, Password: getAdminRole().Password + "-wrong"}
		_, err := database.newSource(wrongPasswordRole).TestConnection(t.Context(), newTestLogger(t), DumpCheck{})

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "password")
	})

	t.Run("returns error when nothing listens on port", func(t *testing.T) {
		t.Parallel()

		settings := server.newSource(getAdminRole(), "testdb").settings
		settings.Port = 1

		_, err := NewSource(settings).TestConnection(t.Context(), newTestLogger(t), DumpCheck{})

		assert.Error(t, err)
	})

	t.Run("returns database name error without database name", func(t *testing.T) {
		t.Parallel()

		_, err := server.newSource(getAdminRole(), "").TestConnection(t.Context(), newTestLogger(t), DumpCheck{})

		assert.ErrorIs(t, err, ErrDatabaseNameRequired)
	})
}

func runDetectVersionScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("returns server major version when server is reachable", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)

		detectedVersion, err := database.newAdminSource().DetectVersion(t.Context(), newTestLogger(t))

		require.NoError(t, err)
		assert.Equal(t, server.version, detectedVersion)
	})
}
