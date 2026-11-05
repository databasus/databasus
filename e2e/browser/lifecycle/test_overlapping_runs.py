import re
import subprocess

from leftover_check import assert_nothing_of_the_project_remains
from lifecycle.recovery_resources import assert_private_recovery_is_usable
from probe_run import INTERRUPT_READY, start_probe_run
from supervisor_file import get_run_directory


class TestOverlappingRuns:
    def test_recovery_resources_keep_answering_after_an_overlapping_run_cleans_up(
        self, probes_image: str
    ) -> None:
        first_run = start_probe_run(probes_image, "interrupt")
        second_run = start_probe_run(probes_image, "interrupt")
        first_run.wait_for_output(INTERRUPT_READY)
        second_run.wait_for_output(INTERRUPT_READY)
        assert_private_recovery_is_usable(first_run.get_project_name())
        assert_private_recovery_is_usable(second_run.get_project_name())

        first_run.terminate_supervisor()
        assert first_run.finish().exit_code != 0

        assert_nothing_of_the_project_remains(first_run.get_project_name())
        assert second_run.is_running()
        assert_private_recovery_is_usable(second_run.get_project_name())

        second_run.terminate_supervisor()
        assert second_run.finish().exit_code != 0

        assert_nothing_of_the_project_remains(second_run.get_project_name())

    def test_both_pass_with_their_own_accounts_projects_ports_volumes_and_directories(
        self, probes_image: str
    ) -> None:
        first_run = start_probe_run(probes_image)
        second_run = start_probe_run(probes_image)
        first_run.wait_for_output("browser port: ")
        second_run.wait_for_output("browser port: ")
        first_volumes = _list_project_volumes(first_run.get_project_name())
        second_volumes = _list_project_volumes(second_run.get_project_name())

        assert first_volumes and second_volumes
        assert not first_volumes & second_volumes
        assert get_run_directory(first_run.get_run_id()).is_dir()
        assert get_run_directory(second_run.get_run_id()).is_dir()

        finished_runs = [first_run.finish(), second_run.finish()]

        for finished_run in finished_runs:
            assert finished_run.exit_code == 0, finished_run.output

        first_identity = _read_run_identity(finished_runs[0].output)
        second_identity = _read_run_identity(finished_runs[1].output)

        assert first_identity["project"] != second_identity["project"]
        assert first_identity["port"] != second_identity["port"]
        assert first_identity["account"] != second_identity["account"]
        assert_nothing_of_the_project_remains(first_identity["project"])
        assert_nothing_of_the_project_remains(second_identity["project"])
        assert not get_run_directory(first_run.get_run_id()).exists()
        assert not get_run_directory(second_run.get_run_id()).exists()


def _list_project_volumes(project_name: str) -> set[str]:
    completed = subprocess.run(
        [
            "docker",
            "volume",
            "ls",
            "--format",
            "{{.Name}}",
            "--filter",
            f"label=com.docker.compose.project={project_name}",
        ],
        capture_output=True,
        text=True,
        check=True,
    )

    return set(completed.stdout.split())


def _read_run_identity(output: str) -> dict[str, str]:
    matched_account = re.search(r"account: (\S+)", output)
    assert matched_account is not None

    matched_port = re.search(r"browser port: (\d+);", output)
    assert matched_port is not None

    matched_project = re.search(r"project: (\S+?);", output)
    assert matched_project is not None

    return {
        "project": matched_project.group(1),
        "port": matched_port.group(1),
        "account": matched_account.group(1),
    }
