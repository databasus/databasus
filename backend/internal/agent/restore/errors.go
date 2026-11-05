package restore

import "errors"

// ErrRestoreFailed is the contract boundary the runner keys on: a non-zero
// pg_restore exit is terminal "backup rejected" (modulo the disk-ceiling
// probe), whereas any other RunPgRestore error is an exec-infrastructure
// failure with no usable exit code — a retryable agent-setup failure.
var ErrRestoreFailed = errors.New("pg_restore exited non-zero")
