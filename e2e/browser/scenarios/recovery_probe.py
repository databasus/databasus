from condition_wait import wait_until
from recovery_environment import PageCommand, execute_page_command, recovery_exec


def prepare_recovery_probe() -> None:
    recovery_exec("curl", "--fail", "--silent", "http://databasus:4005/")
    image = recovery_exec("printenv", "E2E_POSTGRES_IMAGE").stdout.strip()
    execute_page_command(
        PageCommand(text="echo 'matching-bind-path' > 'probe marker'", stage="probe file")
    )
    nested_marker = recovery_exec(
        "docker",
        "run",
        "--detach",
        "--volume",
        "/restore-work:/restore-work",
        image,
        "sh",
        "-c",
        "cat '/restore-work/probe marker'; exec sleep 600",
    ).stdout.strip()
    wait_until(
        "nested bind mount marker",
        lambda: recovery_exec("docker", "logs", nested_marker).stdout.strip(),
        lambda marker: marker == "matching-bind-path",
    )
    assert recovery_exec("sh", "-c", "command -v pg_ctl", should_check=False).returncode != 0
    assert (
        execute_page_command(
            PageCommand(text="printf '%s' 'stdin-command-probe'", stage="probe stdin")
        )
        == "stdin-command-probe"
    )
    print("E2E_RECOVERY_READY", flush=True)
