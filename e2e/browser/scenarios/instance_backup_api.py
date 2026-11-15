import base64
import json

from api_response import get_api_object
from playwright.sync_api import Page


def call_api(page: Page, method: str, path: str, body: object = None) -> dict[str, object]:
    """Calls the API inside the page, so the session token never reaches Python or its output."""
    response: object = page.evaluate(
        """async ({ method, path, body }) => {
            const token = localStorage.getItem('databasus_user_token');
            const response = await fetch(path, {
                method,
                headers: {
                    Authorization: `Bearer ${token}`,
                    ...(body === null ? {} : { 'Content-Type': 'application/json' }),
                },
                body: body === null ? undefined : JSON.stringify(body),
            });
            return { isOk: response.ok, status: response.status, body: await response.text() };
        }""",
        {"method": method, "path": path, "body": body},
    )
    assert isinstance(response, dict)
    assert response.get("isOk") is True, f"{method} {path} answered {response.get('status')}"
    text = response.get("body")
    assert isinstance(text, str)

    return get_api_object(json.loads(text)) if text else {}


def download_bytes(page: Page, path: str) -> bytes:
    """Follows a single-use download link inside the page and returns the exact file bytes."""
    encoded: object = page.evaluate(
        """async (path) => {
            const response = await fetch(path);
            if (!response.ok) throw new Error(`download answered ${response.status}`);
            const bytes = new Uint8Array(await response.arrayBuffer());
            let binary = '';
            for (const byte of bytes) binary += String.fromCharCode(byte);
            return btoa(binary);
        }""",
        path,
    )
    assert isinstance(encoded, str)

    return base64.b64decode(encoded)
