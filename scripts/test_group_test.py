import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time
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

    def test_real_groups_cover_every_backend_package_once_without_the_upgrade_package(self):
        groups = ("physical", "logical", "physical-api", "agent-e2e", "browser-e2e", "other")
        selections = {}
        for group in groups:
            packages = subprocess.check_output(["bash", str(ROOT / "mise-tasks/backend/test-group-packages")], cwd=ROOT / "backend", env={**os.environ, "usage_group": group}, text=True).splitlines()
            selections[group] = set(packages)

        all_packages = subprocess.check_output(["go", "list", "./internal/...", "./cmd/..."], cwd=ROOT / "backend", text=True).splitlines()

        self.assertEqual(set(all_packages), set().union(*selections.values()))
        self.assertEqual(len(all_packages), sum(len(packages) for packages in selections.values()))
        self.assertEqual(selections["browser-e2e"], {"databasus-backend/internal/app/browsertests"})
        self.assertNotIn("databasus-backend/internal/app/browsertests", selections["other"])


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
        for name in ("test-group", "test-group-packages", "test-group-tests"):
            shutil.copy(ROOT / "mise-tasks/backend" / name, self.workspace / name)

        self.environment = {**os.environ, "PATH": f"{self.bin}:{os.environ['PATH']}",
                            "TMPDIR": str(self.logs), "CALLS": str(self.calls), "GOFLAGS": ""}

    def stub(self, name, source):
        path = self.bin / name
        path.write_text("#!/usr/bin/env python3\n" + source)
        path.chmod(0o755)

    def start(self, shards):
        process = subprocess.Popen(["bash", str(self.workspace / "test-group")], cwd=self.backend,
                                   env={**self.environment, "usage_shards": shards},
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
                                   start_new_session=True)
        self.addCleanup(self.stop, process)
        return process

    def stop(self, process):
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.communicate()

    def records(self):
        return [json.loads(line) for line in self.calls.read_text().splitlines()]

    def wait_for(self, condition):
        deadline = time.monotonic() + TIMEOUT
        while time.monotonic() < deadline:
            if condition():
                return
            time.sleep(0.02)
        self.fail(f"condition did not become true; calls: {self.records()}")


class ShardOrchestrationTest(ShardFixture):
    def setUp(self):
        super().setUp()
        self.stub("mise", '''import os, sys
selection = sys.argv[-1]
if "backend:test-group-packages" in sys.argv:
    print("./fixture/" + selection)
else:
    print("^(Test" + selection.replace("-", "") + ")$")
''')
        self.stub("go", '''import json, os, pathlib, sys, time
arguments = sys.argv[1:]
package = arguments[-1]
with open(os.environ["CALLS"], "a") as calls:
    calls.write(json.dumps({"event": "start", "args": arguments, "pid": os.getpid(), "goflags": os.environ.get("GOFLAGS")}) + "\\n")
if os.environ.get("HOLD") and (os.environ.get("HOLD_ALL") or package.endswith("2-of-3")):
    while not pathlib.Path(os.environ["HOLD"]).exists():
        time.sleep(0.02)
status = int(bool(os.environ.get("FAIL") and package.endswith("1-of-3")))
with open(os.environ["CALLS"], "a") as calls:
    calls.write(json.dumps({"event": "end", "package": package, "status": status}) + "\\n")
print(package)
sys.exit(status)
''')

    def test_runs_each_requested_shard_concurrently_once_without_a_tag(self):
        release = self.workspace / "release"
        self.environment.update(HOLD=str(release), HOLD_ALL="1")
        process = self.start("physical-api-1-of-3 physical-api-2-of-3 physical-api-3-of-3")

        self.wait_for(lambda: len([record for record in self.records() if record["event"] == "start"]) == 3)
        self.assertFalse(any(record["event"] == "end" for record in self.records()))
        self.assertIsNone(process.poll())

        release.touch()
        output, errors = process.communicate(timeout=TIMEOUT)
        self.assertEqual(process.returncode, 0, errors)

        starts = [record for record in self.records() if record["event"] == "start"]
        self.assertEqual(len(starts), 3)
        self.assertEqual(len({record["args"][-1] for record in starts}), 3)
        for record in starts:
            self.assertEqual(record["goflags"], "")
            self.assertNotIn("-tags", record["args"])
            self.assertIn("40m", record["args"])
            self.assertIn("-race", record["args"])
            self.assertIn("-failfast", record["args"])
            self.assertIn(record["args"][-1], output)

        self.assertEqual(len([record for record in self.records() if record["event"] == "end"]), 3)
        self.assertEqual(list(self.logs.iterdir()), [])

    def test_invalid_or_duplicate_selections_start_nothing(self):
        for shards in ("physical-1-of-3 physical-1-of-3", "other", "physical", "logical-4-of-3", "unknown-1-of-3", "physical-api-0-of-3"):
            with self.subTest(shards=shards):
                process = self.start(shards)
                process.communicate(timeout=TIMEOUT)
                self.assertNotEqual(process.returncode, 0)
                self.assertEqual(self.records(), [])
                self.assertEqual(list(self.logs.iterdir()), [])

    def test_failure_waits_for_the_neighbour_before_returning(self):
        release = self.workspace / "release"
        self.environment.update(FAIL="1", HOLD=str(release))
        process = self.start("physical-1-of-3 physical-2-of-3")

        self.wait_for(lambda: any(record.get("status") == 1 for record in self.records()))
        self.wait_for(lambda: len([record for record in self.records() if record["event"] == "start"]) == 2)
        self.assertIsNone(process.poll())

        release.touch()
        output, _ = process.communicate(timeout=TIMEOUT)

        self.assertNotEqual(process.returncode, 0)
        self.assertIn("physical-1-of-3", output)
        self.assertIn("physical-2-of-3", output)
        self.assertEqual(len([record for record in self.records() if record["event"] == "end"]), 2)

    def test_interrupt_removes_only_its_processes_and_logs(self):
        release = self.workspace / "release"
        self.environment["HOLD"] = str(release)
        neighbour = self.start("logical-2-of-3")
        self.wait_for(lambda: len([record for record in self.records() if record["event"] == "start"]) == 1)
        neighbour_pid = next(record["pid"] for record in self.records() if record["event"] == "start")
        neighbour_logs = set(self.logs.iterdir())

        interrupted = self.start("physical-2-of-3")
        self.wait_for(lambda: len([record for record in self.records() if record["event"] == "start"]) == 2)
        interrupted_pid = next(record["pid"] for record in self.records() if record["event"] == "start" and record["args"][-1].endswith("physical-2-of-3"))
        interrupted.send_signal(signal.SIGTERM)
        interrupted.communicate(timeout=TIMEOUT)

        self.assertNotEqual(interrupted.returncode, 0)
        self.assertIsNone(neighbour.poll())
        os.kill(neighbour_pid, 0)
        self.assertEqual(set(self.logs.iterdir()), neighbour_logs)
        with self.assertRaises(ProcessLookupError):
            os.kill(interrupted_pid, 0)

        release.touch()
        _, errors = neighbour.communicate(timeout=TIMEOUT)
        self.assertEqual(neighbour.returncode, 0, errors)
        self.assertEqual(list(self.logs.iterdir()), [])


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
