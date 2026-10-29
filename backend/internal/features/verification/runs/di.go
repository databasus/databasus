package verification_runs

import "databasus-backend/internal/util/logger"

func GetVerificationAgentController() *VerificationAgentController {
	return &VerificationAgentController{logger: logger.GetLogger()}
}
