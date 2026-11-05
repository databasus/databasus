import subprocess
from collections.abc import Callable

from compose_project import list_remaining_resources


def assert_nothing_of_the_project_remains(
    project_name: str,
    list_resources: Callable[[str], list[str]] = list_remaining_resources,
) -> None:
    remaining_resources = list_resources(project_name)

    assert not remaining_resources, f"{project_name} left behind: {', '.join(remaining_resources)}"


def is_image_present(image: str) -> bool:
    return (
        subprocess.run(["docker", "image", "inspect", image], capture_output=True).returncode == 0
    )
