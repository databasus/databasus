## Verification evidence

All checks run locally with the versions pinned in `mise.toml`. No hosted pipeline, commit or push is part of this verification.

| Check | Result |
| --- | --- |
| `mise run doctor` | Codex selected; eight project hooks trusted and enabled |
| `mise run python:lint` from root, backend and frontend | Ruff lint, formatting and ty pass for the same 84 Python files |
| `mise run python:format` twice | Both pass; SHA-256 comparison confirms the second run changes none of the 84 files |
| `mise run lint` | Backend, frontend, Python, shell and workflow checks pass |
| `hk check --check --all --step python-lint` | One complete Python check passes |
| `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'` | 148 tests pass, including 15 real-tool quality and hk integration tests |
| `mise exec -- python3 -m unittest discover -s .agent-hooks -p '*_test.py'` | 282 tests pass |
| `mise run e2e:unit` | 29 tests pass |
| `mise run e2e:lifecycle` | All 27 tests pass, including image builds, failures, signals, crashes, fresh checkouts, overlapping runs and cleanup |
| `mise exec -- openspec validate python-code-quality --strict` | Passes |
| `mise run e2e:browser` | Four application scenarios pass; five probe-only scenarios skip as designed; cleanup passes |
| `mise exec -- env PYTHONDONTWRITEBYTECODE=1 UV_PYTHON_PREFERENCE=only-system uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption` | Three tests pass |

## Repairs and retained behavior

- A regression test first reproduced acceptance of a supervisor JSON record with invalid field types. The reader now validates those fields and treats an invalid record as dead, matching the existing damaged-file recovery behavior.
- The lifecycle checks exposed readiness against PostgreSQL's temporary initialization server. Their unrelated-container fixture now probes TCP, which becomes available only on the final server, instead of accepting the temporary Unix socket.
- Pytest fixture imports use explicit re-exports where registration through import is intentional. Formatting and annotation changes retain generated failure snippets, secret masking, reviewer approval checks, child restrictions and cleanup contracts.
- A real-command regression test first showed that a blanket `ty: ignore` could hide an invalid return. Enabling `blanket-ignore-comment` now rejects that escape alongside Ruff's blanket `type: ignore` and `noqa` checks.
- The hk integration tests cover eligible new files, hidden paths, paths with spaces, explicit deleted-path input, Python-to-Python renames, configuration-only selection, unrelated-document exclusion, unchanged callers, one invocation and preservation of files, index and stash on failure. The pinned hk version omits deleted paths and rename origins from automatic selection; `docs/testing.md` documents the full-check command for those cases. CI always checks the full selection.
- The existing CI description in `docs/testing.md` mentions pull requests, while the workflows select main/manual and develop events. This pre-existing discrepancy was reported during exploration; workflow triggers are preserved.
