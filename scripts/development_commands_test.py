import json
import os
import shutil
import subprocess
import tempfile
import tomllib
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def get_mise_config() -> dict[str, object]:
    return tomllib.loads((ROOT / "mise.toml").read_text())


def get_prepare_command() -> str:
    tasks = get_mise_config()["tasks"]
    assert isinstance(tasks, dict)
    prepare_task = tasks["dev:prepare"]
    assert isinstance(prepare_task, dict)

    return str(prepare_task["run"])


def is_task_auto_install_enabled() -> bool:
    settings = get_mise_config()["settings"]
    assert isinstance(settings, dict)
    task_settings = settings["task"]
    assert isinstance(task_settings, dict)

    return bool(task_settings["run_auto_install"])


class DevelopmentCommandsTest(unittest.TestCase):
    def test_the_narrow_test_commands_are_gone(self) -> None:
        mise_config = (ROOT / "mise.toml").read_text()

        self.assertNotIn("backend:test-package", mise_config)
        self.assertFalse((ROOT / "mise-tasks/backend/test-group").exists())

    def test_only_the_supported_user_commands_are_visible_from_every_module(self) -> None:
        user_commands = {
            "run",
            "test",
            "test-change",
            "lint",
            "format",
            "doctor",
            "review",
            "generate-api",
            "verify-deps",
        }
        removed_commands = {
            "dev",
            "full",
            "test-full",
            "pull-testcontainers",
            "backend:generate-api",
            "backend:clean-testcontainers",
            "agent:run",
            "agent:build",
        }
        for directory in (ROOT, ROOT / "backend", ROOT / "frontend"):
            with self.subTest(directory=directory):
                tasks = json.loads(
                    subprocess.check_output(
                        ["mise", "tasks", "--hidden", "--json"], cwd=directory, text=True
                    )
                )

                self.assertEqual(
                    {task["name"] for task in tasks if not task["hide"]}, user_commands
                )
                self.assertFalse(removed_commands & {task["name"] for task in tasks})
                self.assertTrue(
                    {"backend:lint", "frontend:lint", "e2e:browser", "agent:test"}
                    <= {task["name"] for task in tasks}
                )
                self.assertTrue(all(Path(task["dir"]).is_dir() for task in tasks))

    def test_the_local_app_installs_missing_tools_before_it_starts(self) -> None:
        dependency_tree = subprocess.check_output(
            ["mise", "tasks", "deps", "run"], cwd=ROOT, text=True
        )

        self.assertIn("dev:prepare", dependency_tree)

    def test_the_local_app_does_not_start_when_a_pinned_tool_cannot_be_installed(self) -> None:
        for prepare_command, is_expected_to_start in (
            ("true", True),
            (get_prepare_command(), False),
        ):
            with self.subTest(prepare_command=prepare_command):
                project = Path(tempfile.mkdtemp(prefix="dev-prepare-"))
                self.addCleanup(shutil.rmtree, project, ignore_errors=True)
                (project / "mise.toml").write_text(
                    '[tools]\njq = "0.0.1"\n\n'
                    f"[settings.task]\nrun_auto_install = {json.dumps(is_task_auto_install_enabled())}\n\n"
                    f"[tasks.prepare]\nrun = {json.dumps(prepare_command)}\n\n"
                    '[tasks.run]\ndepends = ["prepare"]\nrun = "touch started"\n'
                )
                environment = {
                    name: value
                    for name, value in os.environ.items()
                    if not name.startswith("MISE_")
                }
                environment["MISE_TRUSTED_CONFIG_PATHS"] = str(project)

                completed_run = subprocess.run(
                    ["mise", "run", "run"],
                    cwd=project,
                    env=environment,
                    capture_output=True,
                    text=True,
                    timeout=120,
                )

                self.assertEqual(completed_run.returncode == 0, is_expected_to_start)
                self.assertEqual((project / "started").exists(), is_expected_to_start)
                if not is_expected_to_start:
                    self.assertIn("jq@0.0.1 is not in the lockfile", completed_run.stderr)

    def test_frontend_exposes_only_the_supported_scripts(self) -> None:
        scripts = json.loads((ROOT / "frontend/package.json").read_text())["scripts"]

        self.assertEqual(
            set(scripts),
            {"dev", "build", "lint", "format", "typecheck", "test", "test:browser", "generate:api"},
        )


if __name__ == "__main__":
    unittest.main()
