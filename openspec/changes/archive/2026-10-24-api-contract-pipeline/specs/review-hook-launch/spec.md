## MODIFIED Requirements

### Requirement: Hooks run with the repository's pinned tools
Every review hook SHALL start through one launcher that runs the hook with the Python and OpenSpec versions pinned in the repository's tool versions file. The launcher SHALL find the version manager even when the client was started outside a login shell, for example from an IDE. The launcher MUST NOT install a missing tool itself, because an installation would not fit the hook timeout.

#### Scenario: A client started from an IDE
- **WHEN** Claude Code was started from an IDE whose environment does not activate the version manager, and the agent finishes a turn
- **THEN** the hooks run with the pinned Python and OpenSpec versions

#### Scenario: A pinned version is not installed
- **WHEN** the version manager is present but the pinned OpenSpec version is not installed, and the agent finishes a turn
- **THEN** no installation starts, the agent stops, and the user sees that the hooks did not start, that `mise install` installs the pinned tools and that `mise run doctor` checks the environment

### Requirement: A hook that cannot run stops the agent
When the launcher cannot find the version manager, when the hook exits with an error, or when a tool a gate needs is missing, the hook SHALL stop the agent and show the user the reason and the command that checks the environment. The hook MUST NOT let the turn pass as if the review had run.

#### Scenario: OpenSpec is missing
- **WHEN** the OpenSpec CLI cannot be found and a turn finishes that wrote the artifacts of a change
- **THEN** the agent stops, and the user sees that the hooks could not find `openspec` and that `mise run doctor` checks the environment

#### Scenario: The version manager is missing
- **WHEN** the launcher cannot find the version manager
- **THEN** the agent stops, and the user sees that the version manager was not found, that it has to be installed and that `mise run doctor` then checks the environment

#### Scenario: The hook crashes
- **WHEN** the hook process exits with a non-zero code
- **THEN** the agent stops, and the user sees which hook action failed, its exit code, and that `mise run doctor` checks the environment

### Requirement: The environment check reports every tool the hooks need
The repository SHALL have a command, `mise run doctor`, that reports, one line per tool, whether the hooks can find Python, git, OpenSpec and the Claude Code CLI that runs the reviewers. The command SHALL run through the same launcher as the hooks, and SHALL exit with a non-zero code when any of them is missing. The version manager runs that command, so when the version manager itself cannot be found, the launcher run directly as `.agent-hooks/run.sh doctor` SHALL report that and exit with a non-zero code.

#### Scenario: Every tool is present
- **WHEN** the user runs `mise run doctor` with every tool installed
- **THEN** each tool is reported with its path and the command exits with code 0

#### Scenario: No reviewer CLI
- **WHEN** the Claude Code CLI is not installed and the user runs `mise run doctor`
- **THEN** the command reports `claude` as not found and exits with a non-zero code

#### Scenario: No version manager
- **WHEN** the version manager cannot be found and the user runs `.agent-hooks/run.sh doctor`
- **THEN** the command reports that the version manager was not found and exits with a non-zero code
