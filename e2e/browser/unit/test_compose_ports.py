import json
import subprocess

from compose_project import COMPOSE_FILE, ComposeProject


def test_only_the_browser_publishes_a_port():
    completed = subprocess.run(
        ["docker", "compose", "--file", str(COMPOSE_FILE), "config", "--format", "json"],
        env=ComposeProject("ports-check", "image", "password").get_environment(),
        capture_output=True,
        text=True,
        check=True,
    )
    services = json.loads(completed.stdout)["services"]

    publishing_services = sorted(name for name, service in services.items() if service.get("ports"))

    assert publishing_services == ["browser"]
