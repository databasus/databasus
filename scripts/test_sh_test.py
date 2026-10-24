import fcntl
import os
from pathlib import Path
import shutil
import signal
import socket
import subprocess
import tempfile
import textwrap
import time
import unittest


REPOSITORY_ROOT = Path(__file__).resolve().parent.parent
FULL_RUN_LOCK_NAME = "databasus-full-test.lock"
FIXTURES_LOCK_NAME = "databasus-test-fixtures.lock"
WAIT_TIMEOUT_SECONDS = 30
RUN_LABEL_FILTER = "label=databasus.test-run="

# Each stub appends "<epoch seconds> <name> <arguments>" to $STUB_CALLS. The go stub records
# DATABASUS_TEST_RUN_ID with its test call, sleeps for STUB_GO_TEST_SECONDS, so a test can
# interrupt the run while its tests run, and exits with STUB_GO_TEST_EXIT_STATUS. The docker stub
# lists one container, one network and one volume. The just stub runs test-upgrade for
# STUB_UPGRADE_TEST_SECONDS and exits with STUB_UPGRADE_TEST_EXIT_STATUS.
STUB_SOURCES = {
    "docker": """
        echo "$(date +%s.%N) docker $*" >>"$STUB_CALLS"
        case "$1 $2" in
            "ps -aq") echo container-of-run ;;
            "network ls") echo network-of-run ;;
            "volume ls") echo volume-of-run ;;
        esac
    """,
    "go": """
        if [[ "$1" == test ]]; then
            echo "$(date +%s.%N) go test run_id=${DATABASUS_TEST_RUN_ID:-}" >>"$STUB_CALLS"
            sleep "${STUB_GO_TEST_SECONDS:-0}"
            exit "${STUB_GO_TEST_EXIT_STATUS:-0}"
        else
            echo "$(date +%s.%N) go $*" >>"$STUB_CALLS"
        fi
    """,
    "just": """
        echo "$(date +%s.%N) just $*" >>"$STUB_CALLS"
        if [[ "${*: -1}" == test-upgrade ]]; then
            sleep "${STUB_UPGRADE_TEST_SECONDS:-0}"
            exit "${STUB_UPGRADE_TEST_EXIT_STATUS:-0}"
        fi
    """,
}

STUB_FEDORA_SOURCE = """
apply_libedit_shim() {
    :
}
"""


class TestScriptTestCase(unittest.TestCase):
    """Runs a copy of scripts/test.sh in a checkout of its own, so the repository's .env and the
    machine's real locks stay out of the test."""

    def setUp(self):
        self.workspace = Path(tempfile.mkdtemp(prefix="test-sh-"))
        self.addCleanup(shutil.rmtree, self.workspace, ignore_errors=True)

        self.checkout = self.workspace / "checkout"
        (self.checkout / "scripts").mkdir(parents=True)
        (self.checkout / "backend").mkdir()
        shutil.copy(REPOSITORY_ROOT / "scripts" / "test.sh", self.checkout / "scripts" / "test.sh")
        shutil.copy(REPOSITORY_ROOT / "backend" / "Justfile", self.checkout / "backend" / "Justfile")
        (self.checkout / "scripts" / "fedora.sh").write_text(STUB_FEDORA_SOURCE)

        self.lock_directory = self.workspace / "locks"
        self.lock_directory.mkdir()
        self.calls_path = self.workspace / "calls"
        self.calls_path.touch()

        stubs_directory = self.workspace / "stubs"
        stubs_directory.mkdir()
        for name, source in STUB_SOURCES.items():
            stub_path = stubs_directory / name
            stub_path.write_text("#!/usr/bin/env bash\n" + textwrap.dedent(source))
            stub_path.chmod(0o755)

        fixture_port = self.start_fixture_listener()
        self.environment = {
            **os.environ,
            "PATH": f"{stubs_directory}:{os.environ['PATH']}",
            "STUB_CALLS": str(self.calls_path),
            "TMPDIR": str(self.lock_directory),
            "TEST_LOGICAL_POSTGRES_16_PORT": str(fixture_port),
            "TEST_PHYSICAL_POSTGRES_17_PORT": str(fixture_port),
            "TEST_PHYSICAL_POSTGRES_18_PORT": str(fixture_port),
        }
        self.environment.pop("DATABASUS_TEST_RUN_ID", None)
        self.real_just = shutil.which("just")

    def start_fixture_listener(self):
        listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        listener.bind(("127.0.0.1", 0))
        listener.listen(16)
        self.addCleanup(listener.close)

        return listener.getsockname()[1]

    def start_script(self, extra_environment=None):
        script = subprocess.Popen(
            ["bash", str(self.checkout / "scripts" / "test.sh")],
            env={**self.environment, **(extra_environment or {})},
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            start_new_session=True,
        )
        self.addCleanup(self.stop_script, script)

        return script

    def stop_script(self, script):
        if script.poll() is None:
            os.killpg(script.pid, signal.SIGKILL)
        script.communicate()

    def hold_lock(self, lock_name, lock_operation):
        lock_file = open(self.lock_directory / lock_name, "a")
        self.addCleanup(lock_file.close)
        fcntl.flock(lock_file, lock_operation)

        return lock_file

    def read_calls(self):
        recorded_calls = []
        for line in self.calls_path.read_text().splitlines():
            called_at, call = line.split(" ", 1)
            recorded_calls.append((float(called_at), call))

        return recorded_calls

    def get_upgrade_test_call(self):
        return f"just --justfile {self.checkout}/backend/Justfile test-upgrade"

    def find_call(self, prefix):
        return next((recorded_call for recorded_call in self.read_calls() if recorded_call[1].startswith(prefix)), None)

    def wait_for_call(self, prefix):
        deadline = time.monotonic() + WAIT_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            recorded_call = self.find_call(prefix)
            if recorded_call is not None:
                return recorded_call
            time.sleep(0.05)

        self.fail(f"no call starting with {prefix!r}; calls: {self.read_calls()}")

    def wait_for_output_line(self, script, text):
        deadline = time.monotonic() + WAIT_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            line = script.stdout.readline()
            if text in line:
                return
            if not line and script.poll() is not None:
                break

        self.fail(f"the script did not print {text!r}")

    def finish_script(self, script):
        stdout, stderr = script.communicate(timeout=WAIT_TIMEOUT_SECONDS)

        return script.returncode, stdout, stderr

    def test_a_second_full_run_waits_for_the_first_and_says_so(self):
        full_run_lock = self.hold_lock(FULL_RUN_LOCK_NAME, fcntl.LOCK_EX)
        script = self.start_script()

        self.wait_for_output_line(script, "waiting for another full test run to finish")
        released_at = time.time()
        full_run_lock.close()
        exit_status, _, stderr = self.finish_script(script)

        self.assertEqual(exit_status, 0, stderr)
        test_call = self.find_call("go test")
        self.assertIsNotNone(test_call)
        self.assertGreater(test_call[0], released_at)

    def test_a_package_run_does_not_wait_for_a_full_run(self):
        if self.real_just is None:
            self.skipTest("just is not installed")
        self.hold_lock(FULL_RUN_LOCK_NAME, fcntl.LOCK_EX)

        package_run = subprocess.run(
            [self.real_just, "--justfile", str(self.checkout / "backend" / "Justfile"), "test-package", "./internal/lib/..."],
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=WAIT_TIMEOUT_SECONDS,
        )

        self.assertEqual(package_run.returncode, 0, package_run.stderr)
        self.assertIsNotNone(self.find_call("go test"))

    def test_the_upgrade_test_runs_after_the_packages_pass(self):
        script = self.start_script()

        exit_status, _, stderr = self.finish_script(script)

        self.assertEqual(exit_status, 0, stderr)
        test_call = self.find_call("go test")
        upgrade_call = self.find_call(self.get_upgrade_test_call())
        self.assertIsNotNone(test_call)
        self.assertIsNotNone(upgrade_call)
        self.assertLess(test_call[0], upgrade_call[0])

    def test_the_upgrade_test_does_not_run_and_no_image_is_built_when_a_package_fails(self):
        script = self.start_script({"STUB_GO_TEST_EXIT_STATUS": "1"})

        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIsNotNone(self.find_call("go test"))
        self.assertIsNone(self.find_call(self.get_upgrade_test_call()))
        self.assertIsNone(self.find_call("docker build"))

    def test_the_run_fails_when_the_upgrade_test_fails(self):
        script = self.start_script({"STUB_UPGRADE_TEST_EXIT_STATUS": "1"})

        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIsNotNone(self.find_call("go test"))
        self.assertIsNotNone(self.find_call(self.get_upgrade_test_call()))

    def test_an_interrupt_during_the_upgrade_test_removes_its_containers_networks_and_volumes(self):
        script = self.start_script({"STUB_UPGRADE_TEST_SECONDS": "60"})
        test_call = self.wait_for_call("go test")
        run_id = test_call[1].split("run_id=", 1)[1]
        self.wait_for_call(self.get_upgrade_test_call())

        os.killpg(script.pid, signal.SIGINT)
        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        docker_calls = [recorded_call[1] for recorded_call in self.read_calls() if recorded_call[1].startswith("docker")]
        self.assertIn(f"docker ps -aq --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn(f"docker network ls -q --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn(f"docker volume ls -q --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn("docker rm -f container-of-run", docker_calls)
        self.assertIn("docker network rm network-of-run", docker_calls)
        self.assertIn("docker volume rm -f volume-of-run", docker_calls)
        self.assertIsNone(self.find_call("docker compose down"))

    def test_an_interrupt_removes_only_the_containers_networks_and_volumes_of_its_run(self):
        script = self.start_script({"STUB_GO_TEST_SECONDS": "60"})
        test_call = self.wait_for_call("go test")
        run_id = test_call[1].split("run_id=", 1)[1]
        self.assertTrue(run_id)

        os.killpg(script.pid, signal.SIGINT)
        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        docker_calls = [recorded_call[1] for recorded_call in self.read_calls() if recorded_call[1].startswith("docker")]
        listing_calls = [docker_call for docker_call in docker_calls if " ps " in docker_call or " ls " in docker_call]
        self.assertIn(f"docker ps -aq --filter {RUN_LABEL_FILTER}{run_id}", listing_calls)
        self.assertIn(f"docker network ls -q --filter {RUN_LABEL_FILTER}{run_id}", listing_calls)
        self.assertIn(f"docker volume ls -q --filter {RUN_LABEL_FILTER}{run_id}", listing_calls)
        for listing_call in listing_calls:
            self.assertIn(f"{RUN_LABEL_FILTER}{run_id}", listing_call)
        self.assertIn("docker rm -f container-of-run", docker_calls)
        self.assertIn("docker network rm network-of-run", docker_calls)
        self.assertIn("docker volume rm -f volume-of-run", docker_calls)

    def test_the_reset_waits_for_running_tests_and_runs_before_the_tests(self):
        fixtures_lock = self.hold_lock(FIXTURES_LOCK_NAME, fcntl.LOCK_SH)
        script = self.start_script()
        self.wait_for_call("just --justfile")

        released_at = time.time()
        fixtures_lock.close()
        exit_status, _, stderr = self.finish_script(script)

        self.assertEqual(exit_status, 0, stderr)
        reset_call = self.find_call("go run -tags testhelpers ./cmd/cleanup_test_db")
        test_call = self.find_call("go test")
        self.assertIsNotNone(reset_call)
        self.assertIsNotNone(test_call)
        self.assertGreater(reset_call[0], released_at)
        self.assertLess(reset_call[0], test_call[0])

    def test_the_run_stops_without_a_reset_when_tests_keep_the_servers_past_the_limit(self):
        self.hold_lock(FIXTURES_LOCK_NAME, fcntl.LOCK_SH)
        script = self.start_script({"DATABASUS_FIXTURE_RESET_WAIT_SECONDS": "1"})

        exit_status, _, stderr = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIn("other test runs are using the shared test PostgreSQL servers", stderr)
        self.assertIsNone(self.find_call("go run"))
        self.assertIsNone(self.find_call("go test"))


if __name__ == "__main__":
    unittest.main()
