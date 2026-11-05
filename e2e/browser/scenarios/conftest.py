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


def _new_page(browser: Browser) -> Page:
    context = browser.new_context(
        base_url=os.environ["E2E_BASE_URL"],
        locale="en-US",
        timezone_id="UTC",
        viewport={"width": 1280, "height": 900},
    )
    context.set_default_timeout(SCREEN_MILLISECONDS)

    return context.new_page()


@pytest.fixture(scope="session")
def instance_admin(browser: Browser) -> Account:
    registering_page = _new_page(browser)
    account = Account(os.environ["E2E_EMAIL"], os.environ["E2E_PASSWORD"])

    open_application(registering_page)
    register_first_account(registering_page, account)
    registering_page.context.close()

    return account


@pytest.fixture
def page(browser: Browser, instance_admin: Account) -> Iterator[Page]:
    scenario_page = _new_page(browser)

    yield scenario_page

    scenario_page.context.close()
