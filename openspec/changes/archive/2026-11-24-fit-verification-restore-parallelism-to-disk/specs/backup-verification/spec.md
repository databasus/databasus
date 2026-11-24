## ADDED Requirements

### Requirement: A verification restores with the most parallel jobs that fit the agent's free disk

When the server assigns a verification to an agent, the assignment SHALL carry a number of parallel restore jobs and a disk budget. The disk budget is the disk the verification may use on its agent. The restore peak for N jobs is the most disk the restore is expected to occupy at once. It SHALL be the size of the backup file, plus the estimated database size, plus a fixed reserve of 1536 MB for the write-ahead log and the empty database cluster, plus the sizes of the N largest indexes the backup recorded, rounded up to a whole megabyte. When the backup recorded fewer than N indexes, the restore peak SHALL count all of them.

The per-job CPU budget of an agent SHALL be its CPU capacity divided by its number of concurrent jobs, rounded down, and SHALL be at least 1, also when the agent reports zero CPUs or zero concurrent jobs. The agent's free disk SHALL be its disk capacity in gigabytes multiplied by 1024, in megabytes, minus the disk budgets of the verifications it runs.

The number of jobs SHALL be the largest N from 1 up to the per-job CPU budget, and at most 8, whose restore peak is no more than the agent's free disk. A backup made with TimescaleDB SHALL get 1 job. The disk budget of the assignment SHALL be the restore peak for the chosen number of jobs.

When the restore peak for 1 job is more than the agent's free disk, the server SHALL NOT assign that verification to that agent. The verification SHALL stay pending with its attempt count unchanged, and the server SHALL consider the next pending verification for the same claim.

The agent SHALL restore with exactly the number of jobs the assignment names, and SHALL stop the restore when it reaches the assignment's disk budget.

#### Scenario: All of the per-job CPU budget fits

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 100 GB of disk claims a verification of a backup whose file is 10000 MB, whose database is 20000 MB and whose largest indexes are 3000, 2000, 1000, 500 and 100 MB
- **THEN** the assignment names 4 parallel jobs and a disk budget of 38036 MB

#### Scenario: The free disk fits fewer jobs than the CPU allows

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 36 GB of disk, which is 36864 MB, claims a verification of the same backup
- **THEN** the assignment names 2 parallel jobs and a disk budget of 36536 MB

#### Scenario: Not even one job fits

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 33 GB of disk, which is 33792 MB, claims work while the only pending verification is of the same backup
- **THEN** the response is an empty `204`
- **AND** the verification stays pending with no assigned agent and the same attempt count

#### Scenario: The per-job CPU budget comes from the agent's concurrency

- **WHEN** an agent with 8 CPUs, 2 concurrent jobs and 100 GB of disk claims a verification of the same backup
- **THEN** the assignment names 4 parallel jobs

#### Scenario: Fewer CPUs than concurrent jobs

- **WHEN** an agent with 1 CPU, 2 concurrent jobs and 100 GB of disk claims a verification of the same backup
- **THEN** the assignment names 1 parallel job and a disk budget of 34536 MB

#### Scenario: More CPU than the largest number of jobs

- **WHEN** an agent with 16 CPUs, 1 concurrent job and 100 GB of disk claims a verification of the same backup
- **THEN** the assignment names 8 parallel jobs and a disk budget of 38136 MB, which counts all five recorded indexes

#### Scenario: A TimescaleDB backup

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 100 GB of disk claims a verification of a TimescaleDB backup with the same sizes
- **THEN** the assignment names 1 parallel job and a disk budget of 34536 MB

#### Scenario: The agent restores with the assigned jobs

- **WHEN** an agent whose per-job CPU budget is 8 receives an assignment that names 2 parallel jobs
- **THEN** the restore tool runs with 2 parallel jobs

### Requirement: A backup without recorded index sizes or database size is verified with one job

A verification of a backup that recorded no index sizes, or whose estimated database size is unknown, SHALL get 1 parallel job. A backup records no index sizes when it was made before index sizes were recorded or when they could not be read. Its disk budget SHALL be the size of the backup file, plus the estimated database size, plus a gap of half the estimated database size that is at least 1536 MB and at most 5120 MB, rounded up to a whole megabyte. When the estimated database size is unknown, it SHALL count as 0 and the gap SHALL be 5120 MB. Such a verification SHALL be assigned only when that disk budget is no more than the agent's free disk.

A backup that recorded that it has no indexes SHALL count as a backup with recorded index sizes, all of them zero.

#### Scenario: A backup made before index sizes were recorded

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 100 GB of disk claims a verification of a backup whose file is 10000 MB, whose database is 20000 MB and which recorded no index sizes
- **THEN** the assignment names 1 parallel job and a disk budget of 35120 MB

#### Scenario: A backup whose database size is unknown

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 100 GB of disk claims a verification of a backup whose file is 10000 MB, whose estimated database size is unknown and whose largest indexes are 3000 and 2000 MB
- **THEN** the assignment names 1 parallel job and a disk budget of 15120 MB

#### Scenario: A backup with no indexes

- **WHEN** an agent with 4 CPUs, 1 concurrent job and 100 GB of disk claims a verification of a backup whose file is 10000 MB, whose database is 20000 MB and which recorded that it has no indexes
- **THEN** the assignment names 4 parallel jobs and a disk budget of 31536 MB

### Requirement: A running verification keeps its disk budget reserved

The disk budget of a verification SHALL count against its agent's free disk from the moment it is assigned until it is no longer running on that agent. A verification that completes, fails, is cancelled or is returned to pending SHALL stop counting. A verification that was running when the server was upgraded to this behavior SHALL count with the disk budget it was assigned. The disk budgets of the verifications an agent runs SHALL never add up to more than the disk capacity it reported in the claim that received the latest of them.

#### Scenario: A second verification gets the disk that remains

- **WHEN** an agent with 8 CPUs, 2 concurrent jobs and 100 GB of disk runs a verification with a disk budget of 65000 MB, and claims a verification of a backup whose file is 10000 MB, whose database is 20000 MB and whose largest indexes are 3000, 2000, 1000, 500 and 100 MB
- **THEN** the assignment names 2 parallel jobs and a disk budget of 36536 MB

#### Scenario: A second verification waits for the disk

- **WHEN** the same agent runs a verification with a disk budget of 70000 MB and claims a verification of the same backup
- **THEN** the response is an empty `204`, and the verification stays pending
- **AND** after the running verification completes, the next claim of the agent receives the pending verification with 4 parallel jobs

#### Scenario: A cancelled verification releases its disk budget

- **WHEN** the same agent runs a verification with a disk budget of 70000 MB, a pending verification of the same backup waits, and a member cancels the running verification
- **THEN** the next claim of the agent receives the pending verification with 4 parallel jobs and a disk budget of 38036 MB

### Requirement: A verification that ran out of its disk budget says what to change

When a restore reaches its disk budget, the agent SHALL report the failure with a message that names the disk budget in megabytes and the number of parallel jobs, and says that raising `--max-disk-gb` or lowering `--max-cpu` of the agent gives the restore more room.

#### Scenario: The restore reaches its disk budget

- **WHEN** a restore with 2 parallel jobs and a disk budget of 36536 MB writes more than 36536 MB
- **THEN** the agent reports the failure with a message that contains `36536 MB`, `2 parallel jobs`, `--max-disk-gb` and `--max-cpu`
