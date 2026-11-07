## ADDED Requirements

### Requirement: Module contract aliases and constants have consumers

The architecture check SHALL reject an exported alias or constant in a domain module's public contract unless it has a reference outside its owning domain module or is required transitively by the declarations of a public contract with external consumers. Declaration dependencies include signatures, type definitions and declaration initializers; references confined to function bodies inside the owning module SHALL NOT establish a public contract need.

External consumers SHALL include other domain modules, application composition, commands, test-support packages and tests outside the owning module. The owning module's implementation and its own tests SHALL NOT count as external consumers, even when a test uses a separate test package.

Dependencies within the same public contract SHALL be traced from externally used declarations. A reference from an unused declaration, including a chain or cycle of such references without an external consumer, SHALL NOT make an alias or constant acceptable. A used declaration in a group SHALL NOT automatically make the other declarations in that group acceptable.

For each rejected declaration, the check SHALL report the declaration's file, line and name and identify the violated rule. The existing backend lint command SHALL fail when these violations exist. The check SHALL NOT modify source files.

#### Scenario: An unused alias

- **WHEN** a module contract declares an exported alias with no external consumer and no dependency path from an externally used contract declaration
- **THEN** backend lint fails and identifies the unused alias and its declaration location

#### Scenario: An unused constant

- **WHEN** a module contract declares an exported constant with no external consumer and no dependency path from an externally used contract declaration
- **THEN** backend lint fails and identifies the unused constant and its declaration location

#### Scenario: A consumer renames an import

- **WHEN** a package outside the owning module references a contract alias or constant through a renamed import
- **THEN** the usage check accepts that declaration

#### Scenario: A matching word does not refer to the declaration

- **WHEN** the only apparent uses of a declaration's name are comments, string literals or references to a different declaration with the same name
- **THEN** those occurrences do not make the declaration acceptable

#### Scenario: Application composition or a command is the only consumer

- **WHEN** an alias or constant is referenced only by application composition or an application command outside the owning module
- **THEN** the usage check accepts that declaration

#### Scenario: An external test or test-support package is the only consumer

- **WHEN** an alias or constant is referenced only by tests or test support outside the owning domain module
- **THEN** the usage check accepts that declaration

#### Scenario: Only the owning module uses the declaration

- **WHEN** all references to an exported alias or constant are confined to the owning module's implementation or its own tests, and no externally used contract declaration requires it
- **THEN** backend lint rejects the declaration

#### Scenario: A callback alias is needed by a signature

- **WHEN** an externally used contract has a subscription operation whose signature requires a callback alias, and consumers never name the alias directly
- **THEN** the usage check accepts that alias

#### Scenario: A contract dependency is indirect

- **WHEN** an externally used declaration requires another contract declaration which in turn requires an alias or constant
- **THEN** the usage check accepts that alias or constant

#### Scenario: Unused declarations refer to each other

- **WHEN** otherwise unused declarations in one contract refer to an alias or constant directly or through a dependency chain or cycle
- **AND** no declaration in that chain has an external consumer
- **THEN** those references do not make the alias or constant acceptable

#### Scenario: A constant group contains used and unused declarations

- **WHEN** a contract declares a group of constants and only some have consumers or required declaration dependencies
- **THEN** the usage check rejects the unused declarations and accepts the used ones

#### Scenario: A function body is the only local use

- **WHEN** an exported alias is used only in a function body in its owning module and is absent from the externally used contract's declaration dependencies
- **THEN** the usage check rejects the alias

### Requirement: Contract usage analysis reports an incomplete analysis

The contract usage check SHALL consider the default build for the host platform, including test consumers and packages outside the server's production dependency graph. A failure to load or resolve the source needed to analyze contract usage SHALL fail the check with an analysis error rather than report success or invent unused-declaration findings from incomplete information. Diagnostics for the same input SHALL have stable order and SHALL NOT be duplicated by the presence of test variants.

#### Scenario: A test-only consumer is outside the server dependency graph

- **WHEN** a package outside the server's production dependency graph uses an alias or constant in an external test
- **THEN** the usage check counts that reference

#### Scenario: Required source cannot be resolved

- **WHEN** the usage analysis cannot resolve a required package or declaration
- **THEN** the check fails with an analysis error and does not report the affected declarations as unused

#### Scenario: Production and test variants share a declaration

- **WHEN** source loading includes production and test variants of a package that declares one unused contract alias
- **THEN** the check reports that alias once in stable diagnostic order
