import json
import os
import shutil
from dataclasses import asdict, dataclass
from pathlib import Path

SUPERVISORS_DIRECTORY = Path.home() / ".cache" / "databasus-e2e"
SUPERVISOR_FILE_NAME = "supervisor.json"
UNREADABLE_PID = 0
UNREADABLE_START_TIME = "unreadable"


@dataclass(frozen=True)
class SupervisorRecord:
    run_id: str
    pid: int
    process_start_time: str
    built_image: str | None

    def get_project_name(self) -> str:
        return get_project_name(self.run_id)


def get_project_name(run_id: str) -> str:
    return f"databasus-e2e-{run_id}"


def get_run_directory(run_id: str) -> Path:
    return SUPERVISORS_DIRECTORY / run_id


def write_supervisor_record(record: SupervisorRecord) -> None:
    run_directory = get_run_directory(record.run_id)
    run_directory.mkdir(parents=True, exist_ok=True)
    temporary_file = run_directory / f"{SUPERVISOR_FILE_NAME}.tmp"
    temporary_file.write_text(json.dumps(asdict(record)))
    os.replace(temporary_file, run_directory / SUPERVISOR_FILE_NAME)


def read_supervisor_records() -> list[SupervisorRecord]:
    if not SUPERVISORS_DIRECTORY.exists():
        return []

    records = []

    for supervisor_file in SUPERVISORS_DIRECTORY.glob(f"*/{SUPERVISOR_FILE_NAME}"):
        try:
            records.append(SupervisorRecord(**json.loads(supervisor_file.read_text())))
        except FileNotFoundError:
            continue
        except (ValueError, TypeError):
            records.append(_get_record_of_unreadable_file(supervisor_file))

    return records


def _get_record_of_unreadable_file(supervisor_file: Path) -> SupervisorRecord:
    """A writer replaces the file atomically, so damage means no live run owns it and the sweep may remove it."""
    return SupervisorRecord(supervisor_file.parent.name, UNREADABLE_PID, UNREADABLE_START_TIME, None)


def remove_run_directory(run_id: str) -> None:
    try:
        shutil.rmtree(get_run_directory(run_id))
    except FileNotFoundError:
        return


def get_process_start_time(pid: int) -> str | None:
    try:
        process_stat = Path(f"/proc/{pid}/stat").read_text()
    except OSError:
        return None

    fields_after_command = process_stat.rpartition(")")[2].split()

    if fields_after_command[0] in {"Z", "X"}:
        return None

    return fields_after_command[19]


def is_supervisor_alive(record: SupervisorRecord) -> bool:
    return get_process_start_time(record.pid) == record.process_start_time
