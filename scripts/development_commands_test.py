import json
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


class DevelopmentCommandsTest(unittest.TestCase):
    def test_the_narrow_test_commands_are_gone(self) -> None:
        mise_config = (ROOT / "mise.toml").read_text()

        self.assertNotIn("backend:test-package", mise_config)
        self.assertFalse((ROOT / "mise-tasks/backend/test-group").exists())

    def test_only_the_supported_user_commands_are_visible_from_every_module(self) -> None:
        user_commands = {
            "run",
            "test",
            "test-full",
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

    def test_frontend_exposes_only_the_supported_scripts(self) -> None:
        scripts = json.loads((ROOT / "frontend/package.json").read_text())["scripts"]

        self.assertEqual(
            set(scripts),
            {"dev", "build", "lint", "format", "typecheck", "test", "test:browser", "generate:api"},
        )


if __name__ == "__main__":
    unittest.main()
