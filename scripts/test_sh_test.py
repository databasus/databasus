import fcntl
import os
import re
import shutil
import signal
import subprocess
import tempfile
import textwrap
import time
import unittest
from collections.abc import Callable, Sequence
from pathlib import Path
from typing import TextIO

from workflow_jobs import get_jobs

REPOSITORY_ROOT = Path(__file__).resolve().parent.parent
BROWSER_SUITE_CALL = "mise run e2e:browser"
UPGRADE_TEST_CALL = "mise run backend:test-upgrade"
IMAGE_BUILD_CALL = "docker build --tag databasus-full-test:local"
FULL_RUN_IMAGE = "databasus-full-test:local"
LIFECYCLE_SHARD_COUNT = 7
LIFECYCLE_SHARD_CALLS = tuple(
    f"mise run e2e:lifecycle --without-crashed-run-probes --shard {shard}/{LIFECYCLE_SHARD_COUNT}"
    for shard in range(1, LIFECYCLE_SHARD_COUNT + 1)
)
CRASHED_RUN_PROBES_CALL = "mise run e2e:lifecycle --only-crashed-run-probes"
INTERRUPTION_CHECKS_CALL = (
    "uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption"
)
TIME_ZERO_CALLS = (
    "python3 -m unittest discover -s scripts -p *_test.py",
    "python3 -m unittest discover -s .agent-hooks -p *_test.py",
    "mise run e2e:unit",
    "pnpm --dir frontend test:browser --project=ui",
    CRASHED_RUN_PROBES_CALL,
    IMAGE_BUILD_CALL,
)
FULL_ONLY_CALLS = (
    *TIME_ZERO_CALLS,
    UPGRADE_TEST_CALL,
    INTERRUPTION_CHECKS_CALL,
    BROWSER_SUITE_CALL,
    *LIFECYCLE_SHARD_CALLS,
)
BROWSER_RUN_CALLS = (BROWSER_SUITE_CALL, *LIFECYCLE_SHARD_CALLS)
FULL_RUN_LOCK_NAME = "databasus-full-test.lock"
WAIT_TIMEOUT_SECONDS = 30
RUN_LABEL_FILTER = "label=databasus.test-run="

# Each stub appends "<epoch seconds> <name> <arguments>" to $STUB_CALLS. The go stub records
# DATABASUS_TEST_RUN_ID with its test call, sleeps for STUB_GO_TEST_SECONDS, so a test can
# interrupt the run while its tests run, and exits with STUB_GO_TEST_EXIT_STATUS, or with
# STUB_GO_SLOW_TEST_EXIT_STATUS when it runs the first slow package. The docker stub lists one
# container, one network and one volume, and builds an image in STUB_DOCKER_BUILD_SECONDS with
# STUB_DOCKER_BUILD_EXIT_STATUS. The mise stub records the image the upgrade test and the browser
# suite receive, runs them for STUB_UPGRADE_TEST_SECONDS and STUB_BROWSER_SUITE_SECONDS, and the
# lifecycle shards without crashed-run probes for STUB_LIFECYCLE_SECONDS and the crashed-run probes for
# STUB_CRASHED_RUN_PROBES_SECONDS. The pnpm stub takes
# STUB_PNPM_SECONDS and exits with STUB_PNPM_EXIT_STATUS. A long-running stub records
# "<name> finished <arguments>" when it ends, and any stub fails for STUB_FAIL_COMMAND.
STUB_SOURCES = {
    "docker": """
        echo "$(date +%s.%N) docker $*" >>"$STUB_CALLS"
        if [[ "$1" == build ]]; then
            sleep "${STUB_DOCKER_BUILD_SECONDS:-0}"
            echo "$(date +%s.%N) docker finished $*" >>"$STUB_CALLS"
            exit "${STUB_DOCKER_BUILD_EXIT_STATUS:-0}"
        fi
        case "$1 $2" in
            "ps -aq") echo container-of-run ;;
            "network ls") echo network-of-run ;;
            "volume ls") echo volume-of-run ;;
        esac
    """,
    "go": """
        if [[ "$1" == test ]]; then
            echo "$(date +%s.%N) go test run_id=${DATABASUS_TEST_RUN_ID:-}" >>"$STUB_CALLS"
            echo "$(date +%s.%N) go selection $*" >>"$STUB_CALLS"
            echo "output of go test"
            sleep "${STUB_GO_TEST_SECONDS:-0}"
            echo "$(date +%s.%N) go finished $*" >>"$STUB_CALLS"
            if [[ " $* " == *" databasus-backend/internal/lib/slow "* ]]; then
                exit "${STUB_GO_SLOW_TEST_EXIT_STATUS:-${STUB_GO_TEST_EXIT_STATUS:-0}}"
            fi
            exit "${STUB_GO_TEST_EXIT_STATUS:-0}"
        elif [[ "$1" == list ]]; then
            echo "$(date +%s.%N) go $*" >>"$STUB_CALLS"
            echo "databasus-backend/internal/lib/ordinary"
            echo "databasus-backend/internal/lib/slow"
            echo "databasus-backend/internal/lib/slower"
            if [[ "$2" == ./... ]]; then
                echo "databasus-backend/migrations"
                echo "databasus-backend/tools/archlint"
                echo "databasus-backend/newdirectory/fresh"
                echo "databasus-backend/internal/lib/e2e/upgrade"
                echo "databasus-backend/e2e/upgrade"
                echo "databasus-backend/e2e/upgrade/upgradecheck"
                echo "databasus-backend/internal/agent/e2e"
                echo "databasus-backend/internal/modules/backups/e2e/mysql"
                echo "databasus-backend/internal/modules/backups/e2e/postgresqlphysical/patroni"
            fi
            exit "${STUB_GO_LIST_EXIT_STATUS:-0}"
        else
            echo "$(date +%s.%N) go $*" >>"$STUB_CALLS"
        fi
    """,
    "pnpm": """
        echo "$(date +%s.%N) pnpm $*" >>"$STUB_CALLS"
        if [[ "pnpm $*" == "${STUB_FAIL_COMMAND:-}" ]]; then exit 1; fi
        if [[ "$3" == install ]]; then
            exit "${STUB_PNPM_INSTALL_EXIT_STATUS:-0}"
        fi
        echo "output of pnpm"
        sleep "${STUB_PNPM_SECONDS:-0}"
        echo "$(date +%s.%N) pnpm finished" >>"$STUB_CALLS"
        exit "${STUB_PNPM_EXIT_STATUS:-0}"
    """,
    "python3": """
        echo "$(date +%s.%N) python3 $*" >>"$STUB_CALLS"
        if [[ "python3 $*" == "${STUB_FAIL_COMMAND:-}" ]]; then exit 1; fi
    """,
    "uv": """
        echo "$(date +%s.%N) uv $*" >>"$STUB_CALLS"
        if [[ "uv $*" == "${STUB_FAIL_COMMAND:-}" ]]; then exit 1; fi
        echo "$(date +%s.%N) uv finished $*" >>"$STUB_CALLS"
    """,
    "mise": """
        echo "$(date +%s.%N) mise $*" >>"$STUB_CALLS"
        if [[ "mise $*" == "${STUB_FAIL_COMMAND:-}" ]]; then exit 1; fi
        case "$*" in
            "run backend:test-upgrade")
                echo "$(date +%s.%N) image ${DATABASUS_IMAGE:-} for $*" >>"$STUB_CALLS"
                sleep "${STUB_UPGRADE_TEST_SECONDS:-0}"
                echo "$(date +%s.%N) mise finished $*" >>"$STUB_CALLS"
                exit "${STUB_UPGRADE_TEST_EXIT_STATUS:-0}"
                ;;
            "run e2e:browser")
                echo "$(date +%s.%N) image ${DATABASUS_IMAGE:-} for $*" >>"$STUB_CALLS"
                sleep "${STUB_BROWSER_SUITE_SECONDS:-0}"
                echo "$(date +%s.%N) mise finished $*" >>"$STUB_CALLS"
                exit "${STUB_BROWSER_SUITE_EXIT_STATUS:-0}"
                ;;
            "run e2e:lifecycle --without-crashed-run-probes --shard "*)
                sleep "${STUB_LIFECYCLE_SECONDS:-0}"
                echo "$(date +%s.%N) mise finished $*" >>"$STUB_CALLS"
                ;;
            "run e2e:lifecycle --only-crashed-run-probes")
                sleep "${STUB_CRASHED_RUN_PROBES_SECONDS:-0}"
                echo "$(date +%s.%N) mise finished $*" >>"$STUB_CALLS"
                ;;
        esac
    """,
}

SLOW_PACKAGE_LIST = """# A comment
databasus-backend/internal/modules/backups/e2e/postgresqlphysical/patroni
databasus-backend/internal/lib/slower
databasus-backend/internal/lib/slow
databasus-backend/internal/lib/removed
databasus-backend/e2e/upgrade
"""

STUB_FEDORA_SOURCE = """
apply_libedit_shim() {
    :
}
"""


def get_package_limit(go_selection: list[str]) -> int:
    return int(
        next(argument for argument in go_selection if argument.startswith("-p="))[len("-p=") :]
    )


def find_stale_entries(listed_packages: list[str], module_packages: list[str]) -> list[str]:
    tested_packages = {
        package
        for package in module_packages
        if not re.search(r"^databasus-backend/e2e/upgrade(/|$)", package)
    }

    return [package for package in listed_packages if package not in tested_packages]


class TestScriptTestCase(unittest.TestCase):
    """Runs a copy of scripts/test.sh in a checkout of its own, so the repository's .env and the
    machine's real locks stay out of the test."""

    def setUp(self) -> None:
        self.workspace = Path(tempfile.mkdtemp(prefix="test-sh-"))
        self.addCleanup(shutil.rmtree, self.workspace, ignore_errors=True)

        self.checkout = self.workspace / "checkout"
        (self.checkout / "scripts").mkdir(parents=True)
        (self.checkout / "backend").mkdir()
        shutil.copy(REPOSITORY_ROOT / "scripts" / "test.sh", self.checkout / "scripts" / "test.sh")
        (self.checkout / "backend" / "test-slow-packages.txt").write_text(SLOW_PACKAGE_LIST)
        (self.checkout / "scripts" / "fedora.sh").write_text(STUB_FEDORA_SOURCE)

        self.lock_directory = self.workspace / "locks"
        self.lock_directory.mkdir()
        self.calls_path = self.workspace / "calls"
        self.calls_path.touch()

        self.stubs_directory = self.workspace / "stubs"
        self.stubs_directory.mkdir()
        for name, source in STUB_SOURCES.items():
            stub_path = self.stubs_directory / name
            stub_path.write_text("#!/usr/bin/env bash\n" + textwrap.dedent(source))
            stub_path.chmod(0o755)

        self.environment = {
            **os.environ,
            "PATH": f"{self.stubs_directory}:{os.environ['PATH']}",
            "STUB_CALLS": str(self.calls_path),
            "TMPDIR": str(self.lock_directory),
        }
        self.environment.pop("DATABASUS_TEST_RUN_ID", None)
        self.environment.pop("CI", None)

    def start_full_script(
        self, extra_environment: dict[str, str] | None = None
    ) -> subprocess.Popen[str]:
        return self.start_script(extra_environment, ["--full"])

    def start_script(
        self, extra_environment: dict[str, str] | None = None, arguments: Sequence[str] = ()
    ) -> subprocess.Popen[str]:
        script = subprocess.Popen(
            ["bash", str(self.checkout / "scripts" / "test.sh"), *arguments],
            env={**self.environment, **(extra_environment or {})},
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            start_new_session=True,
        )
        self.addCleanup(self.stop_script, script)

        return script

    def stop_script(self, script: subprocess.Popen[str]) -> None:
        if script.poll() is None:
            os.killpg(script.pid, signal.SIGKILL)
        script.communicate()

    def hold_lock(self, lock_name: str, lock_operation: int) -> TextIO:
        lock_file = open(self.lock_directory / lock_name, "a")
        self.addCleanup(lock_file.close)
        fcntl.flock(lock_file, lock_operation)

        return lock_file

    def read_calls(self) -> list[tuple[float, str]]:
        recorded_calls = []
        for line in self.calls_path.read_text().splitlines():
            called_at, call = line.split(" ", 1)
            recorded_calls.append((float(called_at), call))

        return recorded_calls

    def find_call(self, prefix: str) -> tuple[float, str] | None:
        return next(
            (
                recorded_call
                for recorded_call in self.read_calls()
                if recorded_call[1].startswith(prefix)
            ),
            None,
        )

    def wait_for_call(self, prefix: str) -> tuple[float, str]:
        deadline = time.monotonic() + WAIT_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            recorded_call = self.find_call(prefix)
            if recorded_call is not None:
                return recorded_call
            time.sleep(0.05)

        self.fail(f"no call starting with {prefix!r}; calls: {self.read_calls()}")

    def wait_for_output_line(self, script: subprocess.Popen[str], text: str) -> None:
        deadline = time.monotonic() + WAIT_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            assert script.stdout is not None
            line = script.stdout.readline()
            if text in line:
                return
            if not line and script.poll() is not None:
                break

        self.fail(f"the script did not print {text!r}")

    def finish_script(self, script: subprocess.Popen[str]) -> tuple[int, str, str]:
        stdout, stderr = script.communicate(timeout=WAIT_TIMEOUT_SECONDS)

        return script.returncode, stdout, stderr

    def test_a_routine_run_waits_for_another_run_and_says_so(self) -> None:
        self.assert_waits_for_another_run(self.start_script)

    def test_a_full_run_waits_for_another_run_and_says_so(self) -> None:
        self.assert_waits_for_another_run(self.start_full_script)

    def assert_waits_for_another_run(self, start_run: Callable[[], subprocess.Popen[str]]) -> None:
        full_run_lock = self.hold_lock(FULL_RUN_LOCK_NAME, fcntl.LOCK_EX)
        script = start_run()

        self.wait_for_output_line(script, "waiting for another test run to finish")
        released_at = time.time()
        full_run_lock.close()
        exit_status, _, stderr = self.finish_script(script)

        self.assertEqual(exit_status, 0, stderr)
        test_call = self.find_call("go test")
        assert test_call is not None
        self.assertGreater(test_call[0], released_at)

    def get_go_selections(self) -> list[list[str]]:
        return [
            call[len("go selection ") :].split()
            for _, call in self.read_calls()
            if call.startswith("go selection ")
        ]

    def get_selected_packages(self) -> list[str]:
        return [
            argument
            for selection in self.get_go_selections()
            for argument in selection
            if argument.startswith("databasus-backend/")
        ]

    def get_selection_of(self, package: str) -> list[str]:
        return next(selection for selection in self.get_go_selections() if package in selection)

    def test_the_run_starts_the_shared_test_postgresql_server_without_recreating_it(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_script())
        self.assertEqual(exit_status, 0, stderr)

        compose_calls = [call for _, call in self.read_calls() if call.startswith("docker compose")]
        self.assertEqual(compose_calls, ["docker compose up -d --no-recreate backend-db-test"])

    def test_the_routine_run_starts_neither_the_upgrade_test_nor_the_browser_suite(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_script())
        self.assertEqual(exit_status, 0, stderr)

        self.assertIsNone(self.find_call(UPGRADE_TEST_CALL))
        self.assertIsNone(self.find_call(BROWSER_SUITE_CALL))

    def test_the_routine_run_starts_no_full_only_suite(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_script())

        self.assertEqual(exit_status, 0, stderr)
        for command in FULL_ONLY_CALLS:
            self.assertIsNone(self.find_call(command), command)

    def count_calls(self, prefix: str) -> int:
        return sum(1 for _, call in self.read_calls() if call.startswith(prefix))

    def get_required_call(self, prefix: str) -> tuple[float, str]:
        recorded_call = self.find_call(prefix)
        assert recorded_call is not None, f"no call starting with {prefix!r}: {self.read_calls()}"

        return recorded_call

    def test_the_full_run_executes_each_extra_suite_once(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_full_script())

        self.assertEqual(exit_status, 0, stderr)
        for command in FULL_ONLY_CALLS:
            self.assertEqual(self.count_calls(command), 1, command)
        self.assertIsNone(self.find_call("mise run test-filesystem"))
        self.assertIsNone(self.find_call("mise run agent:e2e-podman"))

    def test_the_full_run_starts_independent_suites_while_the_packages_run(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_GO_TEST_SECONDS": "2"})
        )

        self.assertEqual(exit_status, 0, stderr)
        first_package_finished_at = self.get_required_call("go finished")[0]
        for command in TIME_ZERO_CALLS:
            self.assertLess(self.get_required_call(command)[0], first_package_finished_at, command)

    def test_the_image_suites_do_not_wait_for_the_crashed_run_probes(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_CRASHED_RUN_PROBES_SECONDS": "2"})
        )

        self.assertEqual(exit_status, 0, stderr)
        crashed_run_probes_finished_at = self.get_required_call(
            f"mise finished {CRASHED_RUN_PROBES_CALL[5:]}"
        )[0]
        self.assertLess(
            self.get_required_call(UPGRADE_TEST_CALL)[0], crashed_run_probes_finished_at
        )

    def test_no_other_browser_run_starts_until_the_interruption_checks_end(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_CRASHED_RUN_PROBES_SECONDS": "1"})
        )

        self.assertEqual(exit_status, 0, stderr)
        crashed_run_probes_finished_at = self.get_required_call(
            f"mise finished {CRASHED_RUN_PROBES_CALL[5:]}"
        )[0]
        interruption_checks_started_at = self.get_required_call(INTERRUPTION_CHECKS_CALL)[0]
        interruption_checks_finished_at = self.get_required_call(
            f"uv finished {INTERRUPTION_CHECKS_CALL[3:]}"
        )[0]
        self.assertLess(crashed_run_probes_finished_at, interruption_checks_started_at)
        for command in BROWSER_RUN_CALLS:
            self.assertLess(interruption_checks_finished_at, self.get_required_call(command)[0])

    def test_the_interruption_checks_run_after_failed_crashed_run_probes(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_FAIL_COMMAND": CRASHED_RUN_PROBES_CALL})
        )

        self.assertNotEqual(exit_status, 0)
        self.assertIn("failed: lifecycle-crashed-runs", stderr)
        self.assertIsNotNone(self.find_call(f"uv finished {INTERRUPTION_CHECKS_CALL[3:]}"))

    def test_a_failed_extra_suite_stops_no_other_suite(self) -> None:
        for failed_command in FULL_ONLY_CALLS:
            if failed_command == IMAGE_BUILD_CALL:
                continue
            with self.subTest(command=failed_command):
                self.calls_path.write_text("")

                exit_status, _, stderr = self.finish_script(
                    self.start_full_script({"STUB_FAIL_COMMAND": failed_command})
                )

                self.assertNotEqual(exit_status, 0)
                self.assertIn("failed: ", stderr)
                for command in FULL_ONLY_CALLS:
                    self.assertEqual(self.count_calls(command), 1, command)

    def test_a_package_failure_lets_the_upgrade_test_run_to_its_end(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_GO_TEST_EXIT_STATUS": "1"})
        )

        self.assertNotEqual(exit_status, 0)
        self.assertIn("backend-other", stderr)
        self.assertIsNotNone(self.find_call(IMAGE_BUILD_CALL))
        self.assertIsNotNone(self.find_call(f"mise finished {UPGRADE_TEST_CALL[5:]}"))

    def test_a_frontend_failure_still_builds_the_image_and_runs_the_upgrade_test(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_PNPM_EXIT_STATUS": "1"})
        )

        self.assertNotEqual(exit_status, 0)
        self.assertIn("frontend", stderr)
        self.assertIsNotNone(self.find_call(IMAGE_BUILD_CALL))
        self.assertIsNotNone(self.find_call(f"mise finished {UPGRADE_TEST_CALL[5:]}"))

    def test_a_failed_build_leaves_only_the_image_suites_not_started(self) -> None:
        exit_status, _, stderr = self.finish_script(
            self.start_full_script({"STUB_DOCKER_BUILD_EXIT_STATUS": "1"})
        )

        self.assertNotEqual(exit_status, 0)
        self.assertIn("failed: image-build", stderr)
        self.assertIn("not started: upgrade e2e-browser", stderr)
        self.assertIsNone(self.find_call(UPGRADE_TEST_CALL))
        self.assertIsNone(self.find_call(BROWSER_SUITE_CALL))
        for command in (*LIFECYCLE_SHARD_CALLS, CRASHED_RUN_PROBES_CALL, INTERRUPTION_CHECKS_CALL):
            self.assertIsNotNone(self.find_call(command), command)

    def test_the_full_run_tests_and_removes_the_image_it_built(self) -> None:
        for extra_environment in ({}, {"STUB_UPGRADE_TEST_EXIT_STATUS": "1"}):
            with self.subTest(environment=extra_environment):
                self.calls_path.write_text("")

                self.finish_script(self.start_full_script(extra_environment))

                build_call = self.get_required_call(IMAGE_BUILD_CALL)
                self.assertEqual(build_call[1], f"{IMAGE_BUILD_CALL} {self.checkout}")
                for command in (UPGRADE_TEST_CALL, BROWSER_SUITE_CALL):
                    self.assertIsNotNone(
                        self.find_call(f"image {FULL_RUN_IMAGE} for {command[5:]}"), command
                    )
                removal_call = self.get_required_call(f"docker image rm {FULL_RUN_IMAGE}")
                self.assertLess(
                    self.get_required_call(f"mise finished {UPGRADE_TEST_CALL[5:]}")[0],
                    removal_call[0],
                )

    def test_a_named_image_is_tested_without_a_build_and_kept(self) -> None:
        exit_status, stdout, stderr = self.finish_script(
            self.start_full_script({"DATABASUS_IMAGE": "named:image"})
        )

        self.assertEqual(exit_status, 0, stderr)
        self.assertIsNone(self.find_call("docker build"))
        self.assertIsNone(self.find_call("docker image rm"))
        for command in (UPGRADE_TEST_CALL, BROWSER_SUITE_CALL):
            self.assertIsNotNone(self.find_call(f"image named:image for {command[5:]}"), command)
        self.assertRegex(stdout, r"(?m)^  image-build +skipped *$")
        for suite_name in ("backend-slow", "upgrade", "e2e-browser", "lifecycle-crashed-runs"):
            self.assertRegex(stdout, rf"(?m)^  {suite_name} +passed +\d+s$", suite_name)
        self.assertRegex(stdout, r"(?m)^total: \d+s$")

    def test_the_summary_lists_each_suite_with_its_duration_and_the_total(self) -> None:
        exit_status, stdout, stderr = self.finish_script(
            self.start_full_script({"STUB_DOCKER_BUILD_SECONDS": "1"})
        )

        self.assertEqual(exit_status, 0, stderr)
        for suite_name in (
            "backend-slow",
            "backend-other",
            "frontend",
            "scripts",
            "review-hooks",
            "e2e-unit",
            "browser-ui",
            *(
                f"lifecycle-{shard}-of-{LIFECYCLE_SHARD_COUNT}"
                for shard in range(1, LIFECYCLE_SHARD_COUNT + 1)
            ),
            "upgrade",
            "e2e-browser",
            "lifecycle-crashed-runs",
            "lifecycle-interruption-checks",
        ):
            self.assertRegex(stdout, rf"(?m)^  {suite_name} +passed +\d+s$", suite_name)
        self.assertRegex(stdout, r"(?m)^  image-build +passed +[1-9]\d*s$")
        self.assertRegex(stdout, r"(?m)^total: \d+s$")

    def test_the_routine_run_selects_no_end_to_end_package(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_script())
        self.assertEqual(exit_status, 0, stderr)

        selected_packages = self.get_selected_packages()
        self.assertNotIn("databasus-backend/internal/agent/e2e", selected_packages)
        self.assertNotIn("databasus-backend/internal/modules/backups/e2e/mysql", selected_packages)

    def test_the_full_run_selects_the_end_to_end_packages_but_not_the_upgrade_test(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_full_script())
        self.assertEqual(exit_status, 0, stderr)

        selected_packages = self.get_selected_packages()
        self.assertIn("databasus-backend/internal/agent/e2e", selected_packages)
        self.assertIn("databasus-backend/internal/modules/backups/e2e/mysql", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade", selected_packages)

    def test_the_full_run_fails_when_the_browser_suite_fails(self) -> None:
        exit_status, _, _ = self.finish_script(
            self.start_full_script({"STUB_BROWSER_SUITE_EXIT_STATUS": "1"})
        )

        self.assertNotEqual(exit_status, 0)

    def test_default_selection_covers_new_directories_of_the_module_except_the_upgrade_test(
        self,
    ) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        self.assertIsNotNone(self.find_call("go list ./..."))
        selected_packages = self.get_selected_packages()
        self.assertIn("databasus-backend/migrations", selected_packages)
        self.assertIn("databasus-backend/tools/archlint", selected_packages)
        self.assertIn("databasus-backend/newdirectory/fresh", selected_packages)
        self.assertIn("databasus-backend/internal/lib/e2e/upgrade", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade/upgradecheck", selected_packages)

    def test_every_package_runs_exactly_once(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        selected_packages = self.get_selected_packages()
        self.assertEqual(len(selected_packages), len(set(selected_packages)))

    def test_slow_packages_run_apart_from_the_others_in_the_order_of_their_list(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        self.assertEqual(len(self.get_go_selections()), 2)
        slow_selection = self.get_selection_of("databasus-backend/internal/lib/slow")
        self.assertLess(
            slow_selection.index("databasus-backend/internal/lib/slower"),
            slow_selection.index("databasus-backend/internal/lib/slow"),
        )
        self.assertNotIn("databasus-backend/internal/lib/ordinary", slow_selection)

    def test_a_package_missing_from_the_list_runs_with_the_others(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        other_selection = self.get_selection_of("databasus-backend/internal/lib/ordinary")
        self.assertIn("databasus-backend/newdirectory/fresh", other_selection)

    def test_list_entries_outside_the_selection_do_not_reach_go_test(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        selected_packages = self.get_selected_packages()
        self.assertNotIn("databasus-backend/internal/lib/removed", selected_packages)
        self.assertNotIn("databasus-backend/e2e/upgrade", selected_packages)

    def test_slow_packages_run_under_a_lower_package_limit_than_the_others(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        slow_limit = get_package_limit(self.get_selection_of("databasus-backend/internal/lib/slow"))
        other_limit = get_package_limit(
            self.get_selection_of("databasus-backend/internal/lib/ordinary")
        )
        self.assertLess(slow_limit, other_limit)

    def test_slow_packages_run_under_a_parallel_test_limit_from_the_runner(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_script())
        self.assertEqual(exit_status, 0, stderr)

        slow_selection = self.get_selection_of("databasus-backend/internal/lib/slow")
        other_selection = self.get_selection_of("databasus-backend/internal/lib/ordinary")
        self.assertIn("-parallel=8", slow_selection)
        self.assertFalse(any(argument.startswith("-parallel") for argument in other_selection))

    def test_slow_and_other_packages_have_15_minutes(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_script())
        self.assertEqual(exit_status, 0, stderr)

        slow_selection = self.get_selection_of("databasus-backend/internal/lib/slow")
        other_selection = self.get_selection_of("databasus-backend/internal/lib/ordinary")
        self.assertEqual(slow_selection[slow_selection.index("-timeout") + 1], "15m")
        self.assertEqual(other_selection[other_selection.index("-timeout") + 1], "15m")

    def test_full_run_selects_patroni_once_with_the_common_timeout(self) -> None:
        exit_status, _, stderr = self.finish_script(self.start_full_script())

        self.assertEqual(exit_status, 0, stderr)

        package = "databasus-backend/internal/modules/backups/e2e/postgresqlphysical/patroni"
        selection = self.get_selection_of(package)

        self.assertEqual(self.get_selected_packages().count(package), 1)
        self.assertIn("databasus-backend/internal/lib/slow", selection)
        self.assertEqual(selection[selection.index("-timeout") + 1], "15m")

    def test_the_upgrade_test_has_a_timeout_of_15_minutes_and_30_in_ci(self) -> None:
        upgrade_task = (REPOSITORY_ROOT / "mise-tasks/backend/test-upgrade").read_text()

        self.assertIn("package_timeout=15m\n", upgrade_task)
        self.assertIn('if [[ -n "${CI:-}" ]]; then\n    package_timeout=30m\n', upgrade_task)
        self.assertIn('-timeout "$package_timeout" ./e2e/upgrade/...', upgrade_task)

    def test_a_failure_does_not_stop_the_tests_of_a_package(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        for selection in self.get_go_selections():
            self.assertNotIn("-failfast", selection)

    def test_the_backend_runs_with_the_race_detector_only_under_ci(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        for selection in self.get_go_selections():
            self.assertNotIn("-race", selection)

        self.calls_path.write_text("")
        script = self.start_script({"CI": "true"})
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        for selection in self.get_go_selections():
            self.assertIn("-race", selection)

    def test_frontend_dependencies_are_installed_from_the_lockfile_before_the_suites(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        install_call = self.find_call(
            f"pnpm --dir {self.checkout}/frontend install --frozen-lockfile"
        )
        assert install_call is not None
        test_call = self.find_call("go test")
        assert test_call is not None
        self.assertLess(install_call[0], test_call[0])
        frontend_call = self.find_call(f"pnpm --dir {self.checkout}/frontend test")
        assert frontend_call is not None
        self.assertLess(install_call[0], frontend_call[0])

    def test_a_failed_install_starts_no_suite(self) -> None:
        script = self.start_script({"STUB_PNPM_INSTALL_EXIT_STATUS": "1"})
        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIsNone(self.find_call("go test"))
        self.assertIsNone(self.find_call(f"pnpm --dir {self.checkout}/frontend test"))

    def test_every_entry_of_the_repository_list_names_a_tested_package(self) -> None:
        listed_packages = [
            line
            for line in (REPOSITORY_ROOT / "backend/test-slow-packages.txt")
            .read_text()
            .splitlines()
            if line and not line.startswith("#")
        ]
        module_packages = subprocess.check_output(
            ["go", "list", "./..."], cwd=REPOSITORY_ROOT / "backend", text=True
        ).splitlines()

        self.assertEqual(find_stale_entries(listed_packages, module_packages), [])
        self.assertEqual(len(listed_packages), len(set(listed_packages)))

    def test_a_stale_entry_is_named(self) -> None:
        self.assertEqual(
            find_stale_entries(["pkg/kept", "pkg/removed"], ["pkg/kept"]), ["pkg/removed"]
        )
        self.assertEqual(
            find_stale_entries(
                ["databasus-backend/e2e/upgrade"], ["databasus-backend/e2e/upgrade"]
            ),
            ["databasus-backend/e2e/upgrade"],
        )

    def test_the_suites_run_at_the_same_time(self) -> None:
        script = self.start_script({"STUB_GO_TEST_SECONDS": "1", "STUB_PNPM_SECONDS": "1"})
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        started_at = [
            called_at
            for called_at, call in self.read_calls()
            if call.startswith("go selection") or call.endswith("/frontend test")
        ]
        finished_at = [
            called_at
            for called_at, call in self.read_calls()
            if call.startswith(("go finished", "pnpm finished"))
        ]
        self.assertEqual(len(started_at), 3)
        self.assertLess(max(started_at), min(finished_at))

    def test_the_frontend_tests_run_in_the_frontend_directory(self) -> None:
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        self.assertIsNotNone(self.find_call(f"pnpm --dir {self.checkout}/frontend test"))

    def test_each_suite_marks_its_output_lines(self) -> None:
        script = self.start_script()
        exit_status, stdout, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        self.assertIn("[backend-slow] output of go test", stdout)
        self.assertIn("[backend-other] output of go test", stdout)
        self.assertIn("[frontend] output of pnpm", stdout)

    def test_a_frontend_failure_fails_the_routine_run_and_names_the_suite(self) -> None:
        script = self.start_script({"STUB_PNPM_EXIT_STATUS": "1"})

        exit_status, _, stderr = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIn("frontend", stderr)
        self.assertIsNone(self.find_call(UPGRADE_TEST_CALL))

    def test_a_backend_failure_fails_the_routine_run_and_names_the_suite(self) -> None:
        script = self.start_script({"STUB_GO_TEST_EXIT_STATUS": "1"})

        exit_status, _, stderr = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIn("backend-other", stderr)
        self.assertIsNone(self.find_call(UPGRADE_TEST_CALL))

    def test_a_backend_failure_lets_the_frontend_finish_and_names_the_suite(self) -> None:
        script = self.start_full_script(
            {"STUB_GO_SLOW_TEST_EXIT_STATUS": "1", "STUB_PNPM_SECONDS": "1"}
        )

        exit_status, _, stderr = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIn("backend-slow", stderr)
        self.assertNotIn("backend-other", stderr)
        self.assertIsNotNone(self.find_call("pnpm finished"))

    def test_an_interrupt_stops_every_suite_and_removes_what_the_run_started(self) -> None:
        script = self.start_full_script(
            {
                "STUB_GO_TEST_SECONDS": "60",
                "STUB_PNPM_SECONDS": "60",
                "STUB_LIFECYCLE_SECONDS": "60",
                "STUB_UPGRADE_TEST_SECONDS": "60",
                "STUB_BROWSER_SUITE_SECONDS": "60",
            }
        )
        run_id = self.wait_for_call("go test")[1].split("run_id=", 1)[1]
        for command in (UPGRADE_TEST_CALL, *BROWSER_RUN_CALLS):
            self.wait_for_call(command)
        while len(self.get_go_selections()) < 2:
            time.sleep(0.05)

        script.terminate()
        exit_status, _, _ = self.finish_script(script)

        self.assertEqual(exit_status, 143)
        for finished_call in (
            "go finished",
            "pnpm finished",
            *(
                f"mise finished {command[5:]}"
                for command in (UPGRADE_TEST_CALL, *BROWSER_RUN_CALLS)
            ),
        ):
            self.assertIsNone(self.find_call(finished_call), finished_call)
        remaining_stubs = subprocess.run(
            ["pgrep", "-f", str(self.stubs_directory)], capture_output=True, text=True
        ).stdout
        self.assertEqual(remaining_stubs, "")
        docker_calls = [call for _, call in self.read_calls() if call.startswith("docker")]
        self.assertIn(f"docker ps -aq --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn("docker rm -f container-of-run", docker_calls)
        self.assertIn("docker network rm network-of-run", docker_calls)
        self.assertIn("docker volume rm -f volume-of-run", docker_calls)
        self.assertIn(f"docker image rm {FULL_RUN_IMAGE}", docker_calls)
        self.assertIsNone(self.find_call("docker compose down"))

    def test_discovery_stops_verification_when_go_list_fails_after_partial_output(self) -> None:
        script = self.start_full_script({"STUB_GO_LIST_EXIT_STATUS": "7"})
        exit_status, _, _ = self.finish_script(script)
        self.assertEqual(exit_status, 7)
        self.assertIsNone(self.find_call("go test"))
        self.assertIsNone(self.find_call(UPGRADE_TEST_CALL))

    def test_the_run_fails_when_the_upgrade_test_fails(self) -> None:
        script = self.start_full_script({"STUB_UPGRADE_TEST_EXIT_STATUS": "1"})

        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        self.assertIsNotNone(self.find_call("go test"))
        self.assertIsNotNone(self.find_call(UPGRADE_TEST_CALL))

    def test_an_interrupt_during_the_upgrade_test_removes_its_containers_networks_and_volumes(
        self,
    ) -> None:
        script = self.start_full_script({"STUB_UPGRADE_TEST_SECONDS": "60"})
        test_call = self.wait_for_call("go test")
        run_id = test_call[1].split("run_id=", 1)[1]
        self.wait_for_call(UPGRADE_TEST_CALL)

        os.killpg(script.pid, signal.SIGINT)
        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        docker_calls = [
            recorded_call[1]
            for recorded_call in self.read_calls()
            if recorded_call[1].startswith("docker")
        ]
        self.assertIn(f"docker ps -aq --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn(f"docker network ls -q --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn(f"docker volume ls -q --filter {RUN_LABEL_FILTER}{run_id}", docker_calls)
        self.assertIn("docker rm -f container-of-run", docker_calls)
        self.assertIn("docker network rm network-of-run", docker_calls)
        self.assertIn("docker volume rm -f volume-of-run", docker_calls)
        self.assertIn(f"docker image rm {FULL_RUN_IMAGE}", docker_calls)
        self.assertIsNone(self.find_call("docker compose down"))

    def test_an_interrupt_removes_only_the_containers_networks_and_volumes_of_its_run(self) -> None:
        script = self.start_script({"STUB_GO_TEST_SECONDS": "60"})
        test_call = self.wait_for_call("go test")
        run_id = test_call[1].split("run_id=", 1)[1]
        self.assertTrue(run_id)

        os.killpg(script.pid, signal.SIGINT)
        exit_status, _, _ = self.finish_script(script)

        self.assertNotEqual(exit_status, 0)
        docker_calls = [
            recorded_call[1]
            for recorded_call in self.read_calls()
            if recorded_call[1].startswith("docker")
        ]
        listing_calls = [
            docker_call
            for docker_call in docker_calls
            if " ps " in docker_call or " ls " in docker_call
        ]

        self.assertIn(f"docker ps -aq --filter {RUN_LABEL_FILTER}{run_id}", listing_calls)
        self.assertIn(f"docker network ls -q --filter {RUN_LABEL_FILTER}{run_id}", listing_calls)
        self.assertIn(f"docker volume ls -q --filter {RUN_LABEL_FILTER}{run_id}", listing_calls)
        for listing_call in listing_calls:
            self.assertIn(f"{RUN_LABEL_FILTER}{run_id}", listing_call)
        self.assertIn("docker rm -f container-of-run", docker_calls)
        self.assertIn("docker network rm network-of-run", docker_calls)
        self.assertIn("docker volume rm -f volume-of-run", docker_calls)

    def test_preparation_does_not_reset_sources_beside_an_active_package_run(self) -> None:
        package_run = subprocess.Popen(
            ["go", "test", "./fixture"],
            env={**self.environment, "STUB_GO_TEST_SECONDS": "60"},
            start_new_session=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        self.addCleanup(self.stop_script, package_run)

        self.wait_for_call("go test")
        script = self.start_script()
        exit_status, _, stderr = self.finish_script(script)
        self.assertEqual(exit_status, 0, stderr)

        self.assertIsNone(package_run.poll())
        self.assertIsNone(self.find_call("go run"))
        self.assertFalse(any("cleanup_test_db" in call for _, call in self.read_calls()))

    def test_all_ci_preparations_use_only_the_internal_test_database_wait(self) -> None:
        workflow = (REPOSITORY_ROOT / ".github/workflows/ci-release.yml").read_text()
        workflow = "\n".join(get_jobs(workflow).values())
        self.assertNotIn("cleanup_test_db", workflow)
        self.assertNotIn("Reset the shared", workflow)
        self.assertEqual(workflow.count("docker exec backend-db-test pg_isready"), 2)
        self.assertNotRegex(workflow, r"nc -z[^\n]*5004")

    def test_ci_starts_only_the_test_postgresql_server(self) -> None:
        workflow = (REPOSITORY_ROOT / ".github/workflows/ci-release.yml").read_text()
        workflow = "\n".join(get_jobs(workflow).values())
        compose_starts = re.findall(
            r"docker compose (?:-f docker-compose\.yml )?up -d[^\n]*", workflow
        )
        self.assertEqual(len(compose_starts), 2)
        for compose_start in compose_starts:
            self.assertRegex(compose_start, r"up -d backend-db-test$")
        self.assertNotIn("docker exec backend-db pg_isready", workflow)


if __name__ == "__main__":
    unittest.main()
