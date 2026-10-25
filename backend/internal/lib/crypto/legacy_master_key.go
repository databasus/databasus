package crypto

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"databasus-backend/internal/lib/database"
)

type Deps struct {
	Transactor          *database.Transactor
	LegacyKeyRepository *LegacyMasterKeyRepository
	MasterKeyPath       string
	Logger              *slog.Logger
}

// MoveLegacyMasterKeyToFile writes a key that an old release kept in the database into the key file,
// replacing the file, and then deletes the key from the database. Such a row exists only on an
// installation whose release never wrote a key file, so a file next to it holds no key in use. If
// the deletion does not commit, the next start writes the same key again.
func MoveLegacyMasterKeyToFile(ctx context.Context, deps Deps) error {
	return deps.Transactor.Transaction(ctx, func(ctx context.Context) error {
		secret, isFound, err := deps.LegacyKeyRepository.SelectNonEmptyKeyForUpdate(ctx)
		if err != nil || !isFound {
			return err
		}

		err = replaceMasterKeyFile(deps.MasterKeyPath, secret)
		if err != nil {
			return err
		}

		err = deps.LegacyKeyRepository.DeleteKey(ctx, secret)
		if err != nil {
			return err
		}

		deps.Logger.InfoContext(ctx, "legacy master key moved to file", slog.String("path", deps.MasterKeyPath))

		return nil
	})
}

func replaceMasterKeyFile(path, secret string) error {
	temporaryPath, err := writeTemporaryKeyFile(path, secret)
	if err != nil {
		return err
	}

	err = os.Rename(temporaryPath, path)
	if err != nil {
		_ = os.Remove(temporaryPath)

		return fmt.Errorf("replace the master key file %s: %w", path, err)
	}

	return nil
}
