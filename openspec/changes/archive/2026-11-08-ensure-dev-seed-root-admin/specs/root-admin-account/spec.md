## MODIFIED Requirements

### Requirement: The instance recognizes its bootstrap administrator independently of that account's email address

The instance SHALL record which account is the bootstrap administrator when it creates that account, and SHALL keep recognizing it by that record. Authorization decisions SHALL use that recorded identity rather than the text of any account's email address. The local development seed specified by `local-development-workflow` is the sole exception to preserving this designation: after authenticating the configured dev account, it SHALL explicitly transfer the designation to that account. An email change alone SHALL NOT transfer privileges. This covers, at minimum, every privilege the requirement below reserves for the bootstrap administrator.

#### Scenario: The bootstrap administrator keeps its role after changing address

- **WHEN** the bootstrap administrator changes their email address
- **THEN** the instance still treats that account as the bootstrap administrator

#### Scenario: Restarting the instance

- **WHEN** the backend is restarted without running the local development seed
- **THEN** no account is created, and the recorded bootstrap administrator is unchanged

### Requirement: The bootstrap administrator keeps its powers over other administrators after changing address

Granting or revoking the administrator role, deactivating an account that holds it, and reactivating one SHALL remain reserved for the bootstrap administrator, and SHALL be permitted on the basis of the instance's record of which account that is. Changing the bootstrap administrator's email address SHALL NOT remove any of these powers, and SHALL NOT grant them to any other account.

A production host-console exception is creating a new account with the administrator role from the host console, which SHALL be permitted to a caller who can execute commands inside the running instance without the bootstrap administrator's involvement. It sits at the trust level of the console password reset, which can already take over the bootstrap administrator's own account. The exception covers only new accounts: the console SHALL NOT grant or revoke the role of an existing account, and SHALL NOT deactivate or reactivate one.

The local development seed has the separate, narrow exception specified by `local-development-workflow`: it SHALL grant the dev account the administrator role and transfer the bootstrap designation after successful authentication. The previous bootstrap administrator SHALL retain its administrator role. Ordinary account-management operations and production console commands SHALL NOT expose this transfer.

An overlapping account-management request SHALL NOT apply privileges or bootstrap-protection decisions made stale by this transfer. When the transfer completes before that request's protected administration checks, the request SHALL use the new bootstrap designation: it SHALL NOT demote or deactivate the new holder or allow the previous holder to manage administrators. When administration completes first, the seed SHALL validate the resulting account state before assigning privileges and SHALL refuse an inactive target.

#### Scenario: Development seeding transfers administrator-management powers

- **WHEN** the local development seed transfers the bootstrap designation from another administrator to the dev account
- **THEN** the dev account can grant or revoke another account's administrator role
- **AND** the previous holder remains an administrator but cannot manage administrator roles
- **AND** the audit log identifies the transfer

#### Scenario: An overlapping request cannot demote the new bootstrap administrator

- **WHEN** a request to demote the dev account overlaps a development-seed transfer and the transfer commits before the request's protected administration checks
- **THEN** the request is refused and the dev account remains the active bootstrap administrator with the administrator role

#### Scenario: An overlapping request cannot deactivate the new bootstrap administrator

- **WHEN** a request to deactivate the dev account overlaps a development-seed transfer and the transfer commits before the request's protected administration checks
- **THEN** the request is refused and the dev account remains the active bootstrap administrator with the administrator role

#### Scenario: The previous holder cannot keep its powers through an overlapping request

- **WHEN** the previous bootstrap administrator starts a request to grant another account the administrator role and the seed transfer commits before that request's protected administration checks
- **THEN** the request is refused and the third account's role is unchanged

#### Scenario: Deactivation completes before the seed transfer

- **WHEN** another administrator deactivates the dev account after seed authentication but before the seed validates it for transfer
- **THEN** the seed refuses the assignment and preserves the previous bootstrap administrator

#### Scenario: Appointing an administrator after the address has changed

- **WHEN** the bootstrap administrator, having changed its address, grants the administrator role to another user
- **THEN** the role is granted

#### Scenario: Another administrator still cannot manage administrators

- **WHEN** an account that holds the administrator role but is not the bootstrap administrator tries to grant the role, revoke it, or deactivate or reactivate an account holding it
- **THEN** every one of those attempts is refused, whatever address the bootstrap administrator currently holds

#### Scenario: The bootstrap administrator cannot be stripped of its role

- **WHEN** any account attempts to revoke the administrator role from the bootstrap administrator or to deactivate it
- **THEN** the attempt is refused, so the instance is never left without a recognized bootstrap administrator

#### Scenario: The console cannot promote an existing account

- **WHEN** an operator runs the console create command with the administrator role for an address that a member account already holds
- **THEN** the command fails, and that account is still a member

