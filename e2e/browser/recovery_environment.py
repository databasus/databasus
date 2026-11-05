import os
import subprocess
import uuid
from dataclasses import dataclass

from output_masking import OutputMask

HOST_PATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
COMMAND_LIMIT_SECONDS = 300


@dataclass(frozen=True)
class PageCommand:
    text: str
    stage: str
    user: str = "postgres"


def get_recovery_container() -> str:
    return f"{os.environ['E2E_COMPOSE_PROJECT']}-recovery-1"


def recovery_exec(
    *arguments: str, user: str = "postgres", should_check: bool = True
) -> subprocess.CompletedProcess[str]:
    completed = subprocess.run(
        [
            "docker",
            "exec",
            "--user",
            user,
            "--env",
            f"PATH={HOST_PATH}",
            get_recovery_container(),
            *arguments,
        ],
        capture_output=True,
        text=True,
        timeout=COMMAND_LIMIT_SECONDS,
    )

    if should_check:
        assert completed.returncode == 0, OutputMask([]).apply(completed.stdout + completed.stderr)

    return completed


@dataclass
class RecoveryCommand:
    process: subprocess.Popen[str]
    log_path: str
    stage: str

    def finish(self, *, is_success_required: bool = True) -> str:
        try:
            exit_code = self.process.wait(timeout=COMMAND_LIMIT_SECONDS + 10)
        except subprocess.TimeoutExpired:
            self.process.terminate()
            self.process.wait(timeout=10)
            raise AssertionError(f"{self.stage} exceeded its command limit") from None

        output = recovery_exec("cat", self.log_path, user="root").stdout
        masked_output = OutputMask([]).apply(output)

        if is_success_required:
            assert exit_code == 0, f"{self.stage} exited {exit_code}:\n{masked_output}"
        else:
            assert exit_code != 0, f"{self.stage} unexpectedly succeeded"

        return masked_output


def start_page_command(command: PageCommand) -> RecoveryCommand:
    log_path = f"/restore-work/command-{uuid.uuid4().hex}.log"
    process = subprocess.Popen(
        [
            "docker",
            "exec",
            "--interactive",
            "--user",
            command.user,
            "--workdir",
            "/restore-work",
            "--env",
            f"PATH={HOST_PATH}",
            get_recovery_container(),
            "sh",
            "-c",
            'exec timeout --kill-after=5 "$2" sh -s > "$1" 2>&1',
            "recovery-command",
            log_path,
            str(COMMAND_LIMIT_SECONDS),
        ],
        stdin=subprocess.PIPE,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    assert process.stdin is not None
    process.stdin.write(command.text)
    process.stdin.close()

    return RecoveryCommand(process, log_path, command.stage)


def execute_page_command(command: PageCommand) -> str:
    return start_page_command(command).finish()
