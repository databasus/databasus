from leftover_check import assert_nothing_of_the_project_remains
from public_command import PublicRun
from supervisor_file import get_run_directory

INTERRUPT_READY = "E2E_INTERRUPT_READY"


def start_probe_run(image: str, probe: str = "", **environment: str) -> PublicRun:
    return PublicRun(environment={"DATABASUS_IMAGE": image, "E2E_PROBE": probe, **environment})


def assert_failed_and_released(run: PublicRun, diagnostic: str) -> str:
    project_name = run.get_project_name()
    finished_run = run.finish()

    assert finished_run.exit_code != 0, finished_run.output
    assert diagnostic in finished_run.output, finished_run.output
    assert_nothing_of_the_project_remains(project_name)
    assert not get_run_directory(run.get_run_id()).exists()

    return finished_run.output


def assert_sweep_came_before(output: str, crashed_run_id: str, start_marker: str) -> None:
    """A run from another worktree may have swept the crashed run first, and then this run prints no removal."""
    assert start_marker in output, output

    removal_marker = f"E2E removing what crashed run {crashed_run_id} left behind"

    if removal_marker in output:
        assert output.index(removal_marker) < output.index(start_marker), output
