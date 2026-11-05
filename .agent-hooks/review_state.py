from typing import TypedDict, TypeGuard


class Finding(TypedDict):
    category: str
    file: str
    summary: str


class QuickReviewResult(TypedDict):
    ok: bool
    reason: str


class HookState(TypedDict, total=False):
    baseline: str
    baseline_paths: list[str]
    baseline_files: dict[str, str]
    openspec_files: dict[str, str]
    prompt: str
    blocks: int
    accepted: str
    findings: list[Finding]
    reviewed_files: dict[str, str]


def is_string_map(value: object) -> TypeGuard[dict[str, str]]:
    return isinstance(value, dict) and all(
        isinstance(key, str) and isinstance(item, str) for key, item in value.items()
    )


def is_finding(value: object) -> TypeGuard[Finding]:
    return isinstance(value, dict) and all(
        isinstance(value.get(key), str) for key in ("category", "file", "summary")
    )


def is_hook_state(value: object) -> TypeGuard[HookState]:
    if not isinstance(value, dict):
        return False
    for key, item in value.items():
        if key in ("baseline", "prompt", "accepted") and not isinstance(item, str):
            return False
        if key == "blocks" and not isinstance(item, int):
            return False
        if key in ("baseline_files", "openspec_files", "reviewed_files") and not is_string_map(
            item
        ):
            return False
        if key == "baseline_paths" and (
            not isinstance(item, list) or not all(isinstance(path, str) for path in item)
        ):
            return False
        if key == "findings" and (
            not isinstance(item, list) or not all(is_finding(finding) for finding in item)
        ):
            return False
    return True
