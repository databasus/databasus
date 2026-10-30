## Purpose

Keep frontend dependency boundaries enforceable during gradual restructuring, while allowing only the specific existing violations scheduled for removal.

## ADDED Requirements

### Requirement: Frontend checks reject forbidden dependencies

The frontend lint command SHALL reject a new dependency directed upward through the application, pages, widgets, features, entities and shared layers. It SHALL reject dependencies between different slices of the same layer and access to a slice's private internals from another slice. These checks SHALL cover ordinary, type-only and dynamic imports and re-exports, with equivalent results for relative and configured alias imports. Re-exporting a forbidden dependency through a public entry point MUST NOT authorize it.

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

### Requirement: Existing violations form a finite shrinking baseline

The checks SHALL permit only individually recorded existing findings, each with an owner and a removal milestone. Baseline matching SHALL distinguish the rule, source, dependency or structural location, and occurrence count. The baseline MUST NOT exempt entire layers, slices, files or nested domain groups. New findings, increased occurrence counts and stale entries SHALL fail verification. Unchanged violations SHALL remain recognizable after unrelated line movement.

#### Scenario: A new violation appears beside an accepted one

- **WHEN** a file containing a recorded violation adds another forbidden dependency or another occurrence of the same dependency
- **THEN** frontend lint fails

#### Scenario: A dependency inside an existing nested group violates its intended boundary

- **WHEN** code adds an unrecorded dependency between two scenario slices inside a nested domain group
- **THEN** frontend lint fails even if the enclosing group already has recorded structural findings

#### Scenario: A violation is repaired

- **WHEN** a recorded violation no longer exists but its baseline entry remains
- **THEN** frontend lint fails until the stale entry is removed

#### Scenario: An unrelated edit moves a violation

- **WHEN** an unchanged recorded dependency moves to a different line
- **THEN** the baseline still recognizes the same finding without authorizing any additional finding

### Requirement: Architecture checks preserve the existing verification surface

Adding dependency checks SHALL preserve formatting, source lint, type checking and generated API freshness verification in the frontend lint command. Existing application addresses, screens and domain operations SHALL remain available after the mechanical layer rename.

#### Scenario: Generated API types are stale

- **WHEN** frontend dependency checks pass but generated API types differ from the contract
- **THEN** the frontend lint command fails

#### Scenario: A renamed layer is used by an existing screen

- **WHEN** an existing screen loads after the layer rename
- **THEN** its domain requests and available interactions remain unchanged
