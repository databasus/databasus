package notifier

// TestResult is the outcome of a test send. Error is empty when the channel accepted the message.
type TestResult struct {
	IsDelivered bool
	Error       string
}
