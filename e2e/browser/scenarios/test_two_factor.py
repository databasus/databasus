import uuid
from mailbox import read_latest_sign_in_code

from application_screens import (
    Account,
    application_banner,
    open_application,
    read_api,
    sign_in,
    write_api,
)
from authenticator_code import compute_authenticator_code
from playwright.sync_api import Page, expect
from second_factor_api import assert_password_alone_issues_no_session

INSTANCE_SETTINGS_PATH = "/api/v1/instance-settings"
EDITABLE_SETTINGS = (
    "isAllowExternalRegistrations",
    "isAllowMemberInvitations",
    "isMemberAllowedToCreateWorkspaces",
    "twoFactorRequirement",
)
PASSWORD_PLACEHOLDER = "********"


class TestTwoFactor:
    def test_requires_the_emailed_code_through_the_screens_and_the_api(
        self, mail_installation_page: Page, mail_installation_admin: Account
    ) -> None:
        page = mail_installation_page
        open_application(page)
        sign_in(page, mail_installation_admin)
        _require_second_factor(page, method="Code sent by email")
        _sign_out(page)

        assert_password_alone_issues_no_session(
            page, mail_installation_admin, expected_outcome="CODE_REQUIRED", verify_action="verify"
        )

        _enter_password(page, mail_installation_admin)
        expect(page.get_by_text("Enter the code", exact=True)).to_be_visible()
        _assert_application_stays_closed(page)

        # The repeated password step hands back the live code instead of mailing a new one.
        _enter_password(page, mail_installation_admin)
        page.get_by_label("Sign-in code").fill(
            read_latest_sign_in_code(page, mail_installation_admin.email)
        )
        page.get_by_role("button", name="Sign in", exact=True).click()

        expect(application_banner(page)).to_be_visible()

    def test_requires_an_authenticator_app_through_the_screens_and_the_api(
        self, page: Page, member_page: Page, instance_admin: Account
    ) -> None:
        open_application(page)
        sign_in(page, instance_admin)
        # The other scenarios of the run sign in with a password alone, so the instance is put back.
        original_settings = read_api(page, INSTANCE_SETTINGS_PATH)

        try:
            _require_second_factor(page, method="Authenticator app", is_registration_opened=True)
            # An administrator without an app is taken to connect one right away.
            expect(page.get_by_role("dialog", name="Connect an authenticator app")).to_be_visible()

            member = Account(f"member-{uuid.uuid4()}@example.com", "member-password")
            open_application(member_page)
            _sign_up(member_page, member)
            _sign_out(member_page)

            assert_password_alone_issues_no_session(
                member_page,
                member,
                expected_outcome="AUTHENTICATOR_SETUP_REQUIRED",
                verify_action="confirm-authenticator-app-setup",
            )
            _enter_password(member_page, member)
            expect(
                member_page.get_by_text("Connect an authenticator app", exact=True)
            ).to_be_visible()
            _assert_application_stays_closed(member_page)

            _enter_password(member_page, member)
            secret, recovery_codes = _complete_authenticator_setup(member_page)
            _sign_out(member_page)

            assert_password_alone_issues_no_session(
                member_page,
                member,
                expected_outcome="AUTHENTICATOR_CODE_REQUIRED",
                verify_action="verify",
            )
            _enter_password(member_page, member)
            expect(member_page.get_by_text("Enter the app code", exact=True)).to_be_visible()
            _assert_application_stays_closed(member_page)

            _enter_password(member_page, member)
            # The setup spent the code of the current step, and the instance also accepts the next.
            _enter_second_factor(
                member_page, "Code from the app", compute_authenticator_code(secret, step_offset=1)
            )
            _sign_out(member_page)

            _enter_password(member_page, member)
            member_page.get_by_role("button", name="Use a recovery code instead").click()
            _enter_second_factor(member_page, "Recovery code", recovery_codes[0])
        finally:
            write_api(
                page,
                "PUT",
                INSTANCE_SETTINGS_PATH,
                {key: original_settings[key] for key in EDITABLE_SETTINGS},
            )


def _require_second_factor(page: Page, method: str, is_registration_opened: bool = False) -> None:
    page.get_by_role("link", name="Databasus settings", exact=True).click()
    if is_registration_opened:
        page.get_by_role("switch", name="Allow external registrations").check()

    page.get_by_role("switch", name="Require 2FA").check()
    page.get_by_role("radio", name=method).check()
    page.get_by_role("button", name="Save Changes").click()

    expect(page.get_by_text("Settings updated successfully")).to_be_visible()


def _assert_application_stays_closed(page: Page) -> None:
    page.goto("/databasus-settings")

    expect(page.get_by_placeholder(PASSWORD_PLACEHOLDER, exact=True)).to_be_visible()
    expect(application_banner(page)).to_be_hidden()


def _sign_up(page: Page, account: Account) -> None:
    page.get_by_role("button", name="Sign up", exact=True).click()
    page.get_by_label("Your name").fill("Member")
    page.get_by_label("Your email").fill(account.email)
    page.get_by_label("Password", exact=True).fill(account.password)
    page.get_by_label("Confirm password").fill(account.password)
    page.get_by_role("button", name="Sign up", exact=True).click()

    expect(application_banner(page)).to_be_visible()


def _sign_out(page: Page) -> None:
    page.get_by_role("link", name="Profile", exact=True).click()
    page.get_by_role("button", name="Logout").click()

    expect(page.get_by_placeholder(PASSWORD_PLACEHOLDER, exact=True)).to_be_visible()


def _enter_password(page: Page, account: Account) -> None:
    page.get_by_placeholder("your@email.com").fill(account.email)
    page.get_by_placeholder(PASSWORD_PLACEHOLDER, exact=True).fill(account.password)
    page.get_by_role("button", name="Sign in", exact=True).click()


def _complete_authenticator_setup(page: Page) -> tuple[str, list[str]]:
    """Plays the phone: reads the secret shown as text, answers with the code it computes, and
    saves the recovery codes the instance shows once."""
    expect(page.get_by_text("Connect an authenticator app", exact=True)).to_be_visible()
    expect(
        page.get_by_role("img", name="QR code to scan with an authenticator app")
    ).to_be_visible()
    secret = page.locator("form code").inner_text().strip()

    page.get_by_label("Code from the app").fill(compute_authenticator_code(secret))
    page.get_by_role("button", name="Confirm", exact=True).click()

    recovery_code_list = page.get_by_role("list", name="Save your recovery codes")
    expect(recovery_code_list.get_by_role("listitem")).to_have_count(10)
    recovery_codes = [
        code.strip() for code in recovery_code_list.get_by_role("listitem").all_inner_texts()
    ]
    expect(application_banner(page)).to_be_hidden()

    page.get_by_role("button", name="I've saved the recovery codes").click()
    expect(application_banner(page)).to_be_visible()

    return secret, recovery_codes


def _enter_second_factor(page: Page, field_label: str, code: str) -> None:
    expect(page.get_by_text("Enter the app code", exact=True)).to_be_visible()
    expect(page.get_by_role("button", name="Send another code")).to_have_count(0)

    page.get_by_label(field_label).fill(code)
    page.get_by_role("button", name="Sign in", exact=True).click()

    expect(application_banner(page)).to_be_visible()
