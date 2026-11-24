import json
import os
import re
import shutil
import subprocess
import tempfile
import tomllib
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SETTING_KEY = re.compile(r"^([A-Z_][A-Z0-9_]*)=", re.M)


def get_example_settings() -> dict[str, str]:
    example_settings: dict[str, str] = {}
    for line in (ROOT / ".env.example").read_text().splitlines():
        if not SETTING_KEY.match(line):
            continue

        key, value = line.split("=", 1)
        example_settings[key] = value.strip('"')

    return example_settings


def get_settings_files() -> list[str]:
    mise_config = tomllib.loads((ROOT / "mise.toml").read_text())
    settings_files = mise_config["env"]["_"]["file"]
    assert isinstance(settings_files, list)

    return [str(settings_file) for settings_file in settings_files]


class LocalSettingsTest(unittest.TestCase):
    """Loads the repository's settings files the way mise.toml declares them, in a directory of
    its own, so the checkout's .env stays out of the test."""

    def setUp(self) -> None:
        self.project = Path(tempfile.mkdtemp(prefix="local-settings-"))
        self.addCleanup(shutil.rmtree, self.project, ignore_errors=True)

        shutil.copy(ROOT / ".env.example", self.project / ".env.example")
        settings_files = json.dumps(get_settings_files())
        (self.project / "mise.toml").write_text(
            f'[env]\n_.file = {settings_files}\n\n[tasks.print-settings]\nrun = "env"\n'
        )

        self.example_settings = get_example_settings()
        self.environment = {
            name: value
            for name, value in os.environ.items()
            if name not in self.example_settings and not name.startswith("MISE_")
        }
        self.environment["MISE_TRUSTED_CONFIG_PATHS"] = str(self.project)

    def get_shell_settings(self) -> dict[str, str]:
        shell_settings = json.loads(
            subprocess.check_output(
                ["mise", "env", "--json"], cwd=self.project, env=self.environment, text=True
            )
        )
        assert isinstance(shell_settings, dict)

        return {str(name): str(value) for name, value in shell_settings.items()}

    def get_task_settings(self) -> dict[str, str]:
        task_output = subprocess.check_output(
            ["mise", "run", "--quiet", "print-settings"],
            cwd=self.project,
            env=self.environment,
            text=True,
        )

        return dict(line.split("=", 1) for line in task_output.splitlines() if "=" in line)

    def write_local_settings(self, content: str) -> None:
        (self.project / ".env").write_text(content)

    def test_task_sees_the_example_values_without_a_local_file(self) -> None:
        task_settings = self.get_task_settings()

        for key in ("DATABASE_DSN", "TEST_DATABASE_DSN", "DEV_DB_PASSWORD", "ENV_MODE"):
            with self.subTest(key=key):
                self.assertEqual(task_settings[key], self.example_settings[key])

    def test_local_file_overrides_only_the_keys_it_sets(self) -> None:
        self.write_local_settings("LOG_LEVEL=debug\n")

        task_settings = self.get_task_settings()

        self.assertEqual(task_settings["LOG_LEVEL"], "debug")
        self.assertEqual(task_settings["DATABASE_DSN"], self.example_settings["DATABASE_DSN"])
        self.assertEqual(task_settings["SMTP_PORT"], self.example_settings["SMTP_PORT"])

    def test_full_copy_of_the_example_keeps_its_changed_value(self) -> None:
        example_content = (ROOT / ".env.example").read_text()
        self.write_local_settings(example_content.replace("LOG_LEVEL=info", "LOG_LEVEL=warn"))

        task_settings = self.get_task_settings()

        self.assertEqual(task_settings["LOG_LEVEL"], "warn")
        self.assertEqual(
            task_settings["TEST_DATABASE_DSN"], self.example_settings["TEST_DATABASE_DSN"]
        )

    def test_key_added_to_the_example_reaches_the_next_task(self) -> None:
        self.write_local_settings("LOG_LEVEL=debug\n")
        self.get_task_settings()

        with (self.project / ".env.example").open("a") as example_file:
            example_file.write("NEWLY_ADDED_SETTING=from-example\n")
        task_settings = self.get_task_settings()

        self.assertEqual(task_settings["NEWLY_ADDED_SETTING"], "from-example")

    def test_edit_to_the_local_file_reaches_the_next_task_without_an_export(self) -> None:
        self.write_local_settings("LOG_LEVEL=debug\n")
        first_settings = self.get_task_settings()

        self.write_local_settings("LOG_LEVEL=error\n")
        second_settings = self.get_task_settings()

        self.assertEqual(first_settings["LOG_LEVEL"], "debug")
        self.assertEqual(second_settings["LOG_LEVEL"], "error")

    def test_activated_shell_holds_the_migration_settings(self) -> None:
        shell_settings = self.get_shell_settings()

        self.assertEqual(shell_settings["GOOSE_DRIVER"], self.example_settings["GOOSE_DRIVER"])
        self.assertEqual(shell_settings["GOOSE_DBSTRING"], self.example_settings["GOOSE_DBSTRING"])


if __name__ == "__main__":
    unittest.main()
