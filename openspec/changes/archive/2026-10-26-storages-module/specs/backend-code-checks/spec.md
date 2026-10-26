## MODIFIED Requirements

### Requirement: Code awaiting its move keeps the checks it had
For the rules of the previous requirement, code outside the target layout SHALL be checked exactly as before this change: its production files by the linters that checked them before, and its test files by none. The lint configuration SHALL list the target layout explicitly, so code added outside it falls under the old checks and code added inside it falls under every rule. Two rules of the architecture check also cover code outside the target layout: the production binary contains no test code, and test helpers behind the test helper build tag exist only outside the target layout.

#### Scenario: An old package with an existing violation of a new rule
- **WHEN** a production file of a package outside the target layout calls a fixed sleep, as it did before this change
- **THEN** the backend lint passes

#### Scenario: An old package breaks a rule it was checked by before
- **WHEN** a production file outside the target layout ignores a returned error
- **THEN** the backend lint fails, as it did before this change

#### Scenario: A new directory in the target layout
- **WHEN** a developer adds a new domain module in the target layout
- **THEN** every rule applies to it without a change to the lint configuration

## REMOVED Requirements

### Requirement: Old backup code writes stored files only through the file store

**Reason**: The storage providers live under `internal/` of the storages module, so Go's module boundary already keeps backup code from calling a provider's save or delete method, and no old backup code can reach a provider any more.

**Migration**: The Go compiler's rule for `internal/` packages replaces this check: a package outside the storages module that imports a provider fails to build. Backup code writes, reads and deletes stored files through the file store that `storages.Module` offers.
