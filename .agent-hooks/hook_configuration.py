from typing import NotRequired, TypedDict, TypeGuard


class HookDefinition(TypedDict):
    command: str
    type: str
    timeout: NotRequired[int]
    statusMessage: NotRequired[str]


class HookGroup(TypedDict):
    hooks: list[HookDefinition]
    matcher: NotRequired[str]


def is_hook_definition(value: object) -> TypeGuard[HookDefinition]:
    return (
        isinstance(value, dict)
        and isinstance(value.get("command"), str)
        and isinstance(value.get("type"), str)
        and ("timeout" not in value or isinstance(value["timeout"], int))
        and ("statusMessage" not in value or isinstance(value["statusMessage"], str))
    )


def is_hook_group(value: object) -> TypeGuard[HookGroup]:
    if not isinstance(value, dict):
        return False
    hooks = value.get("hooks")
    return (
        isinstance(hooks, list)
        and all(is_hook_definition(hook) for hook in hooks)
        and ("matcher" not in value or isinstance(value["matcher"], str))
    )


def get_hook_configuration(value: object) -> dict[str, list[HookGroup]]:
    if not isinstance(value, dict):
        raise ValueError("Hook configuration must be an object")
    configuration: dict[str, list[HookGroup]] = {}
    for event, groups in value.items():
        if not isinstance(event, str) or not isinstance(groups, list):
            raise ValueError("Hook events must contain groups")
        parsed_groups = []
        for group in groups:
            if not is_hook_group(group):
                raise ValueError("Hook groups must contain command definitions")
            parsed_groups.append(group)
        configuration[event] = parsed_groups
    return configuration
