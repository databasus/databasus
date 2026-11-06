-- +goose Up
ALTER TABLE verification_runs
    ADD COLUMN failure_diagnostic_output TEXT;

ALTER TABLE verification_runs
    ADD COLUMN is_failure_diagnostic_truncated BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE verification_runs
    ADD CONSTRAINT ck_verification_runs_failure_diagnostic
    CHECK (
        (failure_diagnostic_output IS NULL AND NOT is_failure_diagnostic_truncated)
        OR (failure_diagnostic_output IS NOT NULL AND octet_length(failure_diagnostic_output) BETWEEN 1 AND 65536)
    );

-- +goose Down
ALTER TABLE verification_runs DROP CONSTRAINT ck_verification_runs_failure_diagnostic;

ALTER TABLE verification_runs DROP COLUMN is_failure_diagnostic_truncated;

ALTER TABLE verification_runs DROP COLUMN failure_diagnostic_output;
