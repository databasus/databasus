// Package providers declares what a storage provider is and keeps the registry that maps a storage
// type to its provider. Each provider lives in a subpackage with its settings, its table and its
// file operations.
package providers

import (
	"context"
	"io"
	"log/slog"

	"github.com/google/uuid"
)

// Settings are the provider-specific settings of one storage. Secrets in settings that a repository
// loaded stay encrypted; only the file operations and the connection test decrypt them.
type Settings interface {
	GetType() Type
}

type Provider interface {
	GetType() Type
	// Validate answers an apierror problem naming the setting it refuses.
	Validate(settings Settings) error
	// MergeStoredSettings applies the provider's secret selection and retention policy and
	// refuses a change of a setting fixed at creation, such as the S3 prefix. A secret tied to the
	// provider's resource address is filled only while that address stays the stored one, so a user who
	// may not see a secret cannot send it to a server of their choice. Left empty, the secret then gets
	// the refusal of Validate when the settings use it, and is removed when they do not.
	MergeStoredSettings(merge SettingsMerge) (Settings, error)
	SaveFile(ctx context.Context, file FileAccess, content io.Reader) error
	// OpenFile returns a reader that may keep fetching from the provider while it is read, so ctx
	// must outlive the stream, not just the call.
	OpenFile(ctx context.Context, file FileAccess) (io.ReadCloser, error)
	// DeleteFile succeeds only on a confirmed absence of every object derived from the name, and a
	// missing file counts as absent.
	DeleteFile(ctx context.Context, file FileAccess) error
	TestConnection(ctx context.Context, location Location) error
	GetRepository() SettingsRepository
}

// SettingsMerge names both sides, so a swapped call cannot keep the stored settings over the edit.
type SettingsMerge struct {
	Incoming Settings
	Stored   Settings
}

// Location is where a provider works: the storage's ID keys the session pools, and the settings carry
// the encrypted secrets.
type Location struct {
	StorageID uuid.UUID
	Settings  Settings
}

type FileAccess struct {
	Location
	FileName string
	Logger   *slog.Logger
}

// SettingsRepository keeps the settings of one provider in its own table. Upsert encrypts the secrets
// that are not encrypted yet.
type SettingsRepository interface {
	Upsert(ctx context.Context, storageID uuid.UUID, settings Settings) error
	// ListByStorageIDs leaves out the IDs that have no row.
	ListByStorageIDs(ctx context.Context, storageIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
}

// LocalFolders are the folders a local storage writes through: a file is staged in TempFolder and
// moved into DataFolder once it is complete.
type LocalFolders struct {
	TempFolder string
	DataFolder string
}

// CastSettings returns settings as the settings type S of one provider.
func CastSettings[S Settings](settings Settings) (S, error) {
	typedSettings, isOfType := settings.(S)
	if !isOfType {
		return typedSettings, ErrSettingsOfAnotherType
	}

	return typedSettings, nil
}
