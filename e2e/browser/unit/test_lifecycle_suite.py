import os
import shutil
import subprocess
import sys
from pathlib import Path

import pytest
from lifecycle_suite import GROUPS, get_lifecycle_paths

SUITE = Path(__file__).resolve().parents[1]


def collect_cases(paths: list[Path], environment: dict[str, str]) -> set[str]:
    collected = subprocess.run(
        [sys.executable, "-m", "pytest", "--collect-only", "-q", *(str(path) for path in paths)],
        cwd=SUITE,
        env=environment,
        capture_output=True,
        text=True,
        timeout=30,
    )
    assert collected.returncode == 0, collected.stdout + collected.stderr
    return {line for line in collected.stdout.splitlines() if "::" in line}


class TestLifecycleSelection:
    @pytest.mark.parametrize("image", ["", "candidate:local"])
    def test_covers_all_collected_cases_once_with_matching_image_conditions(
        self, image: str
    ) -> None:
        environment = {**os.environ, "CI": "", "DATABASUS_IMAGE": image}
        complete = collect_cases(get_lifecycle_paths(SUITE / "lifecycle", None), environment)
        selections = [
            collect_cases(get_lifecycle_paths(SUITE / "lifecycle", group), environment)
            for group in GROUPS
        ]

        assert complete
        assert set.union(*selections) == complete
        assert sum(map(len, selections)) == len(complete)

    def test_discovers_new_files_and_tests_without_group_configuration(
        self, tmp_path: Path
    ) -> None:
        (tmp_path / "test_failing_runs.py").write_text(
            "def test_failure(): pass\ndef test_new_failure(): pass\n"
        )
        (tmp_path / "test_crashed_runs.py").write_text("def test_crash(): pass\n")
        (tmp_path / "test_new_probe.py").write_text("def test_new_probe(): pass\n")
        nested = tmp_path / "nested"
        nested.mkdir()
        (nested / "new_nested_test.py").write_text("def test_nested_probe(): pass\n")
        environment = {**os.environ, "CI": ""}
        complete = collect_cases([tmp_path], environment)
        selections = [
            collect_cases(get_lifecycle_paths(tmp_path, group), environment) for group in GROUPS
        ]

        assert len(complete) == 5
        assert set.union(*selections) == complete
        assert sum(map(len, selections)) == 5
        assert set(get_lifecycle_paths(tmp_path, "isolation")) == {
            tmp_path / "test_new_probe.py",
            nested / "new_nested_test.py",
        }

    def test_rejects_unknown_and_empty_groups(self, tmp_path: Path) -> None:
        with pytest.raises(ValueError, match="Unknown lifecycle group"):
            get_lifecycle_paths(tmp_path, "unknown")
        with pytest.raises(ValueError, match="has no test files"):
            get_lifecycle_paths(tmp_path, "failures")

    @pytest.mark.parametrize(
        "group,status", [("failures", 1), ("interruption", 5), ("isolation", 0), (None, 1)]
    )
    def test_preserves_environment_exit_status_and_empty_collection_failure(
        self, tmp_path: Path, group: str | None, status: int
    ) -> None:
        shutil.copy(SUITE / "lifecycle_suite.py", tmp_path)
        lifecycle = tmp_path / "lifecycle"
        lifecycle.mkdir()
        (lifecycle / "test_failing_runs.py").write_text("def test_failure(): assert False\n")
        (lifecycle / "test_interruption.py").write_text("")
        (lifecycle / "test_environment.py").write_text(
            'import os\ndef test_environment(): assert os.environ["LIFECYCLE_SENTINEL"] == "unchanged"\n'
        )
        execution = subprocess.run(
            [sys.executable, str(tmp_path / "lifecycle_suite.py"), *([group] if group else [])],
            cwd=tmp_path,
            env={**os.environ, "LIFECYCLE_SENTINEL": "unchanged"},
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert execution.returncode == status, execution.stdout + execution.stderr

    def test_rejects_unknown_group_before_pytest_or_docker(self) -> None:
        execution = subprocess.run(
            [sys.executable, str(SUITE / "lifecycle_suite.py"), "unknown"],
            env={**os.environ, "PATH": ""},
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert execution.returncode == 2
        assert "invalid choice" in execution.stderr
