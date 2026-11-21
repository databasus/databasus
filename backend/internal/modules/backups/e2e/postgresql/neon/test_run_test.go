package neon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	objectMarkerPrefix = "databasus_e2e_"
	// abandonedObjectAge leaves a younger marked object alone, because another run may still use it.
	abandonedObjectAge = time.Hour
	cleanupTimeout     = 2 * time.Minute
)

// objectMarkerPattern reads the creation time from a marker: the prefix, the Unix time and a random
// part unique to the run.
var objectMarkerPattern = regexp.MustCompile(`^databasus_e2e_(\d+)_[0-9a-f]+(_|$)`)

// testRun owns the databases and roles one run creates on a shared server. Each carries the run's
// marker, in its name or, for a role whose name Databasus chose, in its comment, so concurrent runs
// never touch each other's objects and a later run can sweep what a killed run left behind.
type testRun struct {
	server        managedServer
	admin         *pgx.Conn
	marker        string
	databaseNames []string
	roleNames     []string
}

// startTestRun registers the cleanup before the run creates anything, so the cleanup also runs
// after a failed test.
func startTestRun(reporter testReporter, server managedServer, startedAt time.Time) *testRun {
	reporter.Helper()

	run := &testRun{
		server:        server,
		admin:         nil,
		marker:        fmt.Sprintf("%s%d_%s", objectMarkerPrefix, startedAt.Unix(), newRandomHex(reporter)),
		databaseNames: nil,
		roleNames:     nil,
	}

	reporter.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(reporter.Context()), cleanupTimeout)
		defer cancel()

		for _, failure := range run.dropCreatedObjects(ctx) {
			reporter.Errorf("%s", failure)
		}
	})

	run.admin = server.open(reporter, server.maintenanceDatabase)

	return run
}

func (r *testRun) createDatabase(reporter testReporter, purpose string) string {
	reporter.Helper()

	databaseName := r.marker + "_" + purpose
	r.databaseNames = append(r.databaseNames, databaseName)
	r.server.execute(reporter, r.admin, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize())

	return databaseName
}

func (r *testRun) createRole(reporter testReporter, purpose string) string {
	reporter.Helper()

	roleName := r.marker + "_" + purpose
	r.roleNames = append(r.roleNames, roleName)
	r.server.execute(reporter, r.admin, "CREATE ROLE "+pgx.Identifier{roleName}.Sanitize())

	return roleName
}

// markRole takes over a role Databasus created and named, by writing the run's marker into its
// comment.
func (r *testRun) markRole(reporter testReporter, roleName string) {
	reporter.Helper()

	r.roleNames = append(r.roleNames, roleName)
	r.server.execute(reporter, r.admin, fmt.Sprintf("COMMENT ON ROLE %s IS '%s'",
		pgx.Identifier{roleName}.Sanitize(), r.marker))
}

// dropCreatedObjects drops the databases first, which takes the roles' grants inside them along,
// and then the roles. It names every object it could not drop.
func (r *testRun) dropCreatedObjects(ctx context.Context) []string {
	if len(r.databaseNames) == 0 && len(r.roleNames) == 0 {
		return nil
	}

	conn, err := r.server.connect(ctx, r.server.maintenanceDatabase)
	if err != nil {
		return []string{fmt.Sprintf("could not drop the databases [%s] and the roles [%s] of run %s: %s",
			strings.Join(r.databaseNames, ", "), strings.Join(r.roleNames, ", "), r.marker, err)}
	}
	defer func() { _ = conn.Close(ctx) }()

	var failures []string

	for _, databaseName := range r.databaseNames {
		if err := dropDatabase(ctx, conn, databaseName); err != nil {
			failures = append(failures, fmt.Sprintf("could not drop database %s: %s",
				databaseName, r.server.redactor.redact(err.Error())))
		}
	}

	for _, roleName := range r.roleNames {
		if err := dropRole(ctx, conn, roleName); err != nil {
			failures = append(failures, fmt.Sprintf("could not drop role %s: %s",
				roleName, r.server.redactor.redact(err.Error())))
		}
	}

	return failures
}

// sweepAbandonedObjects removes the marked databases and roles older than abandonedObjectAge, which
// a run killed before its cleanup left behind. It never touches an object without the marker. A
// failed drop is only logged, because another run's sweep may be dropping the same object.
func sweepAbandonedObjects(reporter testReporter, server managedServer, now time.Time) {
	reporter.Helper()

	conn := server.open(reporter, server.maintenanceDatabase)
	ctx := reporter.Context()

	for _, databaseName := range queryNames(reporter, server, conn,
		`SELECT datname FROM pg_database WHERE starts_with(datname, $1) ORDER BY datname`, objectMarkerPrefix) {
		if !isAbandoned(databaseName, now) {
			continue
		}

		if err := dropDatabase(ctx, conn, databaseName); err != nil {
			reporter.Logf("sweep could not drop database %s: %s", databaseName, server.redactor.redact(err.Error()))
		}
	}

	for _, roleName := range queryNames(reporter, server, conn, `
		SELECT rolname FROM pg_roles
		WHERE starts_with(rolname, $1) OR starts_with(COALESCE(shobj_description(oid, 'pg_authid'), ''), $1)
		ORDER BY rolname`, objectMarkerPrefix) {
		if !isRoleAbandoned(reporter, server, conn, roleName, now) {
			continue
		}

		if err := dropRole(ctx, conn, roleName); err != nil {
			reporter.Logf("sweep could not drop role %s: %s", roleName, server.redactor.redact(err.Error()))
		}
	}
}

func isRoleAbandoned(reporter testReporter, server managedServer, conn *pgx.Conn, roleName string, now time.Time) bool {
	reporter.Helper()

	if objectMarkerPattern.MatchString(roleName) {
		return isAbandoned(roleName, now)
	}

	var comment string
	if err := conn.QueryRow(reporter.Context(),
		`SELECT COALESCE(shobj_description(oid, 'pg_authid'), '') FROM pg_roles WHERE rolname = $1`,
		roleName).Scan(&comment); err != nil {
		reporter.Logf("sweep could not read the comment of role %s: %s", roleName, server.redactor.redact(err.Error()))

		return false
	}

	return isAbandoned(comment, now)
}

func isAbandoned(marked string, now time.Time) bool {
	match := objectMarkerPattern.FindStringSubmatch(marked)
	if match == nil {
		return false
	}

	createdAtUnix, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return false
	}

	return now.Sub(time.Unix(createdAtUnix, 0)) > abandonedObjectAge
}

func queryNames(reporter testReporter, server managedServer, conn *pgx.Conn, query string, args ...any) []string {
	reporter.Helper()

	rows, err := conn.Query(reporter.Context(), query, args...)
	if err != nil {
		reporter.Fatalf("list objects on %s: %s", server.describe(conn.Config().Database),
			server.redactor.redact(err.Error()))
	}

	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		reporter.Fatalf("list objects on %s: %s", server.describe(conn.Config().Database),
			server.redactor.redact(err.Error()))
	}

	return names
}

func dropDatabase(ctx context.Context, conn *pgx.Conn, databaseName string) error {
	_, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)")

	return err
}

func dropRole(ctx context.Context, conn *pgx.Conn, roleName string) error {
	_, err := conn.Exec(ctx, "DROP ROLE IF EXISTS "+pgx.Identifier{roleName}.Sanitize())

	return err
}

func newRandomHex(reporter testReporter) string {
	reporter.Helper()

	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		reporter.Fatalf("generate the run marker: %v", err)
	}

	return hex.EncodeToString(randomBytes)
}

func TestSweepAbandonedObjects(t *testing.T) {
	t.Parallel()

	t.Run("removes only the marked databases and roles older than an hour", func(t *testing.T) {
		t.Parallel()
		_, connectionURL := startContainerServer(t)
		server := parseContainerServer(t, connectionURL)
		admin := server.open(t, server.maintenanceDatabase)
		now := time.Now()
		agedMarker := fmt.Sprintf("%s%d_%s", objectMarkerPrefix, now.Add(-2*time.Hour).Unix(), "0a1b2c3d")
		freshMarker := fmt.Sprintf("%s%d_%s", objectMarkerPrefix, now.Add(-5*time.Minute).Unix(), "4e5f6a7b")
		agedDatabase, freshDatabase, unmarkedDatabase := agedMarker+"_source", freshMarker+"_source", "kept_old_db"
		agedNamedRole, agedCommentedRole := agedMarker+"_role", "databasus-0a1b2c3d"
		freshCommentedRole, unmarkedRole := "databasus-4e5f6a7b", "kept_old_role"

		for _, databaseName := range []string{agedDatabase, freshDatabase, unmarkedDatabase} {
			server.execute(t, admin, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize())
		}

		server.execute(t, admin, fmt.Sprintf(`
			CREATE ROLE %s;
			CREATE ROLE %s;
			COMMENT ON ROLE %s IS '%s';
			CREATE ROLE %s;
			COMMENT ON ROLE %s IS '%s';
			CREATE ROLE %s;
			COMMENT ON ROLE %s IS 'owned by another tool since 2020';`,
			pgx.Identifier{agedNamedRole}.Sanitize(),
			pgx.Identifier{agedCommentedRole}.Sanitize(), pgx.Identifier{agedCommentedRole}.Sanitize(), agedMarker,
			pgx.Identifier{freshCommentedRole}.Sanitize(), pgx.Identifier{freshCommentedRole}.Sanitize(), freshMarker,
			pgx.Identifier{unmarkedRole}.Sanitize(), pgx.Identifier{unmarkedRole}.Sanitize()))

		sweepAbandonedObjects(t, server, now)

		assert.ElementsMatch(t, []string{freshDatabase, unmarkedDatabase},
			queryNames(t, server, admin, `SELECT datname FROM pg_database WHERE datname = ANY($1)`,
				[]string{agedDatabase, freshDatabase, unmarkedDatabase}))
		assert.ElementsMatch(t, []string{freshCommentedRole, unmarkedRole},
			queryNames(t, server, admin, `SELECT rolname FROM pg_roles WHERE rolname = ANY($1)`,
				[]string{agedNamedRole, agedCommentedRole, freshCommentedRole, unmarkedRole}))
	})
}

func TestTestRunCleanup(t *testing.T) {
	t.Parallel()

	t.Run("drops every database and role of the run when the test body fails after creating them",
		func(t *testing.T) {
			t.Parallel()
			_, connectionURL := startContainerServer(t)
			server := parseContainerServer(t, connectionURL)
			reporter := newRecordingReporter(t)
			var failedRun *testRun

			reporter.runWithCleanups(func(reporter testReporter) {
				failedRun = startTestRun(reporter, server, time.Now())
				failedRun.createDatabase(reporter, "source")
				failedRun.createDatabase(reporter, "restored")
				failedRun.createRole(reporter, "role")
				server.execute(reporter, failedRun.admin, `CREATE ROLE "databasus-limited"`)
				failedRun.markRole(reporter, "databasus-limited")

				reporter.Fatalf("deliberate failure")
			})

			assert.Equal(t, []string{"deliberate failure"}, reporter.getFailures())
			requireNoObjectOfRun(t, server, failedRun.marker, "databasus-limited")
		})

	t.Run("fails naming the role when a grant in another database keeps the role alive", func(t *testing.T) {
		t.Parallel()
		_, connectionURL := startContainerServer(t)
		server := parseContainerServer(t, connectionURL)
		reporter := newRecordingReporter(t)
		var keptRoleName string

		reporter.runWithCleanups(func(reporter testReporter) {
			run := startTestRun(reporter, server, time.Now())
			keptRoleName = run.createRole(reporter, "role")
			server.execute(reporter, run.admin, fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s",
				pgx.Identifier{server.maintenanceDatabase}.Sanitize(), pgx.Identifier{keptRoleName}.Sanitize()))
		})

		failures := reporter.getFailures()
		require.Len(t, failures, 1)
		assert.Contains(t, failures[0], "could not drop role "+keptRoleName)
	})
}

func requireNoObjectOfRun(t *testing.T, server managedServer, marker string, commentedRoleNames ...string) {
	t.Helper()

	admin := server.open(t, server.maintenanceDatabase)

	assert.Empty(t, queryNames(t, server, admin,
		`SELECT datname FROM pg_database WHERE starts_with(datname, $1)`, marker))
	assert.Empty(t, queryNames(t, server, admin,
		`SELECT rolname FROM pg_roles WHERE starts_with(rolname, $1) OR rolname = ANY($2)`,
		marker, commentedRoleNames))
}
