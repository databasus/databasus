## ADDED Requirements

### Requirement: Every agent reads the module rules of the code it changes
Before the agent changes a file under a folder that holds its own `AGENTS.md`, it SHALL read that `AGENTS.md`, unless the file is already in its context. A folder whose rules extend another module's rules SHALL name that module's `AGENTS.md`, and the agent SHALL read both. The rule SHALL hold for every client, including one that loads only the `AGENTS.md` files on the path from the repository root to the directory it was started in.

#### Scenario: Codex changes backend code
- **WHEN** Codex, started at the repository root, is about to change a file under `backend/`
- **THEN** it reads `backend/AGENTS.md` first

#### Scenario: Codex changes the verification agent
- **WHEN** Codex, started at the repository root, is about to change a file under `backend/internal/agent/` or `backend/cmd/verification-agent/`, the agent's end-to-end scenarios included
- **THEN** it reads `backend/AGENTS.md` first, and no other module doc

## REMOVED Requirements

### Requirement: The module rules reach every agent
**Reason**: Its last scenario covers the old end-to-end suite under `agent/verification/e2e/`, which this change deletes. A delta cannot drop one scenario of a requirement, so the requirement is replaced as a whole.
**Migration**: "Every agent reads the module rules of the code it changes" holds the same rule and the two remaining scenarios. The agent's end-to-end scenarios now live under `backend/internal/agent/`, which the scenario for the verification agent covers.
