# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: The environment check reports every tool the hooks need`
- TO: `### Requirement: mise run doctor reports the hook tools and the development environment`

## MODIFIED Requirements

### Requirement: mise run doctor reports the hook tools and the development environment

The repository SHALL have a command, `mise run doctor [--client <codex|claude>]`. Its first part, the hook tool check, reports the selected review client and, one line per tool, whether the hooks can find Python, git, OpenSpec and the selected client's CLI. It SHALL use the client-selection rules of this capability and MUST NOT require the other client's CLI. The hook tool check SHALL run through the same launcher as the hooks, and the launcher's doctor action SHALL cover the hook tool check only. After it, `mise run doctor`, not the launcher, SHALL run the development environment check and print its report, even when the hook tool check fails. The command SHALL exit with a non-zero code when a required tool is missing, the client cannot be selected or the development environment check fails. The version manager runs that command, so when the version manager itself cannot be found, the launcher run directly with the doctor action SHALL report that and exit with a non-zero code. Finding a CLI SHALL NOT be reported as proof that its authentication or model access works.

#### Scenario: Every tool is present

- **WHEN** the user runs `mise run doctor` with a selected client, every required tool installed and every development environment prerequisite met
- **THEN** the selected client and each required tool's path are reported and the command exits with code 0 without claiming that the client is authenticated or can reach its model

#### Scenario: Only Codex is installed

- **WHEN** `mise run doctor` selects Codex, Python, git, OpenSpec and Codex are available, Claude Code is absent and every development environment prerequisite is met
- **THEN** the command exits with code 0 without requiring Claude Code

#### Scenario: Only Claude Code is installed

- **WHEN** `mise run doctor` selects Claude Code, Python, git, OpenSpec and Claude Code are available, Codex is absent and every development environment prerequisite is met
- **THEN** the command exits with code 0 without requiring Codex

#### Scenario: No reviewer CLI

- **WHEN** the selected client's CLI is absent and the other client's CLI is installed
- **THEN** the command names the missing selected CLI, exits with a non-zero code and does not switch clients

#### Scenario: No version manager

- **WHEN** the version manager cannot be found and the user starts the launcher's doctor action directly
- **THEN** the command reports that the version manager was not found and exits with a non-zero code

#### Scenario: Broken development environment

- **WHEN** the user runs `mise run doctor`, every required tool is installed but a development environment prerequisite is missing
- **THEN** the command reports the tools, then the development environment report with the failed prerequisite, and exits with a non-zero code

#### Scenario: Missing tool and broken environment

- **WHEN** the user runs `mise run doctor`, the selected client's CLI is absent and a development environment prerequisite is missing
- **THEN** the command reports both problems in one run and exits with a non-zero code
