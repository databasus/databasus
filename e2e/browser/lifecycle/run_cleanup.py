import signal
import subprocess
from dataclasses import replace
from typing import Protocol

import pytest
from compose_project import list_remaining_resources, remove_dead_run_project
from public_command import PublicRun
from supervisor_file import SupervisorRecord, read_supervisor_records, remove_run_directory


class RunCleanup(Protocol):
    def __call__(
        self,
        run: PublicRun,
        built_image: str | None = None,
        *,
        record: SupervisorRecord | None = None,
        unrelated_container: str | None = None,
        should_stop_whole_command: bool = False,
    ) -> None: ...


@pytest.fixture
def register_run_cleanup(request: pytest.FixtureRequest) -> RunCleanup:
    """The one way a probe registers the teardown of a run it kills or keeps from cleaning up.
    Requesting it makes the probe a crashed-run probe, which the lifecycle command can run apart."""

    def register(
        run: PublicRun,
        built_image: str | None = None,
        *,
        record: SupervisorRecord | None = None,
        unrelated_container: str | None = None,
        should_stop_whole_command: bool = False,
    ) -> None:
        if record is None:
            # Reading the run ID waits until the supervisor has written its record.
            run_id = run.get_run_id()
            record = next(record for record in read_supervisor_records() if record.run_id == run_id)

        run_record = record

        if built_image is not None:
            run_record = replace(run_record, built_image=built_image)

        def clean_up() -> None:
            try:
                if should_stop_whole_command:
                    run.send_signal_to_the_whole_command(signal.SIGTERM)
                    run.finish()
            finally:
                remove_run_resources(run, run_record, unrelated_container=unrelated_container)

        request.addfinalizer(clean_up)

    return register


def remove_run_resources(
    run: PublicRun, record: SupervisorRecord, *, unrelated_container: str | None = None
) -> None:
    try:
        if run.is_running():
            run.terminate_supervisor()

        run.finish()
    finally:
        project_name = record.get_project_name()
        is_project_removed = remove_dead_run_project(project_name)
        remaining_resources = list_remaining_resources(project_name)

        if not is_project_removed and not remaining_resources:
            remaining_resources.append(f"project {project_name}")

        if unrelated_container:
            container_removal = subprocess.run(
                ["docker", "rm", "--force", "--volumes", unrelated_container],
                capture_output=True,
                text=True,
            )

            if (
                container_removal.returncode != 0
                and "No such container" not in container_removal.stderr
            ):
                remaining_resources.append(f"container {unrelated_container}")

        if record.built_image:
            image_removal = subprocess.run(
                ["docker", "image", "rm", record.built_image], capture_output=True, text=True
            )

            if image_removal.returncode != 0 and "No such image" not in image_removal.stderr:
                remaining_resources.append(f"image {record.built_image}")

        if not remaining_resources:
            remove_run_directory(record.run_id)

        assert not remaining_resources, (
            f"probe cleanup failed, remaining: {', '.join(remaining_resources)}"
        )
