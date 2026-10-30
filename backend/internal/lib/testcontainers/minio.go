package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test MinIO container (the S3-compatible backend).
const (
	MinioRootUser     = "testuser"
	MinioRootPassword = "testpassword"
	MinioRegion       = "us-east-1"
)

const minioPort = "9000/tcp"

// MinIO stopped publishing community images: both `minio/minio` on Docker Hub and
// `quay.io/minio/minio` now deny anonymous pulls. `pgsty/minio` is a community-maintained
// fork of the same server, so the command, env and health endpoint stay the same. The tag
// is pinned because a test that silently follows upstream is a test that breaks on someone
// else's release day.
const MinioImage = "pgsty/minio:RELEASE.2026-08-04T00-00-00Z"

// StartMinio's container starts empty: the caller creates whatever bucket it needs against the
// returned endpoint, or for failure-path tests points at a missing one.
func StartMinio(t *testing.T) Endpoint {
	t.Helper()

	return StartMinioContainer(t).Endpoint
}

func StartMinioContainer(t *testing.T) ContainerHandle {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        MinioImage,
		ExposedPorts: []string{minioPort},
		Env: map[string]string{
			"MINIO_ROOT_USER":     MinioRootUser,
			"MINIO_ROOT_PASSWORD": MinioRootPassword,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort(minioPort).WithStartupTimeout(120 * time.Second),
	}

	return startContainer(t, req, minioPort)
}
