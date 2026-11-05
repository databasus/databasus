"""Scenarios only the lifecycle command selects, through E2E_PROBE, to drive a run into a failure."""

import os
import subprocess
import threading
from pathlib import Path

import pytest
from playwright.sync_api import Page

from application_screens import open_application, sign_in

SYNTHETIC_BEARER_TOKEN = "synthetic-bearer-marker"
SCENARIO_FAILURE = "deliberate E2E scenario failure after readiness"
INTERRUPT_READY = "E2E_INTERRUPT_READY"

_probe = os.environ.get("E2E_PROBE", "")


def _write_to_container_log(service: str, line: str) -> None:
    container_name = f"{os.environ['E2E_COMPOSE_PROJECT']}-{service}-1"
    subprocess.run(
        ["docker", "exec", "--privileged", container_name, "sh", "-c", f"echo '{line}' > /proc/1/fd/1"],
        check=True,
    )


@pytest.mark.skipif(_probe != "scenario", reason="selected by E2E_PROBE=scenario")
def test_fails_after_the_application_and_the_browser_are_ready(page: Page):
    open_application(page)

    pytest.fail(SCENARIO_FAILURE)


@pytest.mark.skipif(_probe != "passing", reason="selected by E2E_PROBE=passing")
def test_passes_after_the_application_and_the_browser_are_ready(page: Page):
    open_application(page)

    secrets_line = f"{os.environ['E2E_PASSWORD']} {os.environ['E2E_DATABASE_PASSWORD']} Bearer {SYNTHETIC_BEARER_TOKEN}"
    _write_to_container_log("databases", f"probe-cleanup-databases {secrets_line}")
    _write_to_container_log("databasus", f"probe-cleanup-application {secrets_line}")


@pytest.mark.skipif(_probe != "interrupt", reason="selected by E2E_PROBE=interrupt")
def test_waits_for_an_interruption_after_the_application_and_the_browser_are_ready(page: Page, tmp_path: Path):
    open_application(page)
    print(f"E2E_PYTEST_TEMP_ROOT: {tmp_path.parent}", flush=True)
    print(INTERRUPT_READY, flush=True)

    threading.Event().wait()


@pytest.mark.skipif(_probe != "secrets", reason="selected by E2E_PROBE=secrets")
def test_fails_with_secrets_in_every_stream_a_run_prints(page: Page, instance_admin):
    database_password = os.environ["E2E_DATABASE_PASSWORD"]
    local_value_marker = "synthetic-local-value"  # noqa: F841 Failure output must not show local values.
    secrets_line = f"password={instance_admin.password} {database_password} Authorization: Bearer {SYNTHETIC_BEARER_TOKEN}"

    open_application(page)
    sign_in(page, instance_admin)
    _write_to_container_log("databases", f"probe-stream-databases {secrets_line}")
    _write_to_container_log("databasus", f"probe-stream-application {secrets_line}")
    print(f"probe-stream-pytest {secrets_line}", flush=True)

    pytest.fail("deliberate E2E secrets failure")
