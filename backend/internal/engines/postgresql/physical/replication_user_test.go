package physical

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestShouldSuggestReplicationOnlyUser(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		// The cloud admin roles are cluster-wide, so these scenarios run one at a time and drop their
		// role before the parallel scenarios start.
		for _, cloudRole := range []cloudAdminRole{
			{Name: "rds_superuser", Label: "rds_superuser (RDS admin)"},
			{Name: "azure_pg_admin", Label: "azure_pg_admin (Azure admin)"},
			{Name: "cloudsqlsuperuser", Label: "cloudsqlsuperuser (GCP admin)"},
		} {
			t.Run("lists membership in "+cloudRole.Name+" when role belongs to it", func(t *testing.T) {
				server.createClusterRole(t, cloudRole.Name, true)

				shouldSuggest, excessivePrivileges, err := server.newSource(getAdminRole()).
					ShouldSuggestReplicationOnlyUser(t.Context(), newTestLogger(t))

				require.NoError(t, err)
				assert.True(t, shouldSuggest)
				assert.Contains(t, excessivePrivileges, cloudRole.Label)
			})
		}

		t.Run("suggests a replication-only user when role is superuser", func(t *testing.T) {
			t.Parallel()

			shouldSuggest, excessivePrivileges, err := server.newSource(getAdminRole()).
				ShouldSuggestReplicationOnlyUser(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.True(t, shouldSuggest)
			assert.Contains(t, excessivePrivileges, "SUPERUSER")
		})

		t.Run("suggests nothing for a freshly created replication-only user", func(t *testing.T) {
			t.Parallel()

			createdUser, err := server.newSource(getAdminRole()).
				CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
			require.NoError(t, err)

			createdRole := testRole{Name: createdUser.Username, Password: createdUser.Password}
			shouldSuggest, excessivePrivileges, err := server.newSource(createdRole).
				ShouldSuggestReplicationOnlyUser(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.False(t, shouldSuggest)
			assert.Empty(t, excessivePrivileges)
		})

		t.Run("lists the table write privilege when role may insert into a table", func(t *testing.T) {
			t.Parallel()
			writerRole := server.createLoginRole(t, "")
			tableName := "write_probe_" + newNameSuffix()

			execStatements(t, server.connectAs(t, getAdminRole()),
				fmt.Sprintf(`CREATE TABLE public."%s" (id int)`, tableName),
				fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO "%s"`, writerRole.Name),
				fmt.Sprintf(`GRANT INSERT ON public."%s" TO "%s"`, tableName, writerRole.Name),
			)

			shouldSuggest, excessivePrivileges, err := server.newSource(writerRole).
				ShouldSuggestReplicationOnlyUser(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.True(t, shouldSuggest)
			assert.Contains(t, excessivePrivileges, "INSERT")
		})
	})
}

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestCreateReplicationOnlyUser(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		// The scenarios that create a user run one at a time: each creation grants EXECUTE on
		// pg_switch_wal, and PostgreSQL fails a second concurrent grant on the same function with
		// "tuple concurrently updated". rds_replication is cluster-wide as well and switches the
		// platform for every scenario, so its scenario drops the role before the parallel ones start.
		t.Run("grants rds_replication membership on a simulated rds source", func(t *testing.T) {
			server.createClusterRole(t, "rds_replication", false)

			createdUser, err := server.newSource(getAdminRole()).
				CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
			require.NoError(t, err)

			var isMember, canReplicate bool

			err = server.connectAs(t, getAdminRole()).QueryRow(t.Context(), `
				SELECT pg_has_role($1, 'rds_replication', 'MEMBER'), rolreplication
				FROM pg_roles
				WHERE rolname = $1
			`, createdUser.Username).Scan(&isMember, &canReplicate)
			require.NoError(t, err)
			assert.True(t, isMember, "the user must be a member of rds_replication on RDS")
			assert.False(t, canReplicate, "RDS takes GRANT rds_replication, not ALTER ROLE REPLICATION")
		})

		t.Run(
			"creates a replication-only user that can stream and rotate wal on a self-managed source",
			func(t *testing.T) {
				createdUser, err := server.newSource(getAdminRole()).
					CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
				require.NoError(t, err)

				assert.True(t, strings.HasPrefix(createdUser.Username, "databasus-"), createdUser.Username)
				assert.NotEmpty(t, createdUser.Password)
				assert.True(t, createdUser.IsForcedWalRotationAvailable)

				createdRole := testRole{Name: createdUser.Username, Password: createdUser.Password}
				assertRoleHasOnlyReplication(t, server, createdRole.Name)

				err = server.newSource(createdRole).
					TestReplicationConnection(t.Context(), newTestLogger(t), getWalStreamCheck())
				assert.NoError(t, err)
			},
		)

		t.Run("grants wal rotation to the created role alone", func(t *testing.T) {
			bystanderRole := server.createLoginRole(t, "REPLICATION")

			_, err := server.newSource(getAdminRole()).CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
			require.NoError(t, err)

			canBystanderRotate, err := canForceWalRotation(
				t.Context(),
				server.connectAs(t, getAdminRole()),
				bystanderRole.Name,
			)
			require.NoError(t, err)
			assert.False(t, canBystanderRotate, "the grant must not reach any other role")
		})

		// A role that may create roles and confer REPLICATION, but owns none of the pg_catalog
		// functions, is what a managed platform's administrator looks like.
		t.Run("returns working credentials without wal rotation when source refuses the grant", func(t *testing.T) {
			t.Parallel()
			platformAdminRole := server.createLoginRole(t, "CREATEROLE REPLICATION")

			createdUser, err := server.newSource(platformAdminRole).
				CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
			require.NoError(t, err)

			assert.False(t, createdUser.IsForcedWalRotationAvailable)

			createdRole := testRole{Name: createdUser.Username, Password: createdUser.Password}
			err = server.newSource(createdRole).
				TestReplicationConnection(t.Context(), newTestLogger(t), getIncrementalCheck())
			assert.NoError(t, err)
		})

		t.Run("returns create roles error when role cannot create roles", func(t *testing.T) {
			t.Parallel()
			ordinaryRole := server.createLoginRole(t, "")

			_, err := server.newSource(ordinaryRole).CreateReplicationOnlyUser(t.Context(), newTestLogger(t))

			require.ErrorIs(t, err, ErrCannotCreateRoles)
		})

		t.Run("retries with a new name when the generated name is taken", func(t *testing.T) {
			takenName := "databasus-" + newNameSuffix()
			freshName := "databasus-" + newNameSuffix()
			execStatements(t, server.connectAs(t, getAdminRole()), fmt.Sprintf(`CREATE ROLE "%s" NOLOGIN`, takenName))

			source := server.newSource(getAdminRole())
			generatedNames := []string{takenName, freshName}
			generateUsernameCalls := 0
			source.generateUsername = func() string {
				generateUsernameCalls++

				return generatedNames[generateUsernameCalls-1]
			}

			createdUser, err := source.CreateReplicationOnlyUser(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.Equal(t, freshName, createdUser.Username)
			assert.Equal(t, 2, generateUsernameCalls)
			assert.True(t, server.isRolePresent(t, freshName))
		})

		// CREATEROLE passes the pre-check, but without REPLICATION of its own the role may not confer
		// it, so the server refuses ALTER ROLE ... REPLICATION inside the attempt.
		t.Run("returns the refusal without retrying when the server refuses the replication grant", func(t *testing.T) {
			t.Parallel()
			roleCreatorWithoutReplication := server.createLoginRole(t, "CREATEROLE")
			attemptedName := "databasus-" + newNameSuffix()

			source := server.newSource(roleCreatorWithoutReplication)
			generateUsernameCalls := 0
			source.generateUsername = func() string {
				generateUsernameCalls++

				return attemptedName
			}

			_, err := source.CreateReplicationOnlyUser(t.Context(), newTestLogger(t))

			require.ErrorIs(t, err, ErrReplicationGrantDeny)
			assert.Equal(t, 1, generateUsernameCalls)
			assert.False(t, server.isRolePresent(t, attemptedName), "the refused attempt must roll back")
		})
	})
}

// detectPlatform reads cluster-wide roles, so its scenarios run one at a time and each drops the
// roles it created when it ends.
//
//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestDetectPlatform(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		for _, scenario := range []struct {
			name             string
			clusterRoleNames []string
			platform         platform
		}{
			{name: "returns self-managed without cloud platform roles", clusterRoleNames: nil, platform: platformSelfManaged},
			{
				name:             "returns rds when rds_replication exists",
				clusterRoleNames: []string{"rds_replication"},
				platform:         platformRds,
			},
			{
				name:             "returns azure when azure_pg_admin exists",
				clusterRoleNames: []string{"azure_pg_admin"},
				platform:         platformAzure,
			},
			{
				name:             "returns gcp when cloudsqlsuperuser exists",
				clusterRoleNames: []string{"cloudsqlsuperuser"},
				platform:         platformGcp,
			},
			{
				name:             "returns rds when roles of every platform exist",
				clusterRoleNames: []string{"rds_replication", "azure_pg_admin", "cloudsqlsuperuser"},
				platform:         platformRds,
			},
		} {
			t.Run(scenario.name, func(t *testing.T) {
				for _, clusterRoleName := range scenario.clusterRoleNames {
					server.createClusterRole(t, clusterRoleName, false)
				}

				assert.Equal(t, scenario.platform, detectPlatform(t.Context(), server.connectAs(t, getAdminRole())))
			})
		}

		t.Run("returns unknown managed when role is not superuser", func(t *testing.T) {
			ordinaryRole := server.createLoginRole(t, "")

			assert.Equal(t, platformUnknownManaged, detectPlatform(t.Context(), server.connectAs(t, ordinaryRole)))
		})
	})
}

func assertRoleHasOnlyReplication(t *testing.T, server testServer, roleName string) {
	t.Helper()

	var isSuperuser, canCreateRole, canCreateDB, canBypassRLS, canReplicate bool

	err := server.connectAs(t, getAdminRole()).QueryRow(t.Context(), `
		SELECT rolsuper, rolcreaterole, rolcreatedb, rolbypassrls, rolreplication
		FROM pg_roles
		WHERE rolname = $1
	`, roleName).Scan(&isSuperuser, &canCreateRole, &canCreateDB, &canBypassRLS, &canReplicate)
	require.NoError(t, err)

	assert.False(t, isSuperuser)
	assert.False(t, canCreateRole)
	assert.False(t, canCreateDB)
	assert.False(t, canBypassRLS)
	assert.True(t, canReplicate, "a self-managed source grants rolreplication")
}
