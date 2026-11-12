import argparse
import os
import sys
from pathlib import Path

import pytest

GROUPS = ("failures", "interruption", "isolation")
# lifecycle/conftest.py marks every probe that requests the run teardown fixture with it.
CRASHED_RUN_MARKER = "crashed_run"
SHARD_OPTION = "--lifecycle-shard"
_EXPLICIT_FILES = {
    "test_failing_runs.py": "failures",
    "test_crashed_runs.py": "interruption",
    "test_interruption.py": "interruption",
}


def get_lifecycle_paths(directory: Path, group: str | None) -> list[Path]:
    if group is None:
        return [directory]
    if group not in GROUPS:
        raise ValueError(f"Unknown lifecycle group: {group}")

    paths = [
        path
        for path in sorted(directory.rglob("*.py"))
        if (path.name.startswith("test_") or path.name.endswith("_test.py"))
        and _EXPLICIT_FILES.get(path.relative_to(directory).as_posix(), "isolation") == group
    ]
    if not paths:
        raise ValueError(f"Lifecycle group {group} has no test files")
    return paths


def get_marker_arguments(
    is_only_crashed_run_probes: bool, is_without_crashed_run_probes: bool
) -> list[str]:
    if is_only_crashed_run_probes:
        return ["-m", CRASHED_RUN_MARKER]
    if is_without_crashed_run_probes:
        return ["-m", f"not {CRASHED_RUN_MARKER}"]
    return []


def parse_shard(value: str) -> tuple[int, int]:
    index_text, separator, count_text = value.partition("/")
    if not (separator and index_text.isdecimal() and count_text.isdecimal()):
        raise ValueError(f"Shard must look like INDEX/COUNT: {value}")
    index, count = int(index_text), int(count_text)
    if not 1 <= index <= count:
        raise ValueError(f"Shard index must be between 1 and the count: {value}")
    return index, count


def _parse_shard_argument(value: str) -> tuple[int, int]:
    try:
        return parse_shard(value)
    except ValueError as error:
        raise argparse.ArgumentTypeError(str(error)) from error


def get_shard_arguments(shard: tuple[int, int] | None) -> list[str]:
    if shard is None:
        return []
    index, count = shard
    return ["-p", "lifecycle_suite", SHARD_OPTION, f"{index}/{count}"]


def pytest_addoption(parser: pytest.Parser) -> None:
    parser.addoption(SHARD_OPTION, metavar="INDEX/COUNT", help="keep one part of the selection")


@pytest.hookimpl(trylast=True)
def pytest_collection_modifyitems(config: pytest.Config, items: list[pytest.Item]) -> None:
    """Runs after marker selection, so the parts split what the selection kept."""
    shard = config.getoption(SHARD_OPTION)
    if shard is None:
        return

    index, count = parse_shard(shard)
    kept = [item for position, item in enumerate(items) if position % count == index - 1]
    deselected = [item for position, item in enumerate(items) if position % count != index - 1]
    config.hook.pytest_deselected(items=deselected)
    items[:] = kept


def main() -> None:
    parser = argparse.ArgumentParser(description="Run all browser lifecycle probes or one group")
    parser.add_argument("group", nargs="?", choices=GROUPS)
    selection = parser.add_mutually_exclusive_group()
    selection.add_argument(
        "--only-crashed-run-probes",
        action="store_true",
        help="run only the probes that leave a crashed run behind for a later run to remove",
    )
    selection.add_argument(
        "--without-crashed-run-probes",
        action="store_true",
        help="run every probe except those, so several selections can overlap",
    )
    parser.add_argument(
        "--shard",
        type=_parse_shard_argument,
        metavar="INDEX/COUNT",
        help="run every COUNT-th probe of the selection starting from INDEX, so COUNT processes share it",
    )
    arguments = parser.parse_args()
    try:
        paths = get_lifecycle_paths(Path(__file__).resolve().parent / "lifecycle", arguments.group)
    except ValueError as error:
        parser.error(str(error))
    marker_arguments = get_marker_arguments(
        arguments.only_crashed_run_probes, arguments.without_crashed_run_probes
    )
    os.execv(
        sys.executable,
        [
            sys.executable,
            "-m",
            "pytest",
            *marker_arguments,
            *get_shard_arguments(arguments.shard),
            *(str(path) for path in paths),
        ],
    )


if __name__ == "__main__":
    main()
