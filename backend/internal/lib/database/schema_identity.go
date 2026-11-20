package database

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// SchemaIdentity names the schema a database has: the digest of the migration bundle of the release
// that wrote it and the migration versions in effect. Two databases with equal identities have the
// same schema, so a snapshot can be restored only into a release whose bundle it matches.
type SchemaIdentity struct {
	MigrationBundleDigest    string
	AppliedMigrationVersions []int64
}

// MigrationBundle is the set of migrations a binary carries.
type MigrationBundle struct {
	Digest   string
	Versions []int64
}

// ReadMigrationBundle hashes every migration file in name order. Each file contributes its name's
// UTF-8 bytes prefixed by their length as an unsigned 32-bit big-endian number, then its content
// prefixed by its length as an unsigned 64-bit big-endian number. The digest is lowercase hex.
func ReadMigrationBundle(files fs.FS) (MigrationBundle, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return MigrationBundle{}, fmt.Errorf("list the migrations: %w", err)
	}

	slices.SortFunc(names, func(left, right string) int { return bytes.Compare([]byte(left), []byte(right)) })

	digest := sha256.New()
	versions := make([]int64, 0, len(names))

	for _, name := range names {
		content, err := fs.ReadFile(files, name)
		if err != nil {
			return MigrationBundle{}, fmt.Errorf("read the migration %s: %w", name, err)
		}

		_ = binary.Write(digest, binary.BigEndian, uint32(len(name)))
		_, _ = digest.Write([]byte(name))
		_ = binary.Write(digest, binary.BigEndian, uint64(len(content)))
		_, _ = digest.Write(content)

		version, err := parseMigrationVersion(name)
		if err != nil {
			return MigrationBundle{}, err
		}

		versions = append(versions, version)
	}

	slices.Sort(versions)

	return MigrationBundle{Digest: hex.EncodeToString(digest.Sum(nil)), Versions: slices.Compact(versions)}, nil
}

// ReadAppliedMigrationVersions reads goose's history in the caller's transaction. For each version
// the newest history row decides, and the version counts only when that row says applied.
func ReadAppliedMigrationVersions(ctx context.Context, db *gorm.DB) ([]int64, error) {
	versions := []int64{}

	err := Conn(ctx, db).Raw(`SELECT version_id FROM (
			SELECT DISTINCT ON (version_id) version_id, is_applied
			FROM goose_db_version
			WHERE version_id > 0
			ORDER BY version_id, id DESC
		) newest
		WHERE is_applied
		ORDER BY version_id`).Scan(&versions).Error
	if err != nil {
		return nil, fmt.Errorf("read the applied migrations: %w", err)
	}

	return versions, nil
}

// CheckCompatibility refuses a snapshot this bundle cannot run: another bundle digest, a carried
// migration the snapshot lacks, or an applied version newer than the bundle's newest. Older applied
// versions the bundle no longer carries are accepted, because releases have deleted migrations.
func (b MigrationBundle) CheckCompatibility(snapshot SchemaIdentity) error {
	if snapshot.MigrationBundleDigest != b.Digest {
		return fmt.Errorf("%w: the migration bundles differ", ErrSchemaIncompatible)
	}

	for _, version := range b.Versions {
		if !slices.Contains(snapshot.AppliedMigrationVersions, version) {
			return fmt.Errorf("%w: migration %d is not applied in the snapshot", ErrSchemaIncompatible, version)
		}
	}

	if len(b.Versions) == 0 {
		return nil
	}

	newestVersion := b.Versions[len(b.Versions)-1]
	for _, version := range snapshot.AppliedMigrationVersions {
		if version > newestVersion {
			return fmt.Errorf("%w: migration %d is newer than this release", ErrSchemaIncompatible, version)
		}
	}

	return nil
}

// CheckNotNewer refuses applied versions newer than the bundle's newest, the same refusal Migrate
// makes before an upgrade, without touching the database they were read from.
func (b MigrationBundle) CheckNotNewer(appliedVersions []int64) error {
	if len(b.Versions) == 0 || len(appliedVersions) == 0 {
		return nil
	}

	databaseVersion := slices.Max(appliedVersions)
	binaryVersion := b.Versions[len(b.Versions)-1]

	if databaseVersion > binaryVersion {
		return fmt.Errorf(
			"%w: the newest migration in the database is %d, the newest one this version carries is %d",
			ErrSchemaNewerThanBinary, databaseVersion, binaryVersion,
		)
	}

	return nil
}

func parseMigrationVersion(name string) (int64, error) {
	prefix, _, isFound := strings.Cut(name, "_")
	if !isFound {
		return 0, fmt.Errorf("%w: %s", ErrMigrationNameInvalid, name)
	}

	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrMigrationNameInvalid, name)
	}

	return version, nil
}
