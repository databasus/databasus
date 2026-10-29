## Context

See `proposal.md` for the problem and scope. The existing machinery already supports both parent clients; the missing part is selecting the subprocess that performs review.

- `.codex/hooks.json:4` registers session start, request snapshot and the same six Stop gates as Claude Code. `.claude/settings.json:10` additionally intercepts `ExitPlanMode`.
- `.agent-hooks/review_gate.py:107` stores Claude-specific model and effort fields. Its quick and full review profiles select Opus `low` and `medium`; `run_reviewer` at line 623 always calls the Claude adapter.
- `.agent-hooks/review_gate.py:579` already supplies the Claude process with a restricted tool list, disabled hooks and no persisted session. The environment helper at line 571 sets the recursion guard and removes nested Claude session variables.
- `.agent-hooks/review_gate.py:403` stores request and gate state using only the session identifier. The full-review state path at line 467 uses only gate and change, so both clients share findings and acceptance.
- `.agent-hooks/review_gate.py:1142` and line 1163 select automatic implementation and proposal reviews from request snapshots and file hashes. Preserve those trigger conditions, including skipping unchanged turns.
- `.agent-hooks/review_gate.py:1347` requires `claude` in `doctor`. `mise.toml:67` exposes doctor and review without a client option.
- `.codex/config.toml:4` and `.claude/settings.json:6` already inject shell environment values, providing an explicit way for agent-launched terminal commands to identify their client.
- `AGENTS.md:256` requires an Opus 5.5 consultation through Claude Code for consequential apply decisions. The apply skill links to that rule at `.agents/skills/openspec-apply-change/SKILL.md:17`.
- `.agent-hooks/review_gate_test.py:23` represents external commands as executable stubs. The configuration tests at line 635 compare both clients, and line 728 explicitly asserts that a Codex turn launches Claude reviewers. Replace that assertion and extend the same subprocess surface.

Only root tooling and documentation are involved. The root `AGENTS.md` and testing skill constrain implementation; no module-specific `AGENTS.md` applies. `docs/architecture.md` describes application boundaries that this change does not alter, so no application architecture edit is needed. Hook mechanics belong in the hook docstring and specs, as the `agent-instructions` capability requires.

## Goals / Non-Goals

**Goals:** keep one review engine, with client-specific execution, deterministic command selection and testable consultation behavior. Preserve the existing validation, reviewer roles, concurrency of at most six reviewers, time budgets, findings and repair protocol.

**Non-Goals:** resolve model catalogues, inherit a one-off interactive model override, emulate `ExitPlanMode` in Codex, or introduce a second review workflow. Codex subprocesses use the configured default model of the developer's existing Codex installation. Automated tests use external CLI stubs and do not consume model calls.

## Decisions

### 1. Select a review client explicitly at the invocation boundary

Introduce the term **Review client**, with values `codex` and `claude`. Add it to the Code table in `docs/glossary.md`, apart from database client tools, contract-checking test clients, product providers and the verification agent. Add `--client` to the Python command and to the `review` and `doctor` mise tasks. Name the inherited declaration `AGENT_REVIEW_CLIENT`.

- Each command in `.codex/hooks.json` passes `--client codex`; each command in `.claude/settings.json` passes `--client claude`, including the plan gates.
- `.codex/config.toml` sets `AGENT_REVIEW_CLIENT=codex` in its shell environment policy; `.claude/settings.json` sets it to `claude` in `env`. Existing build-tag and browser settings remain intact.
- Resolve an explicit option first, then `AGENT_REVIEW_CLIENT`. An explicit valid option overrides even an invalid inherited declaration. Missing or invalid selection is an error, never an implicit Claude default.
- Land the explicit hook arguments and injected environments in the same implementation task as the selector that removes the Claude default. Do not leave active hooks without a client declaration between tasks.
- Keep the action as the first argument to `run.sh` so its distinction between hook and terminal output remains reliable. The launcher forwards the client option unchanged. Hook failures retain the JSON stop response; terminal failures remain text and non-zero exit status.

Examples after implementation:

```sh
mise run review support-codex-review-pipeline --client codex
mise run doctor --client claude
```

Inside either agent, `mise run review` and `mise run doctor` need no option. Error hints printed outside an agent include an explicit client example; an environment check reports executable availability rather than claiming authentication success.

**Rejected alternatives:** infer the client from installed executables or parent-process names, which is ambiguous when both are installed; read a last-used client from Git state, which races across sessions; keep Claude as a default, which retains the unwanted dependency. Use `--client` rather than overloading the product glossary term `provider`.

### 2. Keep review policy shared and make execution client-specific

Extract client selection and immutable review profiles into a small `review_client` unit. Keep existing review roles, prompts, result schemas, target selection and orchestration in the shared engine. A resolved execution request names the client, model selection, effort, schema and timeout; it uses a named structure rather than adding adjacent string parameters to the current Claude runner.

| Purpose                                           | Claude Code      | Codex                              |
| ------------------------------------------------- | ---------------- | ---------------------------------- |
| Quick checks, including the Claude-only plan gate | `opus`, `low`    | configured default model, `medium` |
| Initial and repeat full proposal review           | `opus`, `medium` | configured default model, `high`   |
| Initial and repeat full implementation review     | `opus`, `medium` | configured default model, `high`   |
| Consequential apply consultation                  | `opus`, `medium` | configured default model, `high`   |

The existing quick timeout of 540 seconds, full timeout of 1800 seconds, total budget and parallel pool remain shared. The effort mapping is an explicit project choice supplied by the user, not a claim of model-quality equivalence. A configured Codex model that rejects the requested effort fails visibly; the runner does not lower effort or change models.

**Rejected alternatives:** duplicate the gates for Codex, which would let the criteria drift; invoke `codex review`, which does not express all proposal roles and the existing result contracts; hardcode a new Codex model or copy the main agent's interactive conversation, which is unnecessary for an independent reviewer.

### 3. Add a Codex execution adapter with bounded, read-only access

Keep the existing Claude execution restrictions. Add a separate Codex adapter, with private helpers next to that adapter, that launches a fresh `codex exec` process at the repository root. Use an argument array and standard input for the prompt. Do not interpolate the prompt, schema or answer into a shell command.

The Codex invocation supplies:

- `--sandbox read-only` and approval policy `never`, so inherited permissive settings cannot enable writes or escalation;
- `model_reasoning_effort` from the selected review profile;
- `--ephemeral`, `--output-schema` and `--output-last-message`, with a unique temporary output directory owned by this invocation;
- hooks disabled in the child through the supported `hooks` feature switch, and `AGENT_REVIEW_HOOK_ACTIVE=1` as the shared guard;
- app connectors and multi-agent tools disabled for this child, and disabled configured MCP servers, including plugin-bundled servers, so filesystem sandboxing is not mistaken for protection against external tool mutations.

Obtain configured MCP and plugin identities through the CLI's machine-readable configuration listings; pass per-server disabling overrides to the child without logging server configuration, environment values or credentials. Bound metadata queries within the same reviewer timeout. If the installed CLI or managed policy cannot enforce these restrictions, return an execution failure rather than starting a less restricted reviewer. Keep ordinary model transport and the existing authentication available. Never change project or user configuration to restrict the child.

Read only the final answer file, not progress output, and validate it through the existing quick/full result parsers. Reject an absent, empty, malformed or schema-invalid result, even after exit code 0. Remove temporary output in every exit path. Process failure, timeout and authentication or model-access errors use the existing unaccepted-review path; diagnostics identify the selected client without exposing credentials. Account for setup time when giving the model process its remaining timeout.

The local CLI is version 0.164.0. `codex exec --help` exposes read-only sandboxing, ephemeral execution and schema-constrained final output; `codex features list` exposes the hooks feature. The [official command reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli), [hook documentation](https://learn.chatgpt.com/docs/hooks) and [configuration reference](https://developers.openai.com/codex/config-reference) document the corresponding interfaces. These are capability observations, not proof of a live authenticated model call.

**Rejected alternatives:** trust a read-only prompt alone; inherit connectors merely because the shell is sandboxed; use an isolated login that copies credentials; disable the parent's hooks; bypass sandbox or hook trust. These either fail to bound the child or disturb the main session.

### 4. Separate state by client and review profile

Keep state under the worktree's existing Git-directory location; do not add tracked state or a cross-worktree service. Request snapshots and per-request gate counters keep their existing paths keyed by unique session IDs, preserving the user's original baseline throughout automatic continuations. Only full-review findings and acceptance are namespaced by review client, review profile, gate and change and remain reusable across sessions. Add **Review profile** to the Code table in `docs/glossary.md`: the selected model, reasoning effort, review contract and execution policy that identify which full-review results can be reused.

The review profile identity contains the model selector, effort mapping, review schema and prompt digests, and an execution-policy version. For the Codex default-model selector, include digests of its local model-configuration inputs, including the active Codex home configuration and trusted project configuration, so a local default-model change does not reuse acceptance under the old configuration. Persist digests rather than configuration contents or credentials. Remote changes behind an unchanged model alias are not resolved; explicit manual review already forces a new review of an accepted state.

Apply the identity everywhere full-review state affects behavior: target selection, repeat construction, accepted-state skipping, suppression of quick checks during full-review fixes, summary requests and manual review. Do not reuse client-independent full-review state or fall back to another namespace. Existing full-review cache files remain on disk but are not read as current full-review state; existing per-session snapshot and counter paths stay in use.

Preserve the existing trigger boundary: opening another client without edits does not itself trigger full review. When that client later has an eligible turn or the developer explicitly requests manual review, another client's acceptance cannot satisfy it. A matching review profile's existing findings produce the same focused repeat in manual and automatic review.

**Rejected alternatives:** keep one shared acceptance per change, which lets Claude substitute for Codex; namespace session snapshots and per-request counters by client, which adds paths for an artificial session-ID collision without changing the behavior under the clients' unique session IDs; invalidate every session's acceptance, which repeats expensive review without a relevant change; migrate old full-review state, which has no trustworthy client identity.

### 5. Keep apply consultations as direct client invocations

Change the existing behavioral rule rather than adding a repository command. The agent invokes a separate read-only instance through the active client's native interface, using that client's review profile for full reviews: Codex `high` or Opus `medium`. Consultations receive the decision and relevant repository context and return advisory text. The agent applies the same read-only access and recursion restrictions as reviewer subprocesses, but no review-result schema or acceptance cycle is involved. The hook docstring documents native invocation and child-only restrictions; no `consult` action, advice schema or consultation-specific runner is introduced.

Update only the behavioral consultation rule in `AGENTS.md`: use the active review client, consider the advice and retain responsibility for the decision. When consultation fails, make the decision from available evidence and report that no advice was received; ask the user only when only the user can supply a needed decision, information or authorization. Link to the runner's documentation for native invocation and review profile values instead of duplicating model names or effort constants in agent instructions. The apply skill already links to this rule and needs no duplicate instructions. The glossary terms introduced in decisions 1 and 4 both belong in its existing Code table.

**Rejected alternatives:** retain unconditional Opus delegation, which makes Codex incomplete; add a `consult` action with a stdin contract, advice schema and new tests, which duplicates native client capabilities without adding user behavior; build a separate consultation service or workflow, which is unnecessary for an advisory invocation.

## Risks / Trade-offs

- Child tool exposure and configuration differ by CLI version. Test the exact generated execution restrictions, configured external-server overrides and unsupported-policy failure. Do not quietly relax the read-only execution contract.
- A Codex model may not support the requested effort or may be unavailable to the account. Report the CLI failure without fallback. `doctor` remains a local executable check, not a paid model probe.
- Codex default-model selection inherits local configuration, not a one-off model or configuration chosen in the main interactive session. State this boundary in the hook docstring and hash the configuration inputs used by the reviewer.
- Including configuration digests can invalidate acceptance after an unrelated local configuration edit. This is preferable to treating an uncertain model selection as the same review profile and does not require a general model resolver.
- Higher effort can take longer. Retain the bounded execution and existing full-review budget so slowness produces a clear unaccepted result rather than an unlimited run.
- Existing caches are deliberately ignored, so the next eligible review starts fresh. No tracked data, application state or unrelated change is migrated.
- Stub tests prove dispatch, permissions requested, parsing and orchestration. They do not prove account access or model judgment. A live review can run after implementation when the agent's normal hooks execute; it is not a prerequisite requiring another account during tests.

An optional scope reduction is to namespace full-review state by review client alone and omit review profile digests. The developer would retain independent Codex and Claude review but lose automatic cache invalidation after changing a prompt, review contract, effort or local Codex model configuration. It would save configuration hashing, identity propagation and the corresponding tests in task 3.1. The current requirements and tasks retain those guarantees; this reduction requires the user's decision.

## Migration Plan

1. Introduce the selection and review profile units together with explicit client declarations in both tracked hook configurations and shell environments. Remove the implicit Claude selection only in that same task and update configuration tests, so hooks remain usable between tasks.
2. Add the Codex adapter and namespaced full-review state, then forward `--client` through the mise tasks and launcher. Remove client-independent full-review state lookup; keep no compatibility fallback. Preserve existing per-session paths, hook actions and reviewer contracts.
3. Update the hook docstring, the single apply consultation rule and the glossary. Document direct consultation without adding a command or result contract. Search the tooling and related specs for claims that Codex requires Claude.
4. Run the hook tests, configuration checks, relevant shell lint and strict OpenSpec validation. Leave tasks unchecked until the implementation and its verification are complete; allow the normal implementation hook to review the result.

Rollback restores the prior tooling and tracked configuration together. It does not migrate new review states into the old shared cache. No application deployment or database rollback is involved.
