import signal
import subprocess

from leftover_check import assert_nothing_of_the_project_remains
from probe_run import INTERRUPT_READY, assert_failed_and_released, start_probe_run
from supervisor_file import get_run_directory


class TestInterruption:
    def test_exits_non_zero_and_removes_everything_when_a_run_is_terminated(self, probes_image: str):
        run = start_probe_run(probes_image, "interrupt")
        run.wait_for_output(INTERRUPT_READY)
        run.terminate_supervisor()

        assert_failed_and_released(run, "E2E run interrupted")

    def test_publishes_a_port_only_for_the_browser_service(self, probes_image: str):
        run = start_probe_run(probes_image, "interrupt")
        run.wait_for_output(INTERRUPT_READY)

        assert _list_services_publishing_ports(run.get_project_name()) == ["browser"]

        run.terminate_supervisor()

        assert_failed_and_released(run, "E2E run interrupted")

    def test_removes_everything_when_a_terminal_interrupt_also_ends_the_relaying_command(self, probes_image: str):
        run = start_probe_run(probes_image, "interrupt")
        run.wait_for_output(INTERRUPT_READY)
        project_name = run.get_project_name()

        run.send_signal_to_the_whole_command(signal.SIGINT)

        assert run.finish().exit_code != 0
        assert_nothing_of_the_project_remains(project_name)
        assert not get_run_directory(run.get_run_id()).exists()


def _list_services_publishing_ports(project_name: str) -> list[str]:
    listing = subprocess.run(
        [
            "docker", "ps", "--filter", f"label=com.docker.compose.project={project_name}",
            "--format", '{{.Label "com.docker.compose.service"}} {{.Ports}}',
        ],
        capture_output=True,
        text=True,
        check=True,
    )

    return [line.split()[0] for line in listing.stdout.splitlines() if "->" in line]
