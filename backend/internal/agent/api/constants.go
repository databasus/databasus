package api

import "time"

const (
	versionPath                 = "/api/v1/system/version"
	verificationAgentBinaryPath = "/api/v1/system/verification-agent"

	apiCallTimeout   = 30 * time.Second
	maxRetryAttempts = 3
	retryBaseDelay   = 1 * time.Second

	maxBackoff = 32 * time.Second

	agentVersionHeader = "X-Databasus-Version"
)
