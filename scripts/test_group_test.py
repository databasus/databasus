import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parent.parent
TIMEOUT = 30


class PackageCoverageTest(unittest.TestCase):
    def test_pins_go_when_ci_collects_real_package_coverage(self):
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        job = re.search(r"^  lint-shell:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S).group(1)
        tools = re.search(r"install_args:\s*['\"]?([^'\"\n]+)", job).group(1).split()
        self.assertIn("go", tools)
        self.assertIn("shellcheck", tools)
        self.assertRegex(job, r"jdx/mise-action@[a-f0-9]{40}")
        self.assertIn("python3 -m unittest discover -s scripts -p '*_test.py'", job)

    def test_backend_test_job_installs_the_linter_its_tests_run(self):
        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        job = re.search(r"^  test-backend:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S).group(1)
        tools = re.search(r"install_args:\s*['\"]?([^'\"\n]+)", job).group(1).split()
        self.assertIn("golangci-lint", tools)

    def test_no_backend_package_runs_browser_scenarios(self):
        selection = subprocess.run(
            ["bash", str(ROOT / "mise-tasks/backend/test-group-packages")],
            cwd=ROOT / "backend",
            env={**os.environ, "usage_group": "browser-e2e"},
            capture_output=True,
            text=True,
        )
        self.assertNotEqual(selection.returncode, 0)

        workflow = (ROOT / ".github/workflows/ci-release.yml").read_text()
        job = re.search(r"^  test-backend:\n(.*?)(?=^  [a-zA-Z0-9_-]+:|\Z)", workflow, re.M | re.S).group(1)
        self.assertNotIn("browser", job)

        go_files_naming_playwright = [str(path) for path in (ROOT / "backend").rglob("*.go") if "playwright" in path.read_text(errors="replace").lower()]
        self.assertEqual(go_files_naming_playwright, [])

    def test_every_package_except_the_upgrade_test_belongs_to_exactly_one_group(self):
        groups = ("physical", "logical", "physical-api", "agent-e2e", "other")
        selections = {}
        for group in groups:
            packages = subprocess.check_output(["bash", str(ROOT / "mise-tasks/backend/test-group-packages")], cwd=ROOT / "backend", env={**os.environ, "usage_group": group}, text=True).splitlines()
            selections[group] = set(packages)

        module_packages = subprocess.check_output(["go", "list", "./..."], cwd=ROOT / "backend", text=True).splitlines()
        all_packages = [package for package in module_packages if not re.match(r"databasus-backend/e2e/upgrade(/|$)", package)]

        self.assertEqual(set(all_packages), set().union(*selections.values()))
        self.assertEqual(len(all_packages), sum(len(packages) for packages in selections.values()))
        self.assertIn("databasus-backend/migrations", selections["other"])
        self.assertIn("databasus-backend/tools/archlint", selections["other"])

    def test_upgrade_step_selects_the_upgrade_test_with_its_comparison_helper(self):
        upgrade_task = (ROOT / "mise-tasks/backend/test-upgrade").read_text()

        self.assertIn("./e2e/upgrade/...", upgrade_task)


class UpgradeTaskImageTest(unittest.TestCase):
    def run_upgrade_task(self, extra_environment):
        workspace = Path(tempfile.mkdtemp(prefix="test-upgrade-"))
        self.addCleanup(shutil.rmtree, workspace, ignore_errors=True)
        calls_path = workspace / "calls"
        stubs = {
            "git": 'echo v9.9.9\n',
            "docker": f'echo "docker $*" >>"{calls_path}"\n',
            "go": f'echo "go image=$DATABASUS_IMAGE" >>"{calls_path}"\n',
        }

        for name, body in stubs.items():
            stub_path = workspace / name
            stub_path.write_text("#!/usr/bin/env bash\n" + body)
            stub_path.chmod(0o755)

        environment = {**os.environ, "PATH": f"{workspace}:{os.environ['PATH']}", **extra_environment}
        subprocess.run(["bash", str(ROOT / "mise-tasks/backend/test-upgrade")], cwd=workspace, env=environment, check=True, capture_output=True, timeout=TIMEOUT)

        return calls_path.read_text().splitlines()

    def test_builds_the_image_and_tests_it_without_a_named_image(self):
        calls = self.run_upgrade_task({"DATABASUS_IMAGE": ""})

        self.assertEqual(calls, ["docker build --tag databasus-candidate:local ..", "go image=databasus-candidate:local"])

    def test_tests_a_named_image_without_building(self):
        calls = self.run_upgrade_task({"DATABASUS_IMAGE": "named:image"})

        self.assertEqual(calls, ["go image=named:image"])


class ShardFixture(unittest.TestCase):
    def setUp(self):
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

        self.environment = {**os.environ, "PATH": f"{self.bin}:{os.environ['PATH']}",
                            "TMPDIR": str(self.logs), "CALLS": str(self.calls), "GOFLAGS": ""}

    def stub(self, name, source):
        path = self.bin / name
        path.write_text("#!/usr/bin/env python3\n" + source)
        path.chmod(0o755)

    def records(self):
        return [json.loads(line) for line in self.calls.read_text().splitlines()]


class OtherGroupSelectionTest(ShardFixture):
    def test_selects_a_package_in_a_new_top_level_directory(self):
        self.stub("go", '''import sys
if sys.argv[1:] == ["list", "./..."]:
    print("databasus-backend/internal/lib/ordinary")
    print("databasus-backend/internal/modules/backups/e2e/logical")
    print("databasus-backend/newdirectory/fresh")
    print("databasus-backend/internal/lib/e2e/upgrade")
    print("databasus-backend/e2e/upgrade")
    print("databasus-backend/e2e/upgrade/upgradecheck")
else:
    print("databasus-backend/internal/lib/ordinary")
''')

        selection = subprocess.run(["bash", str(self.workspace / "test-group-packages")], cwd=self.backend,
                                   env={**self.environment, "usage_group": "other"},
                                   capture_output=True, text=True, timeout=TIMEOUT)

        self.assertEqual(selection.returncode, 0, selection.stderr)
        selected_packages = selection.stdout.splitlines()
        self.assertIn("databasus-backend/newdirectory/fresh", selected_packages)
        self.assertIn("databasus-backend/internal/lib/e2e/upgrade", selected_packages)
        self.assertIn("databasus-backend/internal/lib/ordinary", selected_packages)
        self.assertNotIn("databasus-backend/internal/modules/backups/e2e/logical", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade/upgradecheck", selected_packages)


class RealShardSelectionTest(ShardFixture):
    def test_three_real_selectors_cover_every_package_test_pair_once(self):
        fixture_tests = {
            "alpha": ["TestWeighted", "TestRepeated", "TestPlainAlpha"],
            "beta": ["TestRepeated", "TestPlainBeta", "TestAnother"],
            "gamma": ["TestHeavy", "TestPlainGamma"],
        }

        package_directories = {}
        for name, tests in fixture_tests.items():
            directory = self.backend / "internal/modules/backups/physicalapitests" / name
            directory.mkdir(parents=True)
            (directory / "fixture_test.go").write_text("package " + name + "\nimport \"testing\"\n" +
                "\n".join(f"func {test}(t *testing.T) {{}}" for test in tests) + "\nfunc TestMain(m *testing.M) {}\n")
            package_directories["fixture/internal/modules/backups/physicalapitests/" + name] = str(directory)
        (self.backend / "test-shard-weights.txt").write_text("10 TestHeavy\n7 TestWeighted\n")
        self.environment["FIXTURE_PACKAGES"] = json.dumps(package_directories)
        self.environment["PACKAGE_SELECTOR"] = str(self.workspace / "test-group-packages")

        self.stub("go", '''import json, os, sys
arguments = sys.argv[1:]
with open(os.environ["CALLS"], "a") as calls:
    calls.write(json.dumps({"args": arguments, "goflags": os.environ.get("GOFLAGS")}) + "\\n")
packages = json.loads(os.environ["FIXTURE_PACKAGES"])
if arguments == ["list", "./internal/modules/backups/physicalapitests/..."]:
    print("\\n".join(packages))
elif arguments[:3] == ["list", "-f", "{{ .Dir }}"]:
    print("\\n".join(packages[package] for package in arguments[3:]))
else:
    sys.exit("unexpected discovery: " + repr(arguments))
''')
        self.stub("mise", '''import os, subprocess, sys
if sys.argv[1:4] != ["run", "--quiet", "backend:test-group-packages"]:
    sys.exit("unexpected dispatch")
sys.exit(subprocess.call(["bash", os.environ["PACKAGE_SELECTOR"]], env={**os.environ, "usage_group": sys.argv[4]}))
''')

        patterns = []
        for shard in range(1, 4):
            selection = subprocess.run(["bash", str(self.workspace / "test-group-tests")], cwd=self.backend,
                env={**self.environment, "usage_group": f"physical-api-{shard}-of-3"},
                capture_output=True, text=True, timeout=TIMEOUT)
            self.assertEqual(selection.returncode, 0, selection.stderr)
            patterns.append(re.compile(selection.stdout.strip()))

        inventory = [(package, test) for package, directory in package_directories.items()
                     for source in Path(directory).glob("*_test.go")
                     for test in re.findall(r"^func (Test\w+)\(", source.read_text(), re.MULTILINE)
                     if test != "TestMain"]
        self.assertEqual(len(inventory), 8)
        for package, test in inventory:
            self.assertEqual(sum(bool(pattern.fullmatch(test)) for pattern in patterns), 1, (package, test))
        for pattern in patterns:
            selected_names = re.fullmatch(r"\^\((.*)\)\$", pattern.pattern).group(1).split("|")
            self.assertTrue(set(selected_names).issubset({test for _, test in inventory}))
            self.assertTrue(any(pattern.fullmatch(test) for _, test in inventory))
            self.assertFalse(pattern.fullmatch("TestMain"))
            self.assertFalse(pattern.fullmatch("TestAbsent"))

        self.assertEqual(len(self.records()), 6)
        for record in self.records():
            self.assertNotIn("-tags", record["args"])
            self.assertEqual(record["goflags"], "")
