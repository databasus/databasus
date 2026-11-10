## MODIFIED Requirements

### Requirement: Shipped clients stay on a maintained release line

The system SHALL ship clients from a release line that the vendor still patches, or, when the line has reached its end of life, target the final patch that line received. By default, a client refresh SHALL select the newest patch of its designated release line that has completed the fourteen-day waiting period at update selection. A newly published patch within that period SHALL NOT force an immediate refresh. The updater SHALL review official release information and report deferred patches; an offline dependency check SHALL NOT claim to establish publication age. Missing age information or a departure from the normal waiting period SHALL be reported with its reason under the dependency upgrade procedure.

#### Scenario: A line the vendor still patches

- **WHEN** a client refresh is selected for a line the vendor still supports
- **THEN** the updater selects the newest eligible patch for that line and records the source used for release-age review

#### Scenario: A line that has ended

- **WHEN** a client refresh targets an end-of-life line whose final patch has completed the waiting period
- **THEN** it selects that final patch and reports the line's end-of-life status

#### Scenario: The newest patch is still waiting

- **WHEN** the newest patch has not completed fourteen days and an earlier eligible patch exists
- **THEN** the refresh selects the earlier eligible patch and reports the deferred release and its eligibility date

#### Scenario: No patch has completed the waiting period

- **WHEN** no patch in the designated line has completed the normal waiting period
- **THEN** the updater reports the deferral or explicitly explains a departure from the normal period
- **AND** it does not silently substitute another release line
