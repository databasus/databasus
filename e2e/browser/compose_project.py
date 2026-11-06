import os
import subprocess
from dataclasses import dataclass
from pathlib import Path

import playwright
from supervisor_file import get_run_directory

PLAYWRIGHT_DRIVER_DIRECTORY = Path(playwright.__file__).parent / "driver"
COMPOSE_FILE = Path(__file__).parent / "compose.yaml"
PROJECT_LABEL = "com.docker.compose.project"
PROJECT_NAME_PREFIX = "databasus-e2e-"
POSTGRES_IMAGE = (
    "postgres:18@sha256:3a4264520e121176619f54ff867d12774bc81e95e7b84db58d63bccdb6144b81"
)


@dataclass(frozen=True)
class ComposeProject:
    name: str
    image: str
    database_password: str
    recovery_image: str = "unused"

    def get_recovery_archive(self) -> Path:
        return get_run_directory(self.name.removeprefix(PROJECT_NAME_PREFIX)) / "postgres.tar"

    def get_command(self, *arguments: str) -> list[str]:
        return [
            "docker",
            "compose",
            "--project-name",
            self.name,
            "--file",
            str(COMPOSE_FILE),
            *arguments,
        ]

    def get_environment(self) -> dict[str, str]:
        return {
            **os.environ,
            "E2E_IMAGE": self.image,
            "E2E_DATABASE_PASSWORD": self.database_password,
            "E2E_PLAYWRIGHT_DRIVER": str(PLAYWRIGHT_DRIVER_DIRECTORY),
            "E2E_POSTGRES_ARCHIVE": str(self.get_recovery_archive()),
            "E2E_POSTGRES_IMAGE": self.recovery_image,
        }

    def get_browser_port(self) -> str:
        completed = subprocess.run(
            self.get_command("port", "browser", "3000"),
            env=self.get_environment(),
            capture_output=True,
            text=True,
            check=True,
        )

        return completed.stdout.strip().rpartition(":")[2]


def list_remaining_resources(project_name: str) -> list[str]:
    label_filter = f"label={PROJECT_LABEL}={project_name}"
    listings = {
        "container": ["docker", "ps", "--all", "--format", "{{.Names}}", "--filter", label_filter],
        "volume": ["docker", "volume", "ls", "--format", "{{.Name}}", "--filter", label_filter],
        "network": ["docker", "network", "ls", "--format", "{{.Name}}", "--filter", label_filter],
    }
    remaining_resources = []

    for resource_kind, listing in listings.items():
        completed = subprocess.run(listing, capture_output=True, text=True, check=True)
        remaining_resources += [f"{resource_kind} {name}" for name in completed.stdout.split()]

    return remaining_resources


def list_project_names() -> list[str]:
    completed = subprocess.run(
        ["docker", "ps", "--all", "--format", f'{{{{.Label "{PROJECT_LABEL}"}}}}'],
        capture_output=True,
        text=True,
        check=True,
    )
    labelled_projects = {
        name for name in completed.stdout.split() if name.startswith(PROJECT_NAME_PREFIX)
    }

    return sorted(labelled_projects)


def remove_dead_run_project(project_name: str) -> bool:
    # A dead run's image and password are gone, and compose only needs its variables to be set.
    project = ComposeProject(project_name, "unused", "unused")
    completed = subprocess.run(
        project.get_command("down", "--volumes", "--remove-orphans", "--timeout", "5"),
        env=project.get_environment(),
        capture_output=True,
    )

    return completed.returncode == 0
