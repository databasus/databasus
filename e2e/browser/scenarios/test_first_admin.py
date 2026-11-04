from playwright.sync_api import Page, expect

from application_screens import open_application, sign_in


class TestFirstAdmin:
    def test_registers_the_instance_admin_with_the_notice_on_an_empty_instance(self, page: Page, instance_admin):
        open_application(page)
        sign_in(page, instance_admin)
        page.get_by_role("button", name="Databasus settings", exact=True).click()

        expect(page.get_by_role("heading", name="Databasus settings")).to_be_visible()
