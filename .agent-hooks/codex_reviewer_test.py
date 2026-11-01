from dataclasses import replace
import io
import json
import os
from pathlib import Path
import sys
import unittest
from unittest import mock

import review_gate
from review_gate_test import HOOK_PATH, HookRepository


class CodexReviewerTest(unittest.TestCase):
    review_client = "codex"

    def prepare(self, config=None):
        repository = HookRepository(self)
        repository.write(".codex/config.toml", 'sandbox_mode = "danger-full-access"\napproval_policy = "on-request"\n')
        repository.submit_prompt("Implement widgets")
        repository.write("src/widgets.py", "WIDGETS = []\n")
        if config is not None:
            repository.set_stub("codex", config)
        return repository

    def assert_answers_removed(self, repository):
        paths = [Path(call["arguments"][call["arguments"].index("--output-last-message") + 1])
                 for call in repository.reviewer_calls()]
        for path in paths:
            self.assertFalse(path.parent.exists())
        self.assertEqual(len(paths), len(set(paths)))

    def test_accepts_only_final_answer_with_child_restrictions(self):
        repository = self.prepare()
        repository.environment["CLAUDECODE"] = "parent-session"

        response = repository.stop("diff")
        self.assertEqual(response, {})

        [call] = repository.reviewer_calls()
        arguments = call["arguments"]
        self.assertEqual(call["name"], "codex")
        self.assertEqual(arguments[arguments.index("--sandbox") + 1], "read-only")
        for restriction in ('approval_policy="never"', 'sandbox_mode="read-only"',
                            'model_reasoning_effort="medium"',
                            'mcp_servers={"browser"={enabled=false}}', 'plugins={"external@market"={enabled=false}}'):
            self.assertIn(restriction, arguments)
        for feature in ("hooks", "apps", "enable_mcp_apps", "multi_agent", "multi_agent_v2",
                        "multi_agent_v2_dynamic_tools", "plugins"):
            self.assertIn(["--disable", feature], [arguments[index:index + 2] for index in range(len(arguments) - 1)])
        self.assertIn("--ephemeral", arguments)
        self.assertEqual(arguments[-1], "-")
        self.assertEqual(call["hook_active"], "1")
        self.assertIsNone(call["session_env"])
        self.assertEqual(call["answer_mode"], "0o700")
        self.assertEqual(call["cwd"], str(repository.root))
        self.assertIn("Implement widgets", call["prompt"])
        self.assertIn("review-result.schema.json", arguments[arguments.index("--output-schema") + 1])
        self.assertEqual(repository.environment["CLAUDECODE"], "parent-session")
        self.assert_answers_removed(repository)

    def test_quotes_external_tool_identities_inside_toml_tables(self):
        import tomllib
        server_name = 'browser.with.dots and "quotes"'
        plugin_id = 'plugin.with.dots@market'
        repository = self.prepare({"servers": [{"name": server_name}],
                                   "plugins": {"installed": [{"pluginId": plugin_id}]}})

        self.assertEqual(repository.stop("diff"), {})
        [call] = repository.reviewer_calls()

        overrides = {}
        for argument in call["arguments"]:
            if argument.startswith(("mcp_servers=", "plugins=")):
                overrides.update(tomllib.loads(argument))

        self.assertEqual(overrides, {"mcp_servers": {server_name: {"enabled": False}},
                                     "plugins": {plugin_id: {"enabled": False}}})
        self.assert_answers_removed(repository)

    def test_rejects_unusable_final_answers_without_switching_clients(self):
        for config, reason in (
            ({"missing_answer": True}, "final answer"),
            ({"answer_text": ""}, "final answer"),
            ({"answer_text": "not json"}, "Expecting value"),
            ({"answer_text": "[]"}, "non-object"),
            ({"result": {"ok": "true", "reason": ""}}, "schema"),
            ({"result": {"ok": True, "reason": "", "extra": True}}, "schema"),
            ({"exit_code": 1, "output": "authentication failed"}, "authentication failed"),
            ({"exit_code": 1, "output": "model unavailable"}, "model unavailable"),
            ({"exit_code": 1, "output": "unsupported reasoning effort"}, "unsupported reasoning effort"),
            ({"metadata_exit_code": 2}, "restrictions unavailable"),
            ({"reject_overrides": True}, "managed policy rejects reviewer restrictions"),
            ({"servers": {}}, "metadata"),
            ({"plugins": {"installed": [{}]}}, "plugin identity"),
        ):
            with self.subTest(config=config):
                repository = self.prepare(config)

                response = repository.stop("diff")
                self.assertEqual(response["decision"], "block")
                self.assertIn(reason, response["reason"])
                self.assertIn("codex", response["reason"])
                self.assertEqual(repository.calls("claude"), [])
                self.assert_answers_removed(repository)
                self.assertEqual((repository.root / ".codex/config.toml").read_text(),
                                 'sandbox_mode = "danger-full-access"\napproval_policy = "on-request"\n')
                if "metadata_exit_code" in config or "servers" in config or "plugins" in config:
                    self.assertEqual(repository.reviewer_calls(), [])

    def test_rejects_schema_invalid_full_answers(self):
        for gate in ("implementation", "proposal"):
            for answer in ({"findings": "none"}, {"findings": [{}]},
                           {"findings": [{"category": "invalid", "file": "x", "summary": "x"}]},
                           {"findings": [], "extra": True}):
                with self.subTest(gate=gate, answer=answer):
                    repository = HookRepository(self)
                    repository.submit_prompt("Implement widgets" if gate == "implementation" else "Write a spec")
                    if gate == "implementation":
                        repository.complete_tasks()
                    else:
                        repository.write_proposal()
                    repository.set_stub("codex", {"result": answer})

                    response = repository.stop(gate)

                    self.assertIn("schema", response["reason"])
                    self.assert_answers_removed(repository)

    def test_selected_cli_missing_never_starts_other_client(self):
        repository = HookRepository(self, missing_tools=("codex",))
        repository.submit_prompt("Implement widgets")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")
        self.assertIn("codex", response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_nested_hook_does_not_replace_original_snapshot(self):
        repository = self.prepare({"nested_snapshot": str(HOOK_PATH)})
        snapshot_path = repository.root / ".git/agent-review-hooks/session-1.json"
        original_snapshot = snapshot_path.read_bytes()
        self.assertEqual(repository.stop("diff"), {})
        self.assertEqual(snapshot_path.read_bytes(), original_snapshot)

    def test_metadata_and_execution_share_timeout_and_remove_answers(self):
        for config in ({"metadata_seconds": 1}, {"sleep_seconds": 5},
                       {"metadata_seconds": 0.15, "sleep_seconds": 0.3}):
            with self.subTest(config=config):
                repository = self.prepare(config)
                payload = {"cwd": str(repository.root), "session_id": "session-1", "hook_event_name": "Stop"}
                output = io.StringIO()

                with mock.patch.dict(os.environ, repository.environment, clear=True), \
                     mock.patch.object(sys, "argv", ["review_gate.py", "diff"]), \
                     mock.patch.object(sys, "stdin", io.StringIO(json.dumps(payload))), \
                     mock.patch.object(sys, "stdout", output), \
                     mock.patch.object(review_gate, "QUICK_REVIEWER", replace(review_gate.QUICK_REVIEWER, timeout_seconds=0.5)):
                    review_gate.main()

                self.assertIn("timed out", json.loads(output.getvalue())["reason"])
                self.assertEqual(repository.calls("claude"), [])
                self.assert_answers_removed(repository)
