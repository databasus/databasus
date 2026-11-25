import os
from collections.abc import Iterator

import pytest
from application_screens import Account, open_application, register_first_account
from playwright.sync_api import Browser, Page, expect, sync_playwright

SCREEN_MILLISECONDS = 15_000


@pytest.fixture(scope="session")
def browser() -> Iterator[Browser]:
    expect.set_options(timeout=SCREEN_MILLISECONDS)

    with sync_playwright() as playwright:
        connected_browser = playwright.chromium.connect(os.environ["E2E_BROWSER_ENDPOINT"])
        yield connected_browser
        connected_browser.close()


def _new_page(browser: Browser, base_url: str | None = None) -> Page:
    context = browser.new_context(
        base_url=base_url or os.environ["E2E_BASE_URL"],
        locale="en-US",
        timezone_id="UTC",
        viewport={"width": 1280, "height": 900},
    )
    context.set_default_timeout(SCREEN_MILLISECONDS)

    return context.new_page()


@pytest.fixture(scope="session")
def instance_admin(browser: Browser) -> Account:
    registering_page = _new_page(browser)
    account = Account(os.environ["E2E_EMAIL"], os.environ["E2E_PASSWORD"], is_root_admin=True)

    open_application(registering_page)
    register_first_account(registering_page, account)
    registering_page.context.close()

    return account


@pytest.fixture
def page(browser: Browser, instance_admin: Account) -> Iterator[Page]:
    scenario_page = _new_page(browser)

    yield scenario_page

    scenario_page.context.close()


@pytest.fixture
def member_page(browser: Browser, instance_admin: Account) -> Iterator[Page]:
    """A second browser of the same installation, signed in as nobody, for an account other than
    the administrator's."""
    member_scenario_page = _new_page(browser)

    yield member_scenario_page

    member_scenario_page.context.close()


@pytest.fixture(scope="session")
def mail_installation_admin(browser: Browser) -> Account:
    """Root admin of the installation that has a mail server."""
    registering_page = _new_page(browser, os.environ["E2E_MAIL_BASE_URL"])
    account = Account(os.environ["E2E_EMAIL"], os.environ["E2E_PASSWORD"], is_root_admin=True)

    open_application(registering_page)
    register_first_account(registering_page, account)
    registering_page.context.close()

    return account


@pytest.fixture
def mail_installation_page(browser: Browser, mail_installation_admin: Account) -> Iterator[Page]:
    mail_page = _new_page(browser, os.environ["E2E_MAIL_BASE_URL"])

    yield mail_page

    mail_page.context.close()


@pytest.fixture
def recovery_target_page(browser: Browser) -> Iterator[Page]:
    """A page of the separately initialized installation that a recovery replaces."""
    target_page = _new_page(browser, os.environ["E2E_TARGET_BASE_URL"])

    yield target_page

    target_page.context.close()
