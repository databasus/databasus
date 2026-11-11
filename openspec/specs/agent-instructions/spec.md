# agent-instructions Specification

## Purpose
Keeps `AGENTS.md` and the repository's own skills stable: they hold rules of agent behavior, while descriptions of the system, inventories of what exists and the internals of tools live next to what they describe. This way an ordinary feature does not edit these files, and parallel branches do not conflict in them.

## Requirements

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

### Requirement: The diff reviewer checks edits to agent instructions
The diff reviewer SHALL check every edit to agent instructions against the requirement "What belongs in agent instructions". When an edit adds an inventory of what exists, a statement about the current state of code, a description of how something works or a retelling of another document, the reviewer SHALL reject the turn and name the place to move that information to. An edit that adds or changes a rule of agent behavior MUST NOT be rejected under this criterion.

#### Scenario: An edit adds an inventory
- **WHEN** the agent adds "the backend currently has features X and Y" to a module doc
- **THEN** the diff reviewer rejects the turn and says that the state of the code is visible in the code

#### Scenario: An edit adds a rule
- **WHEN** the agent adds a rule that changes agent behavior to `AGENTS.md`
- **THEN** the diff reviewer does not reject the turn under the instruction content criterion

### Requirement: Shared documentation explains lasting concepts and reader procedures

Shared project documentation SHALL explain system boundaries, ownership, cross-cutting invariants, architectural reasons, domain vocabulary and procedures that readers need to carry out. Root agent instructions SHALL define this scope and distinguish it from implementation documentation, behavioral specifications and evidence from individual changes.

Shared documents MUST NOT duplicate internal helper catalogs, signatures, fixture algorithms, dependency selections or reports of individual verification runs. Concrete names, paths, commands and configuration examples SHALL be retained only when they identify an architectural boundary, establish necessary vocabulary, locate an authoritative source or let the reader perform a documented procedure. A glossary SHALL define concepts rather than mirror every identifier in the code.

#### Scenario: Internal implementation changes

- **WHEN** an internal helper is renamed or a fixture is reorganized without changing a documented concept, source reference or reader procedure
- **THEN** shared documentation requires no corresponding edit
- **AND** internal usage instructions remain with the implementation that owns them

#### Scenario: A reader follows an operational procedure

- **WHEN** a shared document explains a procedure that requires a command or configuration setting
- **THEN** it retains the concrete syntax, prerequisites and relevant limitations needed to perform that procedure
- **AND** it does not expand into a catalog of unrelated implementation details

#### Scenario: An architectural rule changes

- **WHEN** a change alters a system boundary, ownership rule or cross-cutting invariant
- **THEN** shared architecture documentation changes with it and preserves the reason for the rule
- **AND** the reason stays beside that rule rather than in a separate decision archive

### Requirement: Documentation cleanup preserves safety and recovery constraints

Reducing documentation detail SHALL preserve access requirements, credential handling, security assumptions, durability prerequisites, recovery limitations and the limits of claimed support or verification. Moving a verification report out of shared guidance MUST NOT turn an exercised configuration into an unconditional support claim. Security exceptions SHALL retain their justification and traceability.

#### Scenario: Verification evidence leaves operational guidance

- **WHEN** dated versions, fixture descriptions or scenario lists are removed from a shared recovery guide
- **THEN** its operational prerequisites and recovery gaps remain explicit
- **AND** it makes no broader claim about supported configurations or tested platforms
- **AND** evidence needed to justify a remaining claim remains available through an authoritative reference

#### Scenario: Security documentation is shortened

- **WHEN** a security explanation contains internal test names and implementation details
- **THEN** shortening it preserves the threat boundary, mitigation or accepted-risk rationale and any reference needed to trace a security exception

### Requirement: Documentation upkeep follows authoritative ownership

The agent SHALL review shared documentation for conceptual or procedural changes, rather than add an explanation for every new implementation mechanism. It SHALL remove redundant descriptions, retain necessary implementation-only guidance with the owning code or tool, and reference the authoritative source instead of copying it into another general document. It MUST NOT relocate unnecessary prose merely to preserve its volume.

When cleanup changes a document address, section anchor or location of guidance, the agent SHALL update affected references in current documentation and agent instructions. Documentation cleanup SHALL preserve existing behavioral and architecture rules; changing their presentation MUST NOT weaken them.

#### Scenario: A feature adds an internal mechanism

- **WHEN** a feature adds an internal mechanism without changing a documented concept or reader procedure
- **THEN** the agent does not add a shared-document section solely to describe that mechanism

#### Scenario: A redundant implementation explanation is removed

- **WHEN** implementation prose repeats information already available in the owning source or its local documentation
- **THEN** the duplicate is removed without creating another document that repeats it
- **AND** any necessary navigation points to the authoritative source

#### Scenario: Cleanup moves referenced guidance

- **WHEN** cleanup removes a section or moves guidance that current documentation or agent instructions reference
- **THEN** those references resolve to the retained explanation or its new authoritative location
- **AND** readers and reviewers can still find the rules and procedures they depend on
