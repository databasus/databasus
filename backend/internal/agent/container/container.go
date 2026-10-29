package container

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/restore"
)

const (
	containerTerminateTimeout = 30 * time.Second

	restoreUser = "postgres"
	restoreDB   = "postgres"
)

// PostgresContainer exposes two distinct Postgres conns. pg_restore runs
// INSIDE the container and reaches the DB on the container's own loopback
// (GetInContainerConn). The agent-process verifier runs OUTSIDE the container
// and reaches the DB via the random 127.0.0.1 host port the container's 5432
// was published to (GetVerifierConn) — populated by Manager.Spawn after the
// container starts.
type PostgresContainer struct {
	engine    *DockerEngine
	id        string
	networkID string
	hostPort  int
	password  string
	log       *slog.Logger
}

func (c *PostgresContainer) Exec(
	ctx context.Context, cmd []string, stdin io.Reader, env []string,
) (restore.ExecResult, error) {
	return c.engine.Exec(ctx, c.id, cmd, stdin, env)
}

func (c *PostgresContainer) GetInContainerConn() dbconn.Conn {
	return dbconn.Conn{
		Host: "127.0.0.1", Port: pgInternalPort,
		User: restoreUser, Password: c.password, Database: restoreDB,
	}
}

func (c *PostgresContainer) GetVerifierConn() dbconn.Conn {
	return dbconn.Conn{
		Host: "127.0.0.1", Port: c.hostPort,
		User: restoreUser, Password: c.password, Database: restoreDB,
	}
}

// GetDiskUsageBytes sums the byte footprint of the dirs the restore writes
// into: PGDATA (an anonymous volume declared by the postgres image, NOT in
// SizeRw) plus /tmp (where the agent stages the dump). Daemon-reported SizeRw
// alone misses PGDATA entirely; an in-container `du -sb` does not.
func (c *PostgresContainer) GetDiskUsageBytes(ctx context.Context) (int64, error) {
	res, err := c.engine.Exec(ctx, c.id,
		[]string{"du", "-sb", "/var/lib/postgresql", "/tmp"}, nil, nil)
	if err != nil {
		return 0, fmt.Errorf("exec du: %w", err)
	}

	if res.ExitCode != 0 {
		return 0, fmt.Errorf("du exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	var total int64
	for line := range strings.SplitSeq(strings.TrimRight(res.Stdout, "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		n, parseErr := strconv.ParseInt(fields[0], 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("parse du line %q: %w", line, parseErr)
		}

		total += n
	}

	return total, nil
}

func (c *PostgresContainer) Terminate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), containerTerminateTimeout)
	defer cancel()

	rmErr := c.engine.RemoveContainer(ctx, c.id)

	if c.networkID != "" {
		if netErr := c.engine.RemoveNetwork(ctx, c.networkID); netErr != nil && rmErr == nil {
			rmErr = netErr
		}
	}

	return rmErr
}
