## Purpose

Defines when a monthly schedule runs in a month shorter than its day, and how the interface shows and saves that day, so the length of a month never skips a monthly backup or verification or changes its day.

## ADDED Requirements

### Requirement: A monthly schedule has a slot in every month

Every calendar month SHALL hold one slot of a monthly schedule: on the schedule's day of the month at its time of day, or on the month's last day at that time when the month has no such day. Times and days are in UTC. The schedule SHALL be due from the slot to the end of that month when its last run is before the slot, so the length of a month SHALL NOT skip a run. A slot missed until the month ends SHALL NOT carry over to the next month. A schedule without a last run SHALL be due at once, and a run before the slot, such as a manual one, SHALL NOT use up the slot. This SHALL hold for logical backups, physical backups and verification alike.

#### Scenario: Day 31 in a 30-day month

- **WHEN** a schedule on day 31 at 08:00 last ran on October 31 and the time reaches November 30 at 08:00
- **THEN** the schedule is due

#### Scenario: Before the last day's time

- **WHEN** a schedule on day 31 at 08:00 last ran on October 31 and the time is November 30 at 07:59
- **THEN** the schedule is not due

#### Scenario: A late scheduler pass on the last day

- **WHEN** a schedule on day 31 at 08:00 last ran on October 31 and the first pass after the slot happens on November 30 at 10:00
- **THEN** the schedule is due

#### Scenario: A slot missed past the end of the month

- **WHEN** a schedule on day 31 at 08:00 last ran on October 31 and no pass happens from November 30 at 08:00 until December 1
- **THEN** the schedule is not due on December 1, and it is next due on December 31 at 08:00

#### Scenario: A manual run before the slot

- **WHEN** a schedule on day 31 at 08:00 ran by hand on November 5
- **THEN** the schedule is due on November 30 at 08:00

#### Scenario: February in a common year and in a leap year

- **WHEN** a schedule on day 30 at 08:00 last ran in January
- **THEN** it is due on February 28 at 08:00 in a common year, and on February 29 at 08:00 in a leap year but not on February 28

#### Scenario: One run for the month

- **WHEN** a schedule on day 31 ran on November 30 at its time and the time reaches December 1
- **THEN** the schedule is not due until December 31

#### Scenario: A day the month has

- **WHEN** a schedule on day 10 at 08:00 last ran in the previous month and the time reaches the 10th at 08:00
- **THEN** the schedule is due

### Requirement: The interface keeps a monthly day whatever the current month is

The interface SHALL show the day of a monthly schedule in the user's local time, and SHALL convert it to and from UTC using only the schedule's time of day and the user's UTC offset at that time of day today. When that offset changes with daylight saving time, the shown day MAY move by one together with the shown time. The current month's length SHALL NOT change the day it shows or saves. A day SHALL survive being shown and saved again unchanged. When the UTC and local times of a schedule fall on different days, the conversion SHALL move the day by one and SHALL wrap it across the month boundary as if every month had 31 days.

#### Scenario: A saved day 31 viewed in a short month

- **WHEN** a monthly schedule saved for day 31 at 12:00 UTC is opened in November or February by a user whose local day is the same as the UTC day
- **THEN** the interface shows day 31

#### Scenario: Choosing day 31 in a short month

- **WHEN** a user whose local day is the same as the UTC day chooses day 31 in November and saves
- **THEN** the saved day of the month is 31

#### Scenario: Changing the time keeps the day

- **WHEN** a user opens a schedule shown as day 31 in November and changes only its time, within the same local and UTC day
- **THEN** the saved day of the month stays 31

#### Scenario: A local day east of UTC

- **WHEN** a user at UTC+3 chooses day 1 at 02:00 in any month
- **THEN** the schedule is saved as day 31 at 23:00 UTC, and the interface shows day 1 at 02:00 in every month

#### Scenario: A local day west of UTC

- **WHEN** a user at UTC-5 chooses day 31 at 22:00 in any month
- **THEN** the schedule is saved as day 1 at 03:00 UTC, and the interface shows day 31 at 22:00 in every month
