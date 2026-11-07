## Purpose

Defines how internal-database queries respect a caller's cancellation and deadlines while preserving the transaction selected by that caller.

## ADDED Requirements

### Requirement: Transactional queries honor the current caller's cancellation

An internal-database query joining an existing transaction SHALL observe the cancellation signal and deadline of the caller making that query. A shorter child deadline SHALL limit the query even when the transaction was opened with a longer-lived parent context. Binding the query to the child context SHALL NOT move it outside the existing transaction or change its commit and rollback ownership.

#### Scenario: A child deadline expires inside a transaction

- **WHEN** a transaction is opened with an active parent context and a query is issued with a child context whose shorter deadline expires
- **THEN** the query fails because its child deadline expired, even though the parent context is still active
- **AND** the transaction owner remains responsible for ending the transaction

#### Scenario: A child context is already cancelled

- **WHEN** a caller issues a query inside a transaction with an already-cancelled child context while the transaction's parent remains active
- **THEN** the query fails because the child was cancelled rather than running successfully under the parent's context

#### Scenario: Context binding preserves rollback

- **WHEN** a transaction writes through an active child context and its owner subsequently rolls back
- **THEN** the child's writes are rolled back with the transaction
- **AND** no query is redirected to an independent connection by the context binding
