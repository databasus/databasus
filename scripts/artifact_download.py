import hashlib
import re
from pathlib import Path
from urllib.parse import urlsplit
from urllib.request import urlopen


def download_verified_artifact(url: str, destination: Path, sha256: str) -> None:
    if not re.fullmatch(r"[a-f0-9]{64}", sha256):
        raise ValueError("artifact download requires an expected SHA256")

    source = urlsplit(url)
    if (
        source.username
        or source.password
        or source.scheme not in {"http", "https"}
        or (source.scheme == "http" and source.hostname not in {"127.0.0.1", "::1"})
    ):
        raise ValueError("artifact download requires a credential-free HTTPS source")

    checksum = hashlib.sha256()
    try:
        # The scheme allowlist above rejects local files; urllib also rejects redirects to file URLs.
        # nosemgrep: python.lang.security.audit.dynamic-urllib-use-detected.dynamic-urllib-use-detected
        with urlopen(url, timeout=60) as response, destination.open("wb") as stream:
            while chunk := response.read(1024 * 1024):
                checksum.update(chunk)
                stream.write(chunk)

        if checksum.hexdigest() != sha256:
            raise ValueError("downloaded artifact SHA256 mismatch")
    except Exception:
        destination.unlink(missing_ok=True)
        raise
