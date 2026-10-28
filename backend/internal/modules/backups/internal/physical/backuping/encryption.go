package backuping

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// setupEncryption wraps the storage-bound writer in the backup encryption writer
// when encryption is enabled. It returns the writer the stream should tee into,
// the EncryptingWriter to Close on teardown (nil when disabled), and the fresh
// salt/nonce (base64) to persist. When encryption is off it returns base
// unchanged with empty salt/nonce.
func setupEncryption(
	base io.Writer,
	enc catalog.BackupEncryption,
	masterKey *crypto.MasterKey,
	backupID uuid.UUID,
) (io.Writer, *crypto.EncryptingWriter, string, string, error) {
	if enc != catalog.BackupEncryptionEncrypted {
		return base, nil, "", "", nil
	}

	encryptingWriter, header, err := crypto.NewEncryptingWriter(base, masterKey, backupID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("setup encryption: %w", err)
	}

	return encryptingWriter, encryptingWriter, base64.StdEncoding.EncodeToString(header.Salt),
		base64.StdEncoding.EncodeToString(header.Nonce), nil
}
