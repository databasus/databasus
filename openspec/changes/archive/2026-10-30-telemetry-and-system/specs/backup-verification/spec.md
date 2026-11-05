## MODIFIED Requirements

### Requirement: A verification agent reports its memory in megabytes

A heartbeat SHALL include the required nonnegative integer `maxRamMb` with the memory the agent may use in megabytes, as given to the agent. The server SHALL store that value and SHALL return it in megabytes in the list of verification agents. An explicit zero SHALL remain valid. The agents list in the interface SHALL show the memory of any agent that reported more than zero, with the unit that fits the value, such as "512 MB" or "4 GB". Stored memory SHALL survive upgrades without being reset or converted again.

A heartbeat without `maxRamMb`, including one that supplies memory only in whole gigabytes, SHALL get 400 with code `invalid_request`. The server SHALL neither infer memory from another unit nor replace a missing value with zero. Rejection SHALL leave the stored capacity and last-seen timestamp unchanged.

#### Scenario: An agent with less than one gigabyte of memory

- **WHEN** an agent started with 512 MB of memory sends a heartbeat
- **THEN** the list of verification agents returns that agent with `512` as its memory in megabytes, and the agents list shows "512 MB RAM" for it

#### Scenario: An agent with whole gigabytes of memory

- **WHEN** an agent started with 4096 MB of memory sends a heartbeat
- **THEN** the list of verification agents returns `4096` as its memory in megabytes, and the agents list shows "4 GB RAM" for it

#### Scenario: Memory is explicitly zero

- **WHEN** a valid agent sends a heartbeat with `maxRamMb` equal to zero
- **THEN** the heartbeat succeeds and stores zero megabytes

#### Scenario: Memory is omitted or reported only in gigabytes

- **WHEN** a valid agent sends a heartbeat without `maxRamMb`, with or without a gigabyte memory value
- **THEN** the response is 400 with code `invalid_request`
- **AND** the stored capacity and last-seen timestamp stay unchanged

#### Scenario: Persisted memory survives the upgrade

- **WHEN** the instance upgrades with an agent whose stored memory is 2048 megabytes
- **THEN** the list of verification agents still returns 2048 megabytes before the next heartbeat

## REMOVED Requirements

### Requirement: An agent of the previous release keeps working until it updates

**Reason**: Heartbeat memory in megabytes is mandatory. Gigabyte conversion and accepting heartbeats without memory are removed as an explicit breaking protocol change; other agent operations are unchanged.

**Migration**: Agents that send only gigabytes must update through the unchanged public version and binary-download operations. Their update check runs independently of heartbeats. Until they update, refused heartbeats do not refresh last-seen time and agents older than the five-minute freshness limit affect instance health. Persisted memory remains in megabytes and is retained, as required by the updated memory requirement.
