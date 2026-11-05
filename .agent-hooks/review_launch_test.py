import json
import shutil
import subprocess
import tomllib
import unittest

import review_gate_test as fixtures
from review_gate_test import CHANGE_NAME, HOOK_PATH


class ReviewLaunchTest(unittest.TestCase):
    def test_doctor_checks_only_selected_cli(self):
        for client in ("codex", "claude"):
            other = "claude" if client == "codex" else "codex"
            for missing in ((), (other,), (client,)):
                with self.subTest(client=client, missing=missing):
                    repository = fixtures.HookRepository(self, missing_tools=missing, is_path_isolated=True)
                    repository.environment["AGENT_REVIEW_CLIENT"] = client

                    process = repository.run_doctor()

                    self.assertEqual(process.returncode, 1 if client in missing else 0, process.stdout + process.stderr)
                    self.assertIn(f"Review client: {client}", process.stdout)
                    self.assertIn(f"{client}:", process.stdout)
                    self.assertNotIn(f"{other}:", process.stdout)
                    self.assertEqual(repository.reviewer_calls(), [])

    def test_launcher_forwards_selection_and_keeps_terminal_failures_as_text(self):
        for client in ("codex", "claude"):
            for action in ("doctor", "review", "snapshot"):
                for failure in ("missing-mise", "missing-tools", "success"):
                    with self.subTest(client=client, action=action, failure=failure):
                        launcher = fixtures.Launcher(self)
                        if failure != "missing-mise":
                            launcher.install_mise(launcher.bin_dir)
                        if failure == "missing-tools":
                            launcher.set_mise_answer("ls", stdout=fixtures.MISSING_OPENSPEC_LINE)
                        if failure == "success":
                            launcher.set_mise_answer("exec", stdout="{}")

                        process = launcher.run(action, "--client", client)

                        if failure == "success":
                            self.assertEqual(process.returncode, 0, process.stderr)
                            [call] = launcher.mise_calls("exec")
                            self.assertEqual(call["arguments"][-3:], [action, "--client", client])
                        elif action == "snapshot":
                            self.assertEqual(process.returncode, 0)
                            self.assertIs(json.loads(process.stdout)["continue"], False)
                        else:
                            self.assertEqual(process.returncode, 1)
                            self.assertEqual(process.stdout, "")
                            self.assertTrue(process.stderr.strip())

    def test_tracked_mise_tasks_forward_option_and_agent_environment(self):
        mise_path = shutil.which("mise")
        configuration = tomllib.loads((HOOK_PATH.parent.parent / "mise.toml").read_text())
        for action in ("doctor", "review"):
            for inherited, explicit, selected in (
                ("codex", None, "codex"), ("claude", None, "claude"),
                ("codex", "claude", "claude"), ("invalid", "codex", "codex"),
                (None, "codex", "codex"), (None, "claude", "claude"), (None, None, None),
            ):
                with self.subTest(action=action, inherited=inherited, explicit=explicit):
                    repository = fixtures.HookConfigurationTest.create_repository(self)
                    task = configuration["tasks"][action]
                    task_config = "[tasks." + action + "]\n" + "\n".join(
                        f"{key} = {json.dumps(value)}" for key, value in task.items()) + "\n"
                    repository.write("mise.toml", task_config)
                    repository.complete_tasks()

                    repository.environment["MISE_TRUSTED_CONFIG_PATHS"] = str(repository.root)
                    repository.environment["MISE_YES"] = "1"
                    repository.environment.pop("AGENT_REVIEW_CLIENT", None)
                    if inherited is not None:
                        repository.environment["AGENT_REVIEW_CLIENT"] = inherited

                    arguments = [mise_path, "run", action]
                    if action == "review":
                        arguments.append(CHANGE_NAME)
                    if explicit is not None:
                        arguments.extend(["--client", explicit])

                    process = subprocess.run(arguments, cwd=repository.root, env=repository.environment,
                                             stdin=subprocess.DEVNULL, text=True, capture_output=True)

                    self.assertEqual(process.returncode, 0 if selected else 2, process.stdout + process.stderr)
                    if selected:
                        self.assertIn(f"Review client: {selected}", process.stdout)
                    else:
                        self.assertIn("--client codex or --client claude", process.stderr)
                    if action == "review" and selected:
                        self.assertTrue(repository.reviewer_calls())
                        self.assertTrue(all(call["name"] == selected for call in repository.reviewer_calls()))
                    else:
                        self.assertEqual(repository.reviewer_calls(), [])
