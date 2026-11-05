import os
import subprocess
import sys
from pathlib import Path

from public_command import REPOSITORY_ROOT


class TestLifecycleCleanup:
    def test_stops_a_public_run_when_a_probe_fails_without_requesting_the_shared_image(self, tmp_path: Path):
        child_test = tmp_path / "test_failed_probe.py"
        child_test.write_text('''
import os
import signal
import sys
from pathlib import Path

import pytest

from public_command import PublicRun

pytest_plugins = ["lifecycle.conftest"]
process_file = Path(__file__).with_suffix(".pid")


@pytest.fixture(autouse=True)
def unrelated_container_keeps_answering():
    yield


def test_deliberate_probe_failure():
    command = (
        "import os, signal; "
        "print(f'E2E run: child; project: databasus-e2e-child-{os.getpid()}; "
        "supervisor PID: {os.getpid()}', flush=True); "
        "signal.pause()"
    )
    run = PublicRun(command=(sys.executable, "-c", command))
    run.wait_for_output("supervisor PID: ")
    process_file.write_text(str(run.get_supervisor_pid()))

    pytest.fail("deliberate probe failure")


def test_next_probe_sees_no_process_from_the_failed_probe():
    with pytest.raises(ProcessLookupError, match="No such process"):
        os.kill(int(process_file.read_text()), 0)
''')

        child_environment = {**os.environ, "PYTHONDONTWRITEBYTECODE": "1"}
        child_environment.pop("CI", None)

        child_run = subprocess.run(
            [
                sys.executable, "-m", "pytest", "--tb=short", "-p", "no:cacheprovider",
                "--basetemp", str(tmp_path / "child-pytest"), str(child_test),
            ],
            cwd=REPOSITORY_ROOT / "e2e/browser",
            env=child_environment,
            capture_output=True,
            text=True,
            timeout=60,
        )

        assert child_run.returncode == 1, child_run.stdout + child_run.stderr
        assert "deliberate probe failure" in child_run.stdout
        assert "1 failed, 1 passed" in child_run.stdout, child_run.stdout + child_run.stderr
