## Purpose

Preserve existing application navigation while making startup recoverable, account capabilities explicit and the shell usable across themes, languages and screen sizes.

## ADDED Requirements

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

The application SHALL retain its existing root, sign-in callback and Google storage callback addresses and existing screen tabs. An unknown address SHALL return to the root. The selected workspace SHALL be validated against the current account's loaded workspaces; an unavailable remembered selection SHALL fall back to an available workspace or the existing empty-workspace experience. Administrative screens SHALL be available only to instance administrators. Instance administrators SHALL retain the workspace capabilities granted by the workspace-access requirements, including when their explicit workspace role is viewer. Creating a workspace SHALL retain the existing instance-setting policy and select the created workspace on success.

#### Scenario: A remembered workspace is unavailable

- **WHEN** startup finishes and the remembered workspace is absent from the current accessible list
- **THEN** the application selects an available workspace or shows the empty-workspace action according to current permissions
- **AND** it does not request the unavailable workspace's content

#### Scenario: An instance administrator has a viewer workspace role

- **WHEN** an instance administrator opens a workspace whose explicit role is viewer
- **THEN** the interface retains the administrator's permitted management actions

#### Scenario: A member opens the application

- **WHEN** an account without the instance administrator role completes startup
- **THEN** administrative users and instance-settings screens remain unavailable

#### Scenario: An existing callback address is opened

- **WHEN** the browser opens an existing sign-in or storage callback address
- **THEN** its corresponding callback flow remains reachable without a new deep link

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
