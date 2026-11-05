import re
from urllib.parse import unquote

MASK = "***"

_RESTORE_TOKEN = re.compile(r"(?i)([?&]token=)[^\s&\"\'<>]+")

_BEARER_TOKEN = re.compile(r"(?i)(bearer\s+)\S+")


class OutputMask:
    def __init__(self, secrets: list[str]) -> None:
        self._secrets = [secret for secret in secrets if secret]

    def apply(self, line: str) -> str:
        for _ in range(3):
            decoded_line = unquote(line)
            if decoded_line == line:
                break
            line = decoded_line

        masked_line = _RESTORE_TOKEN.sub(rf"\g<1>{MASK}", line)
        masked_line = _BEARER_TOKEN.sub(rf"\g<1>{MASK}", masked_line)

        for secret in self._secrets:
            masked_line = masked_line.replace(secret, MASK)

        return masked_line
