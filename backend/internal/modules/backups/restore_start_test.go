package backups_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// hugeBackupSizeMb is more than any disk the test runs on has free.
const hugeBackupSizeMb = 1_000_000_000_000

var insufficientDiskSpaceDetail = regexp.MustCompile(`needs (\d+) bytes .* but (\d+) bytes are free`)

//nolint:tparallel // The subtests share one source server.
func TestStartRestore(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 202 and completes the restore with a reachable target of the same version", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		targetSpec := server.newTargetSpec(t)

		startedRestore := requireStartedRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, targetSpec))

		assert.Equal(t, database.backup.Id, startedRestore.BackupId)
		waitForRestoreStatus(t, tm.owner, startedRestore.Id, apiclient.RestoreStatusCOMPLETED)
		assert.Equal(t, 3, server.countItems(t, targetSpec.databaseName))
	})

	t.Run("returns 400 database_connection_failed when the target refuses the connection", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		unreachableTarget := newPostgresRestoreTarget(t, containers.Endpoint{Host: "127.0.0.1", Port: 1},
			server.newTargetSpec(t))

		response := startRestore(t, tm.owner, database.backup.Id, unreachableTarget)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeDatabaseConnectionFailed)
		assert.Zero(t, countRestoresOfBackup(t, tm, database.backup.Id))
	})

	t.Run("returns 400 database_connection_failed when the target refuses the credentials", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		targetSpec := server.newTargetSpec(t)
		targetSpec.password = "wrong-password"

		response := startRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, targetSpec))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeDatabaseConnectionFailed)
		assert.Zero(t, countRestoresOfBackup(t, tm, database.backup.Id))
	})

	t.Run("returns 409 already_running when a restore of the same database runs", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		runningRestore, _ := startBlockedRestore(t, tm.owner, server, database.backup.Id)

		response := startRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, server.newTargetSpec(t)))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)
		assert.Equal(t, 1, countRestoresOfBackup(t, tm, database.backup.Id))
		assert.Equal(t, apiclient.RestoreStatusINPROGRESS, getRestore(t, tm.owner, runningRestore.Id).Status)
	})

	t.Run("returns 400 invalid_request when the backup failed", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		server.exec(t, containers.PostgresDatabase, "DROP DATABASE "+database.sourceName+" WITH (FORCE)")
		failedBackup := startRestorableBackup(t, tm.owner, database.databaseID)
		waitForRestorableBackupStatus(t, tm.owner, failedBackup.Id, apiclient.LogicalBackupStatusFAILED)

		response := startRestore(t, tm.owner, failedBackup.Id, server.newRestoreTarget(t, server.newTargetSpec(t)))

		requireInvalidField(t, response.StatusCode(), response.Body, "backupId")
		assert.Zero(t, countRestoresOfBackup(t, tm, failedBackup.Id))
	})

	t.Run("returns 400 invalid_request when the target is of another engine", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		var mysqlTarget apiclient.RestoreTarget
		require.NoError(t, mysqlTarget.FromMysqlRestoreTarget(apiclient.MysqlRestoreTarget{
			Type:      apiclient.MysqlRestoreTargetTypeMYSQL,
			Host:      server.endpoint.Host,
			Port:      server.endpoint.Port,
			Username:  containers.PostgresUsername,
			Password:  new(containers.PostgresPassword),
			Database:  "target",
			IsHttps:   false,
			SshTunnel: newDisabledRestoreSSHTunnel(),
		}))

		response := startRestore(t, tm.owner, database.backup.Id, mysqlTarget)

		requireInvalidField(t, response.StatusCode(), response.Body, "target.type")
		assert.Zero(t, countRestoresOfBackup(t, tm, database.backup.Id))
	})

	t.Run("returns 400 invalid_request when the target runs an older major version", func(t *testing.T) {
		olderServer := startRestoreServer(t, "postgres:15")
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		response := startRestore(t, tm.owner, database.backup.Id,
			olderServer.newRestoreTarget(t, olderServer.newTargetSpec(t)))

		requireInvalidField(t, response.StatusCode(), response.Body, "target")
		assert.Zero(t, countRestoresOfBackup(t, tm, database.backup.Id))
	})

	t.Run("returns 409 insufficient_disk_space with the bytes when a parallel restore needs more disk",
		func(t *testing.T) {
			tm := newTeam(t)
			database := newBackedUpDatabase(t, tm, server)
			tm.app.ExecuteSQL(t, "UPDATE backups_logical SET backup_size_mb = ? WHERE id = ?",
				hugeBackupSizeMb, database.backup.Id)
			targetSpec := server.newTargetSpec(t)
			targetSpec.cpuCount = 2

			response := startRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, targetSpec))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
				apiclient.ProblemCodeInsufficientDiskSpace)

			var problem apiclient.Problem
			require.NoError(t, json.Unmarshal(response.Body, &problem))

			detailBytes := insufficientDiskSpaceDetail.FindStringSubmatch(problem.Detail)
			require.Len(t, detailBytes, 3, problem.Detail)

			neededBytes, err := strconv.ParseInt(detailBytes[1], 10, 64)
			require.NoError(t, err)
			freeBytes, err := strconv.ParseInt(detailBytes[2], 10, 64)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, neededBytes, int64(hugeBackupSizeMb*1024*1024))
			assert.Less(t, freeBytes, neededBytes)
			assert.Zero(t, countRestoresOfBackup(t, tm, database.backup.Id))
		})
}
