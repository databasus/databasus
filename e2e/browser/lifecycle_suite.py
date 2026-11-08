import argparse
import os
import sys
from pathlib import Path

GROUPS = ("failures", "interruption", "isolation")
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


def main() -> None:
    parser = argparse.ArgumentParser(description="Run all browser lifecycle probes or one group")
    parser.add_argument("group", nargs="?", choices=GROUPS)
    arguments = parser.parse_args()
    try:
        paths = get_lifecycle_paths(Path(__file__).resolve().parent / "lifecycle", arguments.group)
    except ValueError as error:
        parser.error(str(error))
    os.execv(sys.executable, [sys.executable, "-m", "pytest", *(str(path) for path in paths)])


if __name__ == "__main__":
    main()
