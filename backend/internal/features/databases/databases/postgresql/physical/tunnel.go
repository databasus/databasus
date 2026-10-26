package postgresql_physical

import (
	"context"
	"log/slog"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

type OpenTunnelSpec struct {
	Database  *PostgresqlPhysicalDatabase
	Logger    *slog.Logger
	Encryptor encryption.FieldEncryptor
}

// The tunnel belongs to the operation, not to the model: a base backup runs for hours and a WAL
// streamer for weeks, both over connections that outlive any single method down there.
type TunneledDatabase struct {
	original              *PostgresqlPhysicalDatabase
	databaseThroughTunnel *PostgresqlPhysicalDatabase
	tunnel                *pgphysical.Tunnel
}

// Ctx bounds opening the tunnel; the forwarder then lives until Close. With the tunnel disabled
// this still returns a usable wrapper around the original, so callers never branch.
func OpenTunnel(ctx context.Context, spec OpenTunnelSpec) (*TunneledDatabase, error) {
	if spec.Database != nil {
		if err := spec.Database.ValidateNotEmbeddedTarget(); err != nil {
			return nil, err
		}
	}

	if spec.Database == nil || !spec.Database.SshTunnel.IsEnabled {
		return &TunneledDatabase{original: spec.Database, databaseThroughTunnel: spec.Database}, nil
	}

	source, err := spec.Database.newSource(spec.Encryptor)
	if err != nil {
		return nil, err
	}

	tunnel, err := source.OpenTunnel(ctx, getTunnelLogger(spec.Logger))
	if err != nil {
		return nil, err
	}

	localEndpoint, _ := tunnel.GetLocalEndpoint()

	// The struct copy is what keeps ID, DatabaseID and ReplicationSlotName intact, and both slot
	// families are named from those. Reaching for CopyForNewDatabase here would blank them and mint
	// a fresh slot on the source cluster on every reconnect.
	databaseThroughTunnel := *spec.Database
	// Host stays the real one: sslmode=verify-full matches the certificate against it, and libpq
	// looks up .pgpass by host and port.
	databaseThroughTunnel.Port = localEndpoint.Port
	databaseThroughTunnel.LocalTunnelEndpoint = &localEndpoint
	// The copy describes an already forwarded connection; leaving this on would make anything that
	// re-reads it open a second tunnel pointed at the first one's local port.
	databaseThroughTunnel.SshTunnel.IsEnabled = false

	return &TunneledDatabase{
		original:              spec.Database,
		databaseThroughTunnel: &databaseThroughTunnel,
		tunnel:                tunnel,
	}, nil
}

func (t *TunneledDatabase) GetDatabaseThroughTunnel() *PostgresqlPhysicalDatabase {
	if t == nil {
		return nil
	}

	return t.databaseThroughTunnel
}

// PopulateDbData fills all three from the cluster, into the model it was handed — which is the
// copy. Carrying back only Version would persist a row with a NULL system_identifier, and
// SystemIdentifierUint64 would then write 0 into every backup manifest.
func (t *TunneledDatabase) CopyDiscoveredMetadataToOriginal() {
	if t == nil || t.original == nil || t.databaseThroughTunnel == nil {
		return
	}

	t.original.Version = t.databaseThroughTunnel.Version
	t.original.SystemIdentifier = t.databaseThroughTunnel.SystemIdentifier
	t.original.WalSegmentSizeBytes = t.databaseThroughTunnel.WalSegmentSizeBytes
}

// No forwarder means no bastion between us and the cluster, so nothing about the transport can be
// at fault and the caller's failure is the source's own.
func (t *TunneledDatabase) IsBastionReachable(ctx context.Context) bool {
	if t == nil || t.tunnel == nil {
		return true
	}

	return t.tunnel.IsBastionReachable(ctx)
}

func (t *TunneledDatabase) Close() {
	if t == nil || t.tunnel == nil {
		return
	}

	t.tunnel.Close()
}

// The old callers and tests open a tunnel without a logger, and lib/sshtunnel requires one.
func getTunnelLogger(tunnelLogger *slog.Logger) *slog.Logger {
	if tunnelLogger == nil {
		return logger.GetLogger()
	}

	return tunnelLogger
}
