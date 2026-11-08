## ADDED Requirements

### Requirement: GFS help explains completed periods and current candidates

The logical and physical backup configuration screens SHALL explain GFS through their existing help controls in every supported interface language. Help SHALL distinguish hourly and daily counts, which include the current calendar period, from weekly, monthly and yearly counts, which reserve completed calendar periods and also protect the current period's newest successful candidate. It SHALL explain that periods use UTC, weeks start on Monday, and the reference is the newest successful eligible backup rather than the user's clock. Physical help SHALL describe full backups as the eligible backups. Help SHALL explain that one backup can satisfy several levels, so the configured counts do not promise a fixed total number of files. Short descriptions elsewhere in these screens SHALL agree with the help. Opening help SHALL NOT change draft values or mark the configuration as changed.

#### Scenario: Logical GFS help with two daily periods

- **WHEN** a user opens GFS help while editing logical retention with two daily periods, one weekly period and one monthly period
- **THEN** the help explains that the weekly and monthly counts preserve completed periods in addition to the current candidates
- **AND** the entered retention values and the draft's changed state stay the same

#### Scenario: Physical GFS help

- **WHEN** a user opens the physical configuration's GFS help
- **THEN** the help describes the same calendar rules for successful full backups
- **AND** it does not imply that selecting a full backup by GFS preserves its incremental backups or WAL

#### Scenario: A different interface language

- **WHEN** the user reads either GFS help in any supported interface language
- **THEN** the explanation conveys the same period, candidate and overlap rules
- **AND** UTC retention boundaries do not change with the interface language or browser region
