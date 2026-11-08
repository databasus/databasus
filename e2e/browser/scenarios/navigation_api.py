import json
from dataclasses import dataclass

from api_response import get_api_object
from playwright.sync_api import Page


@dataclass(frozen=True)
class NavigationRequest:
    path: str
    body: dict[str, object] | None = None
    method: str = "POST"


def request_navigation_api(page: Page, request: NavigationRequest) -> dict[str, object]:
    response: object = page.evaluate(
        """async ({path, method, body}) => {
            const token = localStorage.getItem('databasus_user_token');
            const response = await fetch('/api/v1' + path, {
                method,
                headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
                body: body === null ? undefined : JSON.stringify(body),
            });
            const responseBody = response.status === 204 ? {} : await response.json();
            if (responseBody && typeof responseBody === 'object') delete responseBody.token;
            return { isOk: response.ok, status: response.status, body: JSON.stringify(responseBody) };
        }""",
        {"path": request.path, "method": request.method, "body": request.body},
    )
    parsed_response = get_api_object(response)
    assert parsed_response.get("isOk") is True, (
        f"navigation setup {request.method} {request.path} failed: {parsed_response.get('status')}"
    )
    body = parsed_response.get("body")
    assert isinstance(body, str)

    return get_api_object(json.loads(body))


def get_identity(response: dict[str, object]) -> str:
    identity = response.get("id")
    assert isinstance(identity, str)

    return identity
