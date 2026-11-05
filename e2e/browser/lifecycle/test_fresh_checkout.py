import hashlib
import os
import re
import shutil
import subprocess
from pathlib import Path

import pytest

from leftover_check import assert_nothing_of_the_project_remains, is_image_present
from public_command import REPOSITORY_ROOT, FinishedRun, PublicRun
from supervisor_file import get_run_directory

OWN_DEPENDENCY_DIRECTORY = ("e2e", "browser", ".venv")

pytestmark = pytest.mark.skipif(
    bool(os.environ.get("DATABASUS_IMAGE")),
    reason="needs a run that builds its own image, and CI builds none",
)


@pytest.fixture(autouse=True)
def no_named_image(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("DATABASUS_IMAGE", raising=False)


class TestFreshCheckout:
    def test_runs_without_dependencies_or_an_environment_file_and_writes_only_its_dependency_directory(
        self, tmp_path: Path
    ):
        checkout = tmp_path / "checkout"
        _copy_tracked_and_untracked_files(checkout)
        files_before = _hash_files_outside_dependencies(checkout)

        assert not (checkout / ".env").exists()
        assert not (checkout / Path(*OWN_DEPENDENCY_DIRECTORY)).exists()

        run, finished_run = _run_in_checkout(checkout, {})

        assert finished_run.exit_code == 0, finished_run.output
        assert _hash_files_outside_dependencies(checkout) == files_before

        built_image = re.search(r"E2E image under test: (\S+)", finished_run.output).group(1)

        assert not is_image_present(built_image)
        assert_nothing_of_the_project_remains(run.get_project_name())
        assert not get_run_directory(run.get_run_id()).exists()

    def test_writes_only_its_dependency_directory_when_a_run_fails_after_signing_in(
        self, tmp_path: Path, probes_image: str
    ):
        checkout = tmp_path / "checkout"
        _copy_tracked_and_untracked_files(checkout)
        files_before = _hash_files_outside_dependencies(checkout)

        run, finished_run = _run_in_checkout(checkout, {"DATABASUS_IMAGE": probes_image, "E2E_PROBE": "secrets"})

        assert finished_run.exit_code != 0, finished_run.output
        assert "deliberate E2E secrets failure" in finished_run.output
        assert _hash_files_outside_dependencies(checkout) == files_before
        assert_nothing_of_the_project_remains(run.get_project_name())
        assert not get_run_directory(run.get_run_id()).exists()


def _run_in_checkout(checkout: Path, environment: dict[str, str]) -> tuple[PublicRun, FinishedRun]:
    run = PublicRun(
        working_directory=checkout,
        environment={"MISE_TRUSTED_CONFIG_PATHS": str(checkout), **environment},
    )

    try:
        return run, run.finish()
    finally:
        # mise still records the trust on the machine, which would outlive this temporary checkout.
        subprocess.run(["mise", "trust", "--untrust", "--quiet", str(checkout)], check=True)


def _copy_tracked_and_untracked_files(destination: Path) -> None:
    listing = subprocess.run(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"],
        cwd=REPOSITORY_ROOT,
        capture_output=True,
        text=True,
        check=True,
    )

    for relative_path in filter(None, listing.stdout.split("\0")):
        source = REPOSITORY_ROOT / relative_path
        if not source.is_file():
            continue

        target = destination / relative_path
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target, follow_symlinks=False)


def _hash_files_outside_dependencies(directory: Path) -> dict[str, str]:
    dependency_directory = directory / Path(*OWN_DEPENDENCY_DIRECTORY)

    return {
        str(path.relative_to(directory)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in directory.rglob("*")
        if path.is_file() and not path.is_relative_to(dependency_directory)
    }
