import json
import time
import uuid

from api_response import get_api_object
from application_screens import (
    Account,
    create_workspace,
    open_application,
    read_browser_storage,
    sign_in,
)
from playwright.sync_api import Page, expect

PREFERENCES_SURVIVING_SIGN_OUT = {
    "databasus-locale": "fr",
    "databasus-theme": "dark",
    "lastVersionReload": "123",
}


class TestSignIn:
    def test_signs_in_and_out_without_a_password_reset_when_no_mail_server_is_configured(
        self, page: Page, instance_admin: Account
    ) -> None:
        open_application(page)

        expect(page.get_by_role("button", name="Sign in", exact=True)).to_be_visible()
        expect(page.get_by_role("button", name="Forgot password?", exact=True)).to_have_count(0)

        sign_in(page, instance_admin)

        profile_button = page.get_by_role("link", name="Profile", exact=True)
        expect(profile_button).to_be_visible()

        create_workspace(page, f"Signing out {uuid.uuid4()}")

        assert page.evaluate("() => localStorage.getItem('selected_workspace_id')") is not None

        profile_button.click()

        expect(page.get_by_role("textbox", name="Email", exact=True)).to_have_value(
            instance_admin.email
        )

        page.evaluate(
            """(preferences) => {
                localStorage.setItem('selectedDatabaseId_previous-workspace', 'previous-database');
                localStorage.setItem('logicalBackupListFilter', '{"statuses":["FAILED"]}');
                localStorage.setItem('physicalBackupsFilters', '{"types":["FULL"]}');
                for (const [key, value] of Object.entries(preferences)) {
                    localStorage.setItem(key, value);
                }
            }""",
            {
                **PREFERENCES_SURVIVING_SIGN_OUT,
                "githubLatestVersion": json.dumps(
                    {"version": "0.0.0", "fetchedAt": int(time.time() * 1000)}
                ),
            },
        )
        page.get_by_role("button", name="Logout").click()

        expect(page.get_by_role("button", name="Sign in", exact=True)).to_be_visible()

        remaining_storage_keys = page.evaluate("() => Object.keys(localStorage).sort()")

        assert remaining_storage_keys == sorted(
            [*PREFERENCES_SURVIVING_SIGN_OUT, "githubLatestVersion"]
        )

        cache = read_browser_storage(page, "githubLatestVersion")
        assert cache is not None
        public_version_cache = get_api_object(json.loads(cache))
        assert isinstance(public_version_cache["version"], str)
        assert isinstance(public_version_cache["fetchedAt"], int)

        surviving_preferences = page.evaluate(
            "(keys) => Object.fromEntries(keys.map(key => [key, localStorage.getItem(key)]))",
            list(PREFERENCES_SURVIVING_SIGN_OUT),
        )

        assert surviving_preferences == PREFERENCES_SURVIVING_SIGN_OUT
