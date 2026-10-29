## ADDED Requirements

### Requirement: A verification accepts only ignorable errors that say nothing about the backup

An ignorable error is an error that the PostgreSQL restore tool reported and skipped, and that still lets the restore count as successful. When the restore tool of a verification exits with an error, the verification agent SHALL count the restore as successful only if two conditions hold. First, the tool reports how many errors it skipped, and it listed exactly that many errors. Second, every listed error is an ignorable error of a verification: the `public` schema that already exists in the restore target, or an extension that the verification environment cannot create, together with the comment on that extension. The agent SHALL judge by the tool's whole error output, however long it is.

When the restore tool fails because the restore ran out of the job's disk budget, the agent SHALL report the failure without the tool's exit code, so the backend retries the verification instead of rejecting the backup, as before this change. Any other failure that is not made only of ignorable errors, a count that does not match the listed errors, or a missing count SHALL fail the verification with the tool's exit code.

A missing extension is an ignorable error of a verification because the verification environment does not carry the extensions of the source. A restore into the user's database does not treat it as ignorable, as the `database-tool-failure-output` spec states.

#### Scenario: Only extensions the environment lacks

- **WHEN** the restore tool of a verification exits with an error, reports 4 skipped errors and lists two extensions that are not available, each with the failed comment on it
- **THEN** the verification completes and collects the statistics of the restored database

#### Scenario: Only the existing public schema

- **WHEN** the restore tool of a verification exits with an error, reports 1 skipped error and lists only that the `public` schema already exists
- **THEN** the verification completes

#### Scenario: A missing extension that the data depends on

- **WHEN** the restore tool reports that an extension is not available, and then that a type from it and a table using that type do not exist
- **THEN** the verification fails with the tool's exit code

#### Scenario: The count does not match the listed errors

- **WHEN** the restore tool reports 3 skipped errors but lists only 2, both about an extension that is not available
- **THEN** the verification fails with the tool's exit code

#### Scenario: Ignorable errors that make the output long

- **WHEN** the restore tool lists more than 8 KiB of errors about extensions that are not available and reports their exact count
- **THEN** the verification completes

#### Scenario: A non-zero exit without a count

- **WHEN** the restore tool exits with an error and does not report how many errors it skipped
- **THEN** the verification fails with the tool's exit code

#### Scenario: The restore runs out of the job's disk budget

- **WHEN** the restore tool exits with an error that reports no space left on the device of the restore container
- **THEN** the agent reports the failure without the tool's exit code, and the backend retries the verification instead of rejecting the backup
