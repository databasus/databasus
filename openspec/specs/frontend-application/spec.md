# frontend-application Specification

## Purpose

Preserve existing application navigation while making startup recoverable, account capabilities explicit and the shell usable across themes, languages and screen sizes.

## Requirements

### Requirement: Startup ends in usable content or a recoverable failure

For the active session the application SHALL load the current user, instance settings and accessible workspaces before enabling account-dependent actions. Missing or failed essential data SHALL not grant edit or administrative capability. A failed essential load SHALL end its loading indicator and offer an explicit retry with translated feedback. Failure of supplementary disk or release information SHALL not prevent otherwise usable account screens from opening. Retrying startup SHALL invalidate older startup responses and SHALL not replay a user mutation.

#### Scenario: Essential startup data cannot load

- **WHEN** loading the current user, settings or workspace list fails for the current session
- **THEN** loading ends, a translated error and retry action are reachable, and protected actions remain unavailable
- **AND** retry can produce usable content without accepting an older response

#### Scenario: Supplementary information cannot load

- **WHEN** account startup succeeds but disk or release information cannot load
- **THEN** the user can still navigate the available account screens without a permanent loading indicator

### Requirement: Navigation preserves existing addresses and permission rules

The application SHALL retain its existing root, sign-in callback and Google storage callback addresses and existing screen tabs, and SHALL expose section and resource addresses as defined by frontend-navigation. An unknown address SHALL resolve through the root to the default available destination. The selected workspace SHALL be validated against the current account's loaded workspaces; an unavailable remembered or explicitly addressed selection SHALL fall back to an available workspace or the existing empty-workspace experience, and the address SHALL identify the destination actually shown. Administrative screens SHALL be available only to instance administrators. Instance administrators SHALL retain the workspace capabilities granted by the workspace-access requirements, including when their explicit workspace role is viewer. Creating a workspace SHALL retain the existing instance-setting policy and select the created workspace's databases destination on success.

#### Scenario: A remembered workspace is unavailable

- **WHEN** startup finishes and the remembered workspace is absent from the current accessible list
- **THEN** the application selects an available workspace or shows the empty-workspace action according to current permissions, with the matching address
- **AND** it does not request the unavailable workspace's content

#### Scenario: An instance administrator has a viewer workspace role

- **WHEN** an instance administrator opens a workspace whose explicit role is viewer
- **THEN** the interface retains the administrator's permitted management actions

#### Scenario: A member opens the application

- **WHEN** an account without the instance administrator role completes startup
- **THEN** administrative users and instance-settings screens remain unavailable
- **AND** a link to either screen resolves to an available default destination with its matching address

#### Scenario: An existing callback address is opened

- **WHEN** the browser opens an existing sign-in or storage callback address
- **THEN** its corresponding callback flow remains reachable without a new deep link

#### Scenario: A workspace is created

- **WHEN** a permitted user successfully creates a workspace
- **THEN** that workspace's databases screen is selected and its address becomes current

### Requirement: Presentation changes preserve the active screen

Language and theme changes SHALL update the shell and migrated account screens without replacing their active lifecycle, selected workspace, entered values, open drawers or pending operations. Visible persistent errors SHALL re-render in the selected language; transient notices SHALL follow the frontend-localization lifetime exception. Shell navigation, account forms, errors and actions SHALL remain reachable at 320 CSS pixels and with text enlarged to 200 percent. Ordinary translated content SHALL wrap without page-wide horizontal overflow, clipping or overlapping actions. Each bounded content region SHALL have one vertical scroll owner. Keyboard users SHALL be able to navigate the shell and close drawers with focus restored to an available invoker.

#### Scenario: Language changes with an active profile draft

- **WHEN** a user changes language or theme while profile input or a save is pending
- **THEN** the selected screen, input and pending operation remain and persistent copy follows the selected context

#### Scenario: A narrow sidebar changes language

- **WHEN** a user chooses a language through the open sidebar at 320 CSS pixels
- **THEN** its labels update, the sidebar stays open, focus returns to the language control and the underlying screen keeps its draft

#### Scenario: Long account-form errors are shown with enlarged text

- **WHEN** a migrated account screen shows long translated errors at narrow width or with text enlarged to 200 percent
- **THEN** fields and primary actions remain reachable without overlap or page-wide horizontal overflow

### Requirement: Version checks end with their application lifecycle

A production application SHALL check the server version on startup and every five minutes. A different nonempty server version SHALL trigger at most one automatic reload within ten seconds across page loads. Development builds SHALL not poll for a production version. A failed check SHALL leave the current interface usable. Leaving the application lifecycle SHALL stop polling and prevent its pending check from triggering a reload; setting up a new lifecycle SHALL permit fresh checks.

#### Scenario: A replacement version persists across reload

- **WHEN** a version mismatch remains immediately after an automatic reload
- **THEN** another automatic reload does not occur within the ten-second cooldown

#### Scenario: A pending version check completes after cleanup

- **WHEN** the application lifecycle ends before a version check returns a mismatch
- **THEN** that response causes no reload and the old polling timer remains stopped

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

### Requirement: The shell identifies the current section

The main navigation SHALL distinguish the selected section with a solid background. Its icon and any visible label SHALL use the light selected presentation in both themes, on wide and narrow viewports. Assistive technology SHALL announce the selected section as the current page, and SHALL NOT read its icon separately from its name.

#### Scenario: A section is selected on a wide viewport

- **WHEN** a user opens a section in either theme on a wide viewport
- **THEN** its navigation button has the solid selected background and the light icon
- **AND** it is the only navigation button exposed as the current page

#### Scenario: A section is selected in the narrow menu

- **WHEN** a user opens the menu in either theme on a narrow viewport
- **THEN** the selected section has the solid background with a light icon and a light label
- **AND** it is the only menu entry exposed as the current page

#### Scenario: The user moves to another section

- **WHEN** a user activates another section
- **THEN** the selected presentation and the current-page state move to that section and leave the previous one

### Requirement: The new-version link explains itself

On wide viewports, where the shell shows the running version, the shell SHALL show a small status dot that links to the latest release when a newer release exists. The link SHALL explain that a new version is available in a translated tooltip on pointer hover and keyboard focus, and SHALL have the same text as its accessible name.

#### Scenario: A newer release exists

- **WHEN** the latest release is newer than the running version and the user hovers over or focuses the new-version link on a wide viewport
- **THEN** the translated new-version explanation appears and the link leads to the latest release

#### Scenario: No newer release exists

- **WHEN** the running version is the latest release, the application runs as a development build, or the release lookup fails
- **THEN** no new-version link is shown
