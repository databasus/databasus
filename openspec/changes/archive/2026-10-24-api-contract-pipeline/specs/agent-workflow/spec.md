## MODIFIED Requirements

### Requirement: The agent can see and drive the local app

The repository SHALL register one browser automation server for Claude Code. With it, an agent opens pages of the local app and reads what the page shows, including a screenshot, the page structure, console messages and network requests. It also clicks, types and navigates. Claude Code SHALL start the server from the repository configuration. The server SHALL drive the Chrome browser installed on the host. Each agent session SHALL get its own temporary browser profile, which is discarded when the session's browser closes, so that two sessions running at once do not block each other and no sign-in carries over between sessions.

The repository SHALL hold a skill that tells the agent how to check the app in the browser:

- When the backend and frontend ports already answer from a `mise run dev` started in the agent's own worktree, the agent uses that app.
- When they answer from another worktree, the agent leaves that app running, does not run the browser check, and tells the user the check was skipped because the running app serves another worktree's code.
- When only one of the two ports answers, the agent does not start `mise run dev` and does not run the browser check. It tells the user which port is held and that the check was skipped.
- When neither port answers, the agent starts `mise run dev` without its interactive window and stops it before it finishes the task, leaving neither port held.
- The agent signs in with the seeded dev account.
- When the app of the agent's own worktree lacks the dev account or a seeded database, the agent reruns the seed command against it instead of creating the items by hand.

A browser check SHALL add to the tests a task names and SHALL NOT replace them.

#### Scenario: Agent checks a page with no app running

- **WHEN** an agent needs to see a page and nothing answers on the backend and frontend ports
- **THEN** the agent starts `mise run dev` without its interactive window, signs in as the dev account, opens the page and reads what it shows
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
- **THEN** the agent does not start `mise run dev` and does not open the page
- **AND** the agent tells the user that the frontend port is held and the browser check was skipped

#### Scenario: A seeded database is missing

- **WHEN** an agent uses the app of its own worktree and the WAL-streaming database is missing from it
- **THEN** the agent reruns the seed command, and the database appears with backups switched on

#### Scenario: Two agents use the browser at once

- **WHEN** two agent sessions open the browser at the same time
- **THEN** both browsers start, each with its own empty profile
