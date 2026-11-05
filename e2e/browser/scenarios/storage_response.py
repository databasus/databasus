from typing import TypedDict, TypeGuard


class StorageSettings(TypedDict):
    type: str


class Storage(TypedDict):
    id: str
    name: str
    workspaceId: str
    settings: StorageSettings


def is_storage(value: object) -> TypeGuard[Storage]:
    if not isinstance(value, dict):
        return False
    settings = value.get("settings")
    return (
        all(isinstance(value.get(key), str) for key in ("id", "name", "workspaceId"))
        and isinstance(settings, dict)
        and isinstance(settings.get("type"), str)
    )


def get_storage(value: object) -> Storage:
    assert is_storage(value), "Invalid storage response"
    return value


def get_storages(response: dict[str, object]) -> list[Storage]:
    items = response.get("items")
    assert isinstance(items, list)
    return [get_storage(item) for item in items]
