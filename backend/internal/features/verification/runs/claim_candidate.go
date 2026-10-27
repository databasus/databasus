package verification_runs

import "databasus-backend/internal/modules/backups"

// ClaimCandidate pairs a locked PENDING verification with its backup, so the claim checks the
// disk fit without reading the backup again.
type ClaimCandidate struct {
	Verification *RestoreVerification
	Backup       backups.LogicalBackup
}
