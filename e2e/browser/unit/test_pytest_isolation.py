import os
import subprocess
import sys
import time
from pathlib import Path

from public_command import REPOSITORY_ROOT


class TestPytestIsolation:
    def test_keeps_a_live_process_temporary_files_when_another_pytest_process_finishes(self, tmp_path: Path):
        ready_marker = tmp_path / "ready"
        release_marker = tmp_path / "release"
        child_test = tmp_path / "test_temporary_files.py"
        child_test.write_text('''
import os
import time
from pathlib import Path


def test_temporary_files(tmp_path):
    temporary_file = tmp_path / "owned"
    temporary_file.write_text("preserve while running")

    if os.environ.get("HOLD_TEMPORARY_FILES"):
        Path(os.environ["READY_MARKER"]).write_text(str(temporary_file))
        deadline = time.monotonic() + 30

        while not Path(os.environ["RELEASE_MARKER"]).exists():
            assert time.monotonic() < deadline
            time.sleep(0.05)

    assert temporary_file.read_text() == "preserve while running"
''')
        command = [
            sys.executable, "-m", "pytest", "-p", "conftest", "-p", "no:cacheprovider",
            "--tb=short", str(child_test),
        ]
        environment = {**os.environ, "PYTHONDONTWRITEBYTECODE": "1"}
        held_process = subprocess.Popen(
            command,
            cwd=REPOSITORY_ROOT / "e2e/browser",
            env={
                **environment,
                "HOLD_TEMPORARY_FILES": "1",
                "READY_MARKER": str(ready_marker),
                "RELEASE_MARKER": str(release_marker),
            },
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )

        try:
            deadline = time.monotonic() + 30

            while not ready_marker.exists():
                assert held_process.poll() is None, held_process.communicate()[0]
                assert time.monotonic() < deadline, "the held pytest process did not create its temporary file"
                time.sleep(0.05)

            other_process = subprocess.run(
                command, cwd=REPOSITORY_ROOT / "e2e/browser", env=environment,
                capture_output=True, text=True, timeout=30,
            )

            assert other_process.returncode == 0, other_process.stdout + other_process.stderr
            assert Path(ready_marker.read_text()).exists()
            release_marker.touch()
            held_output = held_process.communicate(timeout=30)[0]

            assert held_process.returncode == 0, held_output
        finally:
            release_marker.touch()

            try:
                held_process.communicate(timeout=30)
            except subprocess.TimeoutExpired:
                held_process.kill()
                held_process.communicate()
