package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const (
	minioNetworkAlias   = "recovery-minio"
	minioPublishedPort  = "9000/tcp"
	backupBucket        = "databasus-recovery-cycle"
	sourceServerImage   = "postgres:17"
	sourceServerAlias   = "recovery-source-db"
	sourceServerPort    = 5432
	sourceServerPublish = "5432/tcp"
	sourceUsername      = "postgres"
	sourcePassword      = "recovery-source-password"
	sourceDatabaseName  = "recovery_source"
	cleanupTimeout      = 3 * time.Minute
	// containerStartLimit matches the testcontainers start-up waits. A Databasus container initializes
	// its internal database and runs migrations or a recovery during this wait.
	containerStartLimit = 5 * time.Minute
)

// recoveryCycle is what outlives the lost Databasus installation: the database server it backed up
// and the remote storage holding the Databasus backup and the database backups. The scenarios share
// it and own everything else they start.
type recoveryCycle struct {
	docker *containers.DockerHost
	image  string
	prefix string

	networkName      string
	storageClient    *minio.Client
	sourceServerPort int
	// postgresImage is the pinned image of the source server, which the physical restore reuses.
	postgresImage string

	// physicalSourceTurn lets one recovered installation at a time stream WAL from the source server:
	// its replication slot serves a single consumer, so two recovered installations would take it
	// from each other.
	physicalSourceTurn sync.Mutex

	resourcesMutex sync.Mutex
	resources      containers.DockerResources
}

func newRecoveryCycle(t *testing.T) *recoveryCycle {
	t.Helper()

	docker, err := containers.NewDockerHost()
	require.NoError(t, err)
	t.Cleanup(docker.Close)

	suffix := make([]byte, 6)
	_, err = rand.Read(suffix)
	require.NoError(t, err)

	cycle := &recoveryCycle{
		docker:        docker,
		image:         getCandidateImage(t),
		prefix:        "databasus-recovery-" + hex.EncodeToString(suffix),
		postgresImage: containers.GetPinnedImage(t, sourceServerImage),
	}
	t.Cleanup(func() { cycle.removeResources(t, cycle.takeResources()) })

	cycle.networkName = cycle.trackNetwork(cycle.prefix + "-network")
	require.NoError(t, docker.CreateNetwork(t.Context(), cycle.networkName))

	cycle.startStorage(t)
	cycle.startSourceServer(t)

	return cycle
}

func (c *recoveryCycle) startStorage(t *testing.T) {
	t.Helper()

	containerName := c.trackContainer(c.prefix + "-minio")
	require.NoError(t, c.docker.StartContainer(t.Context(), containers.RunSpec{
		Name:       containerName,
		Image:      containers.GetPinnedImage(t, containers.MinioImage),
		Entrypoint: []string{"minio", "server", "/data"},
		Env: []string{
			"MINIO_ROOT_USER=" + containers.MinioRootUser,
			"MINIO_ROOT_PASSWORD=" + containers.MinioRootPassword,
		},
		Mount:         "",
		NetworkName:   c.networkName,
		NetworkAlias:  minioNetworkAlias,
		PublishedPort: minioPublishedPort,
	}))

	hostPort := c.waitForPublishedPort(t, containerName, minioPublishedPort)
	storageClient, err := minio.New(fmt.Sprintf("127.0.0.1:%d", hostPort), &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Region: containers.MinioRegion,
	})
	require.NoError(t, err)

	// The health endpoints answer before S3 requests work, so creating the bucket is the wait.
	testwait.Within(containerStartLimit).
		Until(t, "the remote storage creates the bucket", func(ctx context.Context) error {
			return storageClient.MakeBucket(ctx, backupBucket, minio.MakeBucketOptions{Region: containers.MinioRegion})
		})

	c.storageClient = storageClient
}

// startSourceServer starts the database server whose databases the source installation backs up
// logically and physically, and which the recovered installations restore into.
func (c *recoveryCycle) startSourceServer(t *testing.T) {
	t.Helper()

	containerName := c.trackContainer(c.prefix + "-source-db")
	require.NoError(t, c.docker.StartContainer(t.Context(), containers.RunSpec{
		Name:  containerName,
		Image: c.postgresImage,
		Entrypoint: []string{
			"sh",
			"-ec",
			`printf '%s\n' '#!/bin/sh' 'echo "host replication all all scram-sha-256" >> "$PGDATA/pg_hba.conf"' > /docker-entrypoint-initdb.d/replication.sh
exec docker-entrypoint.sh postgres -c summarize_wal=on`,
		},
		Env: []string{
			"POSTGRES_USER=" + sourceUsername,
			"POSTGRES_PASSWORD=" + sourcePassword,
			"POSTGRES_DB=" + sourceDatabaseName,
		},
		Mount:         "",
		NetworkName:   c.networkName,
		NetworkAlias:  sourceServerAlias,
		PublishedPort: sourceServerPublish,
	}))

	c.sourceServerPort = c.waitForPublishedPort(t, containerName, sourceServerPublish)

	testwait.Within(containerStartLimit).
		Until(t, "the source server accepts connections", func(ctx context.Context) error {
			connection, err := c.connectToSourceServer(ctx, sourceDatabaseName)
			if err != nil {
				return err
			}

			return connection.Close(ctx)
		})
}

func (c *recoveryCycle) connectToSourceServer(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	connection, err := pgx.Connect(ctx, fmt.Sprintf(
		"host=127.0.0.1 port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.sourceServerPort, sourceUsername, sourcePassword, databaseName,
	))
	if err != nil {
		return nil, fmt.Errorf("connect to %s on the source server: %w", databaseName, err)
	}

	return connection, nil
}

func (c *recoveryCycle) openSourceDatabase(t *testing.T, databaseName string) *pgx.Conn {
	t.Helper()

	connection, err := c.connectToSourceServer(t.Context(), databaseName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })

	return connection
}

func (c *recoveryCycle) waitForPublishedPort(t *testing.T, containerName, publishedPort string) int {
	t.Helper()

	var hostPort int

	testwait.Within(containerStartLimit).
		Until(t, containerName+" publishes "+publishedPort, func(ctx context.Context) error {
			state, err := c.docker.GetContainerState(ctx, containerName, publishedPort)
			if err != nil {
				return err
			}

			if !state.IsRunning {
				return fmt.Errorf("container %s is not running", containerName)
			}

			hostPort = state.HostPort

			return nil
		})

	return hostPort
}

func (c *recoveryCycle) trackContainer(containerName string) string {
	c.resourcesMutex.Lock()
	defer c.resourcesMutex.Unlock()

	c.resources.ContainerNames = append(c.resources.ContainerNames, containerName)

	return containerName
}

func (c *recoveryCycle) trackNetwork(networkName string) string {
	c.resourcesMutex.Lock()
	defer c.resourcesMutex.Unlock()

	c.resources.NetworkNames = append(c.resources.NetworkNames, networkName)

	return networkName
}

func (c *recoveryCycle) takeResources() containers.DockerResources {
	c.resourcesMutex.Lock()
	defer c.resourcesMutex.Unlock()

	taken := c.resources
	c.resources = containers.DockerResources{}

	return taken
}

func (c *recoveryCycle) removeResources(t *testing.T, resources containers.DockerResources) {
	t.Helper()

	cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
	defer cancelCleanup()

	require.NoError(t, c.docker.RemoveResources(cleanupCtx, resources))

	remainingNames, err := c.docker.GetRemainingResources(cleanupCtx, resources)
	require.NoError(t, err)
	require.Empty(t, remainingNames, "the recovery cycle leaves no Docker resource behind")
}
