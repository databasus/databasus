import json
from dataclasses import dataclass

from api_response import get_api_object
from application_screens import Account
from playwright.sync_api import Page

SIGN_IN_PATH = "/api/v1/sessions"
PROTECTED_PATH = "/api/v1/workspaces"
WRONG_CODE = "000000"


@dataclass(frozen=True)
class ApiAnswer:
    status: int
    body: dict[str, object]


def send_without_session(
    page: Page, method: str, path: str, body: dict[str, object] | None = None
) -> ApiAnswer:
    """Calls the API inside the page with no session token, as a client that knows only the
    password does."""
    response: object = page.evaluate(
        """async ({ method, path, body }) => {
            const response = await fetch(path, {
                method,
                headers: body === null ? {} : { 'Content-Type': 'application/json' },
                body: body === null ? undefined : JSON.stringify(body),
            });
            return { status: response.status, body: await response.text() };
        }""",
        {"method": method, "path": path, "body": body},
    )
    answer = get_api_object(response)
    status = answer.get("status")
    text = answer.get("body")
    assert isinstance(status, int)
    assert isinstance(text, str)

    return ApiAnswer(status, get_api_object(json.loads(text)) if text else {})


def assert_password_alone_issues_no_session(
    page: Page, account: Account, expected_outcome: str, verify_action: str
) -> None:
    password_answer = send_without_session(
        page, "POST", SIGN_IN_PATH, {"email": account.email, "password": account.password}
    )

    assert password_answer.status == 200, password_answer.status
    assert password_answer.body.get("outcome") == expected_outcome, password_answer.body
    assert "token" not in password_answer.body
    pending_sign_in_id = password_answer.body.get("pendingSignInId")
    assert isinstance(pending_sign_in_id, str)

    wrong_code_answer = send_without_session(
        page,
        "POST",
        f"/api/v1/pending-sign-ins/{pending_sign_in_id}/{verify_action}",
        {"code": WRONG_CODE},
    )

    assert wrong_code_answer.status == 400, wrong_code_answer.body
    assert "token" not in wrong_code_answer.body
    assert send_without_session(page, "GET", PROTECTED_PATH).status == 401
