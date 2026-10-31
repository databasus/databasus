## ADDED Requirements

### Requirement: A separate browser suite exercises the production frontend against a real backend

The repository SHALL expose an independently runnable browser end-to-end entry point that serves the production frontend and an isolated real backend at the same origin. The pilot SHALL sign in through the visible interface, reach the application shell and sign out. Account setup SHALL use the public API; browser requests for application operations SHALL reach the real backend rather than synthetic responses. Each test application SHALL own its database and temporary state. Runtime configuration SHALL load before the application bundle. The suite SHALL require no shared development app, production test route or modification of shared frontend build output. Fixture-based UI verification SHALL remain independently runnable without a backend.

#### Scenario: A developer runs the real-API pilot

- **WHEN** the end-to-end entry point runs with its documented local dependencies
- **THEN** the built application signs in, displays the authenticated shell and signs out against the test's real API
- **AND** the command returns success only after the scenario and owned-resource shutdown complete

#### Scenario: Two runs overlap

- **WHEN** independent end-to-end runs execute concurrently
- **THEN** their accounts, database state, ports, build outputs and temporary directories do not interfere

### Requirement: Browser end-to-end failures propagate through every lifecycle owner

The end-to-end entry point SHALL return a nonzero status for a failed scenario, startup/readiness failure or interrupted run. Its owners SHALL stop their child processes, browser containers and gateway and release their temporary resources on success, failure and interruption, following the existing test-database retention policy. Shared database services and other runs' resources SHALL remain running. Isolated negative probes SHALL establish failure propagation and cleanup through the actual entry point, including a probe that fails after resources become ready. Retained failure evidence SHALL exclude passwords, session tokens and authorization codes.

#### Scenario: A scenario fails after startup

- **WHEN** an isolated probe deliberately fails after the real backend and browser are ready
- **THEN** the public entry point returns failure with the intended diagnostic and its gateway, processes, containers and temporary build directories are gone
- **AND** unrelated services remain usable

#### Scenario: A run is interrupted

- **WHEN** an isolated run is interrupted after starting its owned resources
- **THEN** it returns failure, stops those resources and does not signal or remove another run's resources

#### Scenario: Ordinary backend checks run

- **WHEN** the ordinary backend test selections run without the separate browser selection
- **THEN** they require no browser or frontend runtime and do not execute the browser end-to-end scenarios
