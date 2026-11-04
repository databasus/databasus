import signal
import sys
from pathlib import Path

from leftover_check import assert_nothing_of_the_project_remains, is_image_present
from public_command import PublicRun
from supervisor_file import get_run_directory


class TestProbeCleanup:
    def test_removes_the_project_image_and_directory_when_a_recovery_probe_fails(
        self, probes_image: str, tmp_path: Path, register_run_cleanup
    ):
        child_test = tmp_path / "test_failed_recovery.py"
        release_marker = tmp_path / "release"
        child_test.write_text('''
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
''')
        child_run = PublicRun(
            command=(
                sys.executable, "-m", "pytest", "-p", "lifecycle.conftest", "--capture=no",
                "-c", str(Path(__file__).parents[1] / "pyproject.toml"),
                "--basetemp", str(tmp_path / "child-pytest"), str(child_test),
            ),
            environment={
                "FAILED_PROBE_IMAGE": probes_image,
                "FAILED_PROBE_RELEASE": str(release_marker),
                "PYTEST_ADDOPTS": "",
                "PYTHONDONTWRITEBYTECODE": "1",
            },
            working_directory=Path(__file__).parents[1],
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
