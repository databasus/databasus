from typing import TypedDict

from review_state import is_string_map

HookResponse = TypedDict(
    "HookResponse",
    {
        "continue": bool,
        "decision": str,
        "reason": str,
        "stopReason": str,
        "systemMessage": str,
        "hookSpecificOutput": dict[str, str],
    },
    total=False,
)


def get_hook_response(value: object) -> HookResponse:
    if not isinstance(value, dict):
        raise ValueError("The hook response must be an object")
    response: HookResponse = {}
    if "continue" in value:
        should_continue = value["continue"]
        if not isinstance(should_continue, bool):
            raise ValueError("The hook continuation must be a boolean")
        response["continue"] = should_continue
    for field in ("decision", "reason", "stopReason", "systemMessage"):
        if field in value:
            text = value[field]
            if not isinstance(text, str):
                raise ValueError(f"The hook {field} must be text")
            response[field] = text
    if "hookSpecificOutput" in value:
        output = value["hookSpecificOutput"]
        if not is_string_map(output):
            raise ValueError("The hook output must contain string fields")
        response["hookSpecificOutput"] = output
    return response
