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

### Requirement: Frontend internal module references use the source-root alias

Frontend lint SHALL reject a literal relative module reference when both its importing source file and its target are within the application source root. This SHALL cover ordinary imports, type-only declarations, import-type expressions, re-exports, literal dynamic imports and side-effect stylesheet imports, including references to sibling files. Diagnostics SHALL identify the source file and violated rule. The configured source-root alias SHALL be accepted when the dependency is otherwise permitted. A missing target SHALL NOT exempt a relative reference whose normalized location is within that root.

#### Scenario: A sibling module is imported relatively

- **WHEN** an application source file imports a neighboring module using a relative path
- **THEN** frontend lint fails and names the source and alias-convention rule

#### Scenario: A parent module is referenced relatively

- **WHEN** application source uses a parent-relative path to another module within the application source root
- **THEN** frontend lint fails

#### Scenario: A type or re-export uses a relative reference

- **WHEN** an internal module is referenced relatively through a type-only declaration, an import-type expression or a re-export
- **THEN** frontend lint fails for each reference

#### Scenario: A dynamic import uses a relative literal

- **WHEN** application source loads another internal module through a literal relative dynamic import
- **THEN** frontend lint fails

#### Scenario: A stylesheet is imported relatively

- **WHEN** application source imports an internal stylesheet for its side effects through a relative path
- **THEN** frontend lint fails

#### Scenario: A relative internal target does not exist

- **WHEN** application source references a missing module relatively and its normalized target location is within the application source root
- **THEN** frontend lint reports the alias-convention violation independently of module-resolution errors

#### Scenario: An allowed internal reference uses the alias

- **WHEN** a source file references an allowed internal target through the configured source-root alias
- **THEN** the alias-convention check accepts it
- **AND** dependency-boundary checks still reject upward, cross-slice and private-access violations

### Requirement: Alias enforcement respects tooling and resource path semantics

Alias enforcement SHALL leave package references, references originating outside the application source root and source-test references to tooling outside that root outside its scope. It SHALL NOT interpret stylesheet resource URLs, module-relative resource URLs or source-code strings inside test fixtures as module declarations in the containing file. These exclusions SHALL NOT exempt an otherwise in-scope reference in the same file or disable existing dependency checks.

#### Scenario: A source test imports external tooling

- **WHEN** a test within application source imports a repository lint tool outside the application source root using a relative path
- **THEN** the alias-convention check accepts that reference
- **AND** still rejects a relative internal module reference added to the same test

#### Scenario: Standalone tooling uses a relative import

- **WHEN** a frontend configuration, browser-test infrastructure file or standalone tool outside application source uses a relative import
- **THEN** the alias-convention check does not reject it

#### Scenario: A package reference is used

- **WHEN** application source imports a package or a package subpath
- **THEN** the alias-convention check accepts that reference

#### Scenario: A resource is located relative to its owner

- **WHEN** a stylesheet locates a resource relative to itself or source code constructs a resource URL relative to its module
- **THEN** alias enforcement leaves that resource lookup unchanged

#### Scenario: A lint test contains relative source in a fixture

- **WHEN** a test stores source code containing a relative module reference in a string used as a lint fixture
- **THEN** the string is not reported as an import of the containing test file

### Requirement: Alias migration preserves frontend behavior and verification

Migrating existing internal module references to the configured alias SHALL preserve their targets, imported and exported symbols, runtime evaluation order, existing screens and interactions. Frontend lint SHALL retain formatting, source, dependency, style, type and generated-contract freshness checks. Dependency checks SHALL continue to recognize violations expressed through either relative references or the alias; alias enforcement SHALL NOT replace them.

#### Scenario: A migrated screen loads

- **WHEN** an existing screen loads after its module references are converted to the alias
- **THEN** its requests and available interactions remain unchanged

#### Scenario: Both spellings express a forbidden dependency

- **WHEN** dependency checks inspect equivalent forbidden relative and alias references
- **THEN** both references produce dependency violations
- **AND** the relative internal reference also violates the alias convention

#### Scenario: Another frontend check fails

- **WHEN** alias-convention checks pass but an existing formatting, source, dependency, style, type or generated-contract freshness check fails
- **THEN** the frontend lint command still exits with a failure
