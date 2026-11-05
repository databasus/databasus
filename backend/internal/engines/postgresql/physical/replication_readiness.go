package physical

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// ReplicationCheck is what the backup type and the stored cluster ask of the source beyond a
// working replication connection.
type ReplicationCheck struct {
	IsWalSummaryRequired  bool
	IsWalRotationRequired bool
	// ExpectedSystemIdentifier refuses another cluster at the same address; empty skips the check.
	ExpectedSystemIdentifier string
}

type replicationSettings struct {
	WalLevel            string
	SummarizeWal        string
	MaxWalSenders       int
	MaxReplicationSlots int
}

// TestReplicationConnection checks that the cluster can be backed up physically. The
// summarize_wal=on gate and the custom-tablespace refusal are required by the spec
// physical-backups/backup-method. A refusal is a *pgshared.ConnectionTestError with its code.
func (s *Source) TestReplicationConnection(ctx context.Context, logger *slog.Logger, check ReplicationCheck) error {
	replicationConn, err := s.connectReplication(ctx)
	if err != nil {
		return classifyReplicationConnectError(err, s.settings.Username)
	}
	defer func() {
		if err := replicationConn.Close(ctx); err != nil {
			logger.WarnContext(ctx, "failed to close replication connection", slog.Any("error", err))
		}
	}()

	// pg_replication_slots is not in the replication command set, so the inspection queries need an
	// ordinary connection.
	conn, err := s.connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to open inspection connection: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	if err := checkNoCustomTablespaces(ctx, conn); err != nil {
		return err
	}

	if err := checkReplicationSettings(ctx, conn, check); err != nil {
		return err
	}

	// Continuous streaming promises a recovery point between backups, and only a finalized segment
	// is uploaded. Without pg_switch_wal a quiet source leaves its newest WAL local until the segment
	// fills, so the promise cannot be kept.
	if check.IsWalRotationRequired {
		if err := s.checkCanForceWalRotation(ctx, conn); err != nil {
			return err
		}
	}

	if check.ExpectedSystemIdentifier != "" {
		return checkSystemIdentifier(ctx, conn, check.ExpectedSystemIdentifier)
	}

	return nil
}

func checkReplicationSettings(ctx context.Context, conn *pgx.Conn, check ReplicationCheck) error {
	settings, err := readReplicationSettings(ctx, conn)
	if err != nil {
		return err
	}

	if settings.WalLevel != "replica" && settings.WalLevel != "logical" {
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrWalLevelInvalid, Message: ""}
	}

	if settings.MaxWalSenders == 0 {
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrNoWalSenders, Message: ""}
	}

	if settings.MaxReplicationSlots == 0 {
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrNoReplicationSlots, Message: ""}
	}

	if check.IsWalSummaryRequired && settings.SummarizeWal != "on" {
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrWalSummaryDisabled, Message: ""}
	}

	return nil
}

func (s *Source) checkCanForceWalRotation(ctx context.Context, conn *pgx.Conn) error {
	canRotate, err := canForceWalRotation(ctx, conn, s.settings.Username)
	if err != nil {
		return err
	}

	if canRotate {
		return nil
	}

	quotedUsername := pgx.Identifier{s.settings.Username}.Sanitize()

	return &pgshared.ConnectionTestError{
		Code: pgshared.ConnErrNoWalSwitchPrivilege,
		Message: fmt.Sprintf(
			"user %s cannot force a WAL segment switch, so the recovery point between backups "+
				"would advance only as segments fill. Run GRANT EXECUTE ON FUNCTION pg_switch_wal() TO %s "+
				"on the source, or choose a backup type that does not replay archived WAL",
			quotedUsername, quotedUsername,
		),
	}
}

func checkSystemIdentifier(ctx context.Context, conn *pgx.Conn, expectedSystemIdentifier string) error {
	var currentSystemIdentifier string

	if err := conn.QueryRow(ctx, "SELECT system_identifier::text FROM pg_control_system()").
		Scan(&currentSystemIdentifier); err != nil {
		return fmt.Errorf("failed to read cluster system_identifier: %w", err)
	}

	if currentSystemIdentifier != expectedSystemIdentifier {
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrSystemIdentifierMismatch, Message: ""}
	}

	return nil
}

// checkNoCustomTablespaces enforces the spec physical-backups/backup-method: physical backups refuse
// any cluster that has tablespaces outside pg_default / pg_global, because pg_basebackup
// --pgdata=- -F tar cannot multiplex multi-tablespace output into a single stdout stream.
func checkNoCustomTablespaces(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, `
		SELECT spcname
		FROM pg_tablespace
		WHERE spcname NOT IN ('pg_default', 'pg_global')
		ORDER BY spcname
	`)
	if err != nil {
		return fmt.Errorf("query pg_tablespace: %w", err)
	}

	tablespaceNames, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("iterate pg_tablespace: %w", err)
	}

	if len(tablespaceNames) > 0 {
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrCustomTablespaces, Message: ""}
	}

	return nil
}

func readReplicationSettings(ctx context.Context, conn *pgx.Conn) (replicationSettings, error) {
	var settings replicationSettings

	if err := conn.QueryRow(ctx, `
		SELECT
			(SELECT setting FROM pg_settings WHERE name = 'wal_level'),
			COALESCE((SELECT setting FROM pg_settings WHERE name = 'summarize_wal'), 'off'),
			(SELECT setting::int FROM pg_settings WHERE name = 'max_wal_senders'),
			(SELECT setting::int FROM pg_settings WHERE name = 'max_replication_slots')
	`).Scan(
		&settings.WalLevel,
		&settings.SummarizeWal,
		&settings.MaxWalSenders,
		&settings.MaxReplicationSlots,
	); err != nil {
		return replicationSettings{}, fmt.Errorf("failed to read replication settings: %w", err)
	}

	return settings, nil
}

func classifyReplicationConnectError(err error, username string) error {
	switch classifyServerError(err) {
	case serverErrorNoHbaEntry:
		return &pgshared.ConnectionTestError{
			Code: pgshared.ConnErrPgHbaNoEntry,
			Message: fmt.Sprintf(
				"pg_hba.conf has no replication entry for user %q. "+
					"Add \"host replication %s all scram-sha-256\" to pg_hba.conf and reload PostgreSQL",
				username, username,
			),
		}
	case serverErrorBadPassword:
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrBadCredentials, Message: ""}
	case serverErrorNoReplicationPrivilege:
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrNoReplicationPrivilege, Message: ""}
	case serverErrorOther, serverErrorNameTaken, serverErrorObjectMissing, serverErrorPermissionDenied:
		return &pgshared.ConnectionTestError{Code: pgshared.ConnErrConnectionFailed, Message: ""}
	}

	return &pgshared.ConnectionTestError{Code: pgshared.ConnErrConnectionFailed, Message: ""}
}
