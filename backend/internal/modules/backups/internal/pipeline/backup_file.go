package pipeline

import (
	"context"
	"fmt"
	"io"
)

// OpenBackupFile opens the stored file of a backup and decrypts it, so the reader returns the file
// in its stored format: the dump of PostgreSQL and MongoDB, and the zstd-compressed dump of MySQL and
// MariaDB.
func (p *Pipeline) OpenBackupFile(ctx context.Context, backup StoredBackup) (io.ReadCloser, error) {
	storageReader, err := p.storages.GetStorageReader(ctx, backup.StorageID)
	if err != nil {
		return nil, fmt.Errorf("open the backup's storage: %w", err)
	}

	backupFile, err := storageReader.OpenFile(ctx, backup.FileName)
	if err != nil {
		return nil, fmt.Errorf("open the backup file: %w", err)
	}

	decryptedStream, err := p.decryptIfEncrypted(backupFile, backup)
	if err != nil {
		_ = backupFile.Close()

		return nil, err
	}

	return &decryptedBackupFile{Reader: decryptedStream, storedFile: backupFile}, nil
}

type decryptedBackupFile struct {
	io.Reader

	storedFile io.Closer
}

func (f *decryptedBackupFile) Close() error {
	return f.storedFile.Close()
}
