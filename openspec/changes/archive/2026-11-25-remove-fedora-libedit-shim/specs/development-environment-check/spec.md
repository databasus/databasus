# Spec Delta

## ADDED Requirements

### Requirement: The check verifies that the client tool bundles find their libraries

The check SHALL fail when a client tool bundle of the machine's architecture holds a program that needs a shared library the system cannot provide. It SHALL look up libraries as the dynamic loader finds them in the caller's environment, without creating aliases for missing ones, and SHALL name each program and missing library.

#### Scenario: A system library is missing

- **WHEN** a program of a client tool bundle needs a shared library that is not installed
- **THEN** the client tool bundles check fails and names the program and the missing library

## REMOVED Requirements

### Requirement: The check verifies that the client tool bundles can start

**Reason**: The run and test commands no longer apply a library compatibility step, so the check has none to account for.

**Migration**: None. The check keeps failing on a missing library and naming it, under the requirement "The check verifies that the client tool bundles find their libraries".
