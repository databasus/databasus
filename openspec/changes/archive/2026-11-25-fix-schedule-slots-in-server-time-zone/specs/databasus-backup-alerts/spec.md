## ADDED Requirements

### Requirement: Overdue deadlines follow the schedule in UTC

A destination's due times and overdue deadlines SHALL follow the schedule's slots in UTC, the same slots the backup runs at. The time zone the server runs in SHALL NOT move a due time, so a backup that ran on schedule SHALL NOT be reported as overdue and SHALL NOT cause an incident email.

#### Scenario: A backup on schedule on a server west of UTC

- **WHEN** the server runs in the America/Sao_Paulo time zone, system SMTP is configured, root admin has an email address, a daily schedule at 04:00 UTC makes a successful copy at 04:00 UTC, and the incident check runs at 07:16 UTC
- **THEN** the destination is healthy, no overdue incident opens, and no incident email reaches root admin

#### Scenario: The next deadline after a copy on schedule

- **WHEN** the server runs in a time zone other than UTC, a daily schedule at 04:00 UTC makes a successful copy at 04:00 UTC, no other copy is made, and root admin reads the backup status at 04:01 UTC on the following day
- **THEN** the destination's overdue time is 04:15 UTC on that following day
