import os
import re
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from typing import NotRequired, TypedDict, TypeGuard

from api_response import get_api_object
from application_screens import read_api
from condition_wait import wait_until
from database_server import (
    BACKED_UP_DATABASE,
    SERVER_HOST_ON_COMPOSE_NETWORK,
    SERVER_PORT,
    SERVER_USER,
    execute_sql,
)
from playwright.sync_api import Page, Request, expect

MARKER_TABLE = "e2e_physical_markers"
PHYSICAL_ROWS = [
    "A before full",
    "B before incremental",
    "C first archived WAL",
    "D second archived WAL",
]


@dataclass(frozen=True)
class ArchivedWrite:
    filename: str
    position: str


@dataclass(frozen=True)
class BackupCadence:
    label: str
    cron: str


@dataclass(frozen=True)
class BackupSelection:
    database_id: str
    kind: str


def read_physical_rows() -> list[str]:
    return execute_sql(BACKED_UP_DATABASE, f"SELECT label FROM {MARKER_TABLE} ORDER BY id")


def write_marker(index: int) -> None:
    execute_sql(
        BACKED_UP_DATABASE, f"INSERT INTO {MARKER_TABLE} VALUES ({index}, '{PHYSICAL_ROWS[index]}')"
    )


def _set_future_cadence(page: Page, schedule: BackupCadence) -> None:
    cadence = page.get_by_role("group", name=schedule.label, exact=True)
    schedule_type = cadence.get_by_role("combobox", name=schedule.label, exact=True)
    schedule_type.press("ArrowDown")
    expect(schedule_type).to_have_attribute("aria-expanded", "true")
    page.get_by_role("option", name="Cron", exact=True).click()
    expect(schedule_type).to_have_attribute("aria-expanded", "false")
    expect(schedule_type).to_contain_text("Cron")
    cadence.get_by_role("textbox", name="Cron expression (UTC)", exact=True).fill(schedule.cron)


def add_physical_database(page: Page, storage_name: str) -> str:
    page.get_by_role("link", name="Databases", exact=True).click()
    page.get_by_role("button", name="Add database").click()
    page.get_by_placeholder("My favourite DB").fill("Critical path physical database")
    page.get_by_role("button", name="Continue", exact=True).click()
    page.get_by_role("button", name=re.compile("^Physical")).click()
    page.get_by_role("button", name="Continue", exact=True).click()
    page.get_by_role("radio", name=re.compile(r"^Full \+ incremental \+ WAL streaming")).check()
    page.get_by_label("Host", exact=True).fill(SERVER_HOST_ON_COMPOSE_NETWORK)
    page.get_by_label("Port", exact=True).fill(SERVER_PORT)
    page.get_by_label("Username", exact=True).fill(SERVER_USER)
    page.get_by_label("Password", exact=True).fill(os.environ["E2E_DATABASE_PASSWORD"])
    page.get_by_role("button", name="Test connection", exact=True).click()
    expect(page.get_by_text("Connection test passed", exact=True)).to_be_visible()
    page.get_by_role("button", name="Continue", exact=True).click()
    page.get_by_role("button", name="Skip", exact=True).click()
    page.get_by_role("button", name="Yes, I accept risks", exact=True).click()

    future = datetime.now(UTC) + timedelta(days=2)
    _set_future_cadence(
        page, BackupCadence(label="Full backup cadence", cron=f"0 0 {future.day} {future.month} *")
    )
    _set_future_cadence(
        page,
        BackupCadence(
            label="Incremental backup cadence", cron=f"0 */6 {future.day} {future.month} *"
        ),
    )
    page.get_by_role("dialog").get_by_role("combobox", name="Storage", exact=True).click()
    page.get_by_role("option", name=storage_name, exact=True).click()
    page.get_by_role("button", name="Continue", exact=True).click()

    manual_backup_requests: list[str] = []

    def record_manual_backup(request: Request) -> None:
        if request.method == "POST" and request.url.endswith("/api/v1/physical-backups"):
            manual_backup_requests.append(request.url)

    page.on("request", record_manual_backup)
    try:
        with page.expect_response(
            lambda response: (
                response.url.endswith("/api/v1/databases") and response.request.method == "POST"
            )
        ) as created_database:
            page.get_by_role("button", name="Complete", exact=True).click()
        expect(page.get_by_role("dialog")).to_be_hidden()
        assert not manual_backup_requests, (
            "the physical wizard requested a manual backup in addition to its scheduled initial full"
        )
    finally:
        page.remove_listener("request", record_manual_backup)

    database = get_api_object(created_database.value.json())
    database_id = database.get("id")
    assert isinstance(database_id, str)
    return database_id


class PhysicalBackup(TypedDict):
    id: str
    type: str
    status: str
    stopLsn: NotRequired[str | None]
    rootFullBackupId: NotRequired[str | None]
    fromWalFilename: NotRequired[str | None]
    toWalFilename: NotRequired[str | None]


def is_physical_backup(value: object) -> TypeGuard[PhysicalBackup]:
    if not isinstance(value, dict):
        return False
    if not all(isinstance(value.get(key), str) for key in ("id", "type", "status")):
        return False
    return all(
        value.get(key) is None or isinstance(value[key], str)
        for key in ("stopLsn", "rootFullBackupId", "fromWalFilename", "toWalFilename")
    )


def get_physical_backups(response: dict[str, object]) -> list[PhysicalBackup]:
    items = response.get("items")
    assert isinstance(items, list)
    backups: list[PhysicalBackup] = []
    for item in items:
        assert is_physical_backup(item), "Invalid physical backup response"
        backups.append(item)
    return backups


def get_archive_progress_lsn(response: dict[str, object]) -> str:
    current = response.get("current")
    if current is None:
        return "0/0"
    current = get_api_object(current)
    position = current.get("archiveProgressLsn")
    assert position is None or isinstance(position, str)
    return position or "0/0"


def contains_archived_write(backup: PhysicalBackup, write: ArchivedWrite) -> bool:
    if backup["type"] != "WAL" or backup["status"] != "COMPLETED":
        return False
    first_filename = backup.get("fromWalFilename")
    last_filename = backup.get("toWalFilename")
    assert first_filename is not None and last_filename is not None
    return first_filename <= write.filename <= last_filename


def list_backups(page: Page, database_id: str) -> list[PhysicalBackup]:
    backups = get_physical_backups(
        read_api(page, f"/api/v1/physical-backups?databaseId={database_id}&limit=100")
    )
    failed = [backup for backup in backups if backup["status"] in ("ERROR", "CHAIN_BROKEN")]
    assert not failed, f"physical backup failed: {failed}"

    return backups


def wait_for_backup(page: Page, selection: BackupSelection) -> PhysicalBackup:
    backups = wait_until(
        f"completed {selection.kind} backup",
        lambda: list_backups(page, selection.database_id),
        lambda rows: any(
            row["type"] == selection.kind and row["status"] == "COMPLETED" for row in rows
        ),
    )
    matching = [backup for backup in backups if backup["type"] == selection.kind]
    assert len(matching) == 1, f"unrelated {selection.kind} backups appeared: {matching}"
    backup = matching[0]
    backup_type_label = "Full" if selection.kind == "FULL" else "Incremental"
    backup_row = page.get_by_role("row").filter(has=page.get_by_text(backup_type_label, exact=True))
    expect(backup_row).to_have_count(1)
    expect(backup_row).to_contain_text("Successful")

    return backup


def build_physical_chain(page: Page, storage_name: str) -> tuple[str, list[ArchivedWrite]]:
    print("E2E physical: full backup", flush=True)
    execute_sql(
        BACKED_UP_DATABASE,
        f"CREATE TABLE {MARKER_TABLE} (id integer PRIMARY KEY, label text NOT NULL)",
    )
    write_marker(0)

    database_id = add_physical_database(page, storage_name)
    full_backup = wait_for_backup(page, BackupSelection(database_id=database_id, kind="FULL"))

    write_marker(1)
    execute_sql(
        BACKED_UP_DATABASE,
        "SELECT pg_logical_emit_message(true, 'browser-e2e', repeat('x', 1024 * 1024)) "
        "FROM generate_series(1, 32)",
    )
    execute_sql(BACKED_UP_DATABASE, "CHECKPOINT")
    execute_sql(BACKED_UP_DATABASE, "SELECT pg_switch_wal()")
    stop_lsn = full_backup.get("stopLsn")
    assert stop_lsn is not None
    assert re.fullmatch(r"[0-9A-Fa-f]+/[0-9A-Fa-f]+", stop_lsn)
    wait_until(
        "WAL summaries through the full backup",
        lambda: execute_sql(
            BACKED_UP_DATABASE,
            f"SELECT EXISTS (SELECT FROM pg_available_wal_summaries() "
            f"WHERE start_lsn <= '{stop_lsn}'::pg_lsn AND end_lsn >= '{stop_lsn}'::pg_lsn)",
        ),
        lambda rows: rows == ["t"],
    )

    print("E2E physical: incremental backup", flush=True)
    page.get_by_role("button", name="Back up now", exact=True).locator("..").get_by_role(
        "button"
    ).last.click()
    with page.expect_response(
        lambda response: (
            response.url.endswith("/api/v1/physical-backups") and response.request.method == "POST"
        )
    ) as started_backup:
        page.get_by_role("menuitem", name="Incremental backup", exact=True).click()
    assert started_backup.value.status == 202, (
        f"UI incremental backup refused: {started_backup.value.text()}"
    )
    incremental_backup = wait_for_backup(
        page, BackupSelection(database_id=database_id, kind="INCREMENTAL")
    )
    assert incremental_backup.get("rootFullBackupId") == full_backup["id"], (
        f"incremental backup {incremental_backup} does not extend full backup {full_backup}"
    )
    assert "parentIncrementalBackupId" not in incremental_backup, incremental_backup
    assert (
        len([backup for backup in list_backups(page, database_id) if backup["type"] == "FULL"]) == 1
    )

    print("E2E physical: two later WAL segments", flush=True)
    archived_writes = []
    for index in (2, 3):
        filename = execute_sql(
            BACKED_UP_DATABASE,
            f"WITH marker AS (INSERT INTO {MARKER_TABLE} VALUES ({index}, '{PHYSICAL_ROWS[index]}') RETURNING id) "
            f"SELECT pg_walfile_name(pg_logical_emit_message(true, 'browser-e2e', '{PHYSICAL_ROWS[index]}')) "
            "FROM marker",
        )[0]
        position = execute_sql(BACKED_UP_DATABASE, "SELECT pg_current_wal_lsn()::text")[0]
        archived_writes.append(ArchivedWrite(filename, position))
        execute_sql(BACKED_UP_DATABASE, "SELECT pg_switch_wal()")

    assert archived_writes[0].filename != archived_writes[1].filename
    wait_until(
        "both post-incremental WAL files in archived ranges",
        lambda: list_backups(page, database_id),
        lambda backups: all(
            any(contains_archived_write(backup, write) for backup in backups)
            for write in archived_writes
        ),
    )
    wait_until(
        "archive progress through the final write",
        lambda: read_api(page, f"/api/v1/wal-streams/{database_id}"),
        lambda live: (
            _parse_lsn(get_archive_progress_lsn(live)) >= _parse_lsn(archived_writes[-1].position)
        ),
    )
    expect(
        page.get_by_role("row")
        .filter(has=page.get_by_text("WAL", exact=True))
        .filter(has=page.get_by_text("Successful", exact=True))
        .first
    ).to_be_visible()

    return database_id, archived_writes


def _parse_lsn(position: str) -> int:
    high, low = position.split("/")
    return (int(high, 16) << 32) + int(low, 16)
