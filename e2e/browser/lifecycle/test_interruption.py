import re
import signal
import subprocess
from pathlib import Path

from leftover_check import assert_nothing_of_the_project_remains
from lifecycle.run_cleanup import RunCleanup
from probe_run import INTERRUPT_READY, assert_failed_and_released, start_probe_run
from supervisor_file import get_run_directory


class TestInterruption:
    def test_exits_non_zero_and_removes_everything_when_a_run_is_terminated(
        self, probes_image: str, register_run_cleanup: RunCleanup
    ) -> None:
        run = start_probe_run(probes_image, "interrupt")
        register_run_cleanup(run)
        run.wait_for_output(INTERRUPT_READY)
        matched_output = re.search(r"E2E_PYTEST_TEMP_ROOT: (\S+)", run.get_output())
        assert matched_output is not None

        pytest_directory = Path(matched_output.group(1))

        assert pytest_directory.exists()
        run.terminate_supervisor()

        assert_failed_and_released(run, "E2E run interrupted")
        assert not pytest_directory.exists(), f"child pytest directory remains: {pytest_directory}"

    def test_publishes_a_port_only_for_the_browser_service(self, probes_image: str) -> None:
        run = start_probe_run(probes_image, "interrupt")
        run.wait_for_output(INTERRUPT_READY)

        assert _list_services_publishing_ports(run.get_project_name()) == ["browser"]

        run.terminate_supervisor()

        assert_failed_and_released(run, "E2E run interrupted")

    def test_removes_everything_when_a_terminal_interrupt_also_ends_the_relaying_command(
        self, probes_image: str, register_run_cleanup: RunCleanup
    ) -> None:
        run = start_probe_run(probes_image, "interrupt")
        register_run_cleanup(run)
        run.wait_for_output(INTERRUPT_READY)
        project_name = run.get_project_name()
        matched_output = re.search(r"E2E_PYTEST_TEMP_ROOT: (\S+)", run.get_output())
        assert matched_output is not None

        pytest_directory = Path(matched_output.group(1))

        assert pytest_directory.exists()

        run.send_signal_to_the_whole_command(signal.SIGINT)

        assert run.finish().exit_code != 0
        assert_nothing_of_the_project_remains(project_name)
        assert not get_run_directory(run.get_run_id()).exists()
        assert not pytest_directory.exists(), f"child pytest directory remains: {pytest_directory}"


def _list_services_publishing_ports(project_name: str) -> list[str]:
    listing = subprocess.run(
        [
            "docker",
            "ps",
            "--filter",
            f"label=com.docker.compose.project={project_name}",
            "--format",
            '{{.Label "com.docker.compose.service"}} {{.Ports}}',
        ],
        capture_output=True,
        text=True,
        check=True,
    )

    return [line.split()[0] for line in listing.stdout.splitlines() if "->" in line]
