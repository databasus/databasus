package mariadb

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReadOnlyUser(t *testing.T) {
	t.Parallel()

	server := startTestServer(t, "mariadb:10.11")

	t.Run("grants read access to tables created after the user", func(t *testing.T) {
		t.Parallel()
		databaseName := server.createDatabase(t, "db_")

		username, password, err := server.newRootSource(t, databaseName).
			CreateReadOnlyUser(t.Context(), newTestLogger())
		require.NoError(t, err)
		server.dropUserOnCleanup(t, username)

		server.createTable(t, databaseName, "future_table")
		server.exec(t, fmt.Sprintf("INSERT INTO `%s`.`future_table` (data) VALUES ('future_data')", databaseName))

		var futureData string
		err = server.connectAs(t, username, password, databaseName).
			QueryRowContext(t.Context(), "SELECT data FROM future_table LIMIT 1").Scan(&futureData)

		require.NoError(t, err)
		assert.Equal(t, "future_data", futureData)
	})

	t.Run("creates a working user with a dash in the database name", func(t *testing.T) {
		t.Parallel()
		databaseName := server.createDatabase(t, "test-db-with-dash-")
		server.createTable(t, databaseName, "dash_test")
		server.exec(t, fmt.Sprintf("INSERT INTO `%s`.`dash_test` (data) VALUES ('test1'), ('test2')", databaseName))

		username, password, err := server.newRootSource(t, databaseName).
			CreateReadOnlyUser(t.Context(), newTestLogger())
		require.NoError(t, err)
		server.dropUserOnCleanup(t, username)
		assert.True(t, strings.HasPrefix(username, "pgs-"))

		readOnlyDB := server.connectAs(t, username, password, databaseName)

		var rowCount int
		require.NoError(t, readOnlyDB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM dash_test").Scan(&rowCount))
		assert.Equal(t, 2, rowCount)

		_, err = readOnlyDB.ExecContext(t.Context(), "INSERT INTO dash_test (data) VALUES ('should-fail')")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "denied")
	})

	t.Run("creates a user that cannot drop, alter or truncate tables", func(t *testing.T) {
		t.Parallel()
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "drop_test")

		username, password, err := server.newRootSource(t, databaseName).
			CreateReadOnlyUser(t.Context(), newTestLogger())
		require.NoError(t, err)
		server.dropUserOnCleanup(t, username)

		readOnlyDB := server.connectAs(t, username, password, databaseName)

		for _, schemaStatement := range []string{
			"DROP TABLE drop_test",
			"ALTER TABLE drop_test ADD COLUMN new_col VARCHAR(100)",
			"TRUNCATE TABLE drop_test",
		} {
			_, err = readOnlyDB.ExecContext(t.Context(), schemaStatement)
			require.Error(t, err, schemaStatement)
			assert.Contains(t, strings.ToLower(err.Error()), "denied", schemaStatement)
		}
	})

	t.Run("retries with a new name when the generated name is taken", func(t *testing.T) {
		t.Parallel()
		databaseName := server.createDatabase(t, "db_")
		takenUsername, _ := server.createUser(t, databaseName)
		freshUsername := "pgs-" + uuid.New().String()[:8]
		server.dropUserOnCleanup(t, freshUsername)
		source := server.newRootSource(t, databaseName)
		generatedNames := []string{takenUsername, freshUsername}
		generateCallCount := 0
		source.generateUsername = func() string {
			generatedName := generatedNames[min(generateCallCount, len(generatedNames)-1)]
			generateCallCount++

			return generatedName
		}

		username, password, err := source.CreateReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.Equal(t, freshUsername, username)
		assert.NotEmpty(t, password)
		assert.Equal(t, 2, generateCallCount)
	})

	t.Run("returns the error on the first attempt when the account may not create users", func(t *testing.T) {
		t.Parallel()
		databaseName := server.createDatabase(t, "db_")
		username, password := server.createUser(t, databaseName, "GRANT SELECT ON `%s`.* TO %s")
		source := server.newSource(t, username, password, databaseName)
		generateCallCount := 0
		source.generateUsername = func() string {
			generateCallCount++

			return "pgs-" + uuid.New().String()[:8]
		}

		_, _, err := source.CreateReadOnlyUser(t.Context(), newTestLogger())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user")
		assert.Equal(t, 1, generateCallCount)
	})
}
