## ADDED Requirements

### Requirement: A healthcheck config made before the upgrade is there after it

Before the upgrade, the test SHALL save through the API of the old release a healthcheck config of the logical PostgreSQL database in which every value differs from the default: checks switched off, notifications switched off, an interval, a number of failed attempts and a number of retention days other than the default's. After the upgrade, through the API of the current code, the test SHALL check that the config is returned with every value written before the upgrade.

#### Scenario: Healthcheck config after the upgrade

- **WHEN** the current code starts on the volume of an old release that has a healthcheck config with non-default values
- **THEN** the config is returned with the same values

#### Scenario: A migration loses the healthcheck config

- **WHEN** a migration of the current code resets the config to the default, or empties or rewrites a column the test reads, while the table stays readable
- **THEN** the upgrade test fails and names the config value that differs
