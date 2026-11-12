## ADDED Requirements

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
