package verifications

import "databasus-backend/internal/modules/backups"

// ClaimCandidate pairs a locked PENDING verification with its backup, so the claim checks the
// disk fit without reading the backup again.
type ClaimCandidate struct {
	Verification *Verification
	Backup       backups.LogicalBackup
}
