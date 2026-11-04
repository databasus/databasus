import os
import uuid
from dataclasses import dataclass

from playwright.sync_api import Locator, Page, expect

from application_screens import create_local_storage, create_workspace, open_application, sign_in
from database_server import (
    BACKED_UP_DATABASE,
    RESTORE_TARGET_DATABASE,
    SERVER_HOST_ON_COMPOSE_NETWORK,
    SERVER_PORT,
    SERVER_USER,
    prepare_databases,
    read_rows,
)
from row_comparison import RowComparison

LONG_OPERATION_MILLISECONDS = 120_000
SOURCE_ROWS = ["first row before the backup", "second row before the backup"]
SUCCESSFUL_STATUS = "Successful"
FAILED_STATUS = "Failed"


@dataclass(frozen=True)
class WizardDatabase:
    name: str
    storage_name: str
    password: str


def _fill_connection(page: Page | Locator, database: str, password: str) -> None:
    page.get_by_label("Host", exact=True).fill(SERVER_HOST_ON_COMPOSE_NETWORK)
    page.get_by_label("Port", exact=True).fill(SERVER_PORT)
    page.get_by_label("Username", exact=True).fill(SERVER_USER)
    page.get_by_label("Password", exact=True).fill(password)
    page.get_by_label("DB name", exact=True).fill(database)


def _add_database_through_the_wizard(page: Page, database: WizardDatabase) -> None:
    page.get_by_role("button", name="Databases", exact=True).click()
    page.get_by_role("button", name="Add database").click()
    page.get_by_placeholder("My favourite DB").fill(database.name)
    page.get_by_role("button", name="Continue").click()
    page.get_by_role("button", name="Continue").click()
    _fill_connection(page, BACKED_UP_DATABASE, database.password)
    page.get_by_role("button", name="Test connection").click()
    _expect_connection_accepted(page)
    page.get_by_role("button", name="Continue").click()

    page.get_by_role("button", name="Skip", exact=True).click()
    page.get_by_role("button", name="Yes, I accept risks", exact=True).click()

    page.get_by_role("dialog").get_by_role("combobox", name="Storage").click()
    page.get_by_title(database.storage_name).click()
    page.get_by_role("button", name="Continue").click()
    page.get_by_role("button", name="Complete").click()

    expect(page.get_by_role("dialog")).to_be_hidden()


def _expect_connection_accepted(page: Page) -> None:
    connection_outcome = page.get_by_role("alert")

    expect(connection_outcome.first).to_be_visible()

    if page.get_by_text("Connection test passed").count() == 0:
        raise AssertionError(f"the application refused the connection form: {connection_outcome.first.inner_text()}")


def _expect_successful_outcome(scope: Page | Locator, operation: str) -> None:
    successful = scope.get_by_text(SUCCESSFUL_STATUS, exact=True).filter(visible=True)
    failed = scope.get_by_text(FAILED_STATUS, exact=True).filter(visible=True)

    expect(successful.or_(failed).first).to_be_visible(timeout=LONG_OPERATION_MILLISECONDS)

    assert failed.count() == 0, f"the {operation} is shown as failed"


def _restore_first_backup(page: Page, password: str) -> None:
    _expect_successful_outcome(page, "first backup")

    page.get_by_role("button", name="Restore from backup").first.click()

    dialog = page.get_by_role("dialog", name="Restore from backup")
    dialog.get_by_role("button", name="Select database to restore to").click()
    _fill_connection(dialog, RESTORE_TARGET_DATABASE, password)
    dialog.get_by_role("button", name="Restore to this DB").click()

    _expect_successful_outcome(dialog, "restore")


class TestCriticalPath:
    def test_restores_the_rows_of_the_first_backup_into_an_empty_database(self, page: Page, instance_admin):
        database_password = os.environ["E2E_DATABASE_PASSWORD"]
        wizard_password = "wrong-database-password" if os.environ.get("E2E_PROBE") == "refused-form" else database_password
        storage_name = f"Critical path storage {uuid.uuid4()}"

        prepare_databases(SOURCE_ROWS)
        open_application(page)
        sign_in(page, instance_admin)
        create_workspace(page, f"Critical path {uuid.uuid4()}")
        create_local_storage(page, storage_name)
        _add_database_through_the_wizard(page, WizardDatabase("Critical path database", storage_name, wizard_password))
        _restore_first_backup(page, database_password)

        backed_up_difference = RowComparison(SOURCE_ROWS, read_rows(BACKED_UP_DATABASE)).describe_difference()
        restored_difference = RowComparison(SOURCE_ROWS, read_rows(RESTORE_TARGET_DATABASE)).describe_difference()

        assert backed_up_difference is None, f"the backed-up database differs: {backed_up_difference}"
        assert restored_difference is None, f"the restored database differs: {restored_difference}"
