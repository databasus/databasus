import json
import os
import re
import shutil
import signal
import subprocess
import tempfile
import time
import unittest
from pathlib import Path
from typing import TypedDict

from pattern_match import get_required_match

ROOT = Path(__file__).resolve().parent.parent
TIMEOUT = 30


class PackageCoverageTest(unittest.TestCase):
    def test_pins_go_when_ci_collects_real_package_coverage(self) -> None:
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        job = get_required_match(
            r"^  test-scripts:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S
        ).group(1)
        tools = get_required_match(r"install_args:\s*['\"]?([^'\"\n]+)", job).group(1).split()
        self.assertIn("go", tools)
        self.assertIn("shellcheck", tools)
        self.assertRegex(job, r"jdx/mise-action@[a-f0-9]{40}")
        self.assertIn("python3 -m unittest discover -s scripts -p '*_test.py'", job)

    def test_backend_test_job_installs_the_linter_its_tests_run(self) -> None:
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        job = get_required_match(
            r"^  test-backend:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S
        ).group(1)
        tools = get_required_match(r"install_args:\s*['\"]?([^'\"\n]+)", job).group(1).split()
        self.assertIn("golangci-lint", tools)

    def test_no_backend_package_runs_browser_scenarios(self) -> None:
        selection = subprocess.run(
            ["bash", str(ROOT / "mise-tasks/backend/test-group-packages")],
            cwd=ROOT / "backend",
            env={**os.environ, "usage_group": "browser-e2e"},
            capture_output=True,
            text=True,
        )
        self.assertNotEqual(selection.returncode, 0)

        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        job = get_required_match(
            r"^  test-backend:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S
        ).group(1)
        self.assertNotIn("browser", job)

        go_files_naming_playwright = [
            str(path)
            for path in (ROOT / "backend").rglob("*.go")
            if "playwright" in path.read_text(errors="replace").lower()
        ]
        self.assertEqual(go_files_naming_playwright, [])

    def test_every_package_except_the_upgrade_test_belongs_to_exactly_one_group(self) -> None:
        groups = (
            "physical",
            "patroni",
            "logical",
            "physical-api",
            "agent-e2e",
            "storage-providers",
            "other",
        )
        selections = {}
        for group in groups:
            packages = subprocess.check_output(
                ["bash", str(ROOT / "mise-tasks/backend/test-group-packages")],
                cwd=ROOT / "backend",
                env={**os.environ, "usage_group": group},
                text=True,
            ).splitlines()
            selections[group] = set(packages)

        module_packages = subprocess.check_output(
            ["go", "list", "./..."], cwd=ROOT / "backend", text=True
        ).splitlines()
        all_packages = [
            package
            for package in module_packages
            if not re.match(r"databasus-backend/e2e/upgrade(/|$)", package)
        ]

        self.assertEqual(set(all_packages), set().union(*selections.values()))
        self.assertEqual(len(all_packages), sum(len(packages) for packages in selections.values()))
        provider = "databasus-backend/internal/modules/storages/internal/providers"
        self.assertEqual(selections["storage-providers"], {provider})
        self.assertNotIn(provider, selections["other"])
        self.assertIn(provider + "/nas", selections["other"])
        self.assertIn("databasus-backend/migrations", selections["other"])
        self.assertIn("databasus-backend/tools/archlint", selections["other"])
        patroni = "databasus-backend/internal/modules/backups/e2e/postgresqlphysical/patroni"
        self.assertEqual(selections["patroni"], {patroni})
        self.assertNotIn(patroni, selections["physical"])
        self.assertIn(
            "databasus-backend/internal/modules/backups/e2e/postgresqlphysical/pg18",
            selections["physical"],
        )

    def test_upgrade_step_selects_the_upgrade_test_with_its_comparison_helper(self) -> None:
        upgrade_task = (ROOT / "mise-tasks/backend/test-upgrade").read_text()

        self.assertIn("./e2e/upgrade/...", upgrade_task)


class UpgradeTaskImageTest(unittest.TestCase):
    def start_upgrade_task(
        self, extra_environment: dict[str, str]
    ) -> tuple[subprocess.Popen[str], Path]:
        workspace = Path(tempfile.mkdtemp(prefix="test-upgrade-"))
        self.addCleanup(shutil.rmtree, workspace, ignore_errors=True)
        calls_path = workspace / "calls"
        calls_path.touch()
        stubs = {
            "git": "echo v9.9.9\n",
            "docker": f'echo "docker $*" >>"{calls_path}"\n',
            "go": (
                f'echo "go image=$DATABASUS_IMAGE" >>"{calls_path}"\n'
                'sleep "${STUB_GO_SECONDS:-0}"\n'
                'exit "${STUB_GO_EXIT_STATUS:-0}"\n'
            ),
        }

        for name, body in stubs.items():
            stub_path = workspace / name
            stub_path.write_text("#!/usr/bin/env bash\n" + body)
            stub_path.chmod(0o755)

        environment = {
            **os.environ,
            "PATH": f"{workspace}:{os.environ['PATH']}",
            **extra_environment,
        }
        environment.pop("DATABASUS_TEST_RUN_ID", None)
        task = subprocess.Popen(
            ["bash", str(ROOT / "mise-tasks/backend/test-upgrade")],
            cwd=workspace,
            env=environment,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            start_new_session=True,
        )
        self.addCleanup(self.stop_task, task)

        return task, calls_path

    def stop_task(self, task: subprocess.Popen[str]) -> None:
        if task.poll() is None:
            os.killpg(task.pid, signal.SIGKILL)
        task.communicate()

    def run_upgrade_task(self, extra_environment: dict[str, str]) -> tuple[int, list[str], str]:
        task, calls_path = self.start_upgrade_task(extra_environment)
        output, _ = task.communicate(timeout=TIMEOUT)

        return task.returncode, calls_path.read_text().splitlines(), output

    def get_built_image(self, calls: list[str]) -> str:
        build_call = next(call for call in calls if call.startswith("docker build"))
        built_image = build_call.removeprefix("docker build --tag ").removesuffix(" ..")
        self.assertRegex(built_image, r"^databasus-upgrade-test:[0-9]+-[0-9]+-[0-9]+$")

        return built_image

    def test_builds_tests_and_removes_an_image_unique_to_the_run(self) -> None:
        exit_status, calls, output = self.run_upgrade_task({"DATABASUS_IMAGE": ""})

        self.assertEqual(exit_status, 0)
        built_image = self.get_built_image(calls)
        self.assertIn(built_image, output)
        self.assertEqual(
            calls,
            [
                f"docker build --tag {built_image} ..",
                f"go image={built_image}",
                f"docker image rm {built_image}",
            ],
        )

    def test_overlapping_runs_build_different_images(self) -> None:
        _, first_calls, first_output = self.run_upgrade_task({"DATABASUS_IMAGE": ""})
        _, second_calls, second_output = self.run_upgrade_task({"DATABASUS_IMAGE": ""})
        first_image = self.get_built_image(first_calls)
        second_image = self.get_built_image(second_calls)

        self.assertNotEqual(first_image, second_image)
        self.assertIn(first_image, first_output)
        self.assertNotIn(second_image, first_output)
        self.assertIn(second_image, second_output)

    def test_removes_the_built_image_when_the_test_fails(self) -> None:
        exit_status, calls, _ = self.run_upgrade_task(
            {"DATABASUS_IMAGE": "", "STUB_GO_EXIT_STATUS": "1"}
        )

        self.assertNotEqual(exit_status, 0)
        self.assertEqual(calls[-1], f"docker image rm {self.get_built_image(calls)}")

    def test_removes_the_built_image_when_the_test_is_interrupted(self) -> None:
        task, calls_path = self.start_upgrade_task({"DATABASUS_IMAGE": "", "STUB_GO_SECONDS": "60"})
        deadline = time.monotonic() + TIMEOUT
        while "go image=" not in calls_path.read_text():
            self.assertLess(time.monotonic(), deadline, "the upgrade test did not start")
            time.sleep(0.05)

        os.killpg(task.pid, signal.SIGINT)
        task.communicate(timeout=TIMEOUT)

        self.assertNotEqual(task.returncode, 0)
        calls = calls_path.read_text().splitlines()
        self.assertEqual(calls[-1], f"docker image rm {self.get_built_image(calls)}")

    def test_tests_a_named_image_without_building_or_removing_it(self) -> None:
        for extra_environment in ({}, {"STUB_GO_EXIT_STATUS": "1"}):
            with self.subTest(environment=extra_environment):
                _, calls, _ = self.run_upgrade_task(
                    {"DATABASUS_IMAGE": "named:image", **extra_environment}
                )

                self.assertEqual(calls, ["go image=named:image"])


class CommandRecord(TypedDict):
    args: list[str]
    goflags: str


class ShardFixture(unittest.TestCase):
    def setUp(self) -> None:
        self.workspace = Path(tempfile.mkdtemp(prefix="test-groups-"))
        self.addCleanup(shutil.rmtree, self.workspace, ignore_errors=True)
        self.backend = self.workspace / "backend"
        self.backend.mkdir()
        self.bin = self.workspace / "bin"
        self.bin.mkdir()
        self.logs = self.workspace / "logs"
        self.logs.mkdir()
        self.calls = self.workspace / "calls"
        self.calls.touch()

        (self.workspace / "scripts").mkdir()
        (self.workspace / "scripts/fedora.sh").write_text("apply_libedit_shim() { :; }\n")
        for name in ("test-group-packages", "test-group-tests"):
            shutil.copy(ROOT / "mise-tasks/backend" / name, self.workspace / name)

        self.environment = {
            **os.environ,
            "PATH": f"{self.bin}:{os.environ['PATH']}",
            "TMPDIR": str(self.logs),
            "CALLS": str(self.calls),
            "GOFLAGS": "",
        }

    def stub(self, name: str, source: str) -> None:
        path = self.bin / name
        path.write_text("#!/usr/bin/env python3\n" + source)
        path.chmod(0o755)

    def records(self) -> list[CommandRecord]:
        records: list[CommandRecord] = []
        for line in self.calls.read_text().splitlines():
            record: object = json.loads(line)
            assert isinstance(record, dict)
            arguments = record.get("args")
            goflags = record.get("goflags")
            assert isinstance(arguments, list) and all(
                isinstance(argument, str) for argument in arguments
            )
            assert isinstance(goflags, str)
            records.append({"args": arguments, "goflags": goflags})
        return records


class OtherGroupSelectionTest(ShardFixture):
    def test_selects_a_package_in_a_new_top_level_directory(self) -> None:
        self.stub(
            "go",
            """import sys
if sys.argv[1:] == ["list", "./..."]:
    print("databasus-backend/internal/lib/ordinary")
    print("databasus-backend/internal/modules/backups/e2e/logical")
    print("databasus-backend/newdirectory/fresh")
    print("databasus-backend/internal/modules/storages/internal/providers")
    print("databasus-backend/internal/modules/storages/internal/providers/nas")
    print("databasus-backend/internal/lib/e2e/upgrade")
    print("databasus-backend/e2e/upgrade")
    print("databasus-backend/e2e/upgrade/upgradecheck")
else:
    print("databasus-backend/internal/lib/ordinary")
""",
        )

        selection = subprocess.run(
            ["bash", str(self.workspace / "test-group-packages")],
            cwd=self.backend,
            env={**self.environment, "usage_group": "other"},
            capture_output=True,
            text=True,
            timeout=TIMEOUT,
        )

        self.assertEqual(selection.returncode, 0, selection.stderr)
        selected_packages = selection.stdout.splitlines()
        provider = "databasus-backend/internal/modules/storages/internal/providers"
        self.assertNotIn(provider, selected_packages)
        self.assertIn(provider + "/nas", selected_packages)
        self.assertIn("databasus-backend/newdirectory/fresh", selected_packages)
        self.assertIn("databasus-backend/internal/lib/e2e/upgrade", selected_packages)
        self.assertIn("databasus-backend/internal/lib/ordinary", selected_packages)
        self.assertNotIn(
            "databasus-backend/internal/modules/backups/e2e/logical", selected_packages
        )
        self.assertNotIn("databasus-backend/e2e/upgrade", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade/upgradecheck", selected_packages)


class PatroniGroupSelectionTest(ShardFixture):
    def test_patroni_takes_its_nested_packages_and_physical_takes_the_rest(self) -> None:
        self.stub(
            "go",
            """import sys
physical = "databasus-backend/internal/modules/backups/e2e/postgresqlphysical"
packages = [physical + "/pg17", physical + "/fresh", physical + "/patroni", physical + "/patroni/nested"]
if sys.argv[1:] == ["list", "./internal/modules/backups/e2e/postgresqlphysical/..."]:
    print("\\n".join(packages))
elif sys.argv[1:] == ["list", "./internal/modules/backups/e2e/postgresqlphysical/patroni/..."]:
    print("\\n".join(package for package in packages if "/patroni" in package))
else:
    sys.exit("unexpected discovery: " + repr(sys.argv[1:]))
""",
        )

        selected_by_group = {}
        for group in ("physical", "patroni"):
            selection = subprocess.run(
                ["bash", str(self.workspace / "test-group-packages")],
                cwd=self.backend,
                env={**self.environment, "usage_group": group},
                capture_output=True,
                text=True,
                timeout=TIMEOUT,
            )
            self.assertEqual(selection.returncode, 0, selection.stderr)
            selected_by_group[group] = selection.stdout.splitlines()

        physical = "databasus-backend/internal/modules/backups/e2e/postgresqlphysical"
        self.assertEqual(selected_by_group["physical"], [physical + "/pg17", physical + "/fresh"])
        self.assertEqual(
            selected_by_group["patroni"], [physical + "/patroni", physical + "/patroni/nested"]
        )


class PatroniShardBalanceTest(ShardFixture):
    def test_every_patroni_shard_runs_backup_scenarios_with_recorded_weights(self) -> None:
        patroni_directory = ROOT / "backend/internal/modules/backups/e2e/postgresqlphysical/patroni"
        scenario_tests = {
            test
            for source in patroni_directory.glob("*_test.go")
            for test in re.findall(r"^func (Test\w+)\(", source.read_text(), re.MULTILINE)
            if test != "TestPatroniFixtureChild"
        }
        backup_tests = {
            test for test in scenario_tests if re.fullmatch(r"TestPatroni\w+Postgres1[78]", test)
        }
        weighted_tests = set(
            re.findall(
                r"^\d+ (\w+)$", (ROOT / "backend/test-shard-weights.txt").read_text(), re.MULTILINE
            )
        )
        self.assertEqual(len(backup_tests), 10)
        self.assertEqual(scenario_tests - weighted_tests, set())

        self.environment["PACKAGE_SELECTOR"] = str(ROOT / "mise-tasks/backend/test-group-packages")
        self.stub(
            "mise",
            """import os, subprocess, sys
if sys.argv[1:4] != ["run", "--quiet", "backend:test-group-packages"]:
    sys.exit("unexpected dispatch")
sys.exit(subprocess.call(["bash", os.environ["PACKAGE_SELECTOR"]], env={**os.environ, "usage_group": sys.argv[4]}))
""",
        )
        for shard in range(1, 6):
            with self.subTest(shard=shard):
                selection = subprocess.run(
                    ["bash", str(ROOT / "mise-tasks/backend/test-group-tests")],
                    cwd=ROOT / "backend",
                    env={**self.environment, "usage_group": f"patroni-{shard}-of-5"},
                    capture_output=True,
                    text=True,
                    timeout=TIMEOUT,
                )
                self.assertEqual(selection.returncode, 0, selection.stderr)
                pattern = re.compile(selection.stdout.strip())
                self.assertTrue(
                    any(pattern.fullmatch(test) for test in backup_tests), pattern.pattern
                )


class RealShardSelectionTest(ShardFixture):
    def test_three_real_selectors_cover_every_package_test_pair_once(self) -> None:
        for group in ("physical-api", "other"):
            with self.subTest(group=group):
                self.check_shard_coverage(group)

    def check_shard_coverage(self, group: str) -> None:
        self.calls.write_text("")
        fixture_tests = {
            "alpha": ["TestWeighted", "TestRepeated", "TestPlainAlpha"],
            "beta": ["TestRepeated", "TestPlainBeta", "TestAnother"],
            "gamma": ["TestHeavy", "TestHeavySecond", "TestPlainGamma"],
        }

        package_directories = {}
        for name, tests in fixture_tests.items():
            directory = self.backend / "internal/modules/backups/physicalapitests" / name
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "fixture_test.go").write_text(
                "package "
                + name
                + '\nimport "testing"\n'
                + "\n".join(f"func {test}(t *testing.T) {{}}" for test in tests)
                + "\nfunc TestMain(m *testing.M) {}\n"
            )
            package_directories["fixture/" + name] = str(directory)
        (self.backend / "test-shard-weights.txt").write_text(
            "10 TestHeavy\n9 TestHeavySecond\n7 TestWeighted\n100 TestDeleted\n"
        )
        self.environment["FIXTURE_PACKAGES"] = json.dumps(package_directories)
        self.environment["PACKAGE_SELECTOR"] = str(self.workspace / "test-group-packages")

        self.stub(
            "go",
            """import json, os, sys
arguments = sys.argv[1:]
with open(os.environ["CALLS"], "a") as calls:
    calls.write(json.dumps({"args": arguments, "goflags": os.environ.get("GOFLAGS")}) + "\\n")
packages = json.loads(os.environ["FIXTURE_PACKAGES"])
if arguments in (["list", "./internal/modules/backups/physicalapitests/..."], ["list", "./..."]):
    print("\\n".join(packages))
elif arguments[:3] == ["list", "-f", "{{ .Dir }}"]:
    print("\\n".join(packages[package] for package in arguments[3:]))
else:
    sys.exit("unexpected discovery: " + repr(arguments))
""",
        )
        self.stub(
            "mise",
            """import os, subprocess, sys
if sys.argv[1:4] != ["run", "--quiet", "backend:test-group-packages"]:
    sys.exit("unexpected dispatch")
sys.exit(subprocess.call(["bash", os.environ["PACKAGE_SELECTOR"]], env={**os.environ, "usage_group": sys.argv[4]}))
""",
        )

        patterns = []
        for shard in range(1, 4):
            selection = subprocess.run(
                ["bash", str(self.workspace / "test-group-tests")],
                cwd=self.backend,
                env={**self.environment, "usage_group": f"{group}-{shard}-of-3"},
                capture_output=True,
                text=True,
                timeout=TIMEOUT,
            )
            self.assertEqual(selection.returncode, 0, selection.stderr)
            patterns.append(re.compile(selection.stdout.strip()))

        inventory = [
            (package, test)
            for package, directory in package_directories.items()
            for source in Path(directory).glob("*_test.go")
            for test in re.findall(r"^func (Test\w+)\(", source.read_text(), re.MULTILINE)
            if test != "TestMain"
        ]
        self.assertEqual(len(inventory), 9)
        heavy_shard = next(
            index for index, pattern in enumerate(patterns) if pattern.fullmatch("TestHeavy")
        )
        second_heavy_shard = next(
            index for index, pattern in enumerate(patterns) if pattern.fullmatch("TestHeavySecond")
        )
        self.assertNotEqual(heavy_shard, second_heavy_shard)
        for package, test in inventory:
            self.assertEqual(
                sum(bool(pattern.fullmatch(test)) for pattern in patterns), 1, (package, test)
            )
        for pattern in patterns:
            selected_pattern = re.fullmatch(r"\^\((.*)\)\$", pattern.pattern)
            assert selected_pattern is not None
            selected_names = selected_pattern.group(1).split("|")
            self.assertTrue(set(selected_names).issubset({test for _, test in inventory}))
            self.assertTrue(any(pattern.fullmatch(test) for _, test in inventory))
            self.assertFalse(pattern.fullmatch("TestMain"))
            self.assertFalse(pattern.fullmatch("TestAbsent"))

        self.assertEqual(len(self.records()), 6)
        for record in self.records():
            self.assertNotIn("-tags", record["args"])
            self.assertEqual(record["goflags"], "")

    def test_pins_collation_when_the_runner_uses_another_locale(self) -> None:
        sort_path = shutil.which("sort")
        assert sort_path is not None
        self.environment["LC_ALL"] = "C.UTF-8"
        self.stub(
            "sort",
            f"""import os, sys
if os.environ.get("LC_ALL") != "C":
    sys.exit("test partitioning requires LC_ALL=C")
os.execv({sort_path!r}, [{sort_path!r}, *sys.argv[1:]])
""",
        )

        self.check_shard_coverage("other")

    def test_rejects_invalid_and_empty_shards(self) -> None:
        self.check_shard_coverage("other")
        for group in ("other-0-of-3", "other-4-of-3", "other-1-of-0", "other-30-of-30"):
            with self.subTest(group=group):
                selection = subprocess.run(
                    ["bash", str(self.workspace / "test-group-tests")],
                    cwd=self.backend,
                    env={**self.environment, "usage_group": group},
                    capture_output=True,
                    text=True,
                    timeout=TIMEOUT,
                )
                self.assertNotEqual(selection.returncode, 0)
                self.assertTrue("outside" in selection.stderr or "has no tests" in selection.stderr)
