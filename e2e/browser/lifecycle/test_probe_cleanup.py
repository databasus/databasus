import os
import secrets
import signal
import subprocess
import sys
from pathlib import Path

import pytest
from leftover_check import assert_nothing_of_the_project_remains, is_image_present
from lifecycle.run_cleanup import RunCleanup, remove_run_resources
from public_command import PublicRun
from supervisor_file import SupervisorRecord, get_run_directory


def _start_cleanup_probe(test_file: Path, *, plugin: str, environment: dict[str, str]) -> PublicRun:
    return PublicRun(
        command=(
            sys.executable,
            "-m",
            "pytest",
            "-p",
            plugin,
            "--capture=no",
            "-c",
            str(Path(__file__).parents[1] / "pyproject.toml"),
            "--basetemp",
            str(test_file.parent / "child-pytest"),
            str(test_file),
        ),
        environment={"PYTEST_ADDOPTS": "", "PYTHONDONTWRITEBYTECODE": "1", **environment},
        working_directory=Path(__file__).parents[1],
    )


class TestProbeCleanup:
    def test_removes_the_project_image_and_directory_when_a_recovery_probe_fails(
        self, probes_image: str, tmp_path: Path, register_run_cleanup: RunCleanup
    ) -> None:
        child_test = tmp_path / "test_failed_recovery.py"
        release_marker = tmp_path / "release"
        child_test.write_text("""
import os
import signal
import subprocess
import time
from dataclasses import replace
from pathlib import Path

import pytest

from probe_run import INTERRUPT_READY, start_probe_run
from supervisor_file import read_supervisor_records, write_supervisor_record


@pytest.fixture(autouse=True)
def unrelated_container_keeps_answering():
    yield


def test_deliberate_recovery_failure(register_run_cleanup):
    run = start_probe_run(os.environ["FAILED_PROBE_IMAGE"], "interrupt")
    owned_image = f"{run.get_project_name()}:failed-probe"
    register_run_cleanup(run, built_image=owned_image)
    run.wait_for_output(INTERRUPT_READY)
    subprocess.run(["docker", "image", "tag", os.environ["FAILED_PROBE_IMAGE"], owned_image], check=True)
    record = next(record for record in read_supervisor_records() if record.run_id == run.get_run_id())
    write_supervisor_record(replace(record, built_image=owned_image))
    print(run.get_output(), flush=True)
    print("FAILED_PROBE_RESOURCES_READY", flush=True)
    deadline = time.monotonic() + 30

    while not Path(os.environ["FAILED_PROBE_RELEASE"]).exists():
        assert time.monotonic() < deadline
        time.sleep(0.05)

    run.send_signal_to_the_whole_command(signal.SIGKILL)
    run.finish()

    pytest.fail("deliberate failure before recovery")
""")
        child_run = _start_cleanup_probe(
            child_test,
            plugin="lifecycle.conftest",
            environment={
                "FAILED_PROBE_IMAGE": probes_image,
                "FAILED_PROBE_RELEASE": str(release_marker),
            },
        )
        try:
            child_run.wait_for_output("FAILED_PROBE_RESOURCES_READY")
            owned_image = f"{child_run.get_project_name()}:failed-probe"
            register_run_cleanup(child_run, built_image=owned_image)
            release_marker.touch()
            finished_run = child_run.finish()

            assert finished_run.exit_code == 1, finished_run.output
            assert "deliberate failure before recovery" in finished_run.output
            assert "1 failed" in finished_run.output
            assert_nothing_of_the_project_remains(child_run.get_project_name())
            assert not is_image_present(owned_image)
            assert not get_run_directory(child_run.get_run_id()).exists()
        finally:
            child_run.send_signal_to_the_whole_command(signal.SIGTERM)
            child_run.finish()

    def test_removes_lifecycle_resources_when_recovery_of_a_killed_command_fails(
        self, probes_image: str, tmp_path: Path, request: pytest.FixtureRequest
    ) -> None:
        run_id = secrets.token_hex(6)
        owned_image = f"databasus-e2e-{run_id}:probes"
        unrelated_container = f"unrelated-to-e2e-{run_id}"
        supervisor = tmp_path / "supervisor.py"
        supervisor.write_text("""
import os
import signal

from supervisor_file import SupervisorRecord, get_process_start_time, write_supervisor_record

run_id = os.environ["FAILED_LIFECYCLE_RUN_ID"]
write_supervisor_record(SupervisorRecord(
    run_id, os.getpid(), get_process_start_time(os.getpid()), os.environ["FAILED_LIFECYCLE_IMAGE"]
))
print(f"E2E run: {run_id}; project: databasus-e2e-{run_id}; supervisor PID: {os.getpid()}", flush=True)
signal.pause()
""")
        mise = tmp_path / "mise"
        mise.write_text("""#!/usr/bin/env bash
if [[ "$*" == "run e2e:lifecycle" ]]; then
    exec "$FAILED_LIFECYCLE_PYTHON" "$FAILED_LIFECYCLE_SUPERVISOR"
fi

echo 'deliberate recovery failure'
exit 1
""")
        mise.chmod(0o755)
        child_test = tmp_path / "test_failed_lifecycle_recovery.py"
        child_test.write_text("""
import pytest

from lifecycle_interruption.test_lifecycle_interruption import TestLifecycleInterruption as LifecycleInterruption


def test_deliberate_recovery_failure(lifecycle_run):
    print(lifecycle_run.get_output(), flush=True)
    try:
        LifecycleInterruption().test_the_next_run_removes_its_image_and_directory_when_killed(lifecycle_run)
    finally:
        pytest.fail("deliberate recovery failure")
""")
        child_run = _start_cleanup_probe(
            child_test,
            plugin="lifecycle_interruption.test_lifecycle_interruption",
            environment={
                "PATH": f"{tmp_path}:{os.environ['PATH']}",
                "PYTHONPATH": str(Path(__file__).parents[1]),
                "FAILED_LIFECYCLE_RUN_ID": run_id,
                "FAILED_LIFECYCLE_IMAGE": owned_image,
                "FAILED_LIFECYCLE_PYTHON": sys.executable,
                "FAILED_LIFECYCLE_SUPERVISOR": str(supervisor),
            },
        )
        record = SupervisorRecord(run_id, 0, "unreadable", owned_image)

        def cleanup_child_run() -> None:
            try:
                child_run.send_signal_to_the_whole_command(signal.SIGTERM)
                child_run.finish()
            finally:
                remove_run_resources(child_run, record, unrelated_container=unrelated_container)

        request.addfinalizer(cleanup_child_run)
        subprocess.run(
            [
                "docker",
                "run",
                "--detach",
                "--rm",
                "--name",
                unrelated_container,
                "--entrypoint",
                "/bin/sleep",
                probes_image,
                "300",
            ],
            check=True,
            capture_output=True,
        )
        # The image releases the child's readiness wait, so all resources must exist first.
        subprocess.run(["docker", "image", "tag", probes_image, owned_image], check=True)
        finished_run = child_run.finish()

        assert finished_run.exit_code == 1, finished_run.output
        assert "deliberate recovery failure" in finished_run.output
        assert not is_image_present(owned_image), finished_run.output
        assert not get_run_directory(run_id).exists()
        container_status = subprocess.run(
            ["docker", "inspect", unrelated_container], capture_output=True
        )
        assert container_status.returncode != 0
