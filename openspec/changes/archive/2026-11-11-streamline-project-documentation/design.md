## Context

See `proposal.md` for the motivation and scope. A design is useful because the cleanup spans all shared documents and must distinguish redundant implementation prose from operationally necessary detail.

- `AGENTS.md:223-244` defines documentation ownership, but its placement table permits current-code descriptions and helper signatures in shared documents.
- `docs/architecture.md:183-205` mixes HTTP boundaries with a pinned generator-template version and update instructions. `docs/architecture.md:275-303` mixes physical-backup design reasons with scheduler and publication mechanics.
- `docs/testing.md:37-66` lists test-harness calls; `docs/testing.md:90-140` describes individual schema-recovery and Patroni scenarios and their fixture mechanics.
- `docs/postgresql-patroni.md:106-125` records a dated verification environment and scenario matrix inside operational guidance. Its fixture already has dependency documentation at `backend/internal/lib/testcontainers/testdata/patroni/README.md:1-38`.
- `docs/frontend-localization.md:30-55` names internal helpers alongside the lasting requirements they implement. `docs/glossary.md:104-125` includes implementation identifiers as well as concepts.
- `openspec/specs/agent-instructions/spec.md:9-45` already separates agent rules from descriptions of the system. The delta extends that separation to shared documentation without changing the existing instruction-content or review requirements.

Root `AGENTS.md` constrains every edit. No application architecture changes: the ownership boundaries and invariants in `docs/architecture.md` remain in force. Read the applicable module instructions before placing necessary guidance in a module-local README or updating a module instruction's reference. No module code or test changes are planned.

## Goals / Non-Goals

**Goals:**

- Make shared documents useful after internal identifiers and algorithms change.
- Preserve reasons and constraints that code alone does not explain.
- Keep actionable guidance complete, including concrete syntax when readers need it.
- Give existing reviews a clear documentation-placement rule in one authoritative location.

**Non-Goals:**

- Make every document abstract or prohibit all paths, names and examples.
- Remove an architectural invariant because its explanation contains implementation terms.
- Achieve a line-count target or move all removed text into new READMEs.
- Change how review hooks execute or add automatic documentation classification.

## Decisions

### 1. Put the placement rule in root agent instructions

Extend Documentation consistency in `AGENTS.md` with the shared-document scope and revise the existing table. Distinguish concepts and architecture reasons, reader procedures, internal usage documentation, behavioral specifications and change-specific evidence. Keep one rule and use references from skills where needed.

Internal renames ordinarily do not change shared documents. Changes to a referenced source, a system boundary, an invariant or a reader procedure still require documentation upkeep. Stability never permits a stale command or broken link.

**Rejected alternatives:** A second policy document would make the rule harder to discover and duplicate its maintenance. Adding the same rule to every skill would create competing copies. Keeping the table unchanged would continue to authorize the content this cleanup removes.

### 2. Classify each passage by the reader's need

Use four outcomes:

| Passage | Outcome |
| --- | --- |
| A boundary, invariant, domain concept or architectural reason | Retain; describe it independently of a particular helper where possible |
| Syntax or prerequisites needed for a real procedure | Retain the necessary concrete detail |
| Useful internal instructions with no authoritative local home | Place a concise explanation in the owning tool or module's local documentation and link to it |
| A source-code retelling, duplicated inventory or one-run report | Remove the duplicate; point to existing evidence when a remaining claim needs it |

Do not add comments to production code as a destination for discarded prose. Historical OpenSpec evidence stays unchanged. If an existing change already records a verification run, reference that evidence rather than reproduce it.

**Rejected alternatives:** A blanket identifier ban would break actionable instructions and obscure ownership. Moving every paragraph would preserve the maintenance burden. Keeping all material in place but shortening sentences would not remove the duplication.

### 3. Audit all nine shared documents in place

Keep the existing document addresses and useful section anchors where practical. Apply the following treatment, leaving passages unchanged when they already meet the rule:

| Document | Preserve | Remove, generalize or refer to its owner |
| --- | --- | --- |
| `architecture.md` | System and module boundaries, ownership, invariants, data flow and reasons | Helper catalogs, generator-version bookkeeping, detailed scheduler or publication algorithms, screen-specific behavioral inventories |
| `testing.md` | Test surfaces, isolation, clock and lifecycle principles, check selection, commands and prerequisites | Harness-option catalogs, fixture algorithms, exhaustive lists of assertions and dated verification evidence |
| `glossary.md` | Domain vocabulary and distinctions needed across documents and code | Entries that merely index a helper, job or file; unnecessary identifier mappings |
| `frontend-design.md` | Theme ownership, semantic roles, interaction, accessibility and responsive presentation conventions | Detailed component/file inventory and screen-specific exceptions already owned by behavioral specs |
| `frontend-localization.md` | Dictionary completeness, full sentences, safe interpolation, reactive language changes and regional formatting | Internal helper catalogs and implementation recipes that can be found in the owning source; retain minimal safe examples when necessary |
| `postgresql-patroni.md` | Endpoint and privilege requirements, external slot ownership, durability, independent recovery and permanent recovery gaps | Test-fixture behavior, dated version matrices, verification reports and parser implementation detail |
| `how-to-upgrade-deps.md` | Release eligibility, authoritative version sources, locked installation, integrity checks and verification procedure | Redundant internal script behavior and copied selections; retain source-location tables that let an updater find the pins |
| `security-scan-triage.md` | Threat boundaries, mitigations, accepted-risk reasons, finding references and verification entry points | Narration of individual test implementations and redundant helper-level explanation |
| `machine-setup.md` | Host/storage recommendations and their reasons | Incidental implementation detail, if any; do not edit merely to touch every file |

Architecture reasons remain beside their rules in `architecture.md`. Behavioral details already specified elsewhere can become a concise explanation plus a reference; their architectural implications remain explicit. A glossary may retain a code mapping when it prevents confusion about a lasting domain concept.

**Rejected alternatives:** Cleaning only the latest Patroni additions would leave the same pattern elsewhere. Reorganizing the document tree would add link churn without improving the policy. Deleting operational documents would remove guidance the user chose to retain.

### 4. Preserve safeguards and evidence limits explicitly

Use the cleanup diff to check that access checks, credentials, source identity, durability, storage publication safety, recovery gaps and unsupported-environment caveats retain their meaning. A dated test report can leave shared guidance, but its removal cannot imply support for untested platforms. Keep security exception identifiers and rationale where they are needed for traceability.

The Patroni guide still needs exact configuration examples for external slot ownership and enough information to prepare independent recovery. The security guide still needs finding references. These are justified concrete details, not violations of the policy.

**Rejected alternatives:** Reducing every limitation to a generic warning would hide information needed for safe operation. Treating all numbers or identifiers as clutter would remove security traceability and necessary syntax.

### 5. Verify prose, ownership and references without application test runs

Review every scoped document and compare the diff with the source rules. Search current repository documentation and repository-owned skills for changed headings, moved explanations and removed names. Check each changed relative link's target and, where present, its section anchor. Existing documentation may link into source code; those references must still resolve.

Use `git diff --check` for changed prose and `openspec validate streamline-project-documentation --strict` for the artifacts. The backend hook selects lint for any backend path (`hk.pkl`, backend-lint), so run `mise run backend:lint` for the new template README. Existing formatters target code rather than these Markdown files (`mise.toml`, format tasks); do not format unrelated files. No new checker is introduced. Application tests are unnecessary because this change does not alter executable behavior.

**Rejected alternatives:** Running the application suites would not establish documentation quality or link correctness. Adding a permanent checker would turn a prose cleanup into a tooling project. A link-only check would miss lost constraints and duplicated explanations.

## Risks / Trade-offs

- [An invariant is mistaken for an implementation detail] -> Preserve the rule and reason in conceptual language; compare against existing architecture and behavioral specs before cutting it.
- [A guide becomes too vague to use] -> Keep exact commands, configuration and prerequisites whenever the reader must execute a procedure.
- [Skills or reviewers lose their navigation] -> Search and repair references in current instructions and documents whenever guidance or anchors move.
- [Local READMEs become another duplicate catalog] -> Add only necessary local instructions not already explained by an authoritative source; delete redundant prose outright.
- [A shorter verification section implies broader support] -> Preserve the limits of support and evidence independently of dated reports.
- [A documentation-only task expands into a code refactor] -> Resolve placement through prose and references; keep runtime, tests, dependencies and hook execution unchanged.

## Migration Plan

No deployment or data migration is needed. Update the policy first, clean the shared documents, repair affected references, then verify the complete documentation diff. No historical OpenSpec artifacts or other active changes are rewritten. Rollback, if needed, restores the documentation change as a whole without changing application state.
