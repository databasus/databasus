import uuid

from playwright.sync_api import Page, expect

from application_screens import create_workspace, open_application, sign_in

STORAGE_KEYS_SURVIVING_SIGN_OUT = ["databasus-locale", "databasus-theme", "lastVersionReload"]


class TestSignIn:
    def test_signs_in_and_out_without_a_password_reset_when_no_mail_server_is_configured(
        self, page: Page, instance_admin
    ):
        open_application(page)

        expect(page.get_by_role("button", name="Sign in", exact=True)).to_be_visible()
        expect(page.get_by_role("button", name="Forgot password?", exact=True)).to_have_count(0)

        sign_in(page, instance_admin)
        create_workspace(page, f"Signing out {uuid.uuid4()}")

        assert page.evaluate("() => localStorage.getItem('selected_workspace_id')") is not None

        page.get_by_role("button", name="Profile", exact=True).click()

        expect(page.get_by_role("textbox", name="Email", exact=True)).to_have_value(instance_admin.email)

        page.evaluate(
            """() => {
                localStorage.setItem('selectedDatabaseId_previous-workspace', 'previous-database');
                localStorage.setItem('logicalBackupListFilter', '{"statuses":["FAILED"]}');
                localStorage.setItem('physicalBackupsFilters', '{"types":["FULL"]}');
                localStorage.setItem('databasus-locale', 'en');
                localStorage.setItem('databasus-theme', 'dark');
                localStorage.setItem('lastVersionReload', '123');
            }"""
        )
        page.get_by_role("button", name="Logout").click()

        expect(page.get_by_role("button", name="Sign in", exact=True)).to_be_visible()

        remaining_storage_keys = page.evaluate("() => Object.keys(localStorage).sort()")

        assert remaining_storage_keys == STORAGE_KEYS_SURVIVING_SIGN_OUT
