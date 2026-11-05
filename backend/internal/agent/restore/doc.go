// Package restore drives pg_restore inside the job's ephemeral container. It
// never imports the Docker SDK: it works through the ExecRunner interface so
// the runner's exit-code contract is unit-testable with a fake.
package restore
