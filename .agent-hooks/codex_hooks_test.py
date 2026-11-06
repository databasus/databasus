import json
import unittest
from pathlib import Path

from review_gate_test import HookRepository


class CodexHooksDoctorTest(unittest.TestCase):
    def create_repository(
        self, trust_status: str = "trusted", is_enabled: bool = True
    ) -> HookRepository:
        repository = HookRepository(self)
        repository.environment["AGENT_REVIEW_CLIENT"] = "codex"
        command = (
            '"$(git rev-parse --show-toplevel)/.agent-hooks/run.sh" implementation --client codex'
        )
        repository.write(
            ".codex/hooks.json",
            json.dumps(
                {
                    "hooks": {
                        "Stop": [
                            {
                                "hooks": [
                                    {"type": "command", "command": command},
                                ]
                            }
                        ]
                    }
                }
            ),
        )

        repository.set_stub(
            "codex",
            {
                "hooks_response": {
                    "data": [
                        {
                            "cwd": str(repository.root),
                            "warnings": [],
                            "errors": [],
                            "hooks": [
                                {
                                    "sourcePath": str(repository.root / ".codex/hooks.json"),
                                    "eventName": "stop",
                                    "command": command,
                                    "enabled": is_enabled,
                                    "trustStatus": trust_status,
                                }
                            ],
                        }
                    ]
                }
            },
        )

        return repository

    def create_linked_worktree(self, repository: HookRepository) -> None:
        repository.git("add", ".codex/hooks.json")
        repository.git("commit", "--quiet", "-m", "add project hooks")
        worktree = repository.root.parent / "linked worktree"
        repository.git("worktree", "add", "--quiet", "--detach", str(worktree))
        repository.root = worktree

        configuration = json.loads((repository.stubs_dir / "codex.json").read_text())
        configuration["hooks_response"]["data"][0]["cwd"] = str(worktree)
        repository.set_stub("codex", configuration)

    def set_hook_source(self, repository: HookRepository, source_path: Path) -> None:
        configuration = json.loads((repository.stubs_dir / "codex.json").read_text())
        configuration["hooks_response"]["data"][0]["hooks"][0]["sourcePath"] = str(source_path)
        repository.set_stub("codex", configuration)

    def test_reports_unapproved_hooks_instead_of_a_healthy_environment(self) -> None:
        repository = self.create_repository(trust_status="modified")

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("modified", process.stdout)
        self.assertIn("/hooks", process.stdout)
        self.assertEqual(repository.reviewer_calls(), [])

    def test_accepts_trusted_enabled_hooks_without_starting_a_model(self) -> None:
        repository = self.create_repository()

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertIn("1 trusted and enabled", process.stdout)
        self.assertEqual(repository.reviewer_calls(), [])

    def test_accepts_main_checkout_hooks_when_running_in_a_linked_worktree(self) -> None:
        repository = self.create_repository()
        self.create_linked_worktree(repository)

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertIn("1 trusted and enabled", process.stdout)
        self.assertEqual(repository.reviewer_calls(), [])

    def test_accepts_worktree_hooks_when_running_in_a_linked_worktree(self) -> None:
        repository = self.create_repository()
        self.create_linked_worktree(repository)
        self.set_hook_source(repository, repository.root / ".codex/hooks.json")

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 0, process.stdout + process.stderr)
        self.assertIn("1 trusted and enabled", process.stdout)
        self.assertEqual(repository.reviewer_calls(), [])

    def test_refuses_different_main_checkout_commands_when_running_in_a_linked_worktree(
        self,
    ) -> None:
        repository = self.create_repository()
        self.create_linked_worktree(repository)
        configuration = json.loads((repository.stubs_dir / "codex.json").read_text())
        configuration["hooks_response"]["data"][0]["hooks"][0]["command"] += " --different-command"
        repository.set_stub("codex", configuration)

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("stop: not loaded", process.stdout)

    def test_refuses_unrelated_checkout_hooks_when_running_in_a_linked_worktree(self) -> None:
        repository = self.create_repository()
        self.create_linked_worktree(repository)
        unrelated_repository = HookRepository(self)
        self.set_hook_source(repository, unrelated_repository.root / ".codex/hooks.json")

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("stop: not loaded", process.stdout)

    def test_refuses_other_worktree_hooks_when_running_in_a_linked_worktree(self) -> None:
        repository = self.create_repository()
        self.create_linked_worktree(repository)
        other_worktree = repository.root.parent / "other worktree"
        repository.git("worktree", "add", "--quiet", "--detach", str(other_worktree))
        self.set_hook_source(repository, other_worktree / ".codex/hooks.json")

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("stop: not loaded", process.stdout)

    def test_refuses_disabled_or_unapproved_main_checkout_hooks_in_a_linked_worktree(self) -> None:
        for trust_status, is_enabled, failure in (
            ("modified", True, "modified"),
            ("trusted", False, "disabled"),
        ):
            with self.subTest(trust_status=trust_status, is_enabled=is_enabled):
                repository = self.create_repository(
                    trust_status=trust_status, is_enabled=is_enabled
                )
                self.create_linked_worktree(repository)

                process = repository.run_doctor()

                self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
                self.assertIn(f"stop: {failure}", process.stdout)
                self.assertEqual(repository.reviewer_calls(), [])

    def test_refuses_disabled_or_unloaded_hooks(self) -> None:
        for is_loaded in (True, False):
            with self.subTest(is_loaded=is_loaded):
                repository = self.create_repository(is_enabled=False)
                if not is_loaded:
                    repository.set_stub("codex", {"hooks_response": {"data": []}})

                process = repository.run_doctor()

                self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
                self.assertIn("Codex project hooks:", process.stdout)

    def test_reports_a_protocol_failure_without_accepting_hooks(self) -> None:
        repository = self.create_repository()
        repository.set_stub("codex", {"hooks_response": {"invalid": True}})

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("could not inspect", process.stdout)

    def test_reports_why_app_server_failed_and_how_to_rerun_outside_the_sandbox(self) -> None:
        repository = self.create_repository()
        repository.set_stub(
            "codex", {"app_server_error": "Error: Read-only file system (os error 30)"}
        )

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("Error: Read-only file system (os error 30)", process.stdout)
        self.assertIn("outside the sandbox", process.stdout)

    def test_refuses_trusted_hooks_when_the_feature_is_disabled(self) -> None:
        repository = self.create_repository()
        configuration = json.loads((repository.stubs_dir / "codex.json").read_text())
        configuration["features"] = {"hooks": False}
        repository.set_stub("codex", configuration)

        process = repository.run_doctor()

        self.assertEqual(process.returncode, 1, process.stdout + process.stderr)
        self.assertIn("features.hooks is disabled", process.stdout)
