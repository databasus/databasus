from typing import TypedDict, TypeGuard


class ReviewerCall(TypedDict):
    name: str
    arguments: list[str]
    cwd: str
    prompt: str
    effort_env: str | None
    hook_active: str | None
    session_env: str | None
    answer_mode: str | None
    started_at: int


class MiseCall(TypedDict):
    subcommand: str
    arguments: list[str]
    exec_auto_install: str | None
    stdin: str


def is_reviewer_call(value: object) -> TypeGuard[ReviewerCall]:
    if not isinstance(value, dict):
        return False
    arguments = value.get("arguments")
    return (
        all(isinstance(value.get(key), str) for key in ("name", "cwd", "prompt"))
        and isinstance(arguments, list)
        and all(isinstance(argument, str) for argument in arguments)
        and isinstance(value.get("started_at"), int)
        and all(
            key in value and (value[key] is None or isinstance(value[key], str))
            for key in ("effort_env", "hook_active", "session_env", "answer_mode")
        )
    )


def is_mise_call(value: object) -> TypeGuard[MiseCall]:
    if not isinstance(value, dict):
        return False
    arguments = value.get("arguments")
    return (
        isinstance(value.get("subcommand"), str)
        and isinstance(value.get("stdin"), str)
        and isinstance(arguments, list)
        and all(isinstance(argument, str) for argument in arguments)
        and (
            "exec_auto_install" in value
            and (value["exec_auto_install"] is None or isinstance(value["exec_auto_install"], str))
        )
    )
