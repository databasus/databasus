import subprocess
from collections.abc import Callable
from dataclasses import dataclass

from compose_project import PROJECT_NAME_PREFIX, list_project_names, list_remaining_resources, remove_dead_run_project
from supervisor_file import (
    SupervisorRecord,
    get_project_name,
    is_supervisor_alive,
    read_supervisor_records,
    remove_run_directory,
)


@dataclass(frozen=True)
class DeadRun:
    run_id: str
    built_image: str | None
    has_record: bool


@dataclass(frozen=True)
class KnownRuns:
    records: list[SupervisorRecord]
    project_names: list[str]


def read_known_runs() -> KnownRuns:
    """A run writes its record before its project exists, so projects are listed first."""
    project_names = list_project_names()

    return KnownRuns(read_supervisor_records(), project_names)


def select_dead_runs(
    known_runs: KnownRuns,
    is_alive: Callable[[SupervisorRecord], bool],
    own_run_id: str,
) -> list[DeadRun]:
    other_records = [record for record in known_runs.records if record.run_id != own_run_id]
    recorded_projects = {record.get_project_name() for record in known_runs.records}
    dead_runs = [
        DeadRun(run_id=record.run_id, built_image=record.built_image, has_record=True)
        for record in other_records
        if not is_alive(record)
    ]
    dead_runs += [
        DeadRun(run_id=project_name.removeprefix(PROJECT_NAME_PREFIX), built_image=None, has_record=False)
        for project_name in known_runs.project_names
        if project_name not in recorded_projects
    ]

    return dead_runs


def sweep_dead_runs(own_run_id: str, report: Callable[[str], None]) -> None:
    dead_runs = select_dead_runs(read_known_runs(), is_supervisor_alive, own_run_id)

    for dead_run in dead_runs:
        report(f"E2E removing what crashed run {dead_run.run_id} left behind")

        project_name = get_project_name(dead_run.run_id)
        is_project_removed = remove_dead_run_project(project_name)
        remaining_resources = list_remaining_resources(project_name)

        if not is_project_removed or remaining_resources:
            remaining_description = ", ".join(remaining_resources) or f"project {project_name}"
            raise RuntimeError(f"E2E could not remove crashed run {dead_run.run_id}: {remaining_description}")

        if dead_run.built_image and not _remove_image(dead_run.built_image):
            raise RuntimeError(f"E2E could not remove crashed run {dead_run.run_id}: image {dead_run.built_image}")

        if dead_run.has_record:
            remove_run_directory(dead_run.run_id)


def _remove_image(image: str) -> bool:
    completed = subprocess.run(["docker", "image", "rm", image], capture_output=True, text=True)

    return completed.returncode == 0 or "No such image" in completed.stderr
