package storage

// TestResult is the outcome of a connection test. Error is empty when the storage was reached.
type TestResult struct {
	IsConnected bool
	Error       string
}

func newTestResult(testErr error) TestResult {
	if testErr != nil {
		return TestResult{IsConnected: false, Error: testErr.Error()}
	}

	return TestResult{IsConnected: true, Error: ""}
}
