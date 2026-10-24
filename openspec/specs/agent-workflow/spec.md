# agent-workflow Specification

## Purpose
Defines how an agent finishes work in this repository: it refactors the code it touches instead of listing suggestions, reports the result rather than the review history, packs a finished change into one commit on request, keeps the user able to see and stop what it runs, and keeps lasting rules where every agent and person can read them.

## Requirements

### Requirement: The agent refactors touched code within the change
When code the change touches stands in the way of the change or breaks the rules of the module docs or `docs/architecture.md`, the agent SHALL refactor it while implementing the change, without asking the user and without a separate task for later. The agent's summary MUST NOT list refactoring suggestions for code inside the change. The summary SHALL name only a refactoring that goes beyond the change, together with what it would cost, because such a refactoring needs the user's agreement first.

#### Scenario: Duplicated code in a touched function
- **WHEN** the agent edits a function and finds that it duplicates a helper next to it
- **THEN** the agent replaces the duplicate with the helper in the same change, and its summary lists no refactoring suggestion about it

#### Scenario: A rework beyond the change
- **WHEN** the agent finds that a module the change does not touch needs a larger rework
- **THEN** the agent does not do the rework, and its summary names it with its cost for the user to decide

### Requirement: The user summary describes the result, not the review
After a review hook blocked the turn and the agent fixed or rejected the findings, the summary the user reads SHALL describe the final state of the result: what was done, which decisions were made, which checks passed, what remains and the next step. The summary MUST NOT retell review rounds, the findings or the fixes made for them. After a quick check blocks a finished change, the agent's next reply SHALL be such a summary. After a quick check blocks a plan, the agent SHALL revise the plan and show it again. After a full review blocks the turn, whether a reviewer found problems or a check before the reviewers failed, the agent's next reply SHALL be one short line about its fixes that names every rejected finding with the reason, because the repeat review reads that reply; the summary comes once the full review accepts the result.

#### Scenario: The diff check blocks a turn
- **WHEN** the diff check blocks the turn because a test is missing, and the agent adds the test and finishes the turn again
- **THEN** the agent's reply is a summary of the finished work for the user and does not mention the diff check or its finding

#### Scenario: A quick check blocks a plan
- **WHEN** the necessity check blocks a plan the agent submits through ExitPlanMode
- **THEN** the block text asks the agent to revise the plan and show it again, not to write a summary of a result

#### Scenario: The full implementation review blocks a turn
- **WHEN** the full implementation review blocks the turn with two findings, and the agent fixes one and rejects the other
- **THEN** the agent's reply is one short line that names the fix and the rejected finding with its reason

#### Scenario: A linter fails before the full review
- **WHEN** the backend linter fails before the full implementation review starts its reviewers, and the agent fixes the lint error
- **THEN** the agent's reply is one short line about the fix

### Requirement: The pack command archives and commits a finished change
When the user asks to pack a change, in whatever language the request is written, and every task of the change is checked, the agent SHALL archive the change with its delta specs synced into the main specs, and commit the implementation, the main specs and the archived change in one commit whose message follows the repository's commit format. The request SHALL count as the user's consent to the sync and to that commit, so the agent does not ask about either again. The agent MUST NOT include changes it did not make within the user's requests, and MUST NOT push. When a task of the change is unchecked, the agent MUST NOT sync, archive or commit anything, and SHALL name the unchecked tasks to the user.

#### Scenario: Pack a finished change
- **WHEN** every task of a change is checked and the user writes "pack"
- **THEN** the main specs contain the change's requirements, the change sits in the archive, and one new commit holds the code, the specs and the archive, with nothing pushed

#### Scenario: Pack an unfinished change
- **WHEN** a change has two unchecked tasks and the user writes "pack"
- **THEN** the main specs, the change directory and the Git history stay as they were, and the agent names the two unchecked tasks

#### Scenario: Another agent's edit in the working tree
- **WHEN** the user asks to pack a change while the working tree also holds an edit the agent did not make
- **THEN** the commit leaves that edit out, and it stays uncommitted in the working tree

### Requirement: The user sees and owns what the agent runs
The agent SHALL run tests, linters and builds as ordinary foreground commands, so the user sees which command runs and that the agent is working. When the agent starts the backend, the frontend or another long-lived process to check its work, it SHALL stop that process before it finishes the task. Long-lived processes the user wants running are started by the user.

#### Scenario: The agent runs package tests
- **WHEN** the agent runs the tests of a backend package
- **THEN** the command runs in the foreground of the agent's console, not as a background task

#### Scenario: The agent started the frontend dev server
- **WHEN** the agent started the frontend dev server to check a page
- **THEN** the dev server is no longer running when the agent hands the task over

### Requirement: Lasting rules live in the repository
The agent MUST NOT write rules, preferences or facts to its local memory outside the repository. A rule or preference that must outlive the session SHALL go, with the user's consent, into an `AGENTS.md` file or a repository skill, where every agent and person working in the repository reads it.

#### Scenario: The user states a lasting preference
- **WHEN** the user says that every future change should do something a certain way
- **THEN** the agent proposes the edit to `AGENTS.md` or a skill and writes nothing to its local memory

### Requirement: Every repository skill has one source file
Every skill of the repository, including the generated `openspec-*` skills, SHALL have one source file under `.agents/skills/`. Claude Code SHALL load it through `.claude/skills`, a symlink to that directory, so no copy can drift from the source.

#### Scenario: The commit skill in Claude Code
- **WHEN** Claude Code lists the skills it can load in this repository
- **THEN** the list contains the Git commits skill, read from `.agents/skills/git-commits/SKILL.md`

### Requirement: The agent can see and drive the local app

The repository SHALL register one browser automation server for Claude Code. With it, an agent opens pages of the local app and reads what the page shows, including a screenshot, the page structure, console messages and network requests. It also clicks, types and navigates. Claude Code SHALL start the server from the repository configuration. The server SHALL drive the Chrome browser installed on the host. Each agent session SHALL get its own temporary browser profile, which is discarded when the session's browser closes, so that two sessions running at once do not block each other and no sign-in carries over between sessions.

The repository SHALL hold a skill that tells the agent how to check the app in the browser:

- When the backend and frontend ports already answer from a `just run` started in the agent's own worktree, the agent uses that app.
- When they answer from another worktree, the agent leaves that app running, does not run the browser check, and tells the user the check was skipped because the running app serves another worktree's code.
- When only one of the two ports answers, the agent does not start `just run` and does not run the browser check. It tells the user which port is held and that the check was skipped.
- When neither port answers, the agent starts `just run` without its interactive window and stops it before it finishes the task, leaving neither port held.
- The agent signs in with the seeded dev account.
- When the app of the agent's own worktree lacks the dev account or a seeded database, the agent reruns the seed command against it instead of creating the items by hand.

A browser check SHALL add to the tests a task names and SHALL NOT replace them.

#### Scenario: Agent checks a page with no app running

- **WHEN** an agent needs to see a page and nothing answers on the backend and frontend ports
- **THEN** the agent starts `just run` without its interactive window, signs in as the dev account, opens the page and reads what it shows
- **AND** no process of that run remains when the agent hands the task over

#### Scenario: The developer's app runs in the same worktree

- **WHEN** an agent needs to see a page and the running app was started in the agent's own worktree
- **THEN** the agent uses that app and leaves it running when it hands the task over

#### Scenario: The running app belongs to another worktree

- **WHEN** an agent needs to see a page and the running app was started in another worktree
- **THEN** the agent does not stop that app and does not open the page
- **AND** the agent tells the user that the browser check was skipped because the app serves another worktree's code

#### Scenario: Only one port answers

- **WHEN** an agent needs to see a page, the frontend port answers and the backend port does not
- **THEN** the agent does not start `just run` and does not open the page
- **AND** the agent tells the user that the frontend port is held and the browser check was skipped

#### Scenario: A seeded database is missing

- **WHEN** an agent uses the app of its own worktree and the WAL-streaming database is missing from it
- **THEN** the agent reruns the seed command, and the database appears with backups switched on

#### Scenario: Two agents use the browser at once

- **WHEN** two agent sessions open the browser at the same time
- **THEN** both browsers start, each with its own empty profile

### Requirement: The developer chooses a visible or headless browser

The browser the agent drives SHALL run headless unless the developer asks otherwise. The `BROWSER_HEADLESS` setting in the developer's `.env` file SHALL choose headless when `true` and a visible window when `false`. A value set in the environment the agent client starts from SHALL take precedence over the file. When neither sets it, the browser SHALL run headless. Any value other than `true` or `false` SHALL keep the browser server from starting, with an error that names `BROWSER_HEADLESS`. The example environment file SHALL contain the setting with its default value. The choice SHALL take effect the next time the agent client starts the browser server.

#### Scenario: Default setting

- **WHEN** neither the developer's `.env` file nor the environment sets `BROWSER_HEADLESS` and an agent opens a page
- **THEN** no browser window appears on the developer's screen

#### Scenario: Developer wants to watch

- **WHEN** the developer sets `BROWSER_HEADLESS=false` in `.env` and the agent client restarts the browser server
- **THEN** the agent's browser opens as a visible window

#### Scenario: Environment overrides the file

- **WHEN** `.env` sets `BROWSER_HEADLESS=true` and the agent client is started with `BROWSER_HEADLESS=false` in its environment
- **THEN** the browser opens as a visible window

#### Scenario: Invalid value

- **WHEN** `.env` sets `BROWSER_HEADLESS=maybe` and the agent client starts the browser server
- **THEN** the server does not start
- **AND** the error the client reports names `BROWSER_HEADLESS`

### Requirement: The quick architecture check covers the HTTP contract
The quick architecture check SHALL check a plan that adds or changes an HTTP operation, and a finished turn that edits an HTTP contract file under `backend/api/`, against the repository's API design skill, in addition to `docs/architecture.md`. It SHALL reject the plan or the turn for each broken rule and name the rule of the API design skill.

#### Scenario: A contract edit that breaks an API rule
- **WHEN** a turn adds an operation to `backend/api/openapi.yaml` whose error responses do not use the error format the API design skill requires
- **THEN** the quick architecture check rejects the turn and names the rule of the API design skill

#### Scenario: A plan with a new operation
- **WHEN** a plan proposes an HTTP operation that lists a collection without the pagination the API design skill requires
- **THEN** the quick architecture check rejects the plan and names the pagination rule

#### Scenario: A turn without contract edits
- **WHEN** a turn changes only Go files outside `backend/api/` and adds no HTTP operation
- **THEN** the quick architecture check judges the turn against `docs/architecture.md` and the module docs only
