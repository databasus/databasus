from application_screens import Account, open_application, sign_in
from playwright.sync_api import Page, expect


class TestFirstAdmin:
    def test_opens_instance_settings_when_the_first_admin_visits_its_address(
        self, page: Page, instance_admin: Account
    ) -> None:
        open_application(page)
        sign_in(page, instance_admin)
        page.goto("/databasus-settings")

        expect(page.get_by_role("heading", name="Databasus settings")).to_be_visible()
