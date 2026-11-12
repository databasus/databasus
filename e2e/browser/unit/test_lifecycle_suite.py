import ast
import os
import shutil
import subprocess
import sys
from pathlib import Path

import pytest
from lifecycle_suite import (
    GROUPS,
    get_lifecycle_paths,
    get_marker_arguments,
    get_shard_arguments,
    parse_shard,
)

SUITE = Path(__file__).resolve().parents[1]
NO_TESTS_COLLECTED = 5
TEARDOWN_FIXTURE = "register_run_cleanup"
# A probe that does any of these leaves a crashed run behind for a later run to remove: it kills a run
# or its recorded supervisor, cleans up after one itself, or makes a removal fail.
CRASHED_RUN_SIGNS = (
    "SIGKILL",
    "write_supervisor_record",
    "addfinalizer",
    "remove_run_resources",
    "removal refused",
)


def collect_cases(
    paths: list[Path], environment: dict[str, str], extra_arguments: tuple[str, ...] = ()
) -> set[str]:
    collected = subprocess.run(
        [
            sys.executable,
            "-m",
            "pytest",
            "--collect-only",
            "-q",
            *extra_arguments,
            *(str(path) for path in paths),
        ],
        cwd=SUITE,
        env=environment,
        capture_output=True,
        text=True,
        timeout=30,
    )
    assert collected.returncode in (0, NO_TESTS_COLLECTED), collected.stdout + collected.stderr
    return {line for line in collected.stdout.splitlines() if "::" in line}


def _has_crashed_run_sign(function: ast.FunctionDef, sign_helpers: set[str]) -> bool:
    for statement in function.body:
        if any(sign in ast.unparse(statement) for sign in CRASHED_RUN_SIGNS):
            return True
        for node in ast.walk(statement):
            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name):
                if node.func.id in sign_helpers:
                    return True
    return False


def _find_sign_helpers(module: ast.Module) -> set[str]:
    """A helper that sends a sign itself, or calls another helper that does, hides it from a probe."""
    helpers = [
        node
        for node in module.body
        if isinstance(node, ast.FunctionDef) and not node.name.startswith("test_")
    ]
    sign_helpers: set[str] = set()
    is_growing = True
    while is_growing:
        found = {helper.name for helper in helpers if _has_crashed_run_sign(helper, sign_helpers)}
        is_growing = not found <= sign_helpers
        sign_helpers |= found
    return sign_helpers


def find_probes_without_teardown_fixture(source: str) -> list[str]:
    module = ast.parse(source)
    sign_helpers = _find_sign_helpers(module)
    probes = [
        node
        for node in ast.walk(module)
        if isinstance(node, ast.FunctionDef) and node.name.startswith("test_")
    ]

    return [
        probe.name
        for probe in probes
        if _has_crashed_run_sign(probe, sign_helpers)
        and TEARDOWN_FIXTURE not in [argument.arg for argument in probe.args.args]
    ]


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


class TestCrashedRunSelection:
    @pytest.mark.parametrize("group", [*GROUPS, None])
    def test_the_two_selections_cover_each_group_once(self, group: str | None) -> None:
        environment = {**os.environ, "CI": "", "DATABASUS_IMAGE": ""}
        paths = get_lifecycle_paths(SUITE / "lifecycle", group)
        complete = collect_cases(paths, environment)
        crashed_run_probes = collect_cases(
            paths, environment, tuple(get_marker_arguments(True, False))
        )
        other_probes = collect_cases(paths, environment, tuple(get_marker_arguments(False, True)))

        assert complete
        assert crashed_run_probes | other_probes == complete
        assert not crashed_run_probes & other_probes

    def test_a_probe_requesting_the_teardown_fixture_is_a_crashed_run_probe(
        self, tmp_path: Path
    ) -> None:
        (tmp_path / "test_new_probes.py").write_text(
            f"def test_kills_a_run({TEARDOWN_FIXTURE}): pass\ndef test_overlaps_runs(): pass\n"
        )
        environment = {**os.environ, "CI": ""}
        plugin = ("-p", "lifecycle.conftest")

        crashed_run_probes = collect_cases(
            [tmp_path], environment, (*plugin, *get_marker_arguments(True, False))
        )
        other_probes = collect_cases(
            [tmp_path], environment, (*plugin, *get_marker_arguments(False, True))
        )

        assert [case.rpartition("::")[2] for case in crashed_run_probes] == ["test_kills_a_run"]
        assert [case.rpartition("::")[2] for case in other_probes] == ["test_overlaps_runs"]

    def test_every_lifecycle_probe_that_leaves_a_crashed_run_requests_the_teardown_fixture(
        self,
    ) -> None:
        probe_files = sorted((SUITE / "lifecycle").glob("test_*.py"))
        missing_fixture = {
            probe_file.name: find_probes_without_teardown_fixture(probe_file.read_text())
            for probe_file in probe_files
        }

        assert probe_files
        assert {name: probes for name, probes in missing_fixture.items() if probes} == {}

    @pytest.mark.parametrize("sign", CRASHED_RUN_SIGNS)
    def test_the_check_names_a_probe_that_leaves_a_crashed_run_without_the_fixture(
        self, sign: str
    ) -> None:
        source = f"def test_kills_a_run(probes_image):\n    use({sign!r})\n"

        assert find_probes_without_teardown_fixture(source) == ["test_kills_a_run"]

    def test_the_check_names_a_probe_whose_module_helper_leaves_a_crashed_run(self) -> None:
        source = (
            "def _kill_run(run):\n"
            "    run.send_signal_to_the_whole_command(signal.SIGKILL)\n"
            "def _kill_and_wait(run):\n"
            "    _kill_run(run)\n"
            "def test_kills_a_run(probes_image):\n"
            "    _kill_and_wait(start(probes_image))\n"
            "def test_kills_a_run_with_teardown(probes_image, register_run_cleanup):\n"
            "    _kill_run(start(probes_image))\n"
            "def test_overlaps_runs(probes_image):\n"
            "    start(probes_image)\n"
        )

        assert find_probes_without_teardown_fixture(source) == ["test_kills_a_run"]

    def test_the_two_selections_exclude_each_other(self) -> None:
        execution = subprocess.run(
            [
                sys.executable,
                str(SUITE / "lifecycle_suite.py"),
                "--only-crashed-run-probes",
                "--without-crashed-run-probes",
            ],
            env={**os.environ, "PATH": ""},
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert execution.returncode == 2
        assert "not allowed with argument" in execution.stderr

    def test_an_empty_selection_fails(self, tmp_path: Path) -> None:
        shutil.copy(SUITE / "lifecycle_suite.py", tmp_path)
        lifecycle = tmp_path / "lifecycle"
        lifecycle.mkdir()
        (lifecycle / "test_failing_runs.py").write_text("def test_failure(): pass\n")

        execution = subprocess.run(
            [sys.executable, str(tmp_path / "lifecycle_suite.py"), "--only-crashed-run-probes"],
            cwd=tmp_path,
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert execution.returncode == NO_TESTS_COLLECTED, execution.stdout + execution.stderr


class TestShards:
    @pytest.mark.parametrize("count", [1, 3, 7])
    def test_the_shards_of_a_selection_cover_it_once(self, count: int) -> None:
        environment = {**os.environ, "CI": "", "DATABASUS_IMAGE": ""}
        paths = get_lifecycle_paths(SUITE / "lifecycle", None)
        marker_arguments = tuple(get_marker_arguments(False, True))
        selection = collect_cases(paths, environment, marker_arguments)
        shards = [
            collect_cases(
                paths, environment, (*marker_arguments, *get_shard_arguments((index, count)))
            )
            for index in range(1, count + 1)
        ]

        assert selection
        assert set.union(*shards) == selection
        assert sum(map(len, shards)) == len(selection)

    @pytest.mark.parametrize("shard", ["0/3", "4/3", "3", "a/3", "1/", "-1/3"])
    def test_rejects_a_shard_outside_its_count(self, shard: str) -> None:
        with pytest.raises(ValueError, match="Shard"):
            parse_shard(shard)

        execution = subprocess.run(
            [sys.executable, str(SUITE / "lifecycle_suite.py"), "--shard", shard],
            env={**os.environ, "PATH": ""},
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert execution.returncode == 2
        assert "Shard" in execution.stderr

    def test_an_empty_shard_fails(self, tmp_path: Path) -> None:
        shutil.copy(SUITE / "lifecycle_suite.py", tmp_path)
        lifecycle = tmp_path / "lifecycle"
        lifecycle.mkdir()
        (lifecycle / "test_failing_runs.py").write_text("def test_failure(): pass\n")

        execution = subprocess.run(
            [sys.executable, str(tmp_path / "lifecycle_suite.py"), "--shard", "2/2"],
            cwd=tmp_path,
            env={**os.environ, "PYTHONPATH": str(tmp_path)},
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert execution.returncode == NO_TESTS_COLLECTED, execution.stdout + execution.stderr
