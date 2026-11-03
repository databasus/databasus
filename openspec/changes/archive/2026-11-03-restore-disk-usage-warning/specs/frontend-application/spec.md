## ADDED Requirements

### Requirement: Disk summaries preserve warning visibility and severity

When disk usage is available, the desktop header SHALL show its disk summary only when used space is strictly greater than 85 percent of total space. A visible summary SHALL use the active theme's muted text color when usage is at most 95 percent and its danger text color when usage is strictly greater than 95 percent. Threshold comparisons SHALL use the returned byte counts before rounding the displayed percentage. The summary SHALL retain localized used capacity, total capacity and percentage, together with its explanatory tooltip. In the mobile menu, disk information SHALL remain available at any usage when data is available, with the same 95 percent color threshold.

#### Scenario: Ordinary disk usage is hidden in the desktop header

- **WHEN** disk usage reports 50 percent used space on desktop
- **THEN** the header does not display a disk summary

#### Scenario: The visibility threshold is exclusive

- **WHEN** disk usage reports exactly 85 percent used space on desktop
- **THEN** the header does not display a disk summary
- **WHEN** disk usage reports 86 percent used space on desktop
- **THEN** the header displays the disk summary in the active theme's muted text color

#### Scenario: Elevated disk usage is visible in muted text

- **WHEN** disk usage reports 90 percent used space on desktop
- **THEN** the header displays localized used capacity, total capacity and percentage in the active theme's muted text color
- **AND** the explanatory tooltip remains available

#### Scenario: The danger threshold is exclusive

- **WHEN** disk usage reports exactly 95 percent used space on desktop
- **THEN** the header displays the disk summary in the active theme's muted text color

#### Scenario: Critical disk usage uses danger text

- **WHEN** disk usage reports 96 percent used space on desktop
- **THEN** the header displays the disk summary in the active theme's danger text color

#### Scenario: Display rounding does not determine severity or visibility

- **WHEN** desktop disk usage is strictly greater than 85 percent but its displayed percentage rounds to 85.0 percent
- **THEN** the header displays the disk summary in the active theme's muted text color
- **WHEN** desktop disk usage is strictly greater than 95 percent but its displayed percentage rounds to 95.0 percent
- **THEN** the header displays the disk summary in the active theme's danger text color

#### Scenario: A theme change preserves warning severity

- **WHEN** a visible desktop disk summary is shown and the user changes between light and dark themes
- **THEN** the summary uses the new theme's text color for the same severity and retains its capacity and percentage

#### Scenario: Mobile disk information remains available

- **WHEN** the user opens the mobile menu with disk usage available
- **THEN** disk information is visible even at 50 percent usage
- **AND** it uses muted text at exactly 95 percent and danger text at 96 percent
