## ADDED Requirements

### Requirement: The Helm chart lets a slow start finish

The supplied Helm chart SHALL give the application up to 10 minutes to start answering on its port before it counts as failed, and the liveness and readiness probes SHALL apply only after that start has succeeded. Once started, an instance that stops answering SHALL be restarted as quickly as the liveness probe settings say. An operator SHALL be able to change or disable this startup allowance through the chart's values.

#### Scenario: A start takes longer than the liveness allowance

- **WHEN** the application needs several minutes for its migrations and startup work before it opens its port
- **THEN** the pod is not restarted during that time and receives traffic once the port answers

#### Scenario: An instance hangs after starting

- **WHEN** an instance that has started stops answering
- **THEN** the liveness probe restarts it under its own settings, without waiting for the startup allowance

#### Scenario: An operator disables the allowance

- **WHEN** an operator renders the chart with the startup probe disabled
- **THEN** the workload has no startup probe and the liveness and readiness probes stay as configured

#### Scenario: A start never finishes

- **WHEN** the port has not answered 10 minutes after the container starts
- **THEN** Kubernetes restarts the container

#### Scenario: An operator changes the allowance

- **WHEN** an operator renders the chart with a different startup probe period or failure threshold
- **THEN** the workload's startup probe carries those values
