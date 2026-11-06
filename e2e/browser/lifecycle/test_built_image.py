import os
import re
import signal
from pathlib import Path

import pytest
from docker_wrapper import install_docker_wrapper_and_get_path
from leftover_check import assert_nothing_of_the_project_remains, is_image_present
from lifecycle.run_cleanup import RunCleanup
from probe_run import assert_sweep_came_before
from public_command import PublicRun, get_running_image
from supervisor_file import get_run_directory

INTERRUPT_READY = "E2E_INTERRUPT_READY"

pytestmark = pytest.mark.skipif(
    bool(os.environ.get("DATABASUS_IMAGE")),
    reason="needs a run that builds its own image, and CI builds none",
)


@pytest.fixture(autouse=True)
def no_named_image(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("DATABASUS_IMAGE", raising=False)


def _read_built_image(output: str) -> str:
    matched_output = re.search(r"E2E image under test: (\S+)", output)
    assert matched_output is not None

    return matched_output.group(1)


class TestRunBuildingItsOwnImage:
    def test_builds_an_image_unique_to_the_run_and_removes_it_afterwards(
        self, register_run_cleanup: RunCleanup
    ) -> None:
        run = PublicRun(environment={"E2E_PROBE": "interrupt"})
        register_run_cleanup(run)
        run.wait_for_output(INTERRUPT_READY)
        built_image = _read_built_image(run.get_output())

        assert get_running_image(run.get_project_name(), "databasus") == built_image

        run.terminate_supervisor()
        finished_run = run.finish()

        assert finished_run.exit_code != 0
        assert "E2E run interrupted" in finished_run.output
        assert "E2E building" in finished_run.output
        assert run.get_run_id() in built_image
        assert not is_image_present(built_image)
        assert_nothing_of_the_project_remains(run.get_project_name())

    def test_fails_and_names_the_image_and_keeps_its_directory_when_the_image_cannot_be_removed(
        self, tmp_path: Path, register_run_cleanup: RunCleanup
    ) -> None:
        path_with_refusal = install_docker_wrapper_and_get_path(
            tmp_path, '*" image rm "*) echo "removal refused"; exit 1;;'
        )

        run = PublicRun(environment={"E2E_PROBE": "passing", "PATH": path_with_refusal})
        register_run_cleanup(run)
        finished_run = run.finish()
        built_image = _read_built_image(finished_run.output)

        assert finished_run.exit_code != 0
        assert "1 passed" in finished_run.output, finished_run.output
        assert f"E2E cleanup failed, remaining: image {built_image}" in finished_run.output
        assert "probe-cleanup-databases" in finished_run.output
        assert "probe-cleanup-application" in finished_run.output
        assert "synthetic-bearer-marker" not in finished_run.output
        assert is_image_present(built_image)
        assert get_run_directory(run.get_run_id()).exists()

    def test_a_later_run_removes_the_built_image_of_a_killed_run(
        self, register_run_cleanup: RunCleanup
    ) -> None:
        killed_run = PublicRun(environment={"E2E_PROBE": "interrupt"})
        register_run_cleanup(killed_run)
        killed_run.wait_for_output(INTERRUPT_READY)
        built_image = _read_built_image(killed_run.get_output())
        killed_run.send_signal_to_the_whole_command(signal.SIGKILL)
        killed_run.finish()

        later_run = PublicRun(environment={"E2E_PROBE": "scenario"}).finish()

        assert_sweep_came_before(later_run.output, killed_run.get_run_id(), "E2E building")
        assert not is_image_present(built_image)
        assert_nothing_of_the_project_remains(killed_run.get_project_name())
        assert not get_run_directory(killed_run.get_run_id()).exists()
