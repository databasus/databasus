## MODIFIED Requirements

### Requirement: What belongs in agent instructions
Agent instructions are the git-tracked `AGENTS.md` files at the repository root and in subdirectories, and the repository's own skills, that is every skill except the generated `openspec-*` skills. Agent instructions SHALL hold only rules that change how the agent behaves and do not depend on which features already exist. Agent instructions MUST NOT hold:

- inventories of what already exists in the system: features, endpoints, checks, installed tools;
- statements about the current state of code with words like "currently" and "for now", except the rule about how to write about the current state;
- descriptions of how code, hooks and tools work: the internal steps of commands, hook trigger conditions, constant values, limits, models;
- retellings of another document's sections.

The root `AGENTS.md` SHALL name the place for each of these kinds of information: code and its comments, the API contract, `docs/` including the reasons behind architectural choices in `docs/architecture.md`, a program's README, the hook docstring and prompts, the OpenSpec specs. The repository SHALL keep no separate decision records: the reason for an architectural choice is written in `docs/architecture.md` next to the rule it explains. An instruction MAY link to a file as an example or a source of values, when the link itself does not change with the file's contents. An instruction MAY hold:

- a table of documents with the topic of each;
- a table of commands the agent runs with the purpose of each, without its internal steps;
- the stack of a part: language and main frameworks;
- one sentence saying that a linter enforces a rule, without the list of its checks.

#### Scenario: A feature adds an endpoint
- **WHEN** a feature adds a backend endpoint
- **THEN** neither `AGENTS.md` nor any module doc changes, and the endpoint is visible in the code and the generated API description

#### Scenario: A module doc gets a new section
- **WHEN** a change adds a section to `website/AGENTS.md`
- **THEN** the `website/AGENTS.md` line in the root `AGENTS.md` does not change, because it names the doc's topic rather than its sections

#### Scenario: A review hook changes
- **WHEN** a review hook changes its trigger, model, effort or reviewer prompt
- **THEN** the hook script, its prompt or its spec changes, and the root `AGENTS.md` does not

#### Scenario: What the agent does with a finding category changes
- **WHEN** what the agent does with a full review finding category changes, for example it stops blocking the turn
- **THEN** the "Review hooks" section of the root `AGENTS.md` changes, because that is a rule of agent behavior

#### Scenario: A new rule of agent behavior appears
- **WHEN** the team introduces a rule that changes how the agent writes code or answers
- **THEN** the rule is written in one place in the agent instructions, and other documents link to it

#### Scenario: A change explains an architectural choice
- **WHEN** a change makes an architectural choice and records why
- **THEN** the reason is written in `docs/architecture.md` next to the rule it explains, and no separate decision record is created
