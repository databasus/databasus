# Verification

## Scope

Reviewed all nine shared documents. Eight changed; `docs/machine-setup.md` already met the placement rule and remains unchanged. Root documentation rules changed, and necessary generator-upgrade instructions moved to `backend/api/templates/README.md`. No production code, contracts, dependencies, executable tests or review-hook behavior changed.

## Local checks

| Check | Result |
| --- | --- |
| `mise run doctor` | Passed; Codex project hooks reported trusted and enabled |
| `mise run backend:lint` | Passed, including contract and generated-output checks; Go lint reported zero issues. Existing informational duplicate-description findings and deprecation notices remain |
| `git diff --check` | Passed |
| `openspec validate streamline-project-documentation --strict` | Passed |
| Temporary standard-library link audit | All 166 local links and incoming references to shared documents resolved, including section anchors |
| Comparison of existing shell command blocks against HEAD | All existing shell command blocks in the cleaned guides are unchanged |
| Removed glossary term search in current documentation and repository-owned skills | No remaining references to removed entries |

Application tests were not run: the change affects prose only. Existing formatters target code rather than these Markdown files; no unrelated formatting was applied.

## Requirement review

- Root instructions now distinguish shared concepts, actionable procedures, local implementation guidance, dependency records, behavioral specifications and change-specific evidence. They require upkeep for conceptual changes and valid references without demanding documentation edits for internal renames.
- Architecture retains module and data ownership, authorization, transaction and after-commit boundaries, cancellation, storage publication safety, lock order, source identity checks and recovery prerequisites. Detailed frontend behavior links to its existing specifications; generator-specific instructions have one local home.
- Testing retains isolation, real-time versus movable-clock behavior, lifecycle cleanup, suite selection and runnable commands. Fixture algorithms, assertion catalogs, CI topology details and dated timing reports no longer occupy shared guidance.
- Frontend guidance retains semantic theme ownership, keyboard and responsive interaction, safe interpolation, exhaustive dictionary checks, live language changes and regional formatting.
- Patroni guidance retains routing and privileges, exact external-slot configuration, durability prerequisites, recovery gaps and independent recovery instructions. Recorded versions and scenario results are referenced from their existing archived change; native ARM and deployment-scope caveats remain explicit.
- Dependency guidance retains release holds, locked installation, provenance checks and verification limits. Security guidance retains finding references, accepted-risk boundaries, credential restrictions and escaping rationale.

Existing section headings were retained, so no incoming anchor migrations were needed. Repository-owned skills still point to the retained explanations. Other active changes and historical OpenSpec artifacts were left untouched.
