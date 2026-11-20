import re
import uuid
from dataclasses import dataclass

from application_screens import Account, open_application, register_first_account, sign_in
from condition_wait import wait_until
from instance_backup_api import call_api, download_bytes
from playwright.sync_api import Locator, Page, expect

BACKUP_MILLISECONDS = 300_000
RESTORE_MILLISECONDS = 300_000
ARCHIVE_FILE_NAME = "databasus-recovery.dbsbackup"
WRONG_MASTER_KEY = "this-is-not-the-source-master-key-at-all"


@dataclass(frozen=True)
class SourceRecoveryMaterial:
    workspace_name: str
    archive: bytes
    master_key: str


def _get_items(response: dict[str, object]) -> list[dict[str, object]]:
    items = response.get("items")
    assert isinstance(items, list)
    return [item for item in items if isinstance(item, dict)]


def _get_string(value: dict[str, object], key: str) -> str:
    field = value.get(key)
    assert isinstance(field, str), f"{key} is {field!r}"
    return field


def _make_source_backup(page: Page) -> SourceRecoveryMaterial:
    """Seeds the source through the API, stores one Databasus backup and keeps the archive and the
    exported master key outside both installations, as an owner who saved them would."""
    workspace_name = f"Recovered workspace {uuid.uuid4()}"
    workspace = call_api(page, "POST", "/api/v1/workspaces", {"name": workspace_name})
    storage = call_api(
        page,
        "POST",
        "/api/v1/storages",
        {
            "workspaceId": _get_string(workspace, "id"),
            "name": f"Databasus backups {uuid.uuid4()}",
            "settings": {"type": "LOCAL"},
        },
    )
    call_api(
        page,
        "PUT",
        "/api/v1/instance-backup-settings",
        {
            "schedule": {"type": "HOURLY"},
            "retentionDays": 30,
            "storageIds": [_get_string(storage, "id")],
        },
    )

    completed_backup = wait_until(
        "a stored Databasus backup",
        lambda: _get_items(call_api(page, "GET", "/api/v1/instance-backups?limit=1")),
        lambda backups: bool(backups) and backups[0].get("status") == "COMPLETED",
        limit_seconds=BACKUP_MILLISECONDS // 1000,
    )[0]
    copies = completed_backup.get("copies")
    assert isinstance(copies, list) and copies and isinstance(copies[0], dict)
    archive_token = call_api(
        page,
        "POST",
        f"/api/v1/instance-backups/{_get_string(completed_backup, 'id')}/download-tokens",
        {"copyId": _get_string(copies[0], "id")},
    )
    archive = download_bytes(
        page, f"/api/v1/instance-backup-downloads?token={_get_string(archive_token, 'token')}"
    )

    key_export = call_api(
        page,
        "POST",
        "/api/v1/master-key-exports",
        {},
    )
    master_key = download_bytes(
        page, f"/api/v1/master-key-downloads?token={_get_string(key_export, 'token')}"
    ).decode()

    return SourceRecoveryMaterial(workspace_name, archive, master_key)


def _paste_master_key(page: Page, master_key: str) -> None:
    """The page is served over HTTP, where the Clipboard API is unavailable, so the paste reaches the
    rendered field as the event a browser dispatches for Ctrl+V."""
    key_field = page.get_by_label("Master key of the source installation", exact=True)
    key_field.fill("")
    key_field.evaluate(
        """(field, key) => {
            const clipboardData = new DataTransfer();
            clipboardData.setData('text/plain', key);
            field.dispatchEvent(
                new ClipboardEvent('paste', { clipboardData, bubbles: true, cancelable: true }),
            );
        }""",
        master_key,
    )


def _upload(page: Page, archive: bytes, master_key: str) -> None:
    with page.expect_file_chooser() as file_chooser:
        page.get_by_role("button", name=re.compile(r"^Databasus backup file")).click()
    file_chooser.value.set_files(
        files=[
            {
                "name": ARCHIVE_FILE_NAME,
                "mimeType": "application/octet-stream",
                "buffer": archive,
            }
        ]
    )
    _paste_master_key(page, master_key)
    page.get_by_role("button", name="Upload and check", exact=True).click()


def _open_recovery(page: Page) -> None:
    page.get_by_role("button", name="Recover from backup", exact=True).click()
    expect(page.get_by_role("dialog", name="Recover from a Databasus backup file")).to_be_visible()


def _prepare_replacement(page: Page, archive: bytes, master_key: str) -> Locator:
    """A prepared archive turns the recovery dialog into its replacement step."""
    _upload(page, archive, master_key)
    replacement = page.get_by_role("dialog", name="Replace this installation")
    expect(
        replacement.get_by_text("The Databasus backup is ready to replace this installation.")
    ).to_be_visible(timeout=RESTORE_MILLISECONDS)
    return replacement


class TestDatabasusRecovery:
    def test_replaces_an_initialized_target_through_the_import_form(
        self, page: Page, recovery_target_page: Page, instance_admin: Account
    ) -> None:
        open_application(page)
        sign_in(page, instance_admin)
        source = _make_source_backup(page)

        target = recovery_target_page
        temporary_root = Account(f"target-{uuid.uuid4()}@example.com", uuid.uuid4().hex, True)
        open_application(target)
        register_first_account(target, temporary_root)
        target.goto("/databasus-settings")

        _open_recovery(target)
        _upload(target, source.archive, WRONG_MASTER_KEY)
        expect(
            target.get_by_text(
                "This installation was not changed. Choose another file or correct the key and try again."
            )
        ).to_be_visible(timeout=RESTORE_MILLISECONDS)

        replacement = _prepare_replacement(target, source.archive, source.master_key)
        replacement.get_by_role("button", name="Cancel", exact=True).click()
        expect(replacement).to_be_hidden()
        target_workspaces = _get_items(call_api(target, "GET", "/api/v1/workspaces?limit=100"))
        assert source.workspace_name not in [item.get("name") for item in target_workspaces]

        _open_recovery(target)
        replacement = _prepare_replacement(target, source.archive, source.master_key)
        replacement.get_by_role(
            "checkbox", name="I understand that the current state of this installation is replaced"
        ).check()
        replacement.get_by_role("button", name="Replace", exact=True).click()

        expect(
            target.get_by_text("Databasus was recovered from a backup.", exact=False)
        ).to_be_visible(timeout=RESTORE_MILLISECONDS)

        sign_in(target, instance_admin)
        recovered_workspaces = _get_items(call_api(target, "GET", "/api/v1/workspaces?limit=100"))
        assert source.workspace_name in [item.get("name") for item in recovered_workspaces]
        expect(
            target.get_by_text("Databasus is paused after a recovery", exact=True)
        ).to_have_count(0)
        expect(target.get_by_role("button", name="Activate", exact=True)).to_have_count(0)
        status = call_api(target, "GET", "/api/v1/instance-backup-statuses")
        assert "isRecoveryPaused" not in status
        recovery = status.get("recovery")
        assert isinstance(recovery, dict)
        assert recovery.get("status") == "COMPLETED"
