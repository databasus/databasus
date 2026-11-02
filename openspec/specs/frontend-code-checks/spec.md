# Frontend Code Checks Specification

## Purpose

Frontend checks reject boundary, style, effect and interaction violations.

## Requirements

### Requirement: Frontend checks reject forbidden dependencies

The frontend lint command SHALL reject a dependency directed upward through the application, pages, widgets, features, entities and shared layers. It SHALL reject dependencies between different slices of the same layer and access to a slice's private internals from another slice. These checks SHALL cover ordinary, type-only and dynamic imports and re-exports, with equivalent results for relative and configured alias imports. Re-exporting a forbidden dependency through a public entry point MUST NOT authorize it.

#### Scenario: An upward dependency is added

- **WHEN** shared infrastructure imports an entity's API
- **THEN** frontend lint fails and identifies the dependency and violated rule

#### Scenario: A feature imports another feature

- **WHEN** one feature imports a different feature on the same layer
- **THEN** frontend lint fails

#### Scenario: A private module is imported

- **WHEN** a consumer bypasses another slice's public entry point
- **THEN** frontend lint fails even if the import is used only as a type or loaded dynamically

#### Scenario: A barrel attempts to authorize a forbidden dependency

- **WHEN** a public entry point re-exports a dependency its slice is forbidden to import
- **THEN** frontend lint fails

#### Scenario: A permitted import uses an alias

- **WHEN** a consumer imports a lower layer through its public entry point using the configured alias
- **THEN** lint, type checking, tests and production build resolve the dependency consistently

### Requirement: Architecture checks preserve the existing verification surface

Adding dependency checks SHALL preserve formatting, source lint, type checking and generated API freshness verification in the frontend lint command. Existing application addresses, screens and domain operations SHALL remain available after the mechanical layer rename.

#### Scenario: Generated API types are stale

- **WHEN** frontend dependency checks pass but generated API types differ from the contract
- **THEN** the frontend lint command fails

#### Scenario: A renamed layer is used by an existing screen

- **WHEN** an existing screen loads after the layer rename
- **THEN** its domain requests and available interactions remain unchanged

### Requirement: Frontend checks reject new style bypasses

Frontend lint SHALL reject direct color values outside approved theme and chart-palette owners, and removal of visible focus without an accessible replacement. Coverage SHALL include component styles, style sheets and utility-class colors, including named colors, functional color notation and default color-palette utilities. Layout utilities, semantic theme references and technical data SHALL remain permitted. Diagnostics SHALL identify the source and violated rule.

#### Scenario: A utility class bypasses the theme

- **WHEN** a contributor adds a direct palette color to an ordinary shared control
- **THEN** frontend lint fails even if the color contains no hexadecimal literal

#### Scenario: A theme owner defines a color

- **WHEN** an approved theme or chart-palette owner defines a color for its declared role
- **THEN** the color is accepted without exempting unrelated component styles

#### Scenario: Focus is removed without a replacement

- **WHEN** a contributor disables a control's focus indication without supplying a visible replacement
- **THEN** frontend lint fails

### Requirement: Frontend checks accept no recorded exceptions

Frontend lint SHALL fail on every dependency, public API and style finding. The repository SHALL provide no list of tolerated findings, and the checks SHALL report each finding with its rule and source.

#### Scenario: A forbidden dependency is added

- **WHEN** a contributor adds a dependency that the architecture checks forbid
- **THEN** frontend lint exits with a failure and names the rule and the source file

#### Scenario: A direct color is added

- **WHEN** a contributor adds a direct color outside an approved theme or chart-palette owner
- **THEN** frontend lint exits with a failure and names the rule and the source file

### Requirement: Frontend script modules belong to a layer

Frontend lint SHALL reject a script module that sits outside the application, pages, widgets, features, entities and shared layers, except the application's entry module and its ambient type declaration. Dependencies of such a module could not otherwise be checked. This SHALL hold whether or not the module imports anything. The global style sheet is not a script module and stays at the source root.

#### Scenario: A module is added beside the layers

- **WHEN** a contributor adds a module with no imports and no importers directly under the frontend source root
- **THEN** frontend lint fails and names that file

#### Scenario: A module is added in a new folder

- **WHEN** a contributor adds a module in a folder beside the layers
- **THEN** frontend lint fails and names that file

#### Scenario: The allowed root files stay

- **WHEN** frontend lint checks the application's entry module, ambient type declaration and global style sheet
- **THEN** it reports no finding for them

### Requirement: Frontend checks enforce effect, rendering and interaction rules

Frontend lint SHALL fail when an effect or memoized value omits a value it reads, when a reference is read or written during rendering, when rendering calls an impure function or mutates a value, when a component file exports something other than components, when a non-interactive element handles a click, or when an image has no text alternative. A single line MAY be exempted from one of these rules with a disable comment, and code review SHALL require that comment to state its reason.

#### Scenario: An effect omits a dependency

- **WHEN** a contributor writes an effect that reads a changing value it does not declare
- **THEN** frontend lint fails

#### Scenario: Rendering touches a reference or the clock

- **WHEN** a component reads or writes a reference, calls an impure function or mutates a value during rendering
- **THEN** frontend lint fails

#### Scenario: A component file exports a non-component

- **WHEN** a component file also exports a value that is not a component or a constant
- **THEN** frontend lint fails

#### Scenario: A plain element handles a click

- **WHEN** a contributor adds a click handler to a non-interactive element
- **THEN** frontend lint fails

#### Scenario: An image has no text alternative

- **WHEN** a contributor adds an image with no text alternative attribute
- **THEN** frontend lint fails

#### Scenario: One line is exempted

- **WHEN** a single line carries a disable comment for one of these rules
- **THEN** frontend lint accepts that line and still rejects the same finding on the next line
