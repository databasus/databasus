import json
import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

from workflow_jobs import get_jobs

ROOT = Path(__file__).resolve().parents[1]
PROVIDER_PACKAGE = "databasus-backend/internal/modules/storages/internal/providers"
OTHER_PACKAGES = [
    "databasus-backend/internal/app/apptest",
    PROVIDER_PACKAGE + "/nas",
]


class BackendCiExecutionTest(unittest.TestCase):
    def run_backend_step(
        self, group: str, packages: list[str], failing_package: str = ""
    ) -> tuple[int, list[list[str]], str]:
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        step = get_jobs(workflow)["test-backend"].split("- name: Run Go tests\n", 1)[1]
        script = textwrap.dedent(step.split("        run: |\n", 1)[1])
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "backend").mkdir()
            binary = workspace / "bin"
            binary.mkdir()
            calls_path = workspace / "calls.jsonl"
            stubs = {
                "mise": 'case "$2" in\n'
                'backend:test-group-packages) printf "%s\\n" "$SELECTED_PACKAGES" ;;\n'
                'backend:test-group-tests) case "$TEST_GROUP" in *-of-*) echo "^(TestSelected)$";; esac ;;\n'
                "*) exit 99 ;;\nesac\n",
                "docker": "exit 0\n",
                "go": "exec python3 - \"$@\" <<'PYTHON'\n"
                "import json, os, sys\n"
                'with open(os.environ["CALLS_PATH"], "a") as calls:\n'
                '    calls.write(json.dumps(sys.argv[1:]) + "\\n")\n'
                'if os.environ["FAILING_PACKAGE"] in sys.argv[1:]:\n'
                "    sys.exit(17)\n"
                "PYTHON\n",
            }
            for name, body in stubs.items():
                executable = binary / name
                executable.write_text("#!/bin/sh\n" + body)
                executable.chmod(0o755)

            invocation = subprocess.run(
                ["bash", "-euo", "pipefail", "-c", script],
                cwd=workspace,
                env={
                    **os.environ,
                    "PATH": f"{binary}:{os.environ['PATH']}",
                    "CALLS_PATH": str(calls_path),
                    "SELECTED_PACKAGES": "\n".join(packages),
                    "FAILING_PACKAGE": failing_package,
                    "TEST_GROUP": group,
                    "TEST_PARALLELISM": "2",
                },
                capture_output=True,
                text=True,
                timeout=20,
            )
            calls: list[list[str]] = []
            if calls_path.exists():
                for line in calls_path.read_text().splitlines():
                    arguments: object = json.loads(line)
                    assert isinstance(arguments, list) and all(
                        isinstance(argument, str) for argument in arguments
                    )
                    calls.append(arguments)

        return invocation.returncode, calls, invocation.stderr

    def test_runs_storage_performance_once_without_fallthrough(self) -> None:
        status, calls, diagnostic = self.run_backend_step("storage-providers", [PROVIDER_PACKAGE])

        self.assertEqual(status, 0, diagnostic)
        self.assertEqual(len(calls), 1)
        self.assertIn("-p=1", calls[0])
        self.assertIn("-parallel=1", calls[0])
        self.assertIn("-race", calls[0])
        self.assertEqual(
            [arg for arg in calls[0] if arg.startswith("databasus-")], [PROVIDER_PACKAGE]
        )

    def test_runs_every_other_shard_once_with_bounded_parallelism(self) -> None:
        for group in ("other-1-of-3", "other-2-of-3", "other-3-of-3", "logical"):
            with self.subTest(group=group):
                status, calls, diagnostic = self.run_backend_step(group, OTHER_PACKAGES)

                self.assertEqual(status, 0, diagnostic)
                self.assertEqual(len(calls), 1)
                self.assertIn("-p=8", calls[0])
                self.assertIn("-parallel=2", calls[0])
                self.assertIn("-race", calls[0])
                self.assertIn("-count=1", calls[0])
                self.assertIn("-timeout", calls[0])
                self.assertIn("40m", calls[0])
                self.assertCountEqual(
                    [arg for arg in calls[0] if arg.startswith("databasus-")], OTHER_PACKAGES
                )
                self.assertNotIn(PROVIDER_PACKAGE, calls[0])
                if group.startswith("other-"):
                    self.assertIn("-run", calls[0])
                    self.assertIn("^(TestSelected)$", calls[0])
                else:
                    self.assertNotIn("-run", calls[0])

    def test_propagates_provider_and_shard_failures(self) -> None:
        for group, packages in (
            ("storage-providers", [PROVIDER_PACKAGE]),
            ("other-1-of-3", OTHER_PACKAGES),
        ):
            with self.subTest(group=group):
                status, calls, diagnostic = self.run_backend_step(group, packages, packages[0])

                self.assertEqual(status, 17, diagnostic)
                self.assertEqual(len(calls), 1)

    def test_rejects_incorrect_provider_package_selections(self) -> None:
        for packages in ([], OTHER_PACKAGES, [PROVIDER_PACKAGE, *OTHER_PACKAGES]):
            with self.subTest(packages=packages):
                status, calls, diagnostic = self.run_backend_step("storage-providers", packages)

                self.assertNotEqual(status, 0, diagnostic)
                self.assertEqual(calls, [])
