## ADDED Requirements

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
