## MODIFIED Requirements

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
