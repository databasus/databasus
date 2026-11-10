import json
import os
import shutil
import subprocess
import tempfile
import tomllib
import unittest
from collections.abc import Sequence
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK_NAMES = ("formatting", "source", "architecture", "styles", "types", "generated-api")
LINT_COMMANDS = (
    ("pnpm", "--dir", "frontend", "lint"),
    ("mise", "run", "frontend:lint"),
    ("mise", "run", "lint"),
)
CHECK_PROBE = """#!/usr/bin/env node
import { basename } from 'node:path';

const checkNames = {
    'oxfmt': 'formatting',
    'oxlint': 'source',
    'fsd.ts': 'architecture',
    'styles.ts': 'styles',
    'tsc': 'types',
    'generated-api-probe.mjs': 'generated-api',
};
const checkName = checkNames[basename(process.argv[1])];
if (!checkName) throw new Error('Unknown lint check probe');

if (process.env.FAIL_FRONTEND_CHECK === checkName) {
    console.error(`intentional frontend check failure: ${checkName}`);
    process.exit(23);
}

console.log(`frontend check passed: ${checkName}`);
"""


class FrontendLintExitStatusTest(unittest.TestCase):
    def setUp(self) -> None:
        temporary_directory = tempfile.TemporaryDirectory(prefix="frontend-lint-")
        self.addCleanup(temporary_directory.cleanup)
        self.workspace = Path(temporary_directory.name)
        frontend = self.workspace / "frontend"
        lint = frontend / "lint"
        binaries = frontend / "node_modules" / ".bin"
        lint.mkdir(parents=True)
        binaries.mkdir(parents=True)
        shutil.copyfile(ROOT / "frontend/lint/check.sh", lint / "check.sh")

        probe = lint / "check-probe.mjs"
        probe.write_text(CHECK_PROBE)
        probe.chmod(0o755)
        for binary_name in ("oxfmt", "oxlint", "tsc"):
            (binaries / binary_name).symlink_to(probe)
        for script_name in ("fsd.ts", "styles.ts", "generated-api-probe.mjs"):
            (lint / script_name).symlink_to(probe)

        package = json.loads((ROOT / "frontend/package.json").read_text())
        (frontend / "package.json").write_text(
            json.dumps(
                {
                    "packageManager": package["packageManager"],
                    "scripts": {
                        "lint": package["scripts"]["lint"],
                        "typecheck": package["scripts"]["typecheck"],
                        "generate:api": "node lint/generated-api-probe.mjs",
                    },
                }
            )
        )

        repository_configuration = tomllib.loads((ROOT / "mise.toml").read_text())
        tasks = repository_configuration["tasks"]
        configuration = ["[tools]"]
        for tool_name in ("node", "pnpm"):
            configuration.append(
                f"{json.dumps(tool_name)} = {json.dumps(repository_configuration['tools'][tool_name])}"
            )
        configuration.extend(
            ("[tool_alias]", f"pnpm = {json.dumps(repository_configuration['tool_alias']['pnpm'])}")
        )
        for task_name in ("lint", "frontend:lint"):
            configuration.append(f"[tasks.{json.dumps(task_name)}]")
            for field_name in ("run", "dir"):
                if field_name in tasks[task_name]:
                    configuration.append(
                        f"{field_name} = {json.dumps(tasks[task_name][field_name])}"
                    )
        for task_name in ("backend:lint", "python:lint", "lint-shell", "lint-workflows"):
            configuration.extend((f"[tasks.{json.dumps(task_name)}]", 'run = "true"'))
        (self.workspace / "mise.toml").write_text("\n".join(configuration) + "\n")

    def run_lint(
        self, command: Sequence[str], failing_check: str
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            command,
            cwd=self.workspace,
            env={
                **os.environ,
                "MISE_TRUSTED_CONFIG_PATHS": str(self.workspace),
                "FAIL_FRONTEND_CHECK": failing_check,
            },
            text=True,
            capture_output=True,
            timeout=30,
        )

    def test_succeeds_through_every_entry_point_when_all_frontend_checks_pass(self) -> None:
        for command in LINT_COMMANDS:
            with self.subTest(command=command):
                lint_process = self.run_lint(command, "")

                self.assertEqual(
                    lint_process.returncode, 0, lint_process.stdout + lint_process.stderr
                )
                for check_name in CHECK_NAMES:
                    self.assertIn(f"frontend check passed: {check_name}", lint_process.stdout)

    def test_fails_through_every_entry_point_when_any_frontend_check_fails(self) -> None:
        for check_name in CHECK_NAMES:
            for command in LINT_COMMANDS:
                with self.subTest(check=check_name, command=command):
                    lint_process = self.run_lint(command, check_name)

                    self.assertNotEqual(
                        lint_process.returncode, 0, lint_process.stdout + lint_process.stderr
                    )
                    self.assertIn(
                        f"intentional frontend check failure: {check_name}",
                        lint_process.stdout + lint_process.stderr,
                    )


if __name__ == "__main__":
    unittest.main()
