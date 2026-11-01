## ADDED Requirements

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
