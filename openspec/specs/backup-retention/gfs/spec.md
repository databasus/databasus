# backup-retention/gfs Specification

## Purpose

Define GFS selection for logical backups and physical full backups so short daily retention preserves older weekly, monthly and yearly restore points through repeated cleanup.

## Requirements

### Requirement: GFS uses independent UTC calendar periods

GFS SHALL select only successfully completed backups: logical backups for logical retention, and full backups for physical full-backup retention. The reference instant SHALL be the creation time of the newest such backup. Every calendar boundary SHALL be computed in UTC. Weeks SHALL start on Monday at 00:00; months and years SHALL start on their first day at 00:00. A period SHALL include its start and exclude the next period's start. A current period SHALL mean the period containing the reference instant, and a completed period SHALL mean a period before that current period. Without a newer successful backup, the reference instant and the GFS selection SHALL NOT advance with wall-clock time.

Every enabled level SHALL select independently of the other levels. A zero count SHALL disable that level, including its current candidate. Within a selected period, GFS SHALL keep its newest eligible backup by creation time. Equal creation times SHALL resolve to a stable choice independent of listing order. Empty calendar periods SHALL consume their place in the configured calendar range without selecting a backup; GFS SHALL NOT fill them with backups outside that range.

#### Scenario: A short monthly setting does not shorten daily retention

- **WHEN** the newest successful backup was created on 2026-11-05 at 12:00 UTC and GFS keeps 40 daily periods and one completed monthly period
- **THEN** daily selection includes the newest available backup of each date from 2026-09-27 through 2026-11-05, independently of monthly selection

#### Scenario: A week crosses the year boundary

- **WHEN** the reference instant is 2027-01-04 at 12:00 UTC and GFS keeps one completed week
- **THEN** the completed weekly range is from 2026-12-28 at 00:00 UTC inclusive to 2027-01-04 at 00:00 UTC exclusive

#### Scenario: A backup has a non-UTC timestamp offset

- **WHEN** two successful backups were created at 2026-11-01T00:30:00+03:00 and 2026-11-01T01:30:00Z and GFS keeps one completed month
- **THEN** the first backup belongs to October's completed monthly period and the second is November's current candidate

#### Scenario: An empty completed week

- **WHEN** the reference instant is 2026-11-05 at 12:00 UTC, GFS keeps one completed week, and no successful backup exists from 2026-10-26 through 2026-11-01
- **THEN** no completed weekly representative is selected
- **AND** an available backup from 2026-10-25 is not selected by that weekly level

#### Scenario: Successful backups pause

- **WHEN** no successful backup is added while time passes, even if failed or cancelled backups appear
- **THEN** GFS keeps the same successful backups and current candidates as before the pause

### Requirement: Hourly and daily GFS counts include their current period

An hourly count of N SHALL select the current calendar hour and its N minus one immediately preceding calendar hours. A daily count of N SHALL select the current calendar day and its N minus one immediately preceding calendar days. Each selected hour or day SHALL keep at most one representative, without an additional current-period slot.

#### Scenario: Two daily periods

- **WHEN** the reference instant is 2026-11-05 at 12:00 UTC and only daily GFS is enabled with count two
- **THEN** GFS keeps the newest successful backup from November 5 and the newest successful backup from November 4, if present
- **AND** it keeps no backup from November 3 or earlier

#### Scenario: Two hours across midnight

- **WHEN** the reference instant is 2026-11-05 at 00:05 UTC and only hourly GFS is enabled with count two
- **THEN** GFS selects the hours starting at November 5 at 00:00 UTC and November 4 at 23:00 UTC
- **AND** no extra candidate is added outside those two hours

### Requirement: Weekly monthly and yearly counts reserve completed periods

A weekly, monthly or yearly count of N SHALL select the N immediately preceding completed calendar periods of that level. In addition, each enabled level SHALL protect the newest eligible backup in its current period as a candidate. The candidate SHALL NOT consume any of its N completed-period slots. A newer successful backup in the current period SHALL replace that candidate for that level; a failed, cancelled or running backup SHALL NOT replace it. When a successful backup first enters the next calendar period, the former candidate SHALL become the previous period's representative if it lies in the configured completed-period range.

#### Scenario: Daily cleanup preserves weekly and monthly history

- **WHEN** one successful backup is created at 12:00 UTC every day from 2026-10-12 through 2026-11-05, GFS keeps two daily periods, one completed week and one completed month, and cleanup runs after each backup
- **THEN** after cleanup on November 5 the retained backups are exactly November 5, November 4, November 1 and October 31
- **AND** the November 5 backup serves as both current weekly and current monthly candidate

#### Scenario: A candidate survives with only monthly retention enabled

- **WHEN** only monthly GFS is enabled with count one and successful backups are created on October 29 and October 31, followed by a failed backup on November 1 and a successful backup on November 2
- **THEN** October 31 remains protected through the failed attempt
- **AND** after the November 2 success, October 31 is the completed monthly representative and November 2 is the current monthly candidate

#### Scenario: An older candidate is replaced only for its own level

- **WHEN** a newer successful backup replaces the current monthly candidate and the former candidate remains selected by daily or weekly retention
- **THEN** the former candidate remains retained for those other levels

#### Scenario: A short February ends

- **WHEN** the newest successful backup is created on March 31 and GFS keeps one completed month
- **THEN** the newest successful backup from February is selected if one exists, including February 28 in a common year or February 29 in a leap year
- **AND** selection does not exclude it because March has more days

#### Scenario: Yearly retention with no daily weekly or monthly level

- **WHEN** only yearly GFS is enabled with count one and successful backups exist on 2026-12-31 and 2027-01-01
- **THEN** GFS keeps December 31 as the completed yearly representative and January 1 as the current yearly candidate

### Requirement: GFS keeps the union of selected representatives

A backup selected by any enabled level SHALL remain retained even if no other level selects it. A backup selected by several levels SHALL remain one backup and SHALL NOT require duplicate stored files. The total number retained SHALL therefore not be promised as the sum of configured counts; long-period candidates can add a point, and overlapping selections can reduce the total. When a previous representative leaves every configured range, it SHALL become eligible for cleanup subject to the backup method's existing deletion protections. Selection SHALL apply to already existing backups without recreating previously deleted copies.

#### Scenario: Weekly and monthly representatives overlap

- **WHEN** GFS keeps two daily periods, one completed week and one completed month, the reference date is 2026-06-04, and daily successful backups span May and June
- **THEN** June 4 and June 3 are selected as daily points and May 31 satisfies both the weekly and monthly selections
- **AND** exactly three backups remain selected, with May 31 appearing only once

#### Scenario: The old monthly representative expires

- **WHEN** daily successful backups and cleanup continue from October through December 3 with two daily periods, one completed week and one completed month
- **THEN** October 31 is no longer selected after the first successful December backup establishes November as the last completed month
- **AND** November 30 remains the completed monthly representative on December 3

#### Scenario: Physical GFS selects full backups

- **WHEN** physical full-backup GFS keeps two daily periods, one completed week and one completed month, completed full backups exist daily from October 12 through November 5, and existing physical deletion protections permit cleanup
- **THEN** its selected full backups on November 5 are November 5, November 4, November 1 and October 31
- **AND** the existing physical policies still decide retention of chains, incrementals and WAL
