package runner

import (
	"context"
	"io"

	"github.com/google/uuid"

	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/lib/agentclient"
)

type JobContainer interface {
	Exec(ctx context.Context, cmd []string, stdin io.Reader, env []string) (restore.ExecResult, error)
	GetInContainerConn() dbconn.Conn
	GetVerifierConn() dbconn.Conn
	GetDiskUsageBytes(ctx context.Context) (int64, error)
	Terminate(ctx context.Context) error
}

type Spawner interface {
	Spawn(ctx context.Context, req SpawnRequest) (JobContainer, error)
}

type APIClient interface {
	ClaimVerification(
		ctx context.Context,
		capacity agentclient.AgentCapacity,
	) (*agentclient.VerificationAssignment, error)
	DownloadBackup(ctx context.Context, verificationID uuid.UUID) (io.ReadCloser, error)
	Report(ctx context.Context, verificationID uuid.UUID, req agentclient.VerificationReport) error
}

type Restorer interface {
	StageBackupViaExec(ctx context.Context, exec restore.ExecRunner, body io.Reader, destPath string) error
	RunPgRestore(
		ctx context.Context,
		exec restore.ExecRunner,
		spec restore.PgRestoreSpec,
	) (restore.Result, error)
	EnsureArchiveOwnerRoles(
		ctx context.Context,
		exec restore.ExecRunner,
		archivePath string,
		conn dbconn.Conn,
	) ([]string, error)
	RunTimescalePreRestore(ctx context.Context, conn dbconn.Conn) error
	RunTimescalePostRestore(ctx context.Context, conn dbconn.Conn) error
}

type StatsCollector interface {
	CollectStats(ctx context.Context, conn dbconn.Conn) (verifier.Stats, error)
}

// Registrar is the heartbeat registry seam: register before any container
// exists so the ID rides every heartbeat, and re-check the recorded abort set
// before any FAILED POST.
type Registrar interface {
	TrackVerification(id uuid.UUID, cancel context.CancelFunc)
	UntrackVerification(id uuid.UUID)
	IsAborted(id uuid.UUID) bool
}

type diskUsageProber interface {
	GetDiskUsageBytes(ctx context.Context) (int64, error)
}
