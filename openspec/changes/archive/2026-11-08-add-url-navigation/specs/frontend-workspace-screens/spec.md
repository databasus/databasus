## MODIFIED Requirements

### Requirement: Resource selection remains valid within its workspace

Database, storage and notifier selection SHALL be remembered separately for each workspace and revalidated against successfully loaded items. Changing the workspace or preference key SHALL read that scope's preference without writing the previous selection into it. For database navigation, an explicit addressed selection SHALL take priority over a remembered selection and SHALL open details on both desktop and mobile. Without an explicit database selection, desktop initial load SHALL select the remembered available item or the first item; mobile initial load SHALL show the list. Storage and notifier selection SHALL retain this desktop and mobile default behavior.

Explicit selection, back-to-list, create and delete SHALL preserve the existing list/detail behavior, with database addresses following frontend-navigation. Creating an item or copying a still-selected database SHALL select the new item. A copy that finishes after the user selects another database SHALL NOT change that selection. After removing the selected item, by deletion or by transfer to another workspace, selection SHALL move to the first remaining item in list order, or become empty. A background refresh that establishes the selected item is gone SHALL apply the same validity rule. Database selection changes SHALL update the address to the selected database and offered tab, or the list when none remains.

#### Scenario: A preference key changes without remounting

- **WHEN** the selection scope changes to another workspace while the screen remains mounted
- **THEN** the new selection uses only that workspace's available items and its explicit address or preference, and its stored preference is not overwritten by the old ID

#### Scenario: A remembered storage was removed

- **WHEN** a desktop list loads successfully and its remembered storage is absent
- **THEN** the first available storage is selected or an empty state is shown

#### Scenario: A user opens the mobile list

- **WHEN** a database page without an explicit database address, a storage page or a notifier page first loads on mobile
- **THEN** the list is shown until the user selects an item, and back-to-list remains available from details

#### Scenario: The selected database is transferred

- **WHEN** the selected database is successfully transferred to another workspace
- **THEN** it leaves the list and the first remaining database is selected, or the list shows no selection when none remains
- **AND** the address changes to that destination

#### Scenario: A database is copied

- **WHEN** a database copy succeeds while its source remains selected
- **THEN** the copy appears in the list and becomes the selected database with its matching address

#### Scenario: A mobile database link overrides remembered selection

- **WHEN** a mobile user opens an explicit accessible database link with another database remembered
- **THEN** the linked database's details appear at the linked address and Back to list remains available
