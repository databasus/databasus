package mongodb

import (
	"context"
	"log/slog"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

type OpenTunnelSpec struct {
	Database  *MongodbDatabase
	Logger    *slog.Logger
	Encryptor encryption.FieldEncryptor
}

// The tunnel belongs to the operation, not to the model: every method of the model connects on its
// own, so no method down there could own a Close.
type TunneledDatabase struct {
	original              *MongodbDatabase
	databaseThroughTunnel *MongodbDatabase
	tunnel                *mongodb_engine.Tunnel
}

// Ctx bounds opening the tunnel; the forwarder then lives until Close. With the tunnel disabled
// this still returns a usable wrapper around the original, so callers never branch.
func OpenTunnel(ctx context.Context, spec OpenTunnelSpec) (*TunneledDatabase, error) {
	if spec.Database == nil || !spec.Database.SshTunnel.IsEnabled {
		return &TunneledDatabase{original: spec.Database, databaseThroughTunnel: spec.Database}, nil
	}

	// Restore targets arrive as bare models straight from the request DTO, so Validate has not
	// necessarily run and the port may still be unset; the engine refuses a tunnel without one.
	source, err := spec.Database.newSource(spec.Encryptor)
	if err != nil {
		return nil, err
	}

	tunnel, err := source.OpenTunnel(ctx, getTunnelLogger(spec.Logger))
	if err != nil {
		return nil, err
	}

	localEndpoint, _ := tunnel.GetLocalEndpoint()

	databaseThroughTunnel := *spec.Database
	databaseThroughTunnel.Host = localEndpoint.Host
	// A fresh pointer, because the shallow copy shares the original's and the original is what gets
	// persisted.
	databaseThroughTunnel.Port = new(localEndpoint.Port)
	// Only the copy is pinned to a direct connection, so the stored config keeps whatever failover
	// behaviour the user chose.
	databaseThroughTunnel.IsDirectConnection = true
	// The copy describes an already forwarded connection; leaving this on would make anything that
	// re-reads it open a second tunnel pointed at the first one's local port.
	databaseThroughTunnel.SshTunnel.IsEnabled = false

	return &TunneledDatabase{
		original:              spec.Database,
		databaseThroughTunnel: &databaseThroughTunnel,
		tunnel:                tunnel,
	}, nil
}

func (t *TunneledDatabase) GetDatabaseThroughTunnel() *MongodbDatabase {
	if t == nil {
		return nil
	}

	return t.databaseThroughTunnel
}

// TestConnection and PopulateVersion write the detected version into the model they were handed,
// which is the copy. Callers that persist the result need it carried back.
func (t *TunneledDatabase) CopyDiscoveredMetadataToOriginal() {
	if t == nil || t.original == nil || t.databaseThroughTunnel == nil {
		return
	}

	t.original.Version = t.databaseThroughTunnel.Version
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
