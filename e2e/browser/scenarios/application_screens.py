import json
import re
from dataclasses import dataclass

from api_response import get_api_object
from playwright.sync_api import Page, expect

GITHUB_REPOSITORY_ADDRESS = "https://api.github.com/repos/databasus/databasus"
ADMINISTERS_NOTICE = "This is the first account on this instance, so it will administer it."


@dataclass(frozen=True)
class Account:
    email: str
    password: str


def open_application(page: Page) -> None:
    page.route(GITHUB_REPOSITORY_ADDRESS, lambda route: route.fulfill(json={"stargazers_count": 0}))
    page.route(
        f"{GITHUB_REPOSITORY_ADDRESS}/releases/latest",
        lambda route: route.fulfill(json={"tag_name": "dev"}),
    )
    page.goto("/")


def register_first_account(page: Page, account: Account) -> None:
    notice = page.get_by_text(ADMINISTERS_NOTICE)
    expect(notice).to_be_visible()

    page.get_by_label("Your name").fill("Instance Admin")
    page.get_by_label("Your email").fill(account.email)
    page.get_by_label("Password", exact=True).fill(account.password)
    page.get_by_label("Confirm password").fill(account.password)
    page.get_by_role("button", name="Sign up", exact=True).click()

    expect(page.get_by_role("banner")).to_be_visible()


def sign_in(page: Page, account: Account) -> None:
    page.get_by_placeholder("your@email.com").fill(account.email)
    page.get_by_placeholder("********", exact=True).fill(account.password)
    page.get_by_role("button", name="Sign in", exact=True).click()

    expect(page.get_by_role("banner")).to_be_visible()


def create_workspace(page: Page, name: str) -> None:
    workspace_selector = page.get_by_role("combobox", name="Selected workspace")
    create_workspace_button = page.get_by_role("banner").get_by_role(
        "button", name="Create workspace", exact=True
    )

    expect(workspace_selector.or_(create_workspace_button)).to_be_visible()

    if workspace_selector.is_visible():
        workspace_selector.focus()
        workspace_selector.press("ArrowDown")
        page.get_by_role("button", name="Create new workspace").click()
    else:
        create_workspace_button.click()

    dialog = page.get_by_role("dialog", name="Create workspace", exact=True)
    dialog.get_by_role("textbox", name="Workspace name", exact=True).fill(name)
    with page.expect_response(
        lambda response: (
            response.request.method == "POST" and response.url.endswith("/api/v1/workspaces")
        )
    ) as workspace_creation:
        dialog.get_by_role("button", name="Create workspace", exact=True).click()

    created_workspace = get_api_object(workspace_creation.value.json())
    workspace_id = created_workspace.get("id")
    assert isinstance(workspace_id, str)

    expect(dialog).not_to_be_visible()
    expect_selected_workspace(page, name)
    expect(page).to_have_url(re.compile(f"/workspaces/{re.escape(workspace_id)}/databases$"))
    page.wait_for_function(
        "(workspaceId) => localStorage.getItem('selected_workspace_id') === workspaceId",
        arg=workspace_id,
    )


def expect_selected_workspace(page: Page, name: str) -> None:
    selected_workspace = page.get_by_role("combobox", name="Selected workspace")
    expect(selected_workspace).to_contain_text(name)


def create_local_storage(page: Page, name: str) -> None:
    page.get_by_role("link", name="Storages", exact=True).click()
    page.get_by_role("button", name="Add storage", exact=True).click()

    dialog = page.get_by_role("dialog", name="Add storage", exact=True)
    dialog.get_by_role("textbox", name="Name", exact=True).fill(name)
    dialog.get_by_role("button", name="Test connection", exact=True).click()
    dialog.get_by_role("button", name="Save", exact=True).click()

    expect(dialog).not_to_be_visible()
    expect(page.get_by_role("button", name=name, exact=True)).to_be_visible()


def read_api(page: Page, path: str) -> dict[str, object]:
    """Reads the API inside the page, so the session token never reaches Python and its failure output."""
    response: object = page.evaluate(
        """async (path) => {
            const token = localStorage.getItem('databasus_user_token');
            const response = await fetch(path, { headers: { Authorization: `Bearer ${token}` } });
            return { isOk: response.ok, status: response.status, body: await response.text() };
        }""",
        path,
    )
    assert isinstance(response, dict)
    assert isinstance(response.get("isOk"), bool)
    assert isinstance(response.get("status"), int)
    body = response.get("body")
    assert isinstance(body, str)
    assert response["isOk"], f"GET {path} answered {response['status']}"

    return get_api_object(json.loads(body))


def read_browser_storage(page: Page, key: str) -> str | None:
    value: object = page.evaluate("(key) => localStorage.getItem(key)", key)
    assert value is None or isinstance(value, str)
    return value
