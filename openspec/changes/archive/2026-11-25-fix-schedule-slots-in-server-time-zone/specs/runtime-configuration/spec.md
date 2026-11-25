## ADDED Requirements

### Requirement: Times from the internal database stay in UTC whatever the server's time zone

Every time the application reads back from its internal database SHALL be in UTC, whatever time zone the operator sets for the container. The API SHALL return such times in UTC, with the `Z` designator. Showing a time in a user's local time is the interface's job. The tools the application starts keep the operator's time zone setting, as the requirement on their environment says.

#### Scenario: A stored time keeps its UTC calendar

- **WHEN** the container runs with `TZ=America/Sao_Paulo` and the application reads back a time it stored as 04:00 UTC on November 25
- **THEN** the date and time of day it works with are November 25 and 04:00, not November 25 and 01:00

#### Scenario: A time in an API response

- **WHEN** the container runs with `TZ=America/Sao_Paulo` and root admin reads the Databasus backup status after a successful copy
- **THEN** the destination's `lastSuccessfulSnapshotAt` ends with `Z`, not with `-03:00`
