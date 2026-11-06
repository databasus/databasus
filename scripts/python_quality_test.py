import json
import os
import shutil
import subprocess
import sys
import tempfile
import tomllib
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class PythonQualityTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        subprocess.run(
            ["uv", "sync", "--locked", "--project", str(ROOT / "e2e/browser")],
            env={
                **os.environ,
                "UV_PYTHON_PREFERENCE": "only-system",
                "PYTHONDONTWRITEBYTECODE": "1",
            },
            capture_output=True,
            text=True,
            timeout=60,
            check=True,
        )

    def setUp(self) -> None:
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.environment = {
            **os.environ,
            "PYTHONDONTWRITEBYTECODE": "1",
            "UV_PYTHON_PREFERENCE": "only-system",
            "MISE_TASK_RUN_AUTO_INSTALL": "false",
            "MISE_TRUSTED_CONFIG_PATHS": str(self.root),
        }
        self.git("init", "--quiet")
        for path in (".agent-hooks", "e2e/browser/scenarios"):
            (self.root / path).mkdir(parents=True)
        for path in (
            "scripts/python_quality.py",
            "pyproject.toml",
            "e2e/browser/pyproject.toml",
            "e2e/browser/uv.lock",
            "mise.toml",
            "hk.pkl",
        ):
            target = self.root / path
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / path, target)
        (self.root / "e2e/browser/.venv").symlink_to(
            ROOT / "e2e/browser/.venv", target_is_directory=True
        )
        (self.root / ".gitignore").write_text(".venv/\n__pycache__/\n.ruff_cache/\nignored/\n")
        self.git("add", ".")
        self.git(
            "-c",
            "user.name=Quality tests",
            "-c",
            "user.email=quality@example.com",
            "commit",
            "--quiet",
            "--no-verify",
            "-m",
            "Initial fixture",
        )

    def git(self, *arguments: str) -> str:
        return subprocess.check_output(["git", *arguments], cwd=self.root, text=True)

    def write_source(self, path: str, content: str) -> Path:
        source = self.root / path
        source.parent.mkdir(parents=True, exist_ok=True)
        source.write_text(content)
        return source

    def run_quality(self, mode: str = "lint") -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(self.root / "scripts/python_quality.py"), mode],
            cwd=self.root,
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )

    def run_hk(self, *arguments: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["hk", "check", "--check", *arguments],
            cwd=self.root,
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )

    def get_python_step_status(self, *arguments: str) -> str:
        plan = self.run_hk("--plan", "--json", *arguments)
        self.assertEqual(plan.returncode, 0, plan.stdout + plan.stderr)
        parsed_plan: object = json.loads(plan.stdout)
        assert isinstance(parsed_plan, dict)
        steps = parsed_plan.get("steps")
        assert isinstance(steps, list)
        for step in steps:
            assert isinstance(step, dict)
            if step.get("name") == "python-lint":
                status = step.get("status")
                assert isinstance(status, str)
                return status
        self.fail("Python lint is missing from the hk plan")

    def test_selects_the_python_hook_when_sources_or_quality_configuration_change(self) -> None:
        for path in (
            "new area/new.py",
            ".hidden/hook.py",
            "pyproject.toml",
            "new area/pyproject.toml",
            "e2e/browser/uv.lock",
            "new area/requirements-dev.txt",
            "mise.toml",
            "hk.pkl",
            ".gitignore",
            "new area/.gitignore",
            "ruff.toml",
            ".ruff.toml",
            "ty.toml",
            "e2e/browser/.python-version",
        ):
            with self.subTest(path=path):
                if not (self.root / path).exists():
                    self.write_source(path, "")
                self.assertEqual(self.get_python_step_status(path), "included")

        self.write_source("docs/guide.md", "Documentation\n")
        self.assertEqual(self.get_python_step_status("docs/guide.md"), "skipped")

    def test_selects_the_python_hook_when_a_source_is_deleted_or_renamed(self) -> None:
        for path in ("deleted.py", "renamed.py"):
            self.write_source(path, "VALUE = 1\n")
        self.git("add", ".")
        self.git(
            "-c",
            "user.name=Quality tests",
            "-c",
            "user.email=quality@example.com",
            "commit",
            "--quiet",
            "--no-verify",
            "-m",
            "Add sources",
        )
        self.git("rm", "deleted.py")
        self.assertEqual(self.get_python_step_status("deleted.py"), "included")
        self.git("mv", "renamed.py", "renamed_source.py")

        self.assertEqual(self.get_python_step_status("--staged"), "included")

    def test_checks_unchanged_callers_once_when_a_selected_source_changes(self) -> None:
        runner = self.root / "scripts/python_quality.py"
        runner.write_text(
            runner.read_text().replace(
                "from pathlib import Path\n",
                'from pathlib import Path\n\nwith (Path(__file__).parent / "quality-invocations").open("a") as invocations:\n    invocations.write("lint\\n")\n',
            )
        )
        callee = self.write_source(
            "scripts/callee.py", "def read_value(value: int) -> int:\n    return value\n"
        )
        self.write_source("scripts/caller.py", "from callee import read_value\n\nread_value(42)\n")
        self.git("add", "scripts")
        self.git(
            "-c",
            "user.name=Quality tests",
            "-c",
            "user.email=quality@example.com",
            "commit",
            "--quiet",
            "--no-verify",
            "-m",
            "Add compatible caller",
        )
        callee.write_text("def read_value(value: str) -> str:\n    return value\n")

        completed_check = self.run_hk("scripts/callee.py")

        self.assertNotEqual(completed_check.returncode, 0)
        output = completed_check.stdout + completed_check.stderr
        self.assertIn("caller.py", output)
        self.assertIn("invalid-argument-type", output)
        self.assertEqual((self.root / "scripts/quality-invocations").read_text(), "lint\n")
        self.assertEqual(self.git("diff", "--", "scripts/caller.py"), "")

    def test_preserves_files_index_and_stash_when_the_python_hook_fails(self) -> None:
        source = self.write_source("broken.py", "def read_value() -> int:\n  return 42\n")
        self.git("add", "broken.py")
        source.write_text("def read_value() -> int:\n  return 43\n")
        other_source = self.write_source("other.py", "VALUE = 1\n")
        staged_content = self.git("show", ":broken.py")
        index = self.git("ls-files", "--stage")
        stash = self.git("stash", "list")

        completed_check = subprocess.run(
            ["hk", "pre-commit"],
            cwd=self.root,
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertEqual(source.read_text(), "def read_value() -> int:\n  return 43\n")
        self.assertEqual(other_source.read_text(), "VALUE = 1\n")
        self.assertEqual(self.git("show", ":broken.py"), staged_content)
        self.assertEqual(self.git("ls-files", "--stage"), index)
        self.assertEqual(self.git("stash", "list"), stash)

    def test_reports_setup_failure_when_the_dependency_lockfile_is_missing(self) -> None:
        (self.root / "e2e/browser/uv.lock").unlink()
        self.environment["UV_OFFLINE"] = "true"

        completed_check = self.run_quality()

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertIn("All checks passed!", completed_check.stdout)
        self.assertIn("lockfile", completed_check.stderr.lower())

    def test_reports_both_failures_when_root_lint_has_python_and_frontend_errors(self) -> None:
        repository_configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        tasks = repository_configuration["tasks"]
        self.write_source("broken.py", "def read_value() -> str:\n    return 42\n")
        configuration = ["[tools]"]
        for tool_name in ("python", "uv", "ruff", "ty"):
            configuration.append(
                f"{tool_name} = {json.dumps(repository_configuration['tools'][tool_name])}"
            )
        configuration.extend(
            [
                "[tasks.lint]",
                f"run = {json.dumps(tasks['lint']['run'])}",
                '[tasks."python:lint"]',
                'run = "python3 scripts/python_quality.py lint"',
            ]
        )
        for name in ("backend:lint", "frontend:lint", "lint-shell", "lint-workflows"):
            command = (
                "printf 'frontend lint failed\\n'; exit 23" if name == "frontend:lint" else "true"
            )
            configuration.extend((f"[tasks.{json.dumps(name)}]", f"run = {json.dumps(command)}"))
        (self.root / "mise.toml").write_text("\n".join(configuration) + "\n")

        for directory in (self.root, self.root / "backend", self.root / "frontend"):
            directory.mkdir(exist_ok=True)
            completed_check = subprocess.run(
                ["mise", "run", "lint"],
                cwd=directory,
                env=self.environment,
                capture_output=True,
                text=True,
                timeout=60,
            )
            self.assertNotEqual(completed_check.returncode, 0)
            output = completed_check.stdout + completed_check.stderr
            self.assertIn("frontend lint failed", output)
            self.assertIn("invalid-return-type", output)

    def test_passes_when_source_and_external_imports_are_typed(self) -> None:
        self.write_source(
            "new area/typed.py",
            "from playwright.sync_api import Page\n\n\ndef get_title(page: Page) -> str:\n    return page.title()\n",
        )

        completed_check = self.run_quality()

        self.assertEqual(
            completed_check.returncode, 0, completed_check.stdout + completed_check.stderr
        )

    def test_rejects_when_new_hidden_or_untracked_source_has_missing_annotations(self) -> None:
        for path in ("new area/untracked.py", ".hidden/hook.py"):
            with self.subTest(path=path):
                source = self.write_source(path, "def read_value(value):\n    return value\n")

                completed_check = self.run_quality()

                self.assertNotEqual(completed_check.returncode, 0)
                self.assertIn("ANN001", completed_check.stdout)
                self.assertIn(path, completed_check.stdout)
                source.unlink()

    def test_reports_both_failures_when_formatting_and_types_are_invalid(self) -> None:
        self.write_source(".hidden/hook.py", "def read_value() -> str:\n  return 42\n")

        completed_check = self.run_quality()

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertIn("would be reformatted", completed_check.stdout)
        self.assertIn("invalid-return-type", completed_check.stdout)

    def test_rejects_when_a_parameter_uses_any(self) -> None:
        self.write_source(
            "untyped.py",
            "from typing import Any\n\n\ndef read_value(value: Any) -> str:\n    return str(value)\n",
        )

        completed_check = self.run_quality()

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertIn("ANN401", completed_check.stdout)

    def test_rejects_when_a_type_error_uses_a_blanket_suppression(self) -> None:
        self.write_source(
            "suppressed.py", "def read_value() -> str:\n    return 42  # ty: ignore\n"
        )

        completed_check = self.run_quality()

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertIn("blanket-ignore-comment", completed_check.stdout)

    def test_fails_and_checks_types_when_the_linter_is_unavailable(self) -> None:
        tools_directory = self.root / "tools"
        tools_directory.mkdir()
        for name in ("git", "uv", "ty"):
            executable = shutil.which(name)
            assert executable is not None
            (tools_directory / name).symlink_to(executable)
        self.environment["PATH"] = str(tools_directory)
        self.write_source("invalid.py", "def read_value() -> str:\n    return 42\n")

        completed_check = self.run_quality()

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertIn("could not start ruff", completed_check.stderr)
        self.assertIn("invalid-return-type", completed_check.stdout)

    def test_skips_when_source_is_deleted_or_ignored(self) -> None:
        source = self.write_source("deleted.py", "invalid Python")
        self.git("add", "deleted.py")
        source.unlink()
        self.write_source("ignored/generated.py", "invalid Python")

        completed_check = self.run_quality()

        self.assertEqual(
            completed_check.returncode, 0, completed_check.stdout + completed_check.stderr
        )

    def test_preserves_files_index_and_stash_when_check_fails(self) -> None:
        source = self.write_source("unformatted.py", "def read_value() -> int:\n  return 42\n")
        self.git("add", "unformatted.py")
        source.write_text("def read_value() -> int:\n  return 43\n")
        staged_content = self.git("show", ":unformatted.py")
        stash_contents = self.git("stash", "list")

        completed_check = self.run_quality()

        self.assertNotEqual(completed_check.returncode, 0)
        self.assertEqual(source.read_text(), "def read_value() -> int:\n  return 43\n")
        self.assertEqual(self.git("show", ":unformatted.py"), staged_content)
        self.assertEqual(self.git("stash", "list"), stash_contents)

    def test_is_idempotent_when_safe_fixes_and_formatting_are_applied(self) -> None:
        source = self.write_source(
            "unformatted.py", "import os\n\ndef read_value() -> int:\n  return 42\n"
        )

        first_format = self.run_quality("format")
        formatted_content = source.read_text()
        second_format = self.run_quality("format")
        completed_check = self.run_quality()

        self.assertEqual(first_format.returncode, 0, first_format.stdout + first_format.stderr)
        self.assertEqual(second_format.returncode, 0, second_format.stdout + second_format.stderr)
        self.assertEqual(
            completed_check.returncode, 0, completed_check.stdout + completed_check.stderr
        )
        self.assertEqual(source.read_text(), formatted_content)
        self.assertNotIn("import os", formatted_content)


if __name__ == "__main__":
    unittest.main()
