package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
	"databasus-backend/internal/modules/instancebackups/internal/snapshot"
)

type PreparerDeps struct {
	Server      internaldb.Connection
	ClientTools pgshared.ClientTools
	Release     string
	Bundle      database.MigrationBundle
	// Normalizers end the other modules' recovered work in the staging database.
	Normalizers []StateNormalizer
	Clock       clock.Source
	Logger      *slog.Logger
}

// Preparer turns the staged archive of an operation into a staging database marked with that
// recovery. Startup and upload recoveries share it, so both accept exactly the same archives.
type Preparer struct {
	server      internaldb.Connection
	release     string
	bundle      database.MigrationBundle
	normalizers []StateNormalizer
	clock       clock.Source
	importer    *snapshot.Importer
}

func NewPreparer(deps PreparerDeps) *Preparer {
	return &Preparer{
		server:      deps.Server,
		release:     deps.Release,
		bundle:      deps.Bundle,
		normalizers: deps.Normalizers,
		clock:       deps.Clock,
		importer: snapshot.NewImporter(snapshot.ImporterDeps{
			Server:      deps.Server,
			ClientTools: deps.ClientTools,
			Logger:      deps.Logger,
		}),
	}
}

// Prepare authenticates the archive at archivePath with the source key, checks that this release
// can serve it, imports it into the operation's staging database, normalizes it and marks it with
// the recovery. It fills the operation's authenticated fields and leaves writing the journal to the
// caller. A failure is a RefusalError; the caller discards what was staged.
func (p *Preparer) Prepare(
	ctx context.Context,
	operation *Operation,
	archivePath string,
	masterKey *crypto.MasterKey,
) error {
	manifest, err := p.importer.Validate(archivePath, masterKey)
	if err != nil {
		return newArchiveRefusal(err)
	}

	if err := p.checkCompatibility(manifest); err != nil {
		return err
	}

	if err := p.importer.Import(ctx, archivePath, masterKey, operation.StagingDatabase); err != nil {
		return newImportRefusal(err)
	}

	if err := p.normalizeStaging(ctx, operation, manifest); err != nil {
		return newImportRefusal(err)
	}

	snapshotAt := manifest.SnapshotAt
	operation.SourceRelease = manifest.SourceRelease
	operation.SnapshotAt = &snapshotAt

	return nil
}

func (p *Preparer) checkCompatibility(manifest archive.Manifest) error {
	if manifest.SourceRelease != p.release {
		return &RefusalError{
			Reason: RefusalIncompatibleRelease,
			Explanation: fmt.Sprintf("the Databasus backup was made by release %s, and this is release %s",
				manifest.SourceRelease, p.release),
			NextStep: fmt.Sprintf("Start Databasus %s with these files, recover there, then upgrade normally",
				manifest.SourceRelease),
			Cause: nil,
		}
	}

	if err := p.bundle.CheckCompatibility(manifest.SchemaIdentity); err != nil {
		return &RefusalError{
			Reason: RefusalIncompatibleSchema,
			Explanation: fmt.Sprintf("the Databasus backup's schema does not match release %s, though both report it",
				p.release),
			NextStep: "Start the exact image that made the Databasus backup with these files, " +
				"recover there, then upgrade normally",
			Cause: err,
		}
	}

	return nil
}

// normalizeStaging checks the imported migration history against the authenticated snapshot, ends
// the work the lost instance had running and marks the database with the recovery that produced
// it, so a resumed publication recognizes it. The recovered installation runs from its first start.
func (p *Preparer) normalizeStaging(ctx context.Context, operation *Operation, manifest archive.Manifest) error {
	staging, err := openDatabaseRepository(ctx, p.server.WithDatabase(operation.StagingDatabase))
	if err != nil {
		return err
	}
	defer staging.close()

	importedVersions, err := staging.readAppliedVersions(ctx)
	if err != nil {
		return err
	}

	if !slices.Equal(importedVersions, manifest.SchemaIdentity.AppliedMigrationVersions) {
		return fmt.Errorf("%w: its migration history differs from the snapshot's", ErrStagedStateInvalid)
	}

	discardedCount, err := staging.normalize(ctx, p.normalizers, p.clock.Now())
	if err != nil {
		return err
	}

	operation.DiscardedCleanupCount = discardedCount

	return staging.markRecovered(ctx, operation.ID, p.clock.Now())
}
