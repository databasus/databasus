## ADDED Requirements

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
