package storage

// TestResult is the outcome of a connection test. Error is empty when the storage was reached.
// RecoveryValidationStatus is the saved storage's status after the test, nil for unsaved settings.
type TestResult struct {
	IsConnected              bool
	Error                    string
	RecoveryValidationStatus *RecoveryValidationStatus
}

func newTestResult(testErr error, status *RecoveryValidationStatus) TestResult {
	if testErr != nil {
		return TestResult{IsConnected: false, Error: testErr.Error(), RecoveryValidationStatus: status}
	}

	return TestResult{IsConnected: true, Error: "", RecoveryValidationStatus: status}
}
