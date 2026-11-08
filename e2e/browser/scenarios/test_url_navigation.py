import os
import re
import uuid

from application_screens import Account, open_application, sign_in
from navigation_api import NavigationRequest, get_identity, request_navigation_api
from navigation_workspace import create_navigation_workspace
from playwright.sync_api import Browser, Page, expect


class TestUrlNavigation:
    def test_restores_a_database_tab_after_reload_and_browser_history_navigation(
        self, page: Page, instance_admin: Account
    ) -> None:
        open_application(page)
        sign_in(page, instance_admin)
        first_workspace = create_navigation_workspace(page)
        second_workspace = create_navigation_workspace(page)

        page.goto(first_workspace.get_database_path())
        expect(page.get_by_role("tab", name="Config", exact=True)).to_have_attribute(
            "aria-selected", "true"
        )
        page.reload()

        expect(page).to_have_url(re.compile(re.escape(first_workspace.get_database_path()) + "$"))
        expect(page.get_by_role("tab", name="Config", exact=True)).to_have_attribute(
            "aria-selected", "true"
        )
        page.get_by_role("tab", name="Backups", exact=True).click()
        expect(page).to_have_url(
            re.compile(re.escape(first_workspace.get_database_path("backups")) + "$")
        )
        page.go_back()
        expect(page).to_have_url(re.compile(re.escape(first_workspace.get_database_path()) + "$"))
        expect(page.get_by_role("tab", name="Config", exact=True)).to_have_attribute(
            "aria-selected", "true"
        )
        page.go_forward()
        expect(page).to_have_url(
            re.compile(re.escape(first_workspace.get_database_path("backups")) + "$")
        )
        expect(page.get_by_role("tab", name="Backups", exact=True)).to_have_attribute(
            "aria-selected", "true"
        )

        page.goto(second_workspace.get_database_path())
        expect(page).to_have_url(re.compile(re.escape(second_workspace.get_database_path()) + "$"))
        expect(
            page.get_by_role("link", name=re.compile(re.escape(second_workspace.database_name)))
        ).to_have_attribute("aria-current", "true")
        page.goto(
            f"/workspaces/{first_workspace.workspace_id}/databases/{second_workspace.database_id}/config"
        )
        expect(page).to_have_url(re.compile(re.escape(first_workspace.get_database_path()) + "$"))
        expect(
            page.get_by_role("link", name=re.compile(re.escape(first_workspace.database_name)))
        ).to_have_attribute("aria-current", "true")
        expect(
            page.get_by_role("link", name=re.compile(re.escape(second_workspace.database_name)))
        ).to_have_count(0)

    def test_replaces_another_teams_link_with_available_content_for_the_signed_in_account(
        self, page: Page, instance_admin: Account, browser: Browser
    ) -> None:
        open_application(page)
        sign_in(page, instance_admin)
        available_workspace = create_navigation_workspace(page)
        foreign_workspace = create_navigation_workspace(page)
        member_account = Account(f"url-member-{uuid.uuid4()}@example.com", uuid.uuid4().hex)
        membership = request_navigation_api(
            page,
            NavigationRequest(
                "/workspace-members",
                {
                    "workspaceId": available_workspace.workspace_id,
                    "email": member_account.email,
                    "role": "WORKSPACE_ADMIN",
                },
            ),
        )
        request_navigation_api(
            page,
            NavigationRequest(
                "/users",
                {
                    "email": member_account.email,
                    "password": member_account.password,
                    "name": "URL member",
                },
            ),
        )
        member_context = browser.new_context(base_url=os.environ["E2E_BASE_URL"], locale="en-US")
        try:
            member_page = member_context.new_page()
            open_application(member_page)
            member_page.goto(foreign_workspace.get_database_path())
            sign_in(member_page, member_account)

            expect(member_page).to_have_url(
                re.compile(re.escape(available_workspace.get_database_path("backups")) + "$")
            )
            expect(
                member_page.get_by_role(
                    "link", name=re.compile(re.escape(available_workspace.database_name))
                )
            ).to_have_attribute("aria-current", "true")
            expect(
                member_page.get_by_text(foreign_workspace.database_name, exact=True)
            ).to_have_count(0)

            member_page.goto(
                f"/workspaces/{available_workspace.workspace_id}/databases/{uuid.uuid4()}/config"
            )
            expect(member_page).to_have_url(
                re.compile(re.escape(available_workspace.get_database_path()) + "$")
            )
            expect(member_page.get_by_role("tab", name="Config", exact=True)).to_have_attribute(
                "aria-selected", "true"
            )

            request_navigation_api(
                page,
                NavigationRequest(f"/databases/{available_workspace.database_id}", method="DELETE"),
            )
            member_page.reload()
            expect(member_page).to_have_url(
                re.compile(f"/workspaces/{available_workspace.workspace_id}/databases$")
            )
            expect(
                member_page.get_by_role("button", name="Add database", exact=True)
            ).to_be_visible()

            request_navigation_api(
                page,
                NavigationRequest(
                    f"/workspace-members/{get_identity(membership)}", method="DELETE"
                ),
            )
            member_page.reload()
            expect(member_page).to_have_url(re.compile(r"/$"))
            expect(member_page.get_by_role("combobox", name="Selected workspace")).to_have_count(0)
            expect(member_page.get_by_role("link", name="Profile", exact=True)).to_be_visible()
        finally:
            member_context.close()
