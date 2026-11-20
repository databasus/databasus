// Package snapshot produces a Databasus backup file from the internal database and imports one into
// a staging database. Production writes the encrypted file once into a private spool, so every
// destination receives the same snapshot without a second dump; plaintext never reaches the disk.
package snapshot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
)

const (
	privateDirMode  = 0o700
	privateFileMode = 0o600
	// diskHeadroom keeps free space for the rest of the instance after the spool file.
	diskHeadroom = 512 << 20
)

type ProducerDeps struct {
	Repositories Repositories
	Connection   internaldb.Connection
	ClientTools  pgshared.ClientTools
	MasterKey    *crypto.MasterKey
	Release      string
	Bundle       database.MigrationBundle
	SpoolDir     string
	Clock        clock.Source
	Logger       *slog.Logger
	InspectDisk  func(path string) (diskusage.Usage, error)
}

type Producer struct {
	snapshots   *snapshotRepository
	connection  internaldb.Connection
	clientTools pgshared.ClientTools
	masterKey   *crypto.MasterKey
	release     string
	bundle      database.MigrationBundle
	spoolDir    string
	clock       clock.Source
	logger      *slog.Logger
	inspectDisk func(path string) (diskusage.Usage, error)
}

func NewProducer(deps ProducerDeps) *Producer {
	return &Producer{
		snapshots:   deps.Repositories.snapshots,
		connection:  deps.Connection,
		clientTools: deps.ClientTools,
		masterKey:   deps.MasterKey,
		release:     deps.Release,
		bundle:      deps.Bundle,
		spoolDir:    deps.SpoolDir,
		clock:       deps.Clock,
		logger:      deps.Logger,
		inspectDisk: deps.InspectDisk,
	}
}

// SpooledArchive is an encrypted Databasus backup file waiting in the spool.
type SpooledArchive struct {
	Path      string
	SizeBytes int64
	Manifest  archive.Manifest
	Summary   archive.Summary
}

// Produce dumps the internal database into a new spool file named by backupID. On any failure the
// partial file is removed, so a spool never holds an archive without its END record.
func (p *Producer) Produce(ctx context.Context, backupID uuid.UUID) (SpooledArchive, error) {
	if err := os.MkdirAll(p.spoolDir, privateDirMode); err != nil {
		return SpooledArchive{}, fmt.Errorf("create the Databasus backup spool: %w", err)
	}

	if err := os.Chmod(p.spoolDir, privateDirMode); err != nil {
		return SpooledArchive{}, fmt.Errorf("restrict the Databasus backup spool: %w", err)
	}

	if err := p.checkFreeSpace(ctx); err != nil {
		return SpooledArchive{}, err
	}

	path := p.GetSpoolPath(backupID)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, privateFileMode)
	if err != nil {
		return SpooledArchive{}, fmt.Errorf("create the Databasus backup spool file: %w", err)
	}

	spooled, err := p.writeArchive(ctx, file, backupID)
	closeErr := file.Close()

	if err == nil && closeErr != nil {
		err = fmt.Errorf("close the Databasus backup spool file: %w", closeErr)
	}

	if err != nil {
		_ = os.Remove(path)

		return SpooledArchive{}, err
	}

	spooled.Path = path

	return spooled, nil
}

// GetSpoolDirectory holds only spool files, so a restart can remove the ones no backup needs.
func (p *Producer) GetSpoolDirectory() string {
	return p.spoolDir
}

// GetSpoolPath is where the spool file of a backup lives, so a restart can find or remove it.
func (p *Producer) GetSpoolPath(backupID uuid.UUID) string {
	return filepath.Join(p.spoolDir, backupID.String()+archive.FileExtension)
}

func (p *Producer) writeArchive(ctx context.Context, file *os.File, backupID uuid.UUID) (SpooledArchive, error) {
	var spooled SpooledArchive

	err := p.snapshots.withExportedSnapshot(ctx, func(exported database.ExportedSnapshot) error {
		manifest := archive.Manifest{
			FormatVersion: 0,
			BackupID:      backupID,
			SourceRelease: p.release,
			SnapshotAt:    p.clock.Now(),
			SchemaIdentity: database.SchemaIdentity{
				MigrationBundleDigest:    p.bundle.Digest,
				AppliedMigrationVersions: exported.AppliedMigrationVersions,
			},
		}

		writer, err := archive.NewWriter(file, p.masterKey, manifest)
		if err != nil {
			return err
		}

		err = internaldb.Dump(ctx, internaldb.DumpSpec{
			ClientTools: p.clientTools,
			Connection:  p.connection,
			SnapshotID:  exported.ID,
			Logger:      p.logger,
		}, writer)
		if err != nil {
			return err
		}

		summary, err := writer.Finish()
		if err != nil {
			return err
		}

		spooled = SpooledArchive{Path: "", SizeBytes: 0, Manifest: manifest, Summary: summary}

		return nil
	})
	if err != nil {
		return SpooledArchive{}, err
	}

	if err := file.Sync(); err != nil {
		return SpooledArchive{}, fmt.Errorf("sync the Databasus backup spool file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		return SpooledArchive{}, fmt.Errorf("stat the Databasus backup spool file: %w", err)
	}

	spooled.SizeBytes = info.Size()

	return spooled, nil
}

// checkFreeSpace compares the database's size with the free space, because the compressed archive
// is not larger than the database it dumps.
func (p *Producer) checkFreeSpace(ctx context.Context) error {
	databaseSizeBytes, err := p.snapshots.getDatabaseSizeBytes(ctx)
	if err != nil {
		return err
	}

	usage, err := p.inspectDisk(p.spoolDir)
	if err != nil {
		return err
	}

	neededBytes := int64(diskHeadroom) + databaseSizeBytes

	if usage.FreeBytes < neededBytes {
		p.logger.WarnContext(ctx, "too little free space for a Databasus backup",
			slog.Int64("free_bytes", usage.FreeBytes), slog.Int64("needed_bytes", neededBytes))

		return ErrInsufficientDiskSpace
	}

	return nil
}

// copyDiscarding is io.Copy into nowhere, for a full validation pass.
func copyDiscarding(reader io.Reader) error {
	_, err := io.Copy(io.Discard, reader)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}
