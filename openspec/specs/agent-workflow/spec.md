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

In a linked worktree, the agent SHALL also merge the pack commit into the local `develop` branch; the request SHALL count as consent to that merge. The agent SHALL resolve conflicts itself and rerun the tests of the packages they touched. It SHALL ask the user only about a critical conflict that the code and specs do not settle, or when another agent's uncommitted changes block the merge, and SHALL abort the merge before asking.

#### Scenario: Pack a finished change
- **WHEN** every task of a change is checked and the user writes "pack"
- **THEN** the main specs contain the change's requirements, the change sits in the archive, and one new commit holds the code, the specs and the archive, with nothing pushed

#### Scenario: Pack an unfinished change
- **WHEN** a change has two unchecked tasks and the user writes "pack"
- **THEN** the main specs, the change directory and the Git history stay as they were, and the agent names the two unchecked tasks

#### Scenario: Another agent's edit in the working tree
- **WHEN** the user asks to pack a change while the working tree also holds an edit the agent did not make
- **THEN** the commit leaves that edit out, and it stays uncommitted in the working tree

#### Scenario: Pack in a linked worktree
- **WHEN** every task of a change is checked and the user writes "pack" in a linked worktree
- **THEN** the local `develop` branch contains the pack commit, with nothing pushed

#### Scenario: A merge conflict the code resolves
- **WHEN** merging the pack commit into `develop` conflicts and the code and the specs show the right result
- **THEN** the agent resolves the conflict without asking the user and reruns the tests of the packages the conflict touched

#### Scenario: A critical conflict with no clear answer
- **WHEN** merging the pack commit into `develop` hits a critical conflict that neither the code nor the specs settle
- **THEN** the agent aborts the merge, `develop` stays as it was, and the agent asks the user

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
Every skill of the repository, including the generated `openspec-*` skills, SHALL have one source file under `.agents/skills/`. Claude Code SHALL load it through `.claude/skills`, a symlink to that directory, and Codex SHALL load it from `.agents/skills/` itself, so both clients read the same file and no copy can drift from the source.

#### Scenario: The commit skill in Claude Code
- **WHEN** Claude Code lists the skills it can load in this repository
- **THEN** the list contains the Git commits skill, read from `.agents/skills/git-commits/SKILL.md`

#### Scenario: The apply skill in Codex
- **WHEN** Codex lists the skills it can load in this repository
- **THEN** the list contains the OpenSpec apply skill, read from `.agents/skills/openspec-apply-change/SKILL.md`

### Requirement: The agent can see and drive the local app

The repository SHALL register one browser automation server for Claude Code and Codex. With it, an agent opens pages of the local app and reads what the page shows, including a screenshot, the page structure, console messages and network requests. It also clicks, types and navigates. Both clients SHALL start the server from the repository configuration with the same command. The server SHALL drive the Chrome browser installed on the host. Each agent session SHALL get its own temporary browser profile, which is discarded when the session's browser closes, so that two sessions running at once do not block each other and no sign-in carries over between sessions.

The repository SHALL hold a skill that tells the agent how to check the app in the browser:

- When the backend and frontend ports already answer from a `mise run run` started in the agent's own worktree, the agent uses that app.
- When they answer from another worktree, the agent leaves that app running, does not run the browser check, and tells the user the check was skipped because the running app serves another worktree's code.
- When only one of the two ports answers, the agent does not start `mise run run` and does not run the browser check. It tells the user which port is held and that the check was skipped.
- When neither port answers, the agent starts `mise run run` without its interactive window and stops it before it finishes the task, leaving neither port held.
- The agent signs in with the seeded dev account.
- When the app of the agent's own worktree lacks the dev account or a seeded database, the agent reruns the seed command against it instead of creating the items by hand.

A browser check SHALL add to the tests a task names and SHALL NOT replace them.

#### Scenario: Agent checks a page with no app running

- **WHEN** an agent needs to see a page and nothing answers on the backend and frontend ports
- **THEN** the agent starts `mise run run` without its interactive window, signs in as the dev account, opens the page and reads what it shows
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
- **THEN** the agent does not start `mise run run` and does not open the page
- **AND** the agent tells the user that the frontend port is held and the browser check was skipped

#### Scenario: A seeded database is missing

- **WHEN** an agent uses the app of its own worktree and the WAL-streaming database is missing from it
- **THEN** the agent reruns the seed command, and the database appears with backups switched on

#### Scenario: Two agents use the browser at once

- **WHEN** two agent sessions open the browser at the same time
- **THEN** both browsers start, each with its own empty profile

#### Scenario: Codex opens a page

- **WHEN** Codex is started in the repository
- **THEN** it lists the same browser server Claude Code uses, started by the same command

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

### Requirement: The quick code-shape check reviews how new code is split
When a finished turn changes a Go or TypeScript source file of the backend, the verification agent, the frontend or the website, the quick code-shape check SHALL review the code the turn added or changed on two counts. First, the split: each new file and each new function holds one concept, and a turn that adds a second, unrelated concept to an existing file moves it into a file of its own. Second, the names and comments that the rules of the repository govern. The check SHALL reject the turn for each broken rule, name the file or the unit and the rule, and suggest the split or the name that follows the rule. The check SHALL NOT judge where code belongs across packages, modules and layers, which the quick architecture check judges. It SHALL NOT reject an addition to a file whose content the architecture document or a module doc fixes, such as the public contract file of a module or the errors file of a package. It SHALL NOT reject a turn for concepts that were already mixed in a file before the turn.

#### Scenario: A new file holds two concepts
- **WHEN** a turn adds a source file that holds a token store and an unrelated mail template renderer
- **THEN** the quick code-shape check rejects the turn, names the file and the rule, and suggests a file for each concept

#### Scenario: A new function does two things
- **WHEN** a turn adds a function that validates a request and also sends the notification mail
- **THEN** the quick code-shape check rejects the turn, names the function and the rule, and suggests a function for each concept

#### Scenario: An event added to the public contract of a module
- **WHEN** a turn adds an event to the file that holds a module's whole public contract, as the architecture document requires
- **THEN** the quick code-shape check does not reject the turn for that file

#### Scenario: A new concept appended to an existing file
- **WHEN** a turn adds a rate limiter to an existing file that holds only a request ID middleware
- **THEN** the quick code-shape check rejects the turn and suggests moving the rate limiter into a file of its own

#### Scenario: An edit inside a file that was already mixed
- **WHEN** a turn changes one function in a file that already held two concepts before the turn, and adds no new concept to it
- **THEN** the quick code-shape check does not reject the turn for the mixed file

#### Scenario: A unit in the wrong layer
- **WHEN** a turn adds a well-split file with a clear name to a package of a layer that must not hold it
- **THEN** the quick code-shape check does not report the placement, and the quick architecture check judges it

#### Scenario: A comment that restates the code
- **WHEN** a turn adds a comment above a function that only says what the function does
- **THEN** the quick code-shape check rejects the turn and suggests a name for the function that makes the comment redundant

### Requirement: Every agent reads the module rules of the code it changes
Before the agent changes a file under a folder that holds its own `AGENTS.md`, it SHALL read that `AGENTS.md`, unless the file is already in its context. A folder whose rules extend another module's rules SHALL name that module's `AGENTS.md`, and the agent SHALL read both. The rule SHALL hold for every client, including one that loads only the `AGENTS.md` files on the path from the repository root to the directory it was started in.

#### Scenario: Codex changes backend code
- **WHEN** Codex, started at the repository root, is about to change a file under `backend/`
- **THEN** it reads `backend/AGENTS.md` first

#### Scenario: Codex changes the verification agent
- **WHEN** Codex, started at the repository root, is about to change a file under `backend/internal/agent/` or `backend/cmd/verification-agent/`, the agent's end-to-end scenarios included
- **THEN** it reads `backend/AGENTS.md` first, and no other module doc

### Requirement: Go commands of a Codex session see the test build tag
A `go` command that the agent runs in a Codex session SHALL see the test helpers build tag without passing it, the same way it does in a Claude Code session, so a package whose cross-package test helpers sit behind that tag builds and its tests run.

#### Scenario: Codex runs a package's tests directly
- **WHEN** in a Codex session the agent runs `go test` on a backend package whose helpers sit behind the test helpers build tag, without passing the tag
- **THEN** the package builds with its helpers and its tests run

### Requirement: Apply consultations stay within the active client

During apply the agent SHALL resolve routine questions itself. When a decision is both difficult and consequential, it SHALL seek a separate consultation directly through a read-only instance of its active client, consider the advice and make the decision. A Codex session SHALL consult Codex using its review profile for full reviews; a Claude Code session SHALL consult Claude Code using its review profile for full reviews. The consultation MUST NOT require the other client, edit files, start recursive review or replace the main agent's decision-making responsibility. The consultation SHALL use the read-only access and recursion restrictions of the review-hook-launch capability; it returns advisory text rather than a review verdict. A failed consultation SHALL be reported and MUST NOT silently switch clients or be presented as advice that was received. The agent SHALL then make the decision from available evidence, asking the user only when information, a decision or authorization only the user can supply is necessary.

#### Scenario: Codex needs a consequential consultation

- **WHEN** Codex encounters a difficult and consequential decision within an agreed apply scope
- **THEN** it seeks advice directly through a separate read-only Codex instance with `high` reasoning effort, considers the advice and makes the decision without requiring Claude Code

#### Scenario: Claude Code needs a consequential consultation

- **WHEN** Claude Code encounters a difficult and consequential decision within an agreed apply scope
- **THEN** it seeks advice directly through a separate read-only Claude Code instance on Opus with `medium` effort, considers the advice and makes the decision without requiring Codex

#### Scenario: The consultation cannot run

- **WHEN** the selected consultation instance fails or times out
- **THEN** the agent makes the decision from available evidence, reports that it received no consultation and does not switch clients
- **AND** it asks the user only if the decision needs information, a decision or authorization that only the user can supply
