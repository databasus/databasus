# recurring-schedules Specification

## Purpose
Defines when a monthly schedule runs in a month shorter than its day, what happens to a cron schedule with no date to run on, how the interface shows and saves weekly and monthly days, when a schedule counts as complete for saving, and how its fields are named.

## Requirements

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

### Requirement: A cron expression with no date to run on is refused and never runs

Saving a cron schedule whose expression parses but matches no calendar date SHALL get `400` with `code` `invalid_request` and SHALL change nothing. The response SHALL name `settings.schedule.cronExpression` for a logical config, `settings.fullBackupSchedule.cronExpression` or `settings.incrementalBackupSchedule.cronExpression` for a physical config, and `schedule` for a verification config. A stored schedule with such an expression SHALL NOT be due, whether or not it has a last run. An expression that matches a date only in leap years SHALL stay valid. This SHALL hold for logical backups, physical backups and verification alike, with one exception: a stored incremental schedule with such an expression SHALL be read as the schedule the backend builds from the full schedule, or as no schedule when the full schedule is hourly, and that replacement SHALL decide when incremental backups run. The `backup-configs` capability states what happens to a physical backup type change while the full schedule holds such an expression.

#### Scenario: Saving February 31

- **WHEN** a member saves a logical backup config with the cron expression `0 0 31 2 *`
- **THEN** the response is `400` with `code` `invalid_request` that names `settings.schedule.cronExpression`, and the stored config is unchanged

#### Scenario: Saving February 31 as a physical full schedule

- **WHEN** a member saves a physical config with the cron expression `0 0 31 2 *` as its full schedule
- **THEN** the response is `400` with `code` `invalid_request` that names `settings.fullBackupSchedule.cronExpression`, and the stored config is unchanged

#### Scenario: Saving February 31 as a physical incremental schedule

- **WHEN** a member saves a physical config with the cron expression `0 0 31 2 *` as its incremental schedule
- **THEN** the response is `400` with `code` `invalid_request` that names `settings.incrementalBackupSchedule.cronExpression`, and the stored config is unchanged

#### Scenario: Saving a day that none of the listed months has

- **WHEN** a member saves a logical backup config with the cron expression `0 0 31 2,4 *`
- **THEN** the response is `400` with `code` `invalid_request` that names `settings.schedule.cronExpression`, and the stored config is unchanged

#### Scenario: Saving February 31 for verification

- **WHEN** a member saves a verification config with automatic verification on, the interval type and the cron expression `0 0 31 2 *`
- **THEN** the response is `400` with `code` `invalid_request` that names `schedule`, and the stored config is unchanged

#### Scenario: Saving February 29

- **WHEN** a member saves a logical backup config with the cron expression `0 0 29 2 *`
- **THEN** the config is saved

#### Scenario: A stored expression with a last run

- **WHEN** an earlier release stored the cron expression `0 0 31 2 *` and the schedule last ran a minute ago
- **THEN** the schedule is not due

#### Scenario: A stored expression without a last run

- **WHEN** an earlier release stored the cron expression `0 0 31 2 *` and the schedule has never run
- **THEN** the schedule is not due

#### Scenario: A stored incremental expression

- **WHEN** an earlier release stored the cron expression `0 0 31 2 *` as the incremental schedule of a physical config whose full schedule is weekly
- **THEN** reading the config returns the daily schedule the backend builds, and that schedule decides when incremental backups run

#### Scenario: A stored incremental expression under an hourly full schedule

- **WHEN** an earlier release stored the cron expression `0 0 31 2 *` as the incremental schedule of a physical config whose full schedule is hourly
- **THEN** reading the config returns no incremental schedule, and no incremental backup runs

### Requirement: The interface keeps a weekly day through its conversion

The interface SHALL show the day of a weekly schedule in the user's local time, and SHALL convert it to and from UTC using only the schedule's time of day and the user's UTC offset at that time of day today. When that offset changes with daylight saving time, the shown day MAY move by one together with the shown time. The offset on another day of the current week SHALL NOT change the day it shows or saves. A day SHALL survive being shown and saved again unchanged. When the UTC and local times of a schedule fall on different days, the conversion SHALL move the day by one and SHALL wrap it across the week. A saved weekday `0` SHALL show as Sunday, like `7`.

#### Scenario: A local day east of UTC

- **WHEN** a user at UTC+3 chooses Monday at 02:00
- **THEN** the schedule is saved as Sunday at 23:00 UTC, and the interface shows Monday at 02:00

#### Scenario: A local day west of UTC

- **WHEN** a user at UTC-5 chooses Sunday at 22:00
- **THEN** the schedule is saved as Monday at 03:00 UTC, and the interface shows Sunday at 22:00

#### Scenario: An offset change later in the week

- **WHEN** a user in Africa/Cairo chooses Friday at 00:00 on 2026-10-27, while the zone's offset changes on 2026-10-29
- **THEN** the schedule is saved as Thursday at 21:00 UTC, and the interface shows Friday at 00:00

#### Scenario: Opening the schedule after the offset change

- **WHEN** a schedule saved as Thursday at 21:00 UTC is opened by a user in Africa/Cairo after the zone's offset changed from UTC+3 to UTC+2
- **THEN** the interface shows Thursday at 23:00

#### Scenario: Changing the time keeps the local day

- **WHEN** a user at UTC+3 opens a schedule shown as Monday at 12:00 and changes only its time to 02:00
- **THEN** the schedule is saved as Sunday at 23:00 UTC, and the interface shows Monday at 02:00

#### Scenario: Sunday saved as zero

- **WHEN** a weekly schedule saved through the API with weekday `0` at 12:00 UTC is opened by a user at UTC
- **THEN** the interface shows Sunday and allows saving

### Requirement: The interface keeps a day chosen before a time

When a weekly or monthly schedule has no time of day yet, the interface SHALL show and keep the day exactly as the user chose it, whatever the current time is. Choosing the time SHALL convert that day to UTC together with the time.

#### Scenario: A weekday chosen before the time

- **WHEN** a user at UTC+3 at 01:00 local switches an hourly schedule without a time to weekly and chooses Monday
- **THEN** the interface shows Monday, and choosing 12:00 saves Monday at 09:00 UTC

#### Scenario: A day of the month chosen before the time

- **WHEN** a user at UTC+3 at 01:00 local switches an hourly schedule without a time to monthly and chooses day 10
- **THEN** the interface shows day 10, and choosing 02:00 saves day 9 at 23:00 UTC

### Requirement: The interface allows saving only a complete schedule

While backups are on, the interface SHALL keep Save disabled until every schedule on the form is complete. A daily schedule SHALL need a time of day. A weekly schedule SHALL also need a weekday, and a monthly schedule SHALL also need a day of the month. A cron schedule SHALL need an expression of five fields that has a date to run on and uses no `L`, `H` or `#` token and no weekday `7`, because the server refuses them. These letters SHALL stay allowed inside a month or weekday name such as `JUL` or `THU`. The interface SHALL show the next run only for such an expression, and SHALL show the expression as invalid otherwise. A schedule without a type SHALL offer only the choice of a type and SHALL count as incomplete. Turning backups off SHALL lift this rule, as it does for the other backup settings.

#### Scenario: A daily schedule without a time

- **WHEN** a user switches an hourly schedule without a time to daily
- **THEN** Save is disabled until the user chooses a time

#### Scenario: Cron expressions the server refuses

- **WHEN** a user enters `0 0 2 * * *`, `@daily`, `0 2 * *`, `0 0 L * *`, `0 0 * * 1#2`, `0 0 H * *`, `0 0 * * 7`, `0 0 31 2 *` or `0 0 31 2,4 *`
- **THEN** the interface shows the expression as invalid, shows no next run and keeps Save disabled

#### Scenario: Cron expressions the server accepts

- **WHEN** a user enters `0 2 * * *`, `0 0 ? * *`, `0 0 * * MON`, `0 0 29 2 *` or `*/5 1-3 1,15 JUL mon-fri`
- **THEN** the interface shows the next run and Save is enabled

#### Scenario: A saved expression with a time zone prefix

- **WHEN** a user opens a config whose saved cron expression is `TZ=Europe/Moscow 0 2 * * *`
- **THEN** the interface shows the expression as invalid and keeps Save disabled until the user replaces it or turns backups off, and the stored schedule keeps running until then

#### Scenario: An incremental schedule without a type

- **WHEN** a user opens a physical config of a database with incremental backups whose incremental schedule is empty
- **THEN** the interface offers only the choice of a type for that schedule, and Save is disabled until the schedule is complete

### Requirement: Logical and physical schedules use the same field names

The logical and physical backup config forms and their read-only views SHALL name the schedule's day and time fields the same in every interface language: "Weekday", "Day of month", "Time of day" and "Cron expression (UTC)" in English. A form with two schedules SHALL let assistive technology tell their fields apart.

#### Scenario: A logical config

- **WHEN** a user opens a weekly logical backup config
- **THEN** the fields are named "Weekday" and "Time of day", without the word "Backup"

#### Scenario: A read-only logical config

- **WHEN** a user views a weekly logical backup config without editing it
- **THEN** the fields are named "Weekday" and "Time of day", without the word "Backup"

#### Scenario: Two schedules on one form

- **WHEN** a user opens a physical config with weekly full backups and weekly incremental backups
- **THEN** each "Weekday" label identifies the control of its own schedule

### Requirement: The verification schedule follows the backup schedule rules

While automatic verification is on and runs on a schedule, the interface SHALL keep Save disabled until the schedule is complete, by the same rule as a backup schedule. While automatic verification is off or runs after each backup, the schedule SHALL not block Save, because the server does not use or validate it then. A save in those states SHALL send the last acknowledged schedule, so edits the form no longer shows are discarded. The verification form and its read-only view SHALL name the schedule's day and time fields the same as the backup forms in every interface language, and they SHALL convert days and times the same way.

#### Scenario: An expression the server refuses

- **WHEN** a user enters `0 0 31 2 *` as the cron expression of an automatic verification
- **THEN** the interface shows the expression as invalid, shows no next run and keeps Save disabled

#### Scenario: Verification after each backup

- **WHEN** a user on a weekly verification types the invalid cron expression `0 0 31 2 *`, chooses to verify after each backup and saves
- **THEN** Save is enabled, the schedule fields are hidden, and the request carries the acknowledged weekly schedule

#### Scenario: Automatic verification turned off

- **WHEN** a user on a weekly verification types the invalid cron expression `0 0 31 2 *`, turns automatic verification off and saves
- **THEN** Save is enabled, the schedule fields are hidden, and the request carries the acknowledged weekly schedule

#### Scenario: A weekly verification form

- **WHEN** a user opens the editor of a weekly verification config
- **THEN** the fields are named "Weekday" and "Time of day", without the word "Verification"

#### Scenario: A read-only weekly verification

- **WHEN** a user views a weekly verification config without editing it
- **THEN** the fields are named "Weekday" and "Time of day", without the word "Verification"
