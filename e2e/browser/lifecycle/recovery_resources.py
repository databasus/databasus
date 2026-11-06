import subprocess


def assert_private_recovery_is_usable(project_name: str) -> None:
    containers = subprocess.run(
        ["docker", "exec", f"{project_name}-recovery-1", "docker", "ps", "--quiet"],
        capture_output=True,
        text=True,
        check=True,
        timeout=30,
    ).stdout.split()
    assert containers, f"{project_name} lost its nested recovery container"
    marker = subprocess.run(
        [
            "docker",
            "exec",
            f"{project_name}-recovery-1",
            "docker",
            "exec",
            containers[0],
            "cat",
            "/restore-work/probe marker",
        ],
        capture_output=True,
        text=True,
        check=True,
        timeout=30,
    ).stdout.strip()

    assert marker == "matching-bind-path", f"{project_name} lost its recovery data: {marker}"
