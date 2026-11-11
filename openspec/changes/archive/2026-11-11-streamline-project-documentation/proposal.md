## Why

Shared documentation increasingly repeats implementation details, test scenarios and verification reports. Internal changes then require unrelated documentation edits, while the current placement rules explicitly allow helper signatures and descriptions of current code in shared documents.

## What Changes

- Define shared documentation as the home for lasting concepts, system boundaries, architectural reasons and procedures readers need to perform real work.
- Update the root documentation rules and placement table so internal names, helper signatures, dependency pins, fixture internals and one-change verification evidence stay with their authoritative sources.
- Review every document under `docs/`, removing redundant detail or replacing it with a conceptual explanation and a focused reference. Keep necessary commands, configuration examples, supported behavior, security constraints and recovery limitations.
- Keep links and references consistent across agent instructions, repository-owned skills and affected local READMEs. Preserve architecture rules used by reviews.
- Require shared-document changes when the concepts or reader procedures change, rather than whenever an internal implementation changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-instructions`: Extend documentation placement rules to shared documents, define justified concrete details and require upkeep without duplicating implementation or losing operational safeguards.

## Impact

- Primary scope: root `AGENTS.md` and all nine Markdown documents under `docs/`.
- Follow-up scope: repository-owned skills, module instructions and local READMEs only where a moved explanation or changed link requires a corresponding update. Read each applicable module `AGENTS.md` before editing its files; backend-local documentation answers to `backend/AGENTS.md`, frontend-local documentation to `frontend/AGENTS.md`, and website-local documentation to `website/AGENTS.md`.
- Root `AGENTS.md` governs this documentation-only change. Existing review prompts can apply the updated rules; no new review mechanism is required.
- No application, API, schema, dependency, test behavior or supported deployment changes. No compatibility break is intended.

### Out of scope

- Refactoring production code, test fixtures, commands or CI configuration.
- Adding a documentation framework, a new linter or a separate architecture-decision archive.
- Rewriting historical OpenSpec changes, other active changes or unrelated agent instructions.
- Rewriting the public website or multilingual READMEs, or removing operational guidance to meet a size target.
- Claiming broader platform support or verification coverage than existing evidence establishes.
