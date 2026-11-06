import uuid

from application_screens import (
    Account,
    create_local_storage,
    create_workspace,
    expect_selected_workspace,
    open_application,
    read_api,
    read_browser_storage,
    sign_in,
)
from playwright.sync_api import Page, expect
from storage_response import get_storage, get_storages

SELECTED_WORKSPACE_KEY = "selected_workspace_id"


class TestWorkspaceResources:
    def test_keeps_a_saved_storage_after_a_reload_and_isolates_it_between_workspaces(
        self, page: Page, instance_admin: Account
    ) -> None:
        workspace_name = f"Browser archive {uuid.uuid4()}"
        other_workspace_name = f"Browser isolated {uuid.uuid4()}"
        storage_name = f"Local archive {uuid.uuid4()}"

        open_application(page)
        sign_in(page, instance_admin)
        create_workspace(page, workspace_name)
        workspace_id = read_browser_storage(page, SELECTED_WORKSPACE_KEY)

        assert workspace_id

        create_local_storage(page, storage_name)
        listed_storages = get_storages(
            read_api(page, f"/api/v1/storages?workspaceId={workspace_id}&limit=100")
        )
        saved_storage = next(
            storage for storage in listed_storages if storage["name"] == storage_name
        )

        assert saved_storage["workspaceId"] == workspace_id
        assert saved_storage["settings"]["type"] == "LOCAL"
        assert (
            get_storage(read_api(page, f"/api/v1/storages/{saved_storage['id']}"))["name"]
            == storage_name
        )

        page.reload()
        page.get_by_role("button", name="Storages", exact=True).click()

        expect(page.get_by_role("button", name=storage_name, exact=True)).to_have_attribute(
            "aria-pressed", "true"
        )
        assert read_browser_storage(page, SELECTED_WORKSPACE_KEY) == workspace_id

        create_workspace(page, other_workspace_name)
        other_workspace_id = read_browser_storage(page, SELECTED_WORKSPACE_KEY)

        assert other_workspace_id != workspace_id

        page.get_by_role("button", name="Storages", exact=True).click()

        expect(page.get_by_role("button", name="Add storage", exact=True)).to_be_visible()
        expect(page.get_by_role("button", name=storage_name, exact=True)).to_have_count(0)
        assert (
            get_storages(
                read_api(page, f"/api/v1/storages?workspaceId={other_workspace_id}&limit=100")
            )
            == []
        )

        page.reload()

        expect_selected_workspace(page, other_workspace_name)
        assert read_browser_storage(page, SELECTED_WORKSPACE_KEY) == other_workspace_id

        page.get_by_role("button", name="Storages", exact=True).click()

        expect(page.get_by_role("button", name="Add storage", exact=True)).to_be_visible()
        expect(page.get_by_role("button", name=storage_name, exact=True)).to_have_count(0)
        assert (
            get_storages(
                read_api(page, f"/api/v1/storages?workspaceId={other_workspace_id}&limit=100")
            )
            == []
        )

        workspace_selector = page.get_by_role("combobox", name="Selected workspace")
        workspace_selector.focus()
        workspace_selector.press("ArrowDown")
        page.get_by_title(workspace_name, exact=True).click()

        expect(page.get_by_role("button", name=storage_name, exact=True)).to_have_attribute(
            "aria-pressed", "true"
        )
