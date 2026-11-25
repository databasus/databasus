# Test plan

## Impact

| Changed | Tests it can break | Why |
|---|---|---|
| `scripts/test.sh`, `scripts/test_plan.py`, `scripts/go_test_report.py` | the runner, plan parser and report tests in `scripts/` | they define both run modes, the selector grammar and the no-test check |
| `mise.toml` | the command list and CI checks in `scripts/` | `development_commands_test.py` and `frontend_ci_test.py` read the tasks |
| `.agent-hooks/review_gate.py` and its prompts | the hook tests in `.agent-hooks/` | the hook validates the test plan and the tests assert the prompt sentences |
| Documentation | none | prose only |

## Not selected

- Backend and frontend packages: the change edits no Go or TypeScript code.
- `openspec/schemas/databasus/` and `openspec/config.yaml`: no test suite reads them, because the hook tests stub `openspec`. `openspec schema validate databasus` and `openspec validate --strict` check them instead.
- The browser end-to-end suite, its lifecycle checks and the upgrade test: their code, scenarios and the `Dockerfile` are unchanged; the runner tests in `scripts/` cover how `scripts/test.sh` starts them.

## Selection

```tests
suite scripts
suite review-hooks
```
