import argparse
import os
import subprocess
import sys
from pathlib import Path


def get_python_files(root: Path) -> list[str]:
    listed_paths = subprocess.check_output(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", "*.py"],
        cwd=root,
    )
    return sorted(
        {
            os.fsdecode(root / os.fsdecode(path))
            for path in listed_paths.split(b"\0")
            if path and (root / os.fsdecode(path)).is_file()
        }
    )


def run_stage(root: Path, arguments: list[str]) -> int:
    print(f"Python quality: {' '.join(arguments[:2])}", flush=True)
    try:
        return subprocess.run(arguments, cwd=root, check=False).returncode
    except OSError as error:
        print(f"Python quality could not start {arguments[0]}: {error}", file=sys.stderr)
        return 1


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("mode", choices=("lint", "format"))
    arguments = parser.parse_args()
    root = Path(
        subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"],
            cwd=Path(__file__).resolve().parent,
            text=True,
        ).strip()
    )
    python_files = get_python_files(root)
    if not python_files:
        return 0

    configuration = str(root / "pyproject.toml")
    check_arguments = ["ruff", "check", "--config", configuration]
    format_arguments = ["ruff", "format", "--config", configuration]
    if arguments.mode == "format":
        check_arguments.append("--fix")
    else:
        format_arguments.append("--check")

    statuses = [
        run_stage(root, [*check_arguments, "--", *python_files]),
        run_stage(root, [*format_arguments, "--", *python_files]),
    ]
    if arguments.mode == "lint":
        statuses.append(
            run_stage(
                root,
                [
                    "uv",
                    "run",
                    "--locked",
                    "--project",
                    str(root / "e2e/browser"),
                    "ty",
                    "check",
                    "--project",
                    str(root),
                    "--error-on-warning",
                    "--",
                    *python_files,
                ],
            )
        )

    return int(any(status != 0 for status in statuses))


if __name__ == "__main__":
    raise SystemExit(main())
