import subprocess
from dataclasses import dataclass

LOG_LIMIT_SECONDS = 30


@dataclass(frozen=True)
class ContainerLogSource:
    name: str
    command: list[str]
    environment: dict[str, str] | None = None


def get_container_log_output(source: ContainerLogSource) -> str:
    try:
        logs = subprocess.run(
            source.command,
            env=source.environment,
            capture_output=True,
            text=True,
            errors="replace",
            timeout=LOG_LIMIT_SECONDS,
        )

        return logs.stdout + logs.stderr
    except subprocess.TimeoutExpired as failure:
        partial_output = _decode_output(failure.stdout) + _decode_output(failure.stderr)

        return (
            partial_output
            + f"\nE2E {source.name} log collection timed out after {LOG_LIMIT_SECONDS} seconds\n"
        )
    except Exception as failure:
        return f"\nE2E could not read {source.name} logs: {failure}\n"


def _decode_output(output: str | bytes | None) -> str:
    if isinstance(output, bytes):
        return output.decode(errors="replace")

    return output or ""
