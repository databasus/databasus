import signal
import subprocess
from dataclasses import replace
from pathlib import Path

import pytest

from compose_project import list_remaining_resources
from docker_wrapper import install_docker_wrapper_and_get_path
from leftover_check import assert_nothing_of_the_project_remains, is_image_present
from probe_run import INTERRUPT_READY, assert_sweep_came_before, start_probe_run
from public_command import PublicRun
from supervisor_file import get_run_directory, read_supervisor_records, write_supervisor_record


def _kill_run(run: PublicRun) -> None:
    run.send_signal_to_the_whole_command(signal.SIGKILL)
    run.finish()


class TestCrashedRuns:
    @pytest.mark.parametrize("blocked_resource", ["project", "image"])
    def test_stops_before_startup_when_crashed_resources_cannot_be_removed(
        self, probes_image: str, tmp_path: Path, blocked_resource: str, register_run_cleanup
    ):
        killed_run = start_probe_run(probes_image, "interrupt")
        project_name = killed_run.get_project_name()
        owned_image = f"{project_name}:sweep-probe"

        register_run_cleanup(killed_run, built_image=owned_image)
        killed_run.wait_for_output(INTERRUPT_READY)
        subprocess.run(["docker", "image", "tag", probes_image, owned_image], check=True)
        record = next(record for record in read_supervisor_records() if record.run_id == killed_run.get_run_id())
        _kill_run(killed_run)
        write_supervisor_record(replace(record, built_image=owned_image))

        refusal = (
            f'*" --project-name {project_name} "*" down "*) echo "sweep removal refused"; exit 1;;'
            if blocked_resource == "project"
            else f'*" image rm {owned_image} "*) echo "sweep removal refused"; exit 1;;'
        )
        path_with_refusal = install_docker_wrapper_and_get_path(tmp_path, refusal)

        later_run = start_probe_run(probes_image, "passing", PATH=path_with_refusal)
        finished_run = later_run.finish()

        assert finished_run.exit_code != 0, finished_run.output
        assert "E2E could not remove crashed run" in finished_run.output
        assert (project_name if blocked_resource == "project" else owned_image) in finished_run.output
        assert "E2E browser port:" not in finished_run.output
        assert "E2E building " not in finished_run.output
        assert get_run_directory(killed_run.get_run_id()).exists()
        assert is_image_present(owned_image)
        assert_nothing_of_the_project_remains(later_run.get_project_name())

    def test_a_later_run_removes_a_killed_run_and_leaves_a_live_one(self, probes_image: str, register_run_cleanup):
        live_run = start_probe_run(probes_image, "interrupt")
        live_run.wait_for_output(INTERRUPT_READY)

        killed_run = start_probe_run(probes_image, "interrupt")
        register_run_cleanup(killed_run)
        killed_run.wait_for_output(INTERRUPT_READY)
        killed_project_name = killed_run.get_project_name()
        _kill_run(killed_run)

        later_run = start_probe_run(probes_image, "scenario").finish()

        assert_sweep_came_before(later_run.output, killed_run.get_run_id(), "E2E browser port")
        assert_nothing_of_the_project_remains(killed_project_name)
        assert not get_run_directory(killed_run.get_run_id()).exists()
        assert live_run.is_running()
        assert list_remaining_resources(live_run.get_project_name())

        live_run.terminate_supervisor()

        assert live_run.finish().exit_code != 0
        assert_nothing_of_the_project_remains(live_run.get_project_name())
