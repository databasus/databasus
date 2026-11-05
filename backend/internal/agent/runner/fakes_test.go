package runner

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/lib/agentclient"
)

type fakeAPI struct {
	mu sync.Mutex

	claims    []*agentclient.VerificationAssignment
	claimErrs []error
	claimIdx  int

	downloadBody []byte
	downloadErr  error

	reportErr    error
	reportedReqs []agentclient.VerificationReport
	reportedIDs  []uuid.UUID
	reportCalled int
}

func (f *fakeAPI) ClaimVerification(
	_ context.Context, _ agentclient.AgentCapacity,
) (*agentclient.VerificationAssignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	i := f.claimIdx
	f.claimIdx++

	if i < len(f.claimErrs) && f.claimErrs[i] != nil {
		return nil, f.claimErrs[i]
	}

	if i < len(f.claims) {
		return f.claims[i], nil
	}

	return nil, nil
}

func (f *fakeAPI) DownloadBackup(
	_ context.Context, _ uuid.UUID,
) (io.ReadCloser, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}

	return io.NopCloser(bytes.NewReader(f.downloadBody)), nil
}

func (f *fakeAPI) Report(
	_ context.Context, verificationID uuid.UUID, req agentclient.VerificationReport,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.reportCalled++
	f.reportedReqs = append(f.reportedReqs, req)
	f.reportedIDs = append(f.reportedIDs, verificationID)

	return f.reportErr
}

func (f *fakeAPI) lastReport() (agentclient.VerificationReport, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.reportedReqs) == 0 {
		return agentclient.VerificationReport{}, false
	}

	return f.reportedReqs[len(f.reportedReqs)-1], true
}

type fakeContainer struct {
	inContainerConn dbconn.Conn
	verifierConn    dbconn.Conn
	terminated      bool
	diskUsageBytes  atomic.Int64
	diskUsageErr    error
}

func (c *fakeContainer) Exec(
	_ context.Context, _ []string, stdin io.Reader, _ []string,
) (restore.ExecResult, error) {
	if stdin != nil {
		_, _ = io.Copy(io.Discard, stdin)
	}

	return restore.ExecResult{}, nil
}

func (c *fakeContainer) GetInContainerConn() dbconn.Conn { return c.inContainerConn }
func (c *fakeContainer) GetVerifierConn() dbconn.Conn    { return c.verifierConn }

func (c *fakeContainer) GetDiskUsageBytes(context.Context) (int64, error) {
	return c.diskUsageBytes.Load(), c.diskUsageErr
}

func (c *fakeContainer) Terminate(context.Context) error {
	c.terminated = true
	return nil
}

type fakeSpawner struct {
	container JobContainer
	err       error
}

func (s *fakeSpawner) Spawn(_ context.Context, _ SpawnRequest) (JobContainer, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.container, nil
}

type fakeRestorer struct {
	stageErr             error
	ensureRolesErr       error
	ensureRolesBlocks    bool
	onEnsureRolesEntered func()
	runResult            restore.Result
	runErr               error
	runBlocks            bool

	runEntered      atomic.Bool
	runCtxCancelled atomic.Bool

	// calls records the restore-related call order ("roles", "pre", "restore", "post") for a single job.
	calls []string
	// runParallelJobs records the -j value RunPgRestore was invoked with.
	runParallelJobs  int
	runIsTimescaledb bool
}

func (r *fakeRestorer) StageBackupViaExec(
	_ context.Context, _ restore.ExecRunner, body io.Reader, _ string,
) error {
	if body != nil {
		_, _ = io.Copy(io.Discard, body)
	}

	return r.stageErr
}

func (r *fakeRestorer) RunPgRestore(
	ctx context.Context, _ restore.ExecRunner, spec restore.PgRestoreSpec,
) (restore.Result, error) {
	r.calls = append(r.calls, "restore")
	r.runParallelJobs = spec.ParallelJobs
	r.runIsTimescaledb = spec.IsTimescaledb

	if r.runBlocks {
		r.runEntered.Store(true)
		<-ctx.Done()
		r.runCtxCancelled.Store(true)

		return restore.Result{}, ctx.Err()
	}

	return r.runResult, r.runErr
}

func (r *fakeRestorer) EnsureArchiveOwnerRoles(
	ctx context.Context, _ restore.ExecRunner, _ string, _ dbconn.Conn,
) ([]string, error) {
	r.calls = append(r.calls, "roles")

	if r.onEnsureRolesEntered != nil {
		r.onEnsureRolesEntered()
	}

	if r.ensureRolesBlocks {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	if r.ensureRolesErr != nil {
		return nil, r.ensureRolesErr
	}

	return []string{"ts_app"}, nil
}

func (r *fakeRestorer) RunTimescalePreRestore(context.Context, dbconn.Conn) error {
	r.calls = append(r.calls, "pre")

	return nil
}

func (r *fakeRestorer) RunTimescalePostRestore(context.Context, dbconn.Conn) error {
	r.calls = append(r.calls, "post")

	return nil
}

type panickingRestorer struct {
	Restorer
	rolesCalls                atomic.Int32
	blockedRestoreStarted     chan struct{}
	blockedRestoreFinished    chan struct{}
	isBlockedRestoreCancelled atomic.Bool
}

func (r *panickingRestorer) EnsureArchiveOwnerRoles(
	ctx context.Context, _ restore.ExecRunner, _ string, _ dbconn.Conn,
) ([]string, error) {
	if r.rolesCalls.Add(1) == 1 {
		close(r.blockedRestoreStarted)
		<-ctx.Done()
		r.isBlockedRestoreCancelled.Store(true)
		close(r.blockedRestoreFinished)

		return nil, ctx.Err()
	}

	<-r.blockedRestoreStarted
	panic("restore panic")
}

type fakeStats struct {
	stats verifier.Stats
	err   error
}

func (s *fakeStats) CollectStats(context.Context, dbconn.Conn) (verifier.Stats, error) {
	return s.stats, s.err
}

type fakeRegistrar struct {
	mu         sync.Mutex
	registered map[uuid.UUID]struct{}
	aborted    bool
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{registered: make(map[uuid.UUID]struct{})}
}

func (r *fakeRegistrar) TrackVerification(id uuid.UUID, _ context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.registered[id] = struct{}{}
}

func (r *fakeRegistrar) UntrackVerification(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.registered, id)
}

func (r *fakeRegistrar) IsAborted(uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.aborted
}
