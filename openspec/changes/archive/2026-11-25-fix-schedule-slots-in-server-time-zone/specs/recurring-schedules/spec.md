## ADDED Requirements

### Requirement: Schedule slots stay in UTC whatever the server's time zone

The time of day, weekday and day of month of a daily, weekly or monthly schedule SHALL be read on the UTC calendar. The time zone the server runs in SHALL NOT move a slot, whether the system decides that a run is due or works out when the next run is expected. This SHALL hold for logical backups, physical backups, verification and Databasus backups alike.

#### Scenario: A daily slot on a server west of UTC

- **WHEN** the server runs at UTC-3 and a daily schedule at 04:00 UTC last ran on November 24 at 04:00 UTC
- **THEN** the schedule is due on November 25 at 04:00 UTC, and the next run after November 25 at 04:00 UTC is expected on November 26 at 04:00 UTC

#### Scenario: A daily slot on a server east of UTC

- **WHEN** the server runs at UTC+3 and a daily schedule at 04:00 UTC last ran on November 25 at 04:00 UTC
- **THEN** the schedule is not due on November 26 at 01:00 UTC, and the next run after November 25 at 04:00 UTC is expected on November 26 at 04:00 UTC

#### Scenario: A weekly slot near midnight on a server east of UTC

- **WHEN** the server runs at UTC+3 and a weekly schedule on Sunday at 23:00 UTC last ran the Sunday before at 23:00 UTC
- **THEN** the schedule is due on Sunday at 23:00 UTC, which is Monday 02:00 on the server's clock, and not on Sunday at 20:00 UTC

#### Scenario: A monthly slot near midnight on a server west of UTC

- **WHEN** the server runs at UTC-5 and a monthly schedule on day 1 at 03:00 UTC last ran on November 1 at 03:00 UTC
- **THEN** the next run after November 1 at 03:00 UTC is expected on December 1 at 03:00 UTC, which is November 30 at 22:00 on the server's clock
