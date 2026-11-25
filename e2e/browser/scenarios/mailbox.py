import os
import re

from api_response import get_api_object
from condition_wait import wait_until
from playwright.sync_api import Page

SIGN_IN_CODE_SUBJECT = "Sign-In Code"
MAILED_CODE_PATTERN = re.compile(r">(\d{6})<")


def read_latest_sign_in_code(page: Page, email: str) -> str:
    """Reads the code of the newest sign-in message to the address. The browser's own request
    context runs beside the mail server, which the test process cannot reach."""
    mail_server = os.environ["E2E_MAIL_SERVER_URL"]

    def read_messages() -> list[object]:
        response = page.context.request.get(
            f"{mail_server}/api/v1/search",
            params={"query": f'to:"{email}" subject:"{SIGN_IN_CODE_SUBJECT}"'},
        )
        assert response.ok, f"the mail server answered {response.status}"
        messages = get_api_object(response.json()).get("messages")
        assert isinstance(messages, list)
        return messages

    messages = wait_until("a sign-in code mailed to the account", read_messages, bool)
    message_id = get_api_object(messages[0]).get("ID")
    assert isinstance(message_id, str)

    message_response = page.context.request.get(f"{mail_server}/api/v1/message/{message_id}")
    assert message_response.ok, f"the mail server answered {message_response.status}"
    html = get_api_object(message_response.json()).get("HTML")
    assert isinstance(html, str)
    code_match = MAILED_CODE_PATTERN.search(html)
    assert code_match is not None, "the sign-in message carries no code"

    return code_match.group(1)
