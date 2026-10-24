package containers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

// physicalPrimaryAlias is the network alias the standby resolves its primary by. Both containers
// must share a user-defined network for DNS by alias — the default bridge resolves no names.
const physicalPrimaryAlias = "databasus-primary"

// physicalStandbySlotName is the physical replication slot the standby creates on the primary to
// pin WAL until it has streamed it. Distinct from the databasus_* slots Databasus manages on the
// node it backs up.
const physicalStandbySlotName = "databasus_test_standby"

// physicalDataDirTmpfsOptions sizes the tmpfs for physical containers well above the generic
// logical size: a replication source retains WAL (wal_keep_size plus a slot that pins every
// streamed segment until the uploader commits it), and a WAL-stream test forces a 16 MB segment
// per rotation, so the data dir can briefly hold hundreds of MB of un-recycled WAL. The 512m
// logical default fills mid-test and the source crashes with "No space left on device".
const physicalDataDirTmpfsOptions = "rw,size=2g"

type physicalPostgresOptions struct {
	shouldSummarizeWal            bool
	shouldEnableFullPageWrites    bool
	shouldCreateTablespace        bool
	shouldOmitReplicationHbaEntry bool
	shouldSpreadCheckpoints       bool
	maxConnections                int
	maxWalSenders                 int
}

// PhysicalPostgresOption tunes a physical source container away from its replication-ready default.
type PhysicalPostgresOption func(*physicalPostgresOptions)

// WithoutSummarizer starts the cluster with summarize_wal=off, so an incremental pre-flight reaches
// the SUMMARIZER_OFF fallback deterministically.
func WithoutSummarizer() PhysicalPostgresOption {
	return func(o *physicalPostgresOptions) { o.shouldSummarizeWal = false }
}

// WithTablespace pre-creates a custom tablespace at first boot so the pre-flight rejects the cluster
// because physical backups do not support custom tablespaces (spec physical-backups/backup-method).
func WithTablespace() PhysicalPostgresOption {
	return func(o *physicalPostgresOptions) { o.shouldCreateTablespace = true }
}

// WithoutReplicationHbaEntry omits the "host replication" pg_hba line, leaving only the image's
// default "host all all all" rule. Such a cluster accepts ordinary and logical-replication
// connections but refuses physical replication — the mode pg_basebackup / pg_receivewal actually use.
func WithoutReplicationHbaEntry() PhysicalPostgresOption {
	return func(o *physicalPostgresOptions) { o.shouldOmitReplicationHbaEntry = true }
}

// WithMaxConnections runs the source with a raised max_connections so a restored cluster booted at
// the image default (100) must pick up the primary's higher value to replay WAL - the recovery
// "insufficient parameter settings" case.
func WithMaxConnections(connections int) PhysicalPostgresOption {
	return func(o *physicalPostgresOptions) { o.maxConnections = connections }
}

// WithMaxWalSenders caps the cluster's WAL sender slots so a test can hold all of them and observe
// what a further replication connection, or a probe that must not need one, does at the limit.
func WithMaxWalSenders(walSenders int) PhysicalPostgresOption {
	return func(o *physicalPostgresOptions) { o.maxWalSenders = walSenders }
}

// A promotion only requests its checkpoint, so on a node carrying dirty buffers that request stays
// outstanding for the length of a test and pg_control_checkpoint() keeps answering the pre-failover
// timeline — the stale reading a timeline probe must not trust. The completion target is pinned so
// the pacing does not depend on the image's default.
func WithCheckpointsSpreadOverAnHour() PhysicalPostgresOption {
	return func(o *physicalPostgresOptions) { o.shouldSpreadCheckpoints = true }
}

// allowReplicationInitScript appends the pg_hba replication rule at first boot. "host all all" does
// not cover replication connections, so pg_basebackup / pg_receivewal need this explicit line.
func allowReplicationInitScript() testcontainers.ContainerFile {
	body := "#!/bin/bash\nset -e\n" +
		`echo "host replication all all scram-sha-256" >> "$PGDATA/pg_hba.conf"` + "\n"

	return testcontainers.ContainerFile{
		Reader:            strings.NewReader(body),
		ContainerFilePath: "/docker-entrypoint-initdb.d/02_allow_replication.sh",
		FileMode:          0o755,
	}
}

// createTablespaceInitScript pre-creates the custom_ts tablespace at first boot.
func createTablespaceInitScript() testcontainers.ContainerFile {
	body := "#!/bin/bash\nset -e\n" +
		"mkdir -p /var/lib/postgresql/custom_tablespace_dir\n" +
		`psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-SQL` + "\n" +
		"  CREATE TABLESPACE custom_ts LOCATION '/var/lib/postgresql/custom_tablespace_dir';\n" +
		"SQL\n"

	return testcontainers.ContainerFile{
		Reader:            strings.NewReader(body),
		ContainerFilePath: "/docker-entrypoint-initdb.d/01_create_tablespace.sh",
		FileMode:          0o755,
	}
}

func physicalPostgresCmd(opts physicalPostgresOptions) []string {
	summarize := "summarize_wal=on"
	if !opts.shouldSummarizeWal {
		summarize = "summarize_wal=off"
	}

	// pg_basebackup from a standby refuses a cluster whose WAL was generated with full_page_writes=off
	// ("backup ... is corrupt"), so the replication-pair callers turn it on; single-node sources keep
	// it off for the throwaway-durability speedup.
	fullPageWrites := "full_page_writes=off"
	if opts.shouldEnableFullPageWrites {
		fullPageWrites = "full_page_writes=on"
	}

	maxWalSenders := 10
	if opts.maxWalSenders > 0 {
		maxWalSenders = opts.maxWalSenders
	}

	cmd := []string{
		"-c", "fsync=off", "-c", fullPageWrites, "-c", "synchronous_commit=off",
		"-c", "wal_level=logical", "-c", summarize,
		"-c", fmt.Sprintf("max_wal_senders=%d", maxWalSenders),
		"-c", "max_replication_slots=10", "-c", "wal_keep_size=512MB",
	}
	if opts.maxConnections > 0 {
		cmd = append(cmd, "-c", fmt.Sprintf("max_connections=%d", opts.maxConnections))
	}

	if opts.shouldSpreadCheckpoints {
		cmd = append(cmd, "-c", "checkpoint_timeout=1h", "-c", "checkpoint_completion_target=0.9")
	}

	return cmd
}

// StartPhysicalPostgres boots a replication-capable PostgreSQL source from image (e.g. "postgres:17"):
// the logical wal_level, WAL senders/slots and a pg_hba replication rule that pg_basebackup and
// pg_receivewal require, on top of the throwaway-server durability flags.
func StartPhysicalPostgres(t *testing.T, image string, opts ...PhysicalPostgresOption) Endpoint {
	t.Helper()

	return start(t, physicalPostgresRequest(image, resolvePhysicalPostgresOptions(opts)), postgresPort)
}

// The database publishes no ports and the bastion does, so a tunnel is the only way in — a direct
// route would keep the tunnel tests green after the tunnel stopped being used.
func StartPhysicalPostgresBehindSshBastion(
	t *testing.T,
	image string,
	opts ...PhysicalPostgresOption,
) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)
	bastion := StartSshBastionContainerOnNetwork(t, networkName)

	req := physicalPostgresRequest(image, resolvePhysicalPostgresOptions(opts))
	// postgresReady resolves the published host port, and this container has none.
	req.WaitingFor = wait.ForLog("database system is ready to accept connections").
		WithOccurrence(2).WithStartupTimeout(postgresStartupTimeout)

	placement := NetworkPlacement{NetworkName: networkName, Alias: bastionedPhysicalPostgresAlias}
	startUnpublished(t, req, placement)

	return BastionedDatabase{
		Bastion:          bastion.Endpoint,
		BastionContainer: bastion.Container,
		Database:         Endpoint{Host: placement.Alias, Port: getPortNumber(postgresPort)},
	}
}

func resolvePhysicalPostgresOptions(opts []PhysicalPostgresOption) physicalPostgresOptions {
	options := physicalPostgresOptions{shouldSummarizeWal: true}
	for _, apply := range opts {
		apply(&options)
	}

	return options
}

func physicalPostgresRequest(image string, options physicalPostgresOptions) testcontainers.ContainerRequest {
	var files []testcontainers.ContainerFile
	if !options.shouldOmitReplicationHbaEntry {
		files = append(files, allowReplicationInitScript())
	}
	if options.shouldCreateTablespace {
		files = append(files, createTablespaceInitScript())
	}

	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{postgresPort},
		Env:          postgresEnv(),
		Cmd:          physicalPostgresCmd(options),
		Files:        files,
		Tmpfs:        map[string]string{postgresDataDir(image): physicalDataDirTmpfsOptions},
		WaitingFor:   postgresReady(),
	}
}

// StartPhysicalPrimaryWithStandby boots a replication-capable primary and a streaming standby that
// clones it with pg_basebackup over a shared network, returning both endpoints (primary first).
// Promoting the standby (PromoteStandby) bumps its timeline while the postmaster stays up, so a
// pg_receivewal already streaming from it observes the switch and fetches the new timeline's
// .history file — the failover path issue #643 regressed. The network and both containers are torn
// down with the test.
func StartPhysicalPrimaryWithStandby(
	t *testing.T,
	image string,
	opts ...PhysicalPostgresOption,
) (Endpoint, Endpoint) {
	t.Helper()

	options := resolvePhysicalPostgresOptions(opts)

	// A base backup taken on the standby requires full_page_writes — force it on for both nodes.
	options.shouldEnableFullPageWrites = true

	replicationNetwork, err := newLabelledNetwork()
	if err != nil {
		t.Fatalf("failed to create replication test network: %v", err)
	}
	t.Cleanup(func() { _ = replicationNetwork.Remove(context.Background()) })

	primary := startPhysicalPrimaryOnNetwork(t, image, options, replicationNetwork.Name)
	standby := startPhysicalStandbyOnNetwork(t, image, options, replicationNetwork.Name)

	return primary, standby
}

// startPhysicalPrimaryOnNetwork boots the replication source joined to networkName under
// physicalPrimaryAlias, so the standby can reach it by name. It mirrors StartPhysicalPostgres'
// request otherwise (replication pg_hba rule, durability and WAL flags).
func startPhysicalPrimaryOnNetwork(
	t *testing.T,
	image string,
	options physicalPostgresOptions,
	networkName string,
) Endpoint {
	t.Helper()

	req := physicalPostgresRequest(image, options)
	req.Networks = []string{networkName}
	req.NetworkAliases = map[string][]string{networkName: {physicalPrimaryAlias}}

	return start(t, req, postgresPort)
}

// startPhysicalStandbyOnNetwork boots a standby that clones the primary (reached by alias on
// networkName) before postgres starts. A custom entrypoint runs pg_basebackup -R into PGDATA, then
// hands the same WAL flags to the image entrypoint, which serves the cloned cluster as a streaming
// standby. The readiness log differs from a primary ("read-only connections"), so it gets its own
// wait.
func startPhysicalStandbyOnNetwork(
	t *testing.T,
	image string,
	options physicalPostgresOptions,
	networkName string,
) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{postgresPort},
		Env:          postgresEnv(),
		Entrypoint:   []string{"/usr/local/bin/databasus-standby-entrypoint.sh"},
		Cmd:          physicalPostgresCmd(options),
		Files: []testcontainers.ContainerFile{{
			Reader:            strings.NewReader(standbyEntrypointScript()),
			ContainerFilePath: "/usr/local/bin/databasus-standby-entrypoint.sh",
			FileMode:          0o755,
		}},
		Tmpfs:      map[string]string{postgresDataDir(image): physicalDataDirTmpfsOptions},
		Networks:   []string{networkName},
		WaitingFor: standbyReady(),
	}

	return start(t, req, postgresPort)
}

// standbyEntrypointScript clones the primary into an empty PGDATA and writes the standby recovery
// config (-R records the alias conninfo, password included so streaming reconnects). It runs as
// root (the image default) and drops to postgres for the clone so the cluster files are owned by
// the server uid, then exec's the image entrypoint to serve it. The final "$@" forwards the WAL
// flags passed as the container command.
func standbyEntrypointScript() string {
	return `#!/bin/bash
set -euo pipefail
mkdir -p "$PGDATA"
chown -R postgres:postgres "$PGDATA"
echo "databasus-standby: waiting for primary to accept connections"
until pg_isready -h ` + physicalPrimaryAlias + ` -p 5432 -U ` + PostgresUsername + ` >/dev/null 2>&1; do sleep 1; done
echo "databasus-standby: cloning primary via pg_basebackup"
gosu postgres pg_basebackup -D "$PGDATA" -R -X stream -c fast -C -S ` + physicalStandbySlotName + ` -w \
  -d "host=` + physicalPrimaryAlias + ` port=5432 user=` + PostgresUsername +
		` password=` + PostgresPassword + ` dbname=postgres"
echo "databasus-standby: starting as a streaming standby"
exec docker-entrypoint.sh postgres "$@"
`
}

// standbyReady waits on the standby's own readiness line. A node in archive recovery logs "ready to
// accept read-only connections" (once, after reaching consistency), not the primary's twice-logged
// "ready to accept connections", so postgresReady would hang here.
func standbyReady() wait.Strategy {
	return wait.ForAll(
		wait.ForLog("database system is ready to accept read-only connections").
			WithStartupTimeout(postgresStartupTimeout),
		wait.ForListeningPort(postgresPort),
	)
}

// StartPostgresWithBoundDataDir boots a normal postgres container through its image entrypoint (not
// pg_ctl) with hostVolumeDir bind-mounted at the image's data VOLUME, reproducing a docker-compose
// '- ./pgdata:<volume>' mount. This is how a restored cluster is actually served, so it exercises the
// PGDATA layout the recovery script must produce. The data already exists, so the entrypoint skips
// initdb, replays any recovery.signal and serves it. hostVolumeDir lives under the test's TempDir,
// which the (docker-in-docker) daemon shares, so the bind resolves.
func StartPostgresWithBoundDataDir(t *testing.T, image, hostVolumeDir string) Endpoint {
	t.Helper()

	bind := hostVolumeDir + ":" + postgresDataDir(image)

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{postgresPort},
		Env:          postgresEnv(),
		// Run as the host user that built the bind-mounted cluster. As root the entrypoint chowns
		// PGDATA to the postgres uid, which would then break the test's TempDir cleanup; as the
		// owning uid it skips that chown and still exercises the layout detection and data serving.
		User: fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = append(hc.Binds, bind)
		},
		// An already-initialized cluster runs no initdb temp server, so "ready to accept
		// connections" is logged once (after PITR recovery + promotion, when present) - not twice
		// like a fresh boot, so postgresReady()'s 2-occurrence wait would hang here.
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithStartupTimeout(postgresStartupTimeout),
			wait.ForListeningPort(postgresPort),
		),
	}

	return start(t, req, postgresPort)
}

// StartPhysicalPostgresMtls builds and boots the replication-capable mTLS PostgreSQL source from
// contextDir (Dockerfile + server cert/key, ca.crt and a replication-aware pg_hba.conf). Same
// key-permission rationale as StartPostgresMtls: the Dockerfile chowns+chmods the key, which a
// copied-in file cannot. The test connects with sslmode=verify-ca and a client cert.
func StartPhysicalPostgresMtls(t *testing.T, contextDir string) Endpoint {
	t.Helper()

	return startPostgresBuild(t, contextDir, "databasus-test-postgres-physical-mtls")
}

// RestoreTarget is an idle restore-target container: PostgreSQL is not started on boot (the
// entrypoint sleeps), so the e2e test extracts the streamed bundle, runs pg_combinebackup and starts
// the cluster by hand inside it. The data dir lives on tmpfs and is discarded with the container.
type RestoreTarget struct {
	handle ContainerHandle
}

// StartPhysicalRestoreTarget builds (and caches via KeepImage) an idle restore-target image for
// image (e.g. "postgres:17"). zstd is added because the postgres image lacks the CLI; pg_combinebackup
// and pg_verifybackup ship with it. The build context (a two-line Dockerfile) is written to a temp
// dir so the helper needs no committed testdata.
func StartPhysicalRestoreTarget(t *testing.T, image string) RestoreTarget {
	t.Helper()

	contextDir := t.TempDir()
	dockerfile := fmt.Sprintf(
		"FROM %s\nRUN apt-get update && apt-get install -y --no-install-recommends zstd "+
			"&& rm -rf /var/lib/apt/lists/*\n",
		image,
	)
	if err := os.WriteFile(filepath.Join(contextDir, "Dockerfile"), []byte(dockerfile), 0o600); err != nil {
		t.Fatalf("failed to write restore-target Dockerfile: %v", err)
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:   contextDir,
			Repo:      "databasus-test-postgres-restore-target",
			Tag:       restoreTargetTag(image),
			KeepImage: true,
		},
		ExposedPorts: []string{postgresPort},
		Entrypoint:   []string{"sleep", "infinity"},
		Tmpfs:        map[string]string{"/restore": physicalDataDirTmpfsOptions},
		// postgres never starts on its own here, so a port/log wait would hang; confirm the
		// container is up by running a no-op exec instead.
		WaitingFor: wait.ForExec([]string{"true"}),
	}

	return RestoreTarget{handle: startContainer(t, req, postgresPort)}
}

// restoreTargetTag derives a stable per-version image tag so KeepImage caches 17 and 18 separately.
func restoreTargetTag(image string) string {
	if major := postgresMajorVersion(image); major > 0 {
		return fmt.Sprintf("pg%d", major)
	}

	return "latest"
}

func (rt RestoreTarget) Host() string { return rt.handle.Host }

// MappedPort is the host port published for the restored cluster's 5432. Docker publishes it at
// container start, but a connection only succeeds once pg_ctl has started the cluster.
func (rt RestoreTarget) MappedPort() int { return rt.handle.Port }

// Exec runs args in the container and fails the test on a non-zero exit, returning combined output.
func (rt RestoreTarget) Exec(t *testing.T, args ...string) []byte {
	t.Helper()

	out, code := rt.exec(args, "")
	if code != 0 {
		t.Fatalf("restore-target exec %v failed (exit %d):\n%s", args, code, out)
	}

	return out
}

// ExecAs runs args as user (e.g. "postgres") and fails the test on a non-zero exit.
func (rt RestoreTarget) ExecAs(t *testing.T, user string, args ...string) []byte {
	t.Helper()

	out, code := rt.exec(args, user)
	if code != 0 {
		t.Fatalf("restore-target exec --user %s %v failed (exit %d):\n%s", user, args, code, out)
	}

	return out
}

// ExecBestEffort runs args ignoring any error, returning combined output. For cleanup paths that run
// after the test has finished, where failing the test object is neither possible nor wanted.
func (rt RestoreTarget) ExecBestEffort(user string, args ...string) []byte {
	out, _ := rt.exec(args, user)

	return out
}

// CopyFileToContainer copies a host file into the container at mode (replaces docker cp).
func (rt RestoreTarget) CopyFileToContainer(t *testing.T, hostPath, containerPath string, mode int64) {
	t.Helper()

	if err := rt.handle.Container.CopyFileToContainer(
		context.Background(), hostPath, containerPath, mode,
	); err != nil {
		t.Fatalf("failed to copy %s into restore target: %v", hostPath, err)
	}
}

func (rt RestoreTarget) exec(args []string, user string) ([]byte, int) {
	options := []tcexec.ProcessOption{tcexec.Multiplexed()}
	if user != "" {
		options = append(options, tcexec.WithUser(user))
	}

	code, reader, err := rt.handle.Container.Exec(context.Background(), args, options...)
	if err != nil {
		return []byte(err.Error()), -1
	}

	out, _ := io.ReadAll(reader)

	return out, code
}
