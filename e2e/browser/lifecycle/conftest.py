import os
import secrets
import signal
import subprocess
import time
from collections.abc import Iterator

import pytest
from lifecycle.run_cleanup import register_run_cleanup as register_run_cleanup
from public_command import REPOSITORY_ROOT, stop_live_runs
from supervisor_file import (
    SupervisorRecord,
    get_project_name,
    get_required_process_start_time,
    remove_run_directory,
    write_supervisor_record,
)

IMAGE_VARIABLE = "DATABASUS_IMAGE"
UNRELATED_IMAGE = (
    "postgres:18@sha256:3a4264520e121176619f54ff867d12774bc81e95e7b84db58d63bccdb6144b81"
)
# A killed lifecycle command cannot remove this container, so it ends by itself.
UNRELATED_LIFETIME_SECONDS = "7200"
UNRELATED_READY_LIMIT_SECONDS = 60
UNRELATED_POLL_SECONDS = 0.5


def pytest_configure() -> None:
    if os.environ.get("CI") and not os.environ.get(IMAGE_VARIABLE):
        pytest.exit(f"E2E lifecycle refuses to run in CI without {IMAGE_VARIABLE}", returncode=1)


def _raise_interrupt(_signal_number: int, _frame: object) -> None:
    signal.signal(signal.SIGINT, signal.SIG_IGN)
    signal.signal(signal.SIGTERM, signal.SIG_IGN)

    raise KeyboardInterrupt


@pytest.fixture(scope="session", autouse=True)
def interruptible_session() -> None:
    signal.signal(signal.SIGINT, _raise_interrupt)
    signal.signal(signal.SIGTERM, _raise_interrupt)


@pytest.fixture(scope="session")
def lifecycle_run_id() -> str:
    return secrets.token_hex(6)


@pytest.fixture(autouse=True)
def public_runs_are_stopped() -> Iterator[None]:
    try:
        yield
    finally:
        stop_live_runs()


@pytest.fixture(scope="session")
def probes_image(lifecycle_run_id: str) -> Iterator[str]:
    named_image = os.environ.get(IMAGE_VARIABLE)

    if named_image:
        yield named_image
        stop_live_runs()
        return

    built_image = f"{get_project_name(lifecycle_run_id)}:probes"
    write_supervisor_record(
        SupervisorRecord(
            lifecycle_run_id, os.getpid(), get_required_process_start_time(os.getpid()), built_image
        )
    )

    try:
        subprocess.run(
            ["docker", "build", "--quiet", "--tag", built_image, str(REPOSITORY_ROOT)],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        yield built_image
    finally:
        try:
            stop_live_runs()
        finally:
            _remove_probes_image(built_image, lifecycle_run_id)


def _remove_probes_image(built_image: str, lifecycle_run_id: str) -> None:
    """A failed removal keeps the supervisor file, which lets the next run remove the image."""
    removal = subprocess.run(["docker", "image", "rm", built_image], capture_output=True, text=True)

    if removal.returncode != 0 and "No such image" not in removal.stderr:
        pytest.fail(
            f"E2E lifecycle could not remove image {built_image}: {removal.stderr}", pytrace=False
        )

    remove_run_directory(lifecycle_run_id)


@pytest.fixture(scope="session")
def unrelated_container(lifecycle_run_id: str) -> Iterator[str]:
    """Outside every run and without the suite's name, so a sweep reaching too far would remove it."""
    container_name = f"unrelated-to-e2e-{lifecycle_run_id}"
    try:
        subprocess.run(
            [
                "docker",
                "run",
                "--detach",
                "--rm",
                "--name",
                container_name,
                "--env",
                f"POSTGRES_PASSWORD={secrets.token_urlsafe(16)}",
                "--entrypoint",
                "timeout",
                UNRELATED_IMAGE,
                UNRELATED_LIFETIME_SECONDS,
                "docker-entrypoint.sh",
                "postgres",
            ],
            check=True,
            capture_output=True,
        )
        _wait_until_answering(container_name)

        yield container_name

    finally:
        subprocess.run(
            ["docker", "rm", "--force", "--volumes", container_name], capture_output=True
        )


def _query_container(container_name: str) -> subprocess.CompletedProcess[str]:
    # The image's temporary initialization server accepts Unix sockets before the final server starts.
    return subprocess.run(
        [
            "docker",
            "exec",
            container_name,
            "pg_isready",
            "--host",
            "127.0.0.1",
            "--username",
            "postgres",
        ],
        capture_output=True,
        text=True,
    )


def _wait_until_answering(container_name: str) -> None:
    deadline = time.monotonic() + UNRELATED_READY_LIMIT_SECONDS

    while _query_container(container_name).returncode != 0:
        assert time.monotonic() < deadline, f"{container_name} did not start answering"
        time.sleep(UNRELATED_POLL_SECONDS)


@pytest.fixture(autouse=True)
def unrelated_container_keeps_answering(unrelated_container: str) -> Iterator[None]:
    yield

    answer = _query_container(unrelated_container)

    assert answer.returncode == 0, (
        f"a run disturbed the unrelated container: {answer.stdout}{answer.stderr}"
    )
