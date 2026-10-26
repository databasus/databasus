package logical

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generatedUsernames replaces the source's name generator: it hands out names in order and repeats
// the last one, and counts the attempts that asked for a name.
type generatedUsernames struct {
	names     []string
	callCount int
}

func (g *generatedUsernames) next() string {
	name := g.names[min(g.callCount, len(g.names)-1)]
	g.callCount++

	return name
}

func runShouldSuggestReadOnlyUserScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("suggests read-only user with privileged admin role", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)

		shouldSuggest, privileges, err := database.newAdminSource().
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.NoError(t, err)
		assert.True(t, shouldSuggest)
		assert.NotEmpty(t, privileges)
	})

	t.Run("does not suggest read-only user with role it created", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.exec(t, `CREATE TABLE readonly_check_test (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`)
		readOnlyRole := database.createReadOnlyUser(t, nil)

		shouldSuggest, privileges, err := database.newSource(readOnlyRole).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.NoError(t, err)
		assert.False(t, shouldSuggest)
		assert.Empty(t, privileges)
	})

	t.Run("does not suggest read-only user when row-level security table is in scope", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.createRowLevelSecurityTable(t, "public", "rls_in_scope")

		shouldSuggest, privileges, err := database.newAdminSource().
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.NoError(t, err)
		assert.False(t, shouldSuggest, "row-level security must suppress the suggestion")
		assert.NotEmpty(t, privileges, "the admin user is still privileged, only the suggestion is suppressed")
	})

	t.Run("suggests read-only user when row-level security table is outside included schemas", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.createRowLevelSecurityTable(t, "rls_excluded_schema", "rls_out_of_scope")

		shouldSuggest, _, err := database.newAdminSource().
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger(t), []string{"public"})

		require.NoError(t, err)
		assert.True(t, shouldSuggest)
	})
}

func runCreateReadOnlyUserScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("creates user that reads but cannot write", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.exec(t,
			`CREATE TABLE readonly_test (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`INSERT INTO readonly_test (data) VALUES ('test1'), ('test2')`,
		)

		readOnlyRole := database.createReadOnlyUser(t, nil)
		readOnlyConn := database.connectAs(t, readOnlyRole)

		assert.True(t, strings.HasPrefix(readOnlyRole.Name, "databasus-"))
		assert.NotEmpty(t, readOnlyRole.Password)
		assert.Equal(t, 2, countRows(t, readOnlyConn, "readonly_test"))
		assertPermissionDenied(t, readOnlyConn,
			`INSERT INTO readonly_test (data) VALUES ('should-fail')`,
			`UPDATE readonly_test SET data = 'hacked' WHERE id = 1`,
			`DELETE FROM readonly_test WHERE id = 1`,
			`CREATE TABLE hack_table (id INT)`,
			`CREATE TABLE public.hack_table (id INT)`,
		)
	})

	t.Run("creates user that reads tables created after it", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		readOnlyRole := database.createReadOnlyUser(t, nil)

		database.exec(t,
			`CREATE TABLE future_table (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`INSERT INTO future_table (data) VALUES ('future_data')`,
		)

		assert.Equal(t, 1, countRows(t, database.connectAs(t, readOnlyRole), "future_table"))
	})

	t.Run("creates user that reads every schema without included schemas", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.exec(t,
			`CREATE SCHEMA schema_a`,
			`CREATE SCHEMA schema_b`,
			`CREATE TABLE schema_a.table_a (id INT, data TEXT)`,
			`CREATE TABLE schema_b.table_b (id INT, data TEXT)`,
			`INSERT INTO schema_a.table_a VALUES (1, 'data_a')`,
			`INSERT INTO schema_b.table_b VALUES (2, 'data_b')`,
		)

		readOnlyConn := database.connectAs(t, database.createReadOnlyUser(t, nil))

		assert.Equal(t, 1, countRows(t, readOnlyConn, "schema_a.table_a"))
		assert.Equal(t, 1, countRows(t, readOnlyConn, "schema_b.table_b"))
	})

	t.Run("creates user when database name contains a dash", func(t *testing.T) {
		t.Parallel()
		database := server.createNamedDatabase(t, "test-db-with-dash-"+newNameSuffix())
		database.exec(t,
			`CREATE TABLE dash_test (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`INSERT INTO dash_test (data) VALUES ('test1'), ('test2')`,
		)

		readOnlyConn := database.connectAs(t, database.createReadOnlyUser(t, nil))

		assert.Equal(t, 2, countRows(t, readOnlyConn, "dash_test"))
		assertPermissionDenied(t, readOnlyConn, `INSERT INTO dash_test (data) VALUES ('should-fail')`)
	})

	t.Run("creates user that cannot create tables without public schema", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.exec(t,
			`DROP SCHEMA public CASCADE`,
			`CREATE SCHEMA app_schema`,
			`CREATE SCHEMA data_schema`,
			`CREATE TABLE app_schema.users (id SERIAL PRIMARY KEY, username TEXT NOT NULL)`,
			`CREATE TABLE data_schema.records (id SERIAL PRIMARY KEY, info TEXT NOT NULL)`,
			`INSERT INTO app_schema.users (username) VALUES ('user1'), ('user2')`,
			`INSERT INTO data_schema.records (info) VALUES ('record1'), ('record2')`,
		)

		readOnlyRole := database.createReadOnlyUser(t, nil)
		readOnlyConn := database.connectAs(t, readOnlyRole)

		shouldSuggest, privileges, err := database.newSource(readOnlyRole).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger(t), nil)
		require.NoError(t, err)
		assert.False(t, shouldSuggest)
		assert.Empty(t, privileges)

		assert.Equal(t, 2, countRows(t, readOnlyConn, "app_schema.users"))
		assert.Equal(t, 2, countRows(t, readOnlyConn, "data_schema.records"))
		assertPermissionDenied(t, readOnlyConn,
			`INSERT INTO app_schema.users (username) VALUES ('should-fail')`,
			`CREATE TABLE app_schema.hack_table (id INT)`,
			`CREATE TABLE data_schema.hack_table (id INT)`,
		)
	})

	t.Run("creates user that reads new tables of another owner and locks them like pg_dump", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		creatorRole := database.createLoginRole(t, "user_creator")
		database.exec(t, fmt.Sprintf(`GRANT USAGE, CREATE ON SCHEMA public TO "%s"`, creatorRole.Name))
		creatorConn := database.connectAs(t, creatorRole)

		// The creator owns a table before the read-only user exists, which is how the creation
		// finds the owners whose future tables it must cover.
		execAs(t, creatorConn, `CREATE TABLE public.initial_table (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`)

		readOnlyConn := database.connectAs(t, database.createReadOnlyUser(t, nil))

		execAs(t, creatorConn,
			`CREATE TABLE public.future_table (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`INSERT INTO public.future_table (data) VALUES ('test_data_1'), ('test_data_2')`,
		)

		assert.Equal(t, 2, countRows(t, readOnlyConn, "public.future_table"))
		assertPermissionDenied(t, readOnlyConn, `INSERT INTO public.future_table (data) VALUES ('should-fail')`)

		err := pgx.BeginFunc(t.Context(), readOnlyConn, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `LOCK TABLE public.future_table IN ACCESS SHARE MODE`)

			return err
		})
		assert.NoError(t, err, "pg_dump locks every table it dumps")
	})

	t.Run("grants only included schemas with included schemas", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		creatorRole := database.createLoginRole(t, "user_creator")
		database.exec(
			t,
			`CREATE SCHEMA included_schema`,
			`CREATE SCHEMA excluded_schema`,
			`CREATE TABLE public.public_table (id INT, data TEXT)`,
			`CREATE TABLE included_schema.included_table (id INT, data TEXT)`,
			`CREATE TABLE excluded_schema.excluded_table (id INT, data TEXT)`,
			`INSERT INTO public.public_table VALUES (1, 'public_data')`,
			`INSERT INTO included_schema.included_table VALUES (2, 'included_data')`,
			`INSERT INTO excluded_schema.excluded_table VALUES (3, 'excluded_data')`,
			fmt.Sprintf(
				`GRANT USAGE, CREATE ON SCHEMA public, included_schema, excluded_schema TO "%s"`,
				creatorRole.Name,
			),
		)
		creatorConn := database.connectAs(t, creatorRole)
		execAs(t, creatorConn,
			`CREATE TABLE included_schema.user_table (id INT, data TEXT)`,
			`INSERT INTO included_schema.user_table VALUES (4, 'user_included_data')`,
			`CREATE TABLE excluded_schema.user_excluded_table (id INT, data TEXT)`,
		)

		readOnlyConn := database.connectAs(t, database.createReadOnlyUser(t, []string{"public", "included_schema"}))

		execAs(t, creatorConn,
			`CREATE TABLE included_schema.future_table (id INT, data TEXT)`,
			`INSERT INTO included_schema.future_table VALUES (6, 'future_data')`,
			`CREATE TABLE excluded_schema.future_excluded_table (id INT, data TEXT)`,
		)

		assert.Equal(t, 1, countRows(t, readOnlyConn, "public.public_table"))
		assert.Equal(t, 1, countRows(t, readOnlyConn, "included_schema.included_table"))
		assert.Equal(t, 1, countRows(t, readOnlyConn, "included_schema.user_table"))
		assert.Equal(t, 1, countRows(t, readOnlyConn, "included_schema.future_table"))
		assertPermissionDenied(t, readOnlyConn,
			`SELECT * FROM excluded_schema.excluded_table`,
			`SELECT * FROM excluded_schema.user_excluded_table`,
			`SELECT * FROM excluded_schema.future_excluded_table`,
		)
	})

	t.Run("returns error and creates no role when row-level security table is in scope", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		database.createRowLevelSecurityTable(t, "public", "rls_blocks_creation")
		usernames := &generatedUsernames{names: []string{newReadOnlyUsername()}}
		source := database.newAdminSource()
		source.generateUsername = usernames.next

		createdUsername, createdPassword, err := source.CreateReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.ErrorIs(t, err, ErrRowLevelSecurity)
		assert.Contains(t, err.Error(), "public.rls_blocks_creation")
		assert.Empty(t, createdUsername)
		assert.Empty(t, createdPassword)
		assert.Zero(t, usernames.callCount, "the refusal must come before any role is created")
	})

	t.Run("retries with a new name when generated name is taken", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		takenUsername := newReadOnlyUsername()
		freshUsername := newReadOnlyUsername()
		database.exec(t, fmt.Sprintf(`CREATE ROLE "%s"`, takenUsername))
		database.dropRoleOnCleanup(takenUsername)
		database.dropRoleOnCleanup(freshUsername)

		usernames := &generatedUsernames{names: []string{takenUsername, freshUsername}}
		source := database.newAdminSource()
		source.generateUsername = usernames.next

		createdUsername, createdPassword, err := source.CreateReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.NoError(t, err)
		assert.Equal(t, freshUsername, createdUsername)
		assert.Equal(t, 2, usernames.callCount)
		database.connectAs(t, testRole{Name: createdUsername, Password: createdPassword})
	})

	t.Run("returns collision error when every generated name is taken", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		takenUsername := newReadOnlyUsername()
		database.exec(t, fmt.Sprintf(`CREATE ROLE "%s"`, takenUsername))
		database.dropRoleOnCleanup(takenUsername)

		usernames := &generatedUsernames{names: []string{takenUsername}}
		source := database.newAdminSource()
		source.generateUsername = usernames.next

		_, _, err := source.CreateReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.ErrorIs(t, err, ErrUserNameCollision)
		assert.Equal(t, maxReadOnlyUserAttempts, usernames.callCount)
	})

	// The role passes the CREATEROLE pre-check, but GRANT SELECT on a table it holds no privilege on
	// fails inside the attempt. A new name does not fix that, so nothing is retried.
	t.Run("returns error without retry when connecting role is refused a grant", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)
		limitedAdminRole := database.createLoginRole(t, "limited_admin")
		database.exec(t,
			fmt.Sprintf(`ALTER ROLE "%s" CREATEROLE`, limitedAdminRole.Name),
			`CREATE TABLE public.permission_test_table (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`,
			`GRANT CREATE ON SCHEMA public TO PUBLIC`,
		)

		attemptedUsername := newReadOnlyUsername()
		database.dropRoleOnCleanup(attemptedUsername)
		usernames := &generatedUsernames{names: []string{attemptedUsername}}
		source := database.newSource(limitedAdminRole)
		source.generateUsername = usernames.next

		createdUsername, createdPassword, err := source.CreateReadOnlyUser(t.Context(), newTestLogger(t), nil)

		require.ErrorContains(t, err, "failed to grant select on tables in schema public")
		require.ErrorContains(t, err, "permission denied")
		assert.Empty(t, createdUsername)
		assert.Empty(t, createdPassword)
		assert.Equal(t, 1, usernames.callCount)
		assert.False(t, database.isRolePresent(t, attemptedUsername), "the failed attempt must roll back its role")
	})
}

func (d testDatabase) createReadOnlyUser(t *testing.T, includeSchemas []string) testRole {
	t.Helper()

	username, password, err := d.newAdminSource().CreateReadOnlyUser(t.Context(), newTestLogger(t), includeSchemas)
	require.NoError(t, err)

	d.dropRoleOnCleanup(username)

	return testRole{Name: username, Password: password}
}

func (d testDatabase) createRowLevelSecurityTable(t *testing.T, schemaName, tableName string) {
	t.Helper()

	d.exec(t,
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, schemaName),
		fmt.Sprintf(`CREATE TABLE %q.%q (id int)`, schemaName, tableName),
		fmt.Sprintf(`ALTER TABLE %q.%q ENABLE ROW LEVEL SECURITY`, schemaName, tableName),
	)
}

func newReadOnlyUsername() string {
	return "databasus-" + newNameSuffix()
}

func execAs(t *testing.T, conn *pgx.Conn, statements ...string) {
	t.Helper()

	for _, statement := range statements {
		_, err := conn.Exec(t.Context(), statement)
		require.NoError(t, err, statement)
	}
}

func countRows(t *testing.T, conn *pgx.Conn, tableName string) int {
	t.Helper()

	var rowCount int

	err := conn.QueryRow(t.Context(), "SELECT COUNT(*) FROM "+tableName).Scan(&rowCount)
	require.NoError(t, err)

	return rowCount
}

func assertPermissionDenied(t *testing.T, conn *pgx.Conn, statements ...string) {
	t.Helper()

	for _, statement := range statements {
		_, err := conn.Exec(t.Context(), statement)
		assert.ErrorContains(t, err, "permission denied", statement)
	}
}
