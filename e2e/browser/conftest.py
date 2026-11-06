import os
import secrets

import pytest
from supervisor_file import (
    SupervisorRecord,
    get_required_process_start_time,
    get_run_directory,
    remove_run_directory,
    write_supervisor_record,
)


@pytest.hookimpl(tryfirst=True)
def pytest_configure(config: pytest.Config) -> None:
    """With retention disabled, pytest cleanup needs a separate root to leave other live sessions alone."""
    if config.option.basetemp is not None:
        return

    run_id = secrets.token_hex(6)
    write_supervisor_record(
        SupervisorRecord(run_id, os.getpid(), get_required_process_start_time(os.getpid()), None)
    )
    config.option.basetemp = str(get_run_directory(run_id) / "pytest")
    config.add_cleanup(lambda: remove_run_directory(run_id))
