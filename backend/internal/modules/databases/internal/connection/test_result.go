package connection

import (
	"errors"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// TestResult is the outcome of a connection test. Reason classifies a failure where the engine can,
// such as a missing replication privilege; Error is the server's message, empty after a success.
type TestResult struct {
	IsConnected bool
	Reason      *pgshared.ConnectionErrorCode
	Error       string
}

func newTestResult(testErr error) TestResult {
	if testErr == nil {
		return TestResult{IsConnected: true, Reason: nil, Error: ""}
	}

	var connectionTestErr *pgshared.ConnectionTestError
	if errors.As(testErr, &connectionTestErr) {
		return TestResult{IsConnected: false, Reason: &connectionTestErr.Code, Error: testErr.Error()}
	}

	return TestResult{IsConnected: false, Reason: nil, Error: testErr.Error()}
}
