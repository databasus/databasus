## ADDED Requirements

### Requirement: Verification data made before the upgrade is there after it

Before the upgrade, the test SHALL write through the API of the old release, with its account that is the instance admin and the workspace's owner, a verification agent and a verification config of the logical PostgreSQL database. The config SHALL have automatic verification turned on with the interval schedule type, a weekly schedule on a day and at a time other than the default's, and only the success notification, so every value apart from the weekly interval type differs from the default config. It SHALL then start a manual verification of the completed logical backup, which stays pending because no agent runs. After the upgrade, through the API of the current code, the test SHALL check that the agent is listed with its ID and name, that the config is returned with every value written before the upgrade, the schedule type included, and that the verification is returned with its backup, its trigger and a status that is still pending.

#### Scenario: Verification data after the upgrade

- **WHEN** the current code starts on the volume of an old release that has a verification agent, a verification config with automatic verification on a weekly interval schedule and only the success notification, and a pending manual verification
- **THEN** the agent is listed, the config is returned with the same values, and the verification is returned pending, for the same backup, with the manual trigger

#### Scenario: A migration loses verification data

- **WHEN** a migration of the current code deletes verification rows, or empties or rewrites a column the test reads, while the tables stay readable
- **THEN** the upgrade test fails and names the agent, the config or the verification that is missing or differs

#### Scenario: A migration breaks a verification table

- **WHEN** a migration of the current code drops or renames a verification table or column that the current code still reads
- **THEN** the upgrade test fails on the first request that reads it, or because the current code fails to start
