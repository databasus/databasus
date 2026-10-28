package physical

import "databasus-backend/internal/modules/backups/internal/physical/restorestream"

func (s *Service) GetRecoveryScript() string {
	return restorestream.GetRecoveryScript()
}
