import json
import os
import subprocess
import tempfile
import textwrap
import unittest
from dataclasses import dataclass
from pathlib import Path

from workflow_jobs import get_jobs

ROOT = Path(__file__).resolve().parents[1]
PROVIDER_PACKAGE = "databasus-backend/internal/modules/storages/internal/providers"
OTHER_PACKAGES = [
    "databasus-backend/internal/app/apptest",
    PROVIDER_PACKAGE + "/nas",
]


@dataclass(frozen=True)
class BackendCiStepSpec:
    group: str
    packages: list[str]
    failing_package: str = ""
    should_prepare: bool = False
    failing_sysctl: str = ""


class BackendCiExecutionTest(unittest.TestCase):
    def run_backend_step(self, spec: BackendCiStepSpec) -> tuple[int, list[list[str]], str]:
        group = spec.group
        packages = spec.packages
        failing_package = spec.failing_package
        should_prepare = spec.should_prepare
        failing_sysctl = spec.failing_sysctl
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        step = get_jobs(workflow)["test-backend"].split("- name: Run Go tests\n", 1)[1]
        script = textwrap.dedent(step.split("        run: |\n", 1)[1])
        if should_prepare:
            job = get_jobs(workflow)["test-backend"]
            scripts: list[str] = []
            for block in job.split("      - name: ")[1:]:
                name = block.splitlines()[0]
                if name in ("Raise inotify limits for k3s", "Run Go tests"):
                    scripts.append(textwrap.dedent(block.split("        run: |\n", 1)[1]))
            script = "\n".join(scripts)
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
                "sudo": "exec python3 - \"$@\" <<'PYTHON'\n"
                "import json, os, sys\n"
                'with open(os.environ["CALLS_PATH"], "a") as calls:\n'
                '    calls.write(json.dumps(sys.argv[1:]) + "\\n")\n'
                'if os.environ["FAILING_SYSCTL"] and any(\n'
                '    argument.startswith(os.environ["FAILING_SYSCTL"] + "=") for argument in sys.argv[1:]\n'
                "):\n"
                "    sys.exit(23)\n"
                "PYTHON\n",
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
                    "FAILING_SYSCTL": failing_sysctl,
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

    def test_runs_storage_providers_once_without_fallthrough(self) -> None:
        status, calls, diagnostic = self.run_backend_step(
            BackendCiStepSpec(group="storage-providers", packages=[PROVIDER_PACKAGE])
        )

        self.assertEqual(status, 0, diagnostic)
        self.assertEqual(len(calls), 1)
        self.assertIn("-p=1", calls[0])
        self.assertIn("-parallel=1", calls[0])
        self.assertIn("-race", calls[0])
        self.assertEqual(
            [arg for arg in calls[0] if arg.startswith("databasus-")], [PROVIDER_PACKAGE]
        )

    def test_runs_each_group_selection_once_with_bounded_parallelism(self) -> None:
        for group in (
            "other-1-of-3",
            "other-2-of-3",
            "other-3-of-3",
            "patroni-1-of-5",
            "logical",
        ):
            with self.subTest(group=group):
                status, calls, diagnostic = self.run_backend_step(
                    BackendCiStepSpec(group=group, packages=OTHER_PACKAGES)
                )

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
                if "-of-" in group:
                    self.assertIn("-run", calls[0])
                    self.assertIn("^(TestSelected)$", calls[0])
                else:
                    self.assertNotIn("-run", calls[0])

    def test_cnpg_uses_runner_parallelism_and_keeps_race_detection_and_selection(self) -> None:
        status, calls, diagnostic = self.run_backend_step(
            BackendCiStepSpec(group="cnpg-1-of-5", packages=OTHER_PACKAGES)
        )

        self.assertEqual(status, 0, diagnostic)
        self.assertEqual(len(calls), 1)
        self.assertFalse(any(argument.startswith("-parallel") for argument in calls[0]))
        self.assertIn("-race", calls[0])
        self.assertIn("-count=1", calls[0])
        self.assertIn("-run", calls[0])
        self.assertIn("^(TestSelected)$", calls[0])
        self.assertCountEqual(
            [argument for argument in calls[0] if argument.startswith("databasus-")],
            OTHER_PACKAGES,
        )

    def test_raises_inotify_limits_only_for_cnpg_shards(self) -> None:
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        step = get_jobs(workflow)["test-backend"].split(
            "- name: Raise inotify limits for k3s\n", 1
        )[1]
        step = step.split("\n      - name:", 1)[0]

        self.assertIn("if: startsWith(matrix.group, 'cnpg-')", step)
        self.assertIn("sudo sysctl -w fs.inotify.max_user_instances=", step)
        self.assertIn("sudo sysctl -w fs.inotify.max_user_watches=", step)

    def test_prepares_both_inotify_limits_before_go_and_stops_on_each_failure(self) -> None:
        limits = ["fs.inotify.max_user_instances", "fs.inotify.max_user_watches"]
        for failing_limit in ("", *limits):
            with self.subTest(failing_limit=failing_limit):
                status, calls, diagnostic = self.run_backend_step(
                    BackendCiStepSpec(
                        group="cnpg-1-of-5",
                        packages=OTHER_PACKAGES,
                        should_prepare=True,
                        failing_sysctl=failing_limit,
                    )
                )

                self.assertEqual(status, 23 if failing_limit else 0, diagnostic)
                limit_count = limits.index(failing_limit) + 1 if failing_limit else 2
                self.assertEqual(len(calls), limit_count if failing_limit else 3)
                for arguments, limit in zip(calls, limits, strict=False):
                    self.assertEqual(arguments[:2], ["sysctl", "-w"])
                    self.assertTrue(arguments[2].startswith(limit + "="))
                if not failing_limit:
                    self.assertIn("test", calls[-1])

    def test_propagates_provider_and_shard_failures(self) -> None:
        for group, packages in (
            ("storage-providers", [PROVIDER_PACKAGE]),
            ("other-1-of-3", OTHER_PACKAGES),
            ("cnpg-1-of-5", OTHER_PACKAGES),
        ):
            with self.subTest(group=group):
                status, calls, diagnostic = self.run_backend_step(
                    BackendCiStepSpec(group=group, packages=packages, failing_package=packages[0])
                )

                self.assertEqual(status, 17, diagnostic)
                self.assertEqual(len(calls), 1)

    def test_rejects_incorrect_provider_package_selections(self) -> None:
        for packages in ([], OTHER_PACKAGES, [PROVIDER_PACKAGE, *OTHER_PACKAGES]):
            with self.subTest(packages=packages):
                status, calls, diagnostic = self.run_backend_step(
                    BackendCiStepSpec(group="storage-providers", packages=packages)
                )

                self.assertNotEqual(status, 0, diagnostic)
                self.assertEqual(calls, [])
