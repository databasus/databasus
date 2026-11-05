import subprocess
from collections.abc import Callable
from dataclasses import dataclass

from condition_wait import wait_until
from database_server import BACKED_UP_DATABASE, SERVER_USER
from physical_backup_chain import MARKER_TABLE, PHYSICAL_ROWS, ArchivedWrite, read_physical_rows
from playwright.sync_api import Locator, Page, expect
from recovery_environment import (
    PageCommand,
    execute_page_command,
    recovery_exec,
    start_page_command,
)
from row_comparison import RowComparison

PG_BIN = "/usr/lib/postgresql/18/bin"
HOST_DIRECTORY = "./host restore"
DOCKER_DIRECTORY = "/restore-work/docker restore"


@dataclass(frozen=True)
class RecoveryTarget:
    environment: str
    container_id: str = ""
    is_running: Callable[[], bool] = lambda: True


def _open_restore(page: Page) -> Locator:
    page.get_by_role("button", name="Restore", exact=True).click()
    dialog = page.get_by_role("dialog", name="Point-in-time restore", exact=True)
    expect(dialog.get_by_role("radio", name="Via script", exact=True)).to_be_checked()
    expect(dialog.get_by_placeholder("Latest available")).to_have_value("")
    expect(dialog.get_by_role("tabpanel").locator("pre")).to_be_visible()

    return dialog


def _read_script(dialog: Locator) -> str:
    return dialog.get_by_role("tabpanel").locator("pre").inner_text()


def _read_startup_command(dialog: Locator, prefix: str) -> str:
    commands = dialog.get_by_role("tabpanel").locator("code").all_text_contents()
    matching = [command for command in commands if command.startswith(prefix)]
    assert len(matching) == 1, f"startup instructions do not contain one {prefix} command"

    return matching[0]


def _check_downloaded_bundle(directory: str, archived_writes: list[ArchivedWrite]) -> None:
    listing = recovery_exec("tar", "-tf", f"{directory}/restore.tar").stdout.splitlines()
    assert any(path.startswith("full/") or path.startswith("./full/") for path in listing), listing
    assert any("incr-1/" in path for path in listing), listing
    assert not any("incr-2/" in path for path in listing), listing

    for write in archived_writes:
        assert any(f"wal/{write.filename}" in path for path in listing), (
            f"bundle lacks WAL {write.filename}"
        )


def _read_target_sql(target: RecoveryTarget, statement: str) -> subprocess.CompletedProcess[str]:
    psql = [
        "psql",
        "--username",
        SERVER_USER,
        "--dbname",
        BACKED_UP_DATABASE,
        "--no-align",
        "--tuples-only",
        "--set",
        "ON_ERROR_STOP=1",
        "--command",
        statement,
    ]
    if target.environment == "Host":
        return recovery_exec(f"{PG_BIN}/psql", *psql[1:], should_check=False)

    return recovery_exec("docker", "exec", target.container_id, *psql, should_check=False)


def _check_recovered_rows(target: RecoveryTarget) -> None:
    def read_recovery_state() -> tuple[int, str, str]:
        assert target.is_running(), (
            f"{target.environment} startup process ended before PostgreSQL became ready"
        )
        query = _read_target_sql(target, "SELECT pg_is_in_recovery()")
        return query.returncode, query.stdout.strip(), query.stderr

    wait_until(
        f"{target.environment} PostgreSQL to finish recovery",
        read_recovery_state,
        lambda state: state[:2] == (0, "f"),
    )
    rows = _read_target_sql(target, f"SELECT label FROM {MARKER_TABLE} ORDER BY id")
    assert rows.returncode == 0, f"{target.environment} marker query failed: {rows.stderr}"
    difference = RowComparison(PHYSICAL_ROWS, rows.stdout.splitlines()).describe_difference()
    assert difference is None, f"{target.environment} restored rows differ: {difference}"


def restore_both_environments(page: Page, archived_writes: list[ArchivedWrite]) -> None:
    print("E2E physical: Script/Host restore and startup", flush=True)
    dialog = _open_restore(page)
    dialog.get_by_role("textbox", name="Restore directory", exact=True).fill(HOST_DIRECTORY)
    dialog.get_by_role("textbox", name="PostgreSQL bin path", exact=True).fill(PG_BIN)
    expect(dialog.get_by_role("tabpanel").locator("pre")).to_contain_text(PG_BIN)
    expect(dialog.get_by_role("tabpanel").locator("pre")).to_contain_text(HOST_DIRECTORY)

    execute_page_command(PageCommand(text=_read_script(dialog), stage="Script/Host recovery"))

    _check_downloaded_bundle("/restore-work/host restore", archived_writes)

    execute_page_command(
        PageCommand(
            text=_read_startup_command(dialog, "chown "), stage="Host ownership", user="root"
        )
    )
    execute_page_command(
        PageCommand(text=_read_startup_command(dialog, '"' + PG_BIN), stage="Host startup")
    )

    _check_recovered_rows(RecoveryTarget(environment="Host"))

    dialog.get_by_role("button", name="Close", exact=True).last.click()
    expect(dialog).to_be_hidden()

    print("E2E physical: Script/Docker restore and startup", flush=True)
    dialog = _open_restore(page)
    dialog.get_by_role("tab", name="Docker", exact=True).click()
    image = recovery_exec("printenv", "E2E_POSTGRES_IMAGE").stdout.strip()
    dialog.get_by_role("textbox", name="Restore directory", exact=True).fill(DOCKER_DIRECTORY)
    dialog.get_by_role("textbox", name="PostgreSQL image", exact=True).fill(image)
    expect(dialog.get_by_role("tabpanel").locator("pre")).to_contain_text(image)
    expect(dialog.get_by_role("tabpanel").locator("pre")).to_contain_text(DOCKER_DIRECTORY)

    execute_page_command(PageCommand(text=_read_script(dialog), stage="Script/Docker recovery"))

    _check_downloaded_bundle(DOCKER_DIRECTORY, archived_writes)

    startup = start_page_command(
        PageCommand(text=_read_startup_command(dialog, "docker run "), stage="Docker startup")
    )

    try:
        containers = wait_until(
            "Docker startup container",
            lambda: recovery_exec("docker", "ps", "--quiet").stdout.split(),
            lambda containers: len(containers) == 1,
        )
        container_id = containers[0]
        restored_image = recovery_exec(
            "docker", "inspect", "--format", "{{.Config.Image}}", container_id
        ).stdout.strip()
        assert restored_image == image, restored_image

        _check_recovered_rows(
            RecoveryTarget(
                environment="Docker",
                container_id=container_id,
                is_running=lambda: startup.process.poll() is None,
            )
        )
    finally:
        for container_id in recovery_exec("docker", "ps", "--quiet").stdout.split():
            recovery_exec("docker", "stop", "--time", "5", container_id)
        startup.finish()

    difference = RowComparison(PHYSICAL_ROWS, read_physical_rows()).describe_difference()
    assert difference is None, f"physical source rows differ after both restores: {difference}"
    dialog.get_by_role("button", name="Close", exact=True).last.click()


def check_invalid_host_tools(page: Page) -> None:
    dialog = _open_restore(page)
    dialog.get_by_role("textbox", name="Restore directory", exact=True).fill(
        "./invalid tools restore"
    )
    dialog.get_by_role("textbox", name="PostgreSQL bin path", exact=True).fill(
        "/unavailable postgres tools"
    )
    expect(dialog.get_by_role("tabpanel").locator("pre")).to_contain_text(
        "/unavailable postgres tools"
    )
    command = start_page_command(
        PageCommand(text=_read_script(dialog), stage="invalid Host tools from UI")
    )
    diagnostic = command.finish(is_success_required=False)
    assert "has no executable pg_combinebackup" in diagnostic, diagnostic
    raise AssertionError("deliberate invalid Host tools command failure from UI")
