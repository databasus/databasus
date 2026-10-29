## Why

Developing in Codex still requires Claude Code: the review hooks, manual review, environment check and consultation rule all depend on Claude. A developer with only Codex must be able to use the complete OpenSpec workflow, with the same review stages and repair cycle as a developer using Claude Code.

## What Changes

- Select the review client from the active agent: Codex sessions use Codex throughout; Claude Code sessions use Claude Code throughout. A failure never switches clients automatically.
- Preserve the existing proposal validation, reviewer roles, parallelism, finding categories, repeat limits and implementation verification order.
- Run quick Codex checks with `medium` reasoning effort and full proposal, implementation and repeat reviews with `high`. Keep Opus `low` for quick Claude Code checks and Opus `medium` for full reviews.
- Make manual review and the environment check use the active client, with an explicit client option for a terminal outside an agent session. Only the selected client must be installed and authenticated.
- Run difficult apply consultations directly through a separate read-only instance of the active client, using its review profile for full reviews. Add no consultation command or result schema.
- Keep request snapshots and per-request gate counters keyed by their unique session IDs. Separate full-review findings and acceptance by client and review profile, preserving reuse across sessions of the same client and review profile.
- **BREAKING:** Remove the implicit Claude selection for hook and terminal invocations. Hook configurations declare their client; terminal review and environment checks without an agent context require an explicit choice. Remove the shared, client-independent review-cache lookup; existing cache files are not imported as acceptance or findings for the new review profiles.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `review-hook-launch`: Select the client deterministically, require only its CLI, run reviewers without write access or recursive hooks, and isolate review state by client and review profile.
- `implementation-review`: Use client-specific models and effort for quick, full and repeat reviews, and select the same client for manual review.
- `proposal-review`: Apply the selected client's review profile for full reviews to initial and repeat proposal reviews without changing validation or reviewer responsibilities.
- `agent-workflow`: Keep consultations during apply within the active agent client.

## Impact

- Review tooling: `.agent-hooks/`, `.codex/hooks.json`, `.codex/config.toml`, `.claude/settings.json` and the `review` and `doctor` tasks in `mise.toml`.
- Documentation: the review hook docstring, the consultation behavior rule in `AGENTS.md`, and the Review client and Review profile terms in the Code table of `docs/glossary.md`. Keep model and command details next to the tooling, with links from instructions where needed.
- Tests: extend the existing hook and launcher tests with both client CLIs represented as external command stubs. Live model calls are not required to verify dispatch, result handling or the repair cycle.
- Rules: the root `AGENTS.md` applies. No backend, frontend, website or translated README files are changed, so no module-specific `AGENTS.md` applies.
- Dependencies: the selected client uses its existing local login. No API integration, second AI subscription, new service or database migration is introduced.

### Out of scope

- Changing Databasus application behavior, HTTP contracts, database schemas or browser setup.
- Changing review categories, reducing coverage, removing mandatory review or increasing repeat limits.
- Adding an interception equivalent to Claude Code's `ExitPlanMode` for plans shown only in a Codex reply; complete OpenSpec proposals continue to be reviewed at turn completion.
- Mixing clients within a workflow, automatic failover, choosing a new Codex model for the developer or claiming identical model quality.
- Changing unrelated active changes, including `healthcheck-module`, or implementing this proposal during planning.
