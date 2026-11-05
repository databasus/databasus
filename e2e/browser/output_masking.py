import re

MASK = "***"

_BEARER_TOKEN = re.compile(r"(?i)(bearer\s+)\S+")


class OutputMask:
    def __init__(self, secrets: list[str]) -> None:
        self._secrets = [secret for secret in secrets if secret]

    def apply(self, line: str) -> str:
        masked_line = _BEARER_TOKEN.sub(rf"\g<1>{MASK}", line)

        for secret in self._secrets:
            masked_line = masked_line.replace(secret, MASK)

        return masked_line
