## MODIFIED Requirements

### Requirement: Root commands lint and format every module

The repository SHALL provide `mise run lint` and `mise run format`, covering the backend, whose module includes the verification agent, the frontend, and all repository-owned Python. `mise run lint` SHALL exit with a non-zero status when any area fails its checks, and SHALL report the failures of every failing area, not only the first. `mise run format` SHALL rewrite files in these areas to their formatter's output. Each area SHALL also have its own lint and format tasks: `mise run backend:lint`, `mise run frontend:lint`, `mise run python:lint` and the matching `format` tasks. Every task SHALL use its area's working directory whichever directory of the repository it is started from. The backend's lint and format tasks SHALL check and format the verification agent's code together with the server's, with the same rules, its end-to-end scenarios included; there SHALL be no separate lint or format task for the agent. The Python tasks SHALL remain hidden internal tasks and SHALL NOT add everyday commands to the default task listing.

#### Scenario: One module fails lint

- **WHEN** the frontend has a lint error and the other areas are clean
- **THEN** `mise run lint` exits with a non-zero status and reports the frontend error

#### Scenario: Two modules fail lint

- **WHEN** both the backend and the frontend have a lint error
- **THEN** `mise run lint` exits with a non-zero status and reports the errors of both modules

#### Scenario: The verification agent fails lint

- **WHEN** the verification agent's code breaks a rule of the backend lint and the rest of the backend is clean
- **THEN** `mise run backend:lint` and `mise run lint` exit with a non-zero status and report the agent's error

#### Scenario: Developer formats everything

- **WHEN** a developer runs `mise run format`
- **THEN** a later `mise run lint` reports no formatting difference in the backend, the frontend or repository-owned Python, the verification agent included

#### Scenario: Module task started from another directory

- **WHEN** a developer runs `mise run backend:lint` from the frontend directory
- **THEN** the backend checks run against the backend module and pass or fail as they would from the backend directory

#### Scenario: Python and another area fail lint

- **WHEN** Python has a type error and the frontend has a lint error
- **THEN** `mise run lint` fails and reports both errors

#### Scenario: Python task started from a module

- **WHEN** a developer runs `mise run python:lint` or `mise run python:format` from the backend or frontend directory
- **THEN** the task covers the same repository-owned Python as when started from the repository root

### Requirement: Tool versions are pinned in the repository

The repository SHALL pin the exact versions of a defined tool set, so that two hosts set up from the same commit run the same versions of those tools. The pinned set SHALL cover the backend and frontend toolchains, the commit hook runner, the process window, the migration tool, the Go linter, the API contract linter, the backend file watcher, the browser automation server agents use, the package manager of the browser end-to-end suite, and the Python runtime, linter, formatter and type checker. The version manager that installs these tools also runs the repository's tasks, and the repository SHALL name the minimum version of it that the tasks need. An older version manager SHALL refuse to run a task and name the minimum version. Docker, the shell, the Chrome browser and system libraries such as `libedit` come from the host and are not pinned.

#### Scenario: New host setup

- **WHEN** a developer installs the pinned tools on a new Fedora host and runs `mise run lint`
- **THEN** every tool from the pinned set that the task calls resolves to its pinned version

#### Scenario: The version manager is too old

- **WHEN** a developer whose version manager is older than the minimum the repository names runs `mise run lint`
- **THEN** no check runs, and the output names the minimum version

#### Scenario: Browser server starts at its pinned version

- **WHEN** an agent client starts the browser automation server registered in the repository
- **THEN** the server runs at the version pinned in the repository, without downloading a newer one

#### Scenario: The suite's package manager

- **WHEN** a developer runs `mise run e2e:browser` after installing the pinned tools
- **THEN** the suite's dependencies install through the pinned package manager from the suite's lockfile

#### Scenario: A Python quality tool is updated

- **WHEN** a developer changes the repository pin of a Python quality tool
- **THEN** local lint, the commit hook and CI use the new version without independent version changes

## ADDED Requirements

### Requirement: Python checks cover existing and newly added repository code

The Python lint command SHALL check formatting, lint rules, required annotations and type correctness for every repository-owned Python source file, including infrastructure scripts, agent hooks, tests, fixtures and browser scenarios. Its selection SHALL include tracked files and new eligible untracked files, including hidden directories and newly added directories, without maintaining a per-file inventory. It SHALL exclude deleted files, ignored generated artifacts and dependency environments. It SHALL report the results of all Python quality stages even when an earlier stage reports a violation, and SHALL fail if any stage fails. A missing tool or unavailable dependency environment SHALL fail the check rather than silently skip a stage.

#### Scenario: A new script is untyped

- **WHEN** a developer adds an untracked Python script containing a named function without annotations in a new repository directory
- **THEN** Python lint fails and identifies its missing annotations without any configuration update

#### Scenario: A hidden hook has a type error

- **WHEN** a repository-owned Python file in a hidden directory passes a value incompatible with an annotated parameter
- **THEN** Python lint fails and identifies that type error

#### Scenario: Multiple quality stages fail

- **WHEN** Python source has both a formatting difference and a type error
- **THEN** Python lint reports both failures and exits with a non-zero status

#### Scenario: Dependencies exist only in the suite environment

- **WHEN** a developer runs Python lint with no active Python virtual environment
- **THEN** lint prepares the browser suite's dependencies from its committed lockfile and resolves them for type checking
- **AND** it starts no Docker container, application service or browser

#### Scenario: A file is deleted or ignored

- **WHEN** a repository-owned Python file has been deleted and an ignored dependency environment contains Python files
- **THEN** Python lint checks neither the deleted file nor files inside that environment

### Requirement: Python annotations describe the values the code uses

All repository-owned named Python functions and methods SHALL declare parameter and return types, including private functions, test methods, fixtures, generators and callbacks. Conventional instance and class receiver parameters SHALL use inferred receiver types. Collections and process handles SHALL specify their value types where inference does not establish them. Structured values exchanged within repository code SHALL have defined shapes, and external dynamic values SHALL be validated or narrowed before being treated as those shapes. New untyped functions, incompatible assignments, arguments and returns SHALL fail quality checks. The migration SHALL cover existing code without per-directory exemptions, accepted-error baselines, blanket suppressions or dynamic placeholder annotations that bypass checking. Any necessary suppression SHALL identify a specific rule and an evidenced external limitation beside the affected expression.

#### Scenario: A test fixture loses its annotations

- **WHEN** a developer removes a fixture's parameter or return annotation
- **THEN** Python lint fails and identifies the missing annotation

#### Scenario: A typed function returns the wrong value

- **WHEN** a function declares a text return value but returns an integer
- **THEN** Python lint fails and identifies the incompatible return

#### Scenario: An annotation bypasses type checking

- **WHEN** a developer annotates a function parameter with an unrestricted dynamic placeholder
- **THEN** Python lint rejects that annotation

### Requirement: Python commit checks preserve working files

The commit hook SHALL run the complete Python lint command once when selected changes include a repository-owned Python file or configuration that controls Python quality, dependencies or hook selection. Changing one file SHALL check the full Python selection so errors in its callers are visible. A failing check SHALL block the commit. Neither Python lint nor the commit hook SHALL rewrite, stage or stash source or configuration files. Explicit formatting SHALL be the operation that applies formatting and safe automatic lint fixes. Repeating formatting on clean source SHALL produce no further changes.

#### Scenario: One selected file breaks another file

- **WHEN** a selected Python change makes an unchanged caller pass an incompatible argument
- **THEN** the commit hook runs the complete Python check and fails on that caller

#### Scenario: Python configuration changes without source changes

- **WHEN** selected changes alter Python quality rules, tool versions, dependency declarations, the dependency lockfile or hook selection configuration
- **THEN** the commit hook runs the complete Python check once

#### Scenario: A Python commit check fails with unstaged edits present

- **WHEN** a selected Python file fails the commit check while another file has unstaged edits
- **THEN** the commit is blocked and file contents, index contents and the stash remain unchanged

#### Scenario: An unrelated change is selected

- **WHEN** selected changes contain only unrelated documentation and affect no Python quality configuration
- **THEN** the Python commit check is not selected

#### Scenario: Formatting is repeated

- **WHEN** the developer runs Python formatting twice on the same source
- **THEN** the second run changes no file and the source passes the formatting check
