# Single-Instance Runtime Specification

## Purpose

Defines one application process as the supported installation topology so transient coordination remains consistent without a shared service.

## Requirements

### Requirement: Supplied deployments run one application process

Databasus installation artifacts SHALL start exactly one application process for each installation. They SHALL NOT expose a setting that increases the application replica count.

#### Scenario: Operator deploys the Helm chart

- **WHEN** an operator renders or installs the supplied Helm chart
- **THEN** the resulting workload contains exactly one application replica

#### Scenario: Operator uses the Docker image

- **WHEN** an operator starts one instance of the supplied Docker image
- **THEN** that instance runs exactly one Databasus application process

### Requirement: One process owns transient runtime state

The single application process SHALL own cache entries, publish-subscribe subscriptions and rate-limit counters for its installation. The supplied deployment SHALL NOT require another service to share that state.

#### Scenario: Installation starts normally

- **WHEN** the application process starts
- **THEN** its transient runtime providers are available inside that process
- **AND** startup does not wait for a shared transient-state service

### Requirement: The image is the whole installation

The supplied Docker image SHALL contain everything one installation runs: the backend, the web interface, the internal PostgreSQL database and the client tools of every supported engine version. Installing SHALL take one container with one data volume, and upgrading SHALL take a newer image on the same volume. The internal database SHALL run inside that container and SHALL NOT be replaced by an external server.

#### Scenario: Fresh install

- **WHEN** an operator starts the image with an empty data volume
- **THEN** Databasus starts with its internal database and serves the web interface without any other service

#### Scenario: Upgrade

- **WHEN** an operator replaces the container with a newer image on the same data volume
- **THEN** the internal database is migrated and every workspace, database, storage, notifier and backup remains
