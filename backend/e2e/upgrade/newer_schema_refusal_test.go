package upgrade

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

const (
	// newerMigrationVersion is a migration of a release newer than every image the test runs.
	newerMigrationVersion = 29991231235959
	keptLocalBackupPath   = instanceDataPath + "/backups/kept-local-backup.dump"
	installedKeyPath      = instanceDataPath + "/secret.key"
)

func TestNewerSchemaRefusal(t *testing.T) {
	t.Parallel()

	t.Run("refuses a database a newer release migrated and explains the fresh-install alternative", func(t *testing.T) {
		t.Parallel()
		currentImage := getCurrentImage(t)
		resources, err := newUpgradeResources()
		require.NoError(t, err)
		docker, err := newDockerHost()
		require.NoError(t, err)
		t.Cleanup(docker.Close)
		t.Cleanup(func() {
			cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
			defer cancelCleanup()
			require.NoError(t, docker.removeResources(cleanupCtx, resources))
		})
		require.NoError(t, docker.CreateNetwork(t.Context(), resources.networkName))
		require.NoError(t, docker.CreateVolume(t.Context(), resources.volumeName))

		migratedAPI := startRetirementInstance(t, docker, retirementInstanceSpec{
			resources: resources, image: currentImage, name: resources.previousContainerName,
		})
		require.NoError(t, migratedAPI.post(t.Context(), "/users", previousSignUpRequest{
			Email: accountEmail, Password: accountPassword, Name: accountName,
		}, nil))
		_, err = docker.Execute(t.Context(), resources.previousContainerName, []string{
			"gosu", "databasus", "sh", "-ec", "printf 'local backup' > " + keptLocalBackupPath,
		})
		require.NoError(t, err)
		_, err = docker.Execute(t.Context(), resources.previousContainerName, []string{
			"sh", "-c",
			`psql "$(cat /dev/shm/databasus-database-dsn)" -AtXq -v ON_ERROR_STOP=1 -c "INSERT INTO goose_db_version ` +
				`(version_id, is_applied) VALUES (` + strconv.Itoa(newerMigrationVersion) + `, TRUE)"`,
		})
		require.NoError(t, err)
		require.NoError(t, docker.StopContainer(t.Context(), resources.previousContainerName))
		keyBeforeRefusal := readStoppedFile(t, docker, resources.previousContainerName, installedKeyPath)
		removeRetirementContainer(t, docker, resources.previousContainerName)
		snapshotBeforeRefusal := getStoppedRetirementSnapshot(
			t,
			docker,
			retirementSnapshotSpec{resources: resources, image: currentImage},
		)

		_, err = startInstance(t, docker, containers.RunSpec{
			Name: resources.currentContainerName, Image: currentImage, Entrypoint: nil,
			Env:   []string{"IS_DISABLE_ANONYMOUS_TELEMETRY=true"},
			Mount: resources.volumeName + ":" + instanceDataPath, NetworkName: resources.networkName,
			NetworkAlias: resources.currentContainerName, PublishedPort: instancePublishedPort,
		})

		require.ErrorIs(t, err, errInstanceExited)
		refusalLog, err := docker.GetLogs(t.Context(), resources.currentContainerName)
		require.NoError(t, err)
		for _, expectedText := range []string{
			strconv.Itoa(newerMigrationVersion),
			"refusing to start on a database migrated by a newer version",
			"Start the image of the newer version instead to keep this installation",
			"This is not a downgrade: it deletes every account, every setting, the master key",
			"cannot import Databasus backups made by the newer one",
			"copy every Databasus backup, the master key file and the local backup files",
			"sudo rm -rf -- ./databasus-data",
			"docker volume rm <data-volume>",
			"docker rm -f -v databasus",
			"If DATABASE_DSN points to an external database",
		} {
			assert.Contains(t, refusalLog, expectedText)
		}
		assert.NotContains(t, refusalLog, "database migrations completed successfully")
		assert.NotContains(t, refusalLog, "http server listening")
		assert.Equal(t, keyBeforeRefusal, readStoppedFile(t, docker, resources.currentContainerName, installedKeyPath))
		assert.Equal(t, []byte("local backup"),
			readStoppedFile(t, docker, resources.currentContainerName, keptLocalBackupPath))
		removeRetirementContainer(t, docker, resources.currentContainerName)
		assert.Equal(t, snapshotBeforeRefusal, getStoppedRetirementSnapshot(
			t,
			docker,
			retirementSnapshotSpec{resources: resources, image: currentImage},
		), "the refusal changes no record and no migration history")
	})
}

// getCurrentImage is the image under test; this check needs no previous release.
func getCurrentImage(t *testing.T) string {
	t.Helper()

	//nolint:forbidigo // mise run backend:test-upgrade names the image, outside any configuration of ours
	currentImage := os.Getenv(currentImageVariable)
	if currentImage == "" {
		t.Fatalf("%s names the image to check; run the test with `mise run backend:test-upgrade`",
			currentImageVariable)
	}

	return currentImage
}

func readStoppedFile(t *testing.T, docker *dockerHost, containerName, filePath string) []byte {
	t.Helper()

	content, err := docker.ReadFileFromContainer(t.Context(), containerName, filePath)
	require.NoError(t, err)

	return content
}
