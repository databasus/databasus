package snapshot

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
)

type ImporterDeps struct {
	Server      internaldb.Connection
	ClientTools pgshared.ClientTools
	Logger      *slog.Logger
}

// Importer reads only a Databasus backup file and the source key. It needs neither the source
// instance nor its catalog.
type Importer struct {
	server      internaldb.Connection
	clientTools pgshared.ClientTools
	logger      *slog.Logger
}

func NewImporter(deps ImporterDeps) *Importer {
	return &Importer{server: deps.Server, clientTools: deps.ClientTools, logger: deps.Logger}
}

// Validate reads the whole file in bounded memory and returns its authenticated manifest.
func (i *Importer) Validate(path string, masterKey *crypto.MasterKey) (archive.Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return archive.Manifest{}, fmt.Errorf("open the Databasus backup file: %w", err)
	}
	defer func() { _ = file.Close() }()

	reader, err := archive.Open(file, masterKey)
	if err != nil {
		return archive.Manifest{}, err
	}

	if err := copyDiscarding(reader); err != nil {
		return archive.Manifest{}, err
	}

	return reader.GetManifest(), nil
}

// Import restores a validated file into a new staging database. A failure drops it again.
func (i *Importer) Import(ctx context.Context, path string, masterKey *crypto.MasterKey, stagingName string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open the Databasus backup file: %w", err)
	}
	defer func() { _ = file.Close() }()

	reader, err := archive.Open(file, masterKey)
	if err != nil {
		return err
	}

	if err := internaldb.CreateStagingDatabase(ctx, i.server, stagingName); err != nil {
		return err
	}

	err = internaldb.Restore(ctx, internaldb.RestoreSpec{
		ClientTools: i.clientTools,
		Connection:  i.server.WithDatabase(stagingName),
		Logger:      i.logger,
	}, reader)
	if err != nil {
		_ = internaldb.DropStagingDatabase(context.WithoutCancel(ctx), i.server, stagingName)

		return err
	}

	return nil
}

// Release drops a staging database a recovery no longer needs.
func (i *Importer) Release(ctx context.Context, stagingName string) error {
	return internaldb.DropStagingDatabase(ctx, i.server, stagingName)
}
