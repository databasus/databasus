import json
from pathlib import Path
import shutil
import subprocess
import sys
import unittest

from review_gate_test import CHANGE_NAME, HOOK_PATH, PROPOSAL_NAME, HookRepository, finding


class ReviewProfileStateTest(unittest.TestCase):
    def prepare(self, gate):
        repository = HookRepository(self, is_path_isolated=True)
        repository.submit_prompt("Implement widgets" if gate == "implementation" else "Write a spec", session_id="first")
        repository.submit_prompt("Inspect widgets", session_id="second")
        if gate == "implementation":
            repository.complete_tasks()
        else:
            repository.write_proposal()
        return repository

    def test_reuses_acceptance_across_sessions_with_original_baselines(self):
        for client in ("claude", "codex"):
            for gate in ("implementation", "proposal"):
                with self.subTest(client=client, gate=gate):
                    repository = self.prepare(gate)
                    repository.environment["AGENT_REVIEW_CLIENT"] = client

                    repository.stop(gate, session_id="first")
                    repository.clear_calls()

                    self.assertEqual(repository.stop(gate, session_id="second"), {})
                    self.assertEqual(repository.reviewer_calls(), [])
                    snapshots = [repository.root / f".git/agent-review-hooks/{session}.json" for session in ("first", "second")]
                    self.assertTrue(all(path.is_file() for path in snapshots))

    def test_ignores_other_clients_acceptance_and_findings(self):
        for gate in ("implementation", "proposal"):
            for has_findings in (False, True):
                with self.subTest(gate=gate, has_findings=has_findings):
                    repository = self.prepare(gate)
                    category = "bug" if gate == "implementation" else "gap"
                    repository.set_reviewer_result({"findings": [finding(category, "Private Claude finding")] if has_findings else []})

                    repository.stop(gate, session_id="first")
                    repository.environment["AGENT_REVIEW_CLIENT"] = "codex"
                    repository.clear_calls()
                    response = repository.stop(gate, session_id="second")

                    self.assertNotIn("could not run", response.get("reason", ""))
                    calls = repository.reviewer_calls()
                    self.assertEqual(len(calls), 4 if gate == "implementation" else 6)
                    self.assertTrue(all(call["name"] == "codex" for call in calls))
                    self.assertFalse(any("\nRole: repeat review\n" in call["prompt"] for call in calls))
                    self.assertFalse(any("Private Claude finding" in call["prompt"] for call in calls))

    def test_client_switch_without_edits_does_not_trigger_review(self):
        for gate in ("implementation", "proposal"):
            repository = self.prepare(gate)
            repository.stop(gate, session_id="first")
            repository.environment["AGENT_REVIEW_CLIENT"] = "codex"
            repository.submit_prompt("Explain widgets", session_id="fresh")
            repository.clear_calls()

            self.assertEqual(repository.stop(gate, session_id="fresh"), {})
            self.assertEqual(repository.calls(), [])

    def test_local_model_configuration_change_invalidates_codex_acceptance(self):
        for gate in ("implementation", "proposal"):
            for location in ("home", "project"):
                with self.subTest(gate=gate, location=location):
                    repository = self.prepare(gate)
                    repository.environment["AGENT_REVIEW_CLIENT"] = "codex"
                    config_path = (Path(repository.environment["CODEX_HOME"]) if location == "home" else repository.root / ".codex") / "config.toml"
                    config_path.parent.mkdir(parents=True, exist_ok=True)
                    config_path.write_text('model = "first-model"\n')

                    repository.stop(gate, session_id="first")
                    config_path.write_text('model = "second-model"\n')
                    repository.clear_calls()
                    repository.stop(gate, session_id="second")

                    self.assertEqual(len(repository.reviewer_calls()), 4 if gate == "implementation" else 6)
                    state_dir = repository.root / ".git/agent-review-hooks/codex"
                    self.assertEqual(len(list(state_dir.glob(f"*/{gate}/*.json"))), 2)
                    for path in state_dir.glob(f"*/{gate}/*.json"):
                        self.assertNotIn("first-model", path.read_text())
                        self.assertNotIn("second-model", path.read_text())

    def test_profile_contract_model_and_effort_changes_start_full_review(self):
        for relative_path, original, replacement in (
            ("review_client.py", '"opus"', '"different-model"'),
            ("review_client.py", '"low", "medium"', '"medium", "high"'),
            ("common-prompt.md", "", "\nAdditional review rule.\n"),
            ("implementation-result.schema.json", "", "\n"),
            ("review_profile.py", "EXECUTION_POLICY_VERSION = 1", "EXECUTION_POLICY_VERSION = 2"),
        ):
            with self.subTest(path=relative_path, replacement=replacement):
                repository = self.prepare("implementation")
                hooks_dir = repository.root / ".agent-hooks"
                shutil.copytree(HOOK_PATH.parent, hooks_dir, ignore=shutil.ignore_patterns("__pycache__"))
                script_path = hooks_dir / "review_gate.py"

                repository.set_reviewer_result({"findings": [finding("bug", "Old profile finding")]})
                self.stop_script(repository, script_path)

                contract_path = hooks_dir / relative_path
                text = contract_path.read_text()
                contract_path.write_text(text.replace(original, replacement) if original else text + replacement)

                repository.set_reviewer_result({"findings": []})
                repository.clear_calls()
                self.stop_script(repository, script_path)

                self.assertEqual(len(repository.reviewer_calls()), 4)
                self.assertFalse(any("\nRole: repeat review\n" in call["prompt"] for call in repository.reviewer_calls()))

    def stop_script(self, repository, script_path):
        process = subprocess.run([sys.executable, str(script_path), "implementation"], cwd=repository.root,
                                 env=repository.environment, input=json.dumps({"session_id": "first", "hook_event_name": "Stop"}),
                                 text=True, capture_output=True, check=True)
        return json.loads(process.stdout)

    def test_old_unscoped_cache_is_ignored_and_preserved(self):
        for gate, name in (("implementation", CHANGE_NAME), ("proposal", PROPOSAL_NAME)):
            repository = self.prepare(gate)
            old_path = repository.root / f".git/agent-review-hooks/{gate}/{name}.json"
            old_path.parent.mkdir(parents=True)
            old_text = json.dumps({"accepted": "anything", "findings": [finding("bug", "Old unscoped finding")]})
            old_path.write_text(old_text)

            repository.stop(gate, session_id="first")
            self.assertEqual(len(repository.reviewer_calls()), 4 if gate == "implementation" else 6)
            self.assertFalse(any("\nRole: repeat review\n" in call["prompt"] for call in repository.reviewer_calls()))
            self.assertEqual(old_path.read_text(), old_text)

    def test_quick_check_suppression_and_summary_use_selected_profile(self):
        repository = self.prepare("proposal")
        repository.set_reviewer_result({"findings": [finding("gap", "Claude proposal finding")]})
        repository.stop("proposal", session_id="first")

        repository.environment["AGENT_REVIEW_CLIENT"] = "codex"
        repository.clear_calls()

        self.assertEqual(repository.stop("diff", session_id="second"), {})
        self.assertEqual(repository.reviewer_calls(), [])

        response = repository.stop("proposal", session_id="second")
        self.assertNotIn("Claude proposal finding", response.get("reason", ""))
        self.assertEqual(len(repository.reviewer_calls()), 6)

        repository.clear_calls()
        self.assertEqual(repository.stop("proposal", session_id="second"), {})
        self.assertEqual(repository.calls(), [])

    def test_necessity_uses_only_implementation_findings_of_current_profile(self):
        for client in ("claude", "codex"):
            for scenario in ("same-profile", "other-client", "changed-profile"):
                if scenario == "changed-profile" and client != "codex":
                    continue
                with self.subTest(client=client, scenario=scenario):
                    repository = self.prepare("implementation")
                    repository.environment["AGENT_REVIEW_CLIENT"] = client
                    repository.set_stub(client, {"result": {"findings": [finding("spec_mismatch", "Correct the delta spec")]}})

                    response = repository.stop("implementation", session_id="first")
                    self.assertIn("Correct the delta spec", response["reason"])

                    if scenario == "other-client":
                        repository.environment["AGENT_REVIEW_CLIENT"] = "codex" if client == "claude" else "claude"
                    elif scenario == "changed-profile":
                        config_path = Path(repository.environment["CODEX_HOME"]) / "config.toml"
                        config_path.parent.mkdir(parents=True, exist_ok=True)
                        config_path.write_text('model = "another-model"\n')

                    repository.set_stub(client, {})
                    repository.clear_calls()
                    repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n")

                    self.assertEqual(repository.stop("necessity", session_id="second"), {})

                    calls = repository.reviewer_calls()
                    self.assertEqual(len(calls), 0 if scenario == "same-profile" else 1)
                    if calls:
                        self.assertEqual(calls[0]["name"], repository.environment["AGENT_REVIEW_CLIENT"])
